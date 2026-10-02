package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func attachmentFixtures(t *testing.T) (*MockAttachmentsRepo, *MockSubtaskService, *MockTicketAccessChecker, *AttachmentService) {
	t.Helper()
	mockRepo := new(MockAttachmentsRepo)
	mockSubtasks := new(MockSubtaskService)
	mockAccess := new(MockTicketAccessChecker)
	uploadDir := t.TempDir()

	svc := &AttachmentService{
		repo:         mockRepo,
		conf:         &config.FileServerConfig{UploadDir: uploadDir},
		ticketAccess: mockAccess,
		subtasks:     mockSubtasks,
	}
	return mockRepo, mockSubtasks, mockAccess, svc
}

func TestAttachmentService_GetByEntity_Success(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	expected := []*models.Attachment{{ID: uuid.New(), FileName: "file.pdf"}}

	dto := &models.EntityAccessDTO{EntityType: "ticket", EntityID: entityID, ActorID: actorID, Realm: ""}
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Action: string(access.Read)}).Return(nil)
	mockAccess.On("CheckInternalAssigneeAccess", mock.Anything, mock.AnythingOfType("*models.AccessCheckDTO")).Return(nil)
	mockRepo.On("GetByEntity", mock.Anything, "ticket", entityID).Return(expected, nil)

	got, err := svc.GetByEntity(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestAttachmentService_GetByEntity_NoAccess(t *testing.T) {
	_, _, _, svc := attachmentFixtures(t)
	svc.ticketAccess = nil

	dto := &models.EntityAccessDTO{EntityType: "ticket", EntityID: uuid.New(), ActorID: uuid.New()}
	_, err := svc.GetByEntity(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestAttachmentService_GetByEntity_AccessDenied(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	dto := &models.EntityAccessDTO{EntityType: "ticket", EntityID: entityID, ActorID: actorID, Realm: ""}
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Action: string(access.Read)}).Return(models.ErrPermissionDenied)

	_, err := svc.GetByEntity(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "GetByEntity")
}

func TestAttachmentService_GetByEntity_Subtask(t *testing.T) {
	mockRepo, mockSubtasks, mockAccess, svc := attachmentFixtures(t)

	subtaskID := uuid.New()
	ticketID := uuid.New()
	actorID := uuid.New()
	expected := []*models.Attachment{{ID: uuid.New(), FileName: "file.pdf"}}

	mockSubtasks.On("GetRawByID", mock.Anything, &models.GetSubtaskDTO{ID: subtaskID}).Return(&models.Subtask{
		ID: subtaskID, TicketID: ticketID,
	}, nil)
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: ticketID, UserID: actorID, Action: string(access.Read)}).Return(nil)
	mockRepo.On("GetByEntity", mock.Anything, "subtask", subtaskID).Return(expected, nil)

	dto := &models.EntityAccessDTO{EntityType: "subtask", EntityID: subtaskID, ActorID: actorID, Realm: ""}
	got, err := svc.GetByEntity(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestAttachmentService_Upload_InvalidEntityType(t *testing.T) {
	_, _, _, svc := attachmentFixtures(t)

	dto := &models.UploadAttachmentDTO{EntityType: "invalid", EntityID: uuid.New(), FileName: "test.txt", UploadedBy: uuid.New()}
	_, err := svc.Upload(context.Background(), nil, dto)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid entity type")
}

func TestAttachmentService_Upload_Success(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	content := "test file content"

	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID}).Return(nil)
	mockRepo.On("Create", mock.Anything, nil, mock.AnythingOfType("*models.Attachment")).Return(nil)

	dto := &models.UploadAttachmentDTO{
		EntityType: "ticket", EntityID: entityID, FileName: "test.txt",
		File: strings.NewReader(content), UploadedBy: actorID, Realm: "",
	}
	att, err := svc.Upload(context.Background(), nil, dto)
	assert.NoError(t, err)
	assert.NotNil(t, att)
	assert.Equal(t, "test.txt", att.FileName)
	assert.Equal(t, entityID, att.EntityID)
	assert.Equal(t, "ticket", att.EntityType)
	assert.Equal(t, actorID, att.UploadedBy)
	assert.FileExists(t, att.FilePath)

	data, err := os.ReadFile(att.FilePath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))

	mockRepo.AssertExpectations(t)
}

