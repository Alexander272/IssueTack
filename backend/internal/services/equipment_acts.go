package services

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

// GetActs возвращает список актов реалма.
func (s *EquipmentService) GetActs(ctx context.Context, filter *models.EquipmentActsFilter) ([]*models.EquipmentAct, int, error) {
	data, total, err := s.repo.GetActs(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get acts. error: %w", err)
	}
	return data, total, nil
}

// GetActByID возвращает акт по идентификатору.
func (s *EquipmentService) GetActByID(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, error) {
	data, err := s.repo.GetActByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get act. error: %w", err)
	}
	return data, nil
}

// CreateAct формирует акт вручную по текущему составу комплекта либо по единице.
func (s *EquipmentService) CreateAct(ctx context.Context, dto *models.EquipmentActDTO) (*models.EquipmentAct, error) {
	var affected []*models.Equipment
	if dto.KitEquipmentID != nil {
		items, err := s.collectSubtree(ctx, dto.RealmID, *dto.KitEquipmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to collect equipment subtree. error: %w", err)
		}
		affected = items
	}
	if len(affected) == 0 {
		return nil, models.ErrEquipmentActGeneration
	}

	var actID uuid.UUID
	err := s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		act, err := s.buildAct(ctx, tx, dto, affected)
		if err != nil {
			return err
		}
		actID = act.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetActByID(ctx, &models.GetEquipmentActByIDDTO{ID: actID, RealmID: dto.RealmID})
}

// createActTx формирует акт выдачи в существующей транзакции.
func (s *EquipmentService) createActTx(ctx context.Context, tx postgres.Tx, dto *models.EquipmentActDTO, affected []*models.Equipment) error {
	_, err := s.buildAct(ctx, tx, dto, affected)
	return err
}

// buildAct собирает и сохраняет акт со снимком состава.
func (s *EquipmentService) buildAct(ctx context.Context, tx postgres.Tx, dto *models.EquipmentActDTO, affected []*models.Equipment) (*models.EquipmentAct, error) {
	content, err := actContent(affected)
	if err != nil {
		return nil, fmt.Errorf("failed to build act content. error: %w", err)
	}

	seq, err := s.repo.NextActSeq(ctx, tx, dto.RealmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get act seq. error: %w", err)
	}
	number := fmt.Sprintf("АКТ-%d-%04d", time.Now().Year(), seq)

	// Дата акта: момент выдачи из формы, иначе — сейчас при ручном создании
	// акта.
	issuedAt := time.Now()
	if dto.IssuedAt != nil {
		issuedAt = *dto.IssuedAt
	}

	// Снимок ФИО получателя приходит из формы: и выдача, и ручное формирование
	// акта кладут в recipientName выбранного сотрудника или свободное ФИО
	// внешнего. Резолвить имя по recipientUserId не нужно — это лишний запрос к
	// БД, когда подпись уже есть в теле запроса.
	act := &models.EquipmentAct{
		RealmID:         dto.RealmID,
		Number:          number,
		Status:          models.ActStatusFormed,
		KitEquipmentID:  dto.KitEquipmentID,
		RecipientUserID: dto.RecipientUserID,
		RecipientName:   dto.RecipientName,
		SiteID:          dto.SiteID,
		Location:        dto.Location,
		IssuedBy:        actorID(dto.Actor),
		IssuedAt:        issuedAt,
		Content:         content,
		Comment:         dto.Comment,
	}
	if err := s.repo.CreateAct(ctx, tx, act); err != nil {
		return nil, fmt.Errorf("failed to create act. error: %w", err)
	}
	return act, nil
}

// actPosition — позиция снимка состава в акте.
type actPosition struct {
	EquipmentID     uuid.UUID `json:"equipmentId"`
	TypeName        string    `json:"typeName"`
	Name            string    `json:"name"`
	SerialNumber    *string   `json:"serialNumber"`
	InventoryNumber *string   `json:"inventoryNumber"`
}

// actPositions разбирает снимок состава акта.
func actPositions(content []byte) ([]actPosition, error) {
	if len(content) == 0 {
		return []actPosition{}, nil
	}
	positions := []actPosition{}
	if err := json.Unmarshal(content, &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

// GetActDocument собирает docx акта по снимку состава: документ не хранится на
// диске, а собирается по запросу, поэтому правка шаблона не требует пересборки
// старых актов.
func (s *EquipmentService) GetActDocument(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentActDocument, error) {
	act, err := s.repo.GetActByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get act. error: %w", err)
	}

	positions, err := actPositions(act.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse act content. error: %w", err)
	}

	content, err := buildActDocx(act, positions, s.actDocParties(ctx, act))
	if err != nil {
		return nil, fmt.Errorf("failed to build act document. error: %w", err)
	}
	return &models.EquipmentActDocument{FileName: act.Number + ".docx", Content: content}, nil
}

// SignAct сохраняет скан (или фото) подписанного акта и переводит акт в статус
// «подписан». Файл пишется до записи в БД, а прежний удаляется после коммита —
// файловая система в транзакции не участвует.
func (s *EquipmentService) SignAct(ctx context.Context, dto *models.SignEquipmentActDTO) error {
	act, err := s.repo.GetActByID(ctx, &models.GetEquipmentActByIDDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get act. error: %w", err)
	}

	absPath, err := s.saveSignedActFile(dto)
	if err != nil {
		return err
	}
	dto.FilePath = absPath

	if err := s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		return s.repo.SetActSignedFile(ctx, tx, dto)
	}); err != nil {
		// Записи в БД нет — оставленный файл был бы сиротой.
		os.Remove(absPath)
		return err
	}

	if act.SignedFilePath != nil && *act.SignedFilePath != "" && *act.SignedFilePath != absPath {
		if err := os.Remove(*act.SignedFilePath); err != nil && !os.IsNotExist(err) {
			logger.Warn("failed to remove previous signed act file",
				logger.StringAttr("act_id", dto.ID.String()),
				logger.ErrAttr(err),
			)
		}
	}
	return nil
}

// GetActSignedFile отдаёт поток файла подписанного акта.
func (s *EquipmentService) GetActSignedFile(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, io.ReadCloser, error) {
	act, err := s.repo.GetActByID(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get act. error: %w", err)
	}
	if act.SignedFilePath == nil || *act.SignedFilePath == "" {
		return nil, nil, models.ErrEquipmentActNotSigned
	}

	f, err := os.Open(*act.SignedFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	return act, f, nil
}

// actDocParties — подписи сторон для документа. Recipient берётся из снимка имени
// акта (recipient_name), а по recipient_user_id резолвится только как fallback:
// у старых актов снимка нет. Выдавший резолвится по ссылке issued_by — у акта нет
// снимка его имени, а FK объявлен ON DELETE SET NULL. Ошибка резолва не срывает
// документ — акт остаётся выгружаемым, но без подписи, о сбое сообщает Warn-лог.
func (s *EquipmentService) actDocParties(ctx context.Context, act *models.EquipmentAct) actDocParties {
	parties := actDocParties{}
	if act.RecipientName != nil {
		parties.Recipient = *act.RecipientName
	}

	if act.RecipientUserID != nil && parties.Recipient == "" {
		if name, err := s.repo.GetUserFullName(ctx, *act.RecipientUserID); err != nil {
			logger.Warn("failed to resolve act recipient", logger.StringAttr("act_id", act.ID.String()), logger.ErrAttr(err))
		} else if name != "" {
			parties.Recipient = name
		}
	}
	if act.IssuedBy != nil {
		if name, err := s.repo.GetUserFullName(ctx, *act.IssuedBy); err != nil {
			logger.Warn("failed to resolve act issuer", logger.StringAttr("act_id", act.ID.String()), logger.ErrAttr(err))
		} else {
			parties.Issuer = name
		}
	}
	return parties
}

// actUploadDir — подкаталог подписанных актов внутри каталога загрузок.
const actUploadDir = "equipment_act"

// saveSignedActFile кладёт подписанный акт в uploads/equipment_act/<act_id>/.
// Раскладка и лимит — как у вложений заявок (см. AttachmentService.Upload).
func (s *EquipmentService) saveSignedActFile(dto *models.SignEquipmentActDTO) (string, error) {
	if dto.File == nil {
		return "", fmt.Errorf("%w: файл не передан", models.ErrInvalidInput)
	}
	if dto.FileName == "" {
		return "", fmt.Errorf("%w: не указано имя файла", models.ErrInvalidInput)
	}

	ext := filepath.Ext(dto.FileName)
	mimeType := dto.MimeType
	if mimeType == "" {
		mimeType = mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}
	dto.MimeType = mimeType

	base := filepath.Base(strings.TrimSuffix(filepath.Base(dto.FileName), ext))
	safeName := fmt.Sprintf("%s_%s%s", uuid.New().String(), base, ext)
	absPath := filepath.Join(s.conf.UploadDir, actUploadDir, dto.ID.String(), safeName)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	dst, err := os.Create(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	var written int64
	if s.conf.MaxSize > 0 {
		written, err = io.Copy(dst, io.LimitReader(dto.File, s.conf.MaxSize+1))
	} else {
		written, err = io.Copy(dst, dto.File)
	}
	if err != nil {
		dst.Close()
		os.Remove(absPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	if s.conf.MaxSize > 0 && written > s.conf.MaxSize {
		dst.Close()
		os.Remove(absPath)
		return "", fmt.Errorf("%w: максимум %d байт", models.ErrFileTooLarge, s.conf.MaxSize)
	}
	dst.Close()

	dto.FileSize = written
	return absPath, nil
}

// actContent формирует jsonb-снимок состава для акта.
func actContent(affected []*models.Equipment) ([]byte, error) {
	positions := make([]actPosition, 0, len(affected))
	for _, eq := range affected {
		positions = append(positions, actPosition{
			EquipmentID:     eq.ID,
			TypeName:        eq.TypeName,
			Name:            eq.Name,
			SerialNumber:    eq.SerialNumber,
			InventoryNumber: eq.InventoryNumber,
		})
	}
	data, err := json.Marshal(positions)
	if err != nil {
		return nil, err
	}
	return data, nil
}