func TestAttachmentService_Upload_RepoCreateFails(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()

	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID}).Return(nil)
	mockRepo.On("Create", mock.Anything, nil, mock.AnythingOfType("*models.Attachment")).Return(assert.AnError)

	dto := &models.UploadAttachmentDTO{
		EntityType: "ticket", EntityID: entityID, FileName: "test.txt",
		File: strings.NewReader("content"), UploadedBy: actorID, Realm: "",
	}
	_, err := svc.Upload(context.Background(), nil, dto)
	assert.Error(t, err)
}

func TestAttachmentService_Delete_Success(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	attID := uuid.New()
	entityID := uuid.New()
	actorID := uuid.New()

	att := &models.Attachment{
		ID: attID, EntityType: "ticket", EntityID: entityID,
		FilePath: "/tmp/test-file-to-delete.txt",
	}

	mockRepo.On("GetByID", mock.Anything, attID).Return(att, nil)
	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID}).Return(nil)
	mockRepo.On("Delete", mock.Anything, nil, attID).Return(nil)

	dto := &models.DeleteAttachmentDTO{ID: attID, ActorID: actorID, Realm: ""}
	err := svc.Delete(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestAttachmentService_Delete_FileNotFound(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	attID := uuid.New()
	entityID := uuid.New()
	actorID := uuid.New()

	att := &models.Attachment{
		ID: attID, EntityType: "ticket", EntityID: entityID,
		FilePath: "/nonexistent/path/file.txt",
	}

	mockRepo.On("GetByID", mock.Anything, attID).Return(att, nil)
	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID}).Return(nil)
	mockRepo.On("Delete", mock.Anything, nil, attID).Return(nil)

	dto := &models.DeleteAttachmentDTO{ID: attID, ActorID: actorID, Realm: ""}
	err := svc.Delete(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestAttachmentService_Upload_ReadFileContents(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	content := "read check content"

	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID}).Return(nil)
	mockRepo.On("Create", mock.Anything, nil, mock.AnythingOfType("*models.Attachment")).Return(nil)

	dto := &models.UploadAttachmentDTO{
		EntityType: "ticket", EntityID: entityID, FileName: "check.txt",
		File: strings.NewReader(content), UploadedBy: actorID, Realm: "",
	}
	att, err := svc.Upload(context.Background(), nil, dto)
	assert.NoError(t, err)

	assert.Contains(t, att.FilePath, "ticket")
	assert.Contains(t, att.FilePath, entityID.String())

	data, err := os.ReadFile(att.FilePath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))

	os.RemoveAll(att.FilePath[:len(att.FilePath)-len("/check.txt")])
	mockRepo.AssertExpectations(t)
}

func TestAttachmentService_GetForComments_GroupsAndFilters(t *testing.T) {
	mockRepo, _, _, svc := attachmentFixtures(t)

	ticketID := uuid.New()
	publicCommentID := uuid.New()
	internalCommentID := uuid.New()

	internal := map[uuid.UUID]bool{internalCommentID: true}
	atts := []*models.Attachment{
		{ID: uuid.New(), FileName: "public.txt", CommentID: &publicCommentID},
		{ID: uuid.New(), FileName: "internal.txt", CommentID: &internalCommentID},
		{ID: uuid.New(), FileName: "standalone.bin"},
	}

	mockRepo.On("GetByComments", mock.Anything, ticketID).Return(internal, atts, nil)

	// Пользователь без доступа к внутренним комментариям не получает файл
	// внутреннего комментария.
	result, err := svc.GetForComments(context.Background(), ticketID, false)
	assert.NoError(t, err)
	assert.Len(t, result[publicCommentID], 1)
	assert.Nil(t, result[internalCommentID])

	// Пользователь с доступом к внутренним комментариям получает все.
	result, err = svc.GetForComments(context.Background(), ticketID, true)
	assert.NoError(t, err)
	assert.Len(t, result[publicCommentID], 1)
	assert.Len(t, result[internalCommentID], 1)
}

func TestAttachmentService_GetByEntity_HidesInternalCommentFiles(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	internalCID := uuid.New()
	publicCID := uuid.New()
	expected := []*models.Attachment{
		{ID: uuid.New(), FileName: "public.txt"},
		{ID: uuid.New(), FileName: "internal_file.txt", CommentID: &internalCID},
		{ID: uuid.New(), FileName: "public_comment.txt", CommentID: &publicCID},
	}

	dto := &models.EntityAccessDTO{EntityType: "ticket", EntityID: entityID, ActorID: actorID, Realm: ""}
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Action: string(access.Read)}).Return(nil)
	mockAccess.On("CheckInternalAssigneeAccess", mock.Anything, mock.AnythingOfType("*models.AccessCheckDTO")).Return(models.ErrPermissionDenied)
	mockRepo.On("GetByEntity", mock.Anything, "ticket", entityID).Return(expected, nil)
	mockRepo.On("GetByComments", mock.Anything, entityID).Return(map[uuid.UUID]bool{internalCID: true}, nil, nil)

	got, err := svc.GetByEntity(context.Background(), dto)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	names := []string{got[0].FileName, got[1].FileName}
	assert.ElementsMatch(t, []string{"public.txt", "public_comment.txt"}, names)
}

func TestAttachmentService_GetByEntity_ShowAllForInternalUser(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	internalCID := uuid.New()
	expected := []*models.Attachment{
		{ID: uuid.New(), FileName: "public.txt"},
		{ID: uuid.New(), FileName: "internal_file.txt", CommentID: &internalCID},
	}

	dto := &models.EntityAccessDTO{EntityType: "ticket", EntityID: entityID, ActorID: actorID, Realm: ""}
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Action: string(access.Read)}).Return(nil)
	mockAccess.On("CheckInternalAssigneeAccess", mock.Anything, mock.AnythingOfType("*models.AccessCheckDTO")).Return(nil)
	mockRepo.On("GetByEntity", mock.Anything, "ticket", entityID).Return(expected, nil)

	got, err := svc.GetByEntity(context.Background(), dto)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	mockRepo.AssertNotCalled(t, "GetByComments")
}

// writeAttachmentFile создаёт файл вложения внутри uploadDir/entityType/entityID.
func writeAttachmentFile(t *testing.T, uploadDir, entityType string, entityID uuid.UUID, name string) string {
	t.Helper()
	dir := filepath.Join(uploadDir, entityType, entityID.String())
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("payload"), 0o644))
	return path
}

// TestAttachmentService_DeleteByEntity_KeepsFilesInsideTransaction — регрессия на потерю
// файлов: DeleteByEntity вызывается внутри транзакции вызывающего и не должен трогать ФС.
// Раньше он удалял директорию целиком, поэтому откат транзакции оставлял записи в БД
// без файлов — восстановить их было уже нечем.
func TestAttachmentService_DeleteByEntity_KeepsFilesInsideTransaction(t *testing.T) {
	mockRepo, _, _, svc := attachmentFixtures(t)

	ticketID := uuid.New()
	path := writeAttachmentFile(t, svc.conf.UploadDir, "ticket", ticketID, "a.txt")

	mockRepo.On("DeleteByEntity", mock.Anything, mock.Anything, "ticket", ticketID).Return(nil)

	err := svc.DeleteByEntity(context.Background(), nil, "ticket", ticketID)
	assert.NoError(t, err)

	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "файл не должен удаляться до коммита транзакции")
	mockRepo.AssertExpectations(t)
}

// TestAttachmentService_RemoveEntityDir — очистка после коммита сносит директорию целиком.
func TestAttachmentService_RemoveEntityDir(t *testing.T) {
	_, _, _, svc := attachmentFixtures(t)

	ticketID := uuid.New()
	subID := uuid.New()
	path := writeAttachmentFile(t, svc.conf.UploadDir, "ticket", ticketID, "a.txt")
	subPath := writeAttachmentFile(t, svc.conf.UploadDir, "subtask", subID, "b.txt")

	svc.RemoveEntityDir("ticket", ticketID)
	svc.RemoveEntityDir("subtask", subID)

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "директория вложений тикета должна быть снята")
	_, err = os.Stat(subPath)
	assert.True(t, os.IsNotExist(err), "директория вложений подзадачи должна быть снята")
}

// TestAttachmentService_RemoveEntityDir_UnknownTypeIgnored — неизвестный тип сущности
// игнорируется: путь не должен вычисляться от недоверенного значения, и ошибки здесь
// не возвращаются.
func TestAttachmentService_RemoveEntityDir_UnknownTypeIgnored(t *testing.T) {
	_, _, _, svc := attachmentFixtures(t)

	svc.RemoveEntityDir("../../etc", uuid.New()) //nolint:gosec // проверяем защиту от выхода за uploadDir
}

// TestAttachmentService_Delete_FileRemovedAfterCommit — одиночное удаление снимает файл
// только после успешного repo.Delete (при tx == nil он коммитится сразу).
func TestAttachmentService_Delete_FileRemovedAfterCommit(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	attID := uuid.New()
	entityID := uuid.New()
	actorID := uuid.New()
	path := writeAttachmentFile(t, svc.conf.UploadDir, "ticket", entityID, "a.txt")

	att := &models.Attachment{ID: attID, EntityType: "ticket", EntityID: entityID, FilePath: path}
	mockRepo.On("GetByID", mock.Anything, attID).Return(att, nil)
	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID}).Return(nil)
	mockRepo.On("Delete", mock.Anything, nil, attID).Return(nil)

	err := svc.Delete(context.Background(), nil, &models.DeleteAttachmentDTO{ID: attID, ActorID: actorID})
	assert.NoError(t, err)

	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "файл удалённого вложения должен быть снят с диска")
}

// TestAttachmentService_Delete_RepoErrorKeepsFile — если строка не удалена (repo.Delete
// вернул ошибку), файл обязан остаться: иначе получится запись в БД без файла.
func TestAttachmentService_Delete_RepoErrorKeepsFile(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	attID := uuid.New()
	entityID := uuid.New()
	actorID := uuid.New()
	path := writeAttachmentFile(t, svc.conf.UploadDir, "ticket", entityID, "a.txt")

	att := &models.Attachment{ID: attID, EntityType: "ticket", EntityID: entityID, FilePath: path}
	mockRepo.On("GetByID", mock.Anything, attID).Return(att, nil)
	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID}).Return(nil)
	mockRepo.On("Delete", mock.Anything, nil, attID).Return(assert.AnError)

	err := svc.Delete(context.Background(), nil, &models.DeleteAttachmentDTO{ID: attID, ActorID: actorID})
	assert.Error(t, err)

	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "при неудачном удалении записи файл должен остаться")
}

// TestAttachmentService_Delete_WithinCallerTransactionKeepsFile — при переданной транзакции
// момент коммита вызывающему неизвестен, поэтому файл не трогается (только Warn-лог).
func TestAttachmentService_Delete_WithinCallerTransactionKeepsFile(t *testing.T) {
	mockRepo, _, mockAccess, svc := attachmentFixtures(t)

	attID := uuid.New()
	entityID := uuid.New()
	actorID := uuid.New()
	path := writeAttachmentFile(t, svc.conf.UploadDir, "ticket", entityID, "a.txt")

	att := &models.Attachment{ID: attID, EntityType: "ticket", EntityID: entityID, FilePath: path}
	mockRepo.On("GetByID", mock.Anything, attID).Return(att, nil)
	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID}).Return(nil)
	mockRepo.On("Delete", mock.Anything, mock.Anything, attID).Return(nil)

	err := svc.Delete(context.Background(), fakeTx{}, &models.DeleteAttachmentDTO{ID: attID, ActorID: actorID})
	assert.NoError(t, err)

	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "внутри транзакции вызывающего файл не удаляется")
}

// fakeTx — заглушка postgres.Tx: нужна, чтобы проверить ветку Delete с транзакцией
// вызывающего (момент коммита снаружи неизвестен, поэтому файл удаляться не должен).
type fakeTx struct{}

func (fakeTx) TX() pgx.Tx                     { return nil }
func (fakeTx) Commit(context.Context) error   { return nil }
func (fakeTx) Rollback(context.Context) error { return nil }
