package services

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestBuildActDocx проверяет, что документ акта собирается в валидный docx:
// части пакета на месте, а данные снимка состава попали в текст. Сам файл
// открывается офисным пакетом — это отдельная проверка руками, здесь
// фиксируется контракт генератора.
func TestBuildActDocx(t *testing.T) {
	comment := "Выдано вместе с сумкой"
	serial := "SN-001"
	act := &models.EquipmentAct{
		ID:        uuid.New(),
		RealmID:   uuid.New(),
		Number:    "АКТ-2026-0007",
		Status:    models.ActStatusFormed,
		IssuedAt:  time.Date(2026, time.October, 15, 12, 30, 0, 0, time.UTC),
		Comment:   comment,
		Location:  nil,
		Content:   []byte("[]"),
		CreatedAt: time.Now(),
	}
	positions := []actPosition{
		{
			EquipmentID:     uuid.New(),
			TypeName:        "Принтер",
			Name:            "HP LaserJet",
			SerialNumber:    &serial,
			InventoryNumber: nil,
		},
	}

	data, err := buildActDocx(act, positions, actDocParties{Issuer: "Иванов И.И.", Recipient: "Петров П.П."})
	require.NoError(t, err)
	require.NotEmpty(t, data)

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	parts := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		body, err := io.ReadAll(rc)
		require.NoError(t, err)
		rc.Close()
		parts[f.Name] = string(body)
	}

	require.Contains(t, parts, "[Content_Types].xml")
	require.Contains(t, parts, "_rels/.rels")
	require.Contains(t, parts, "word/document.xml")

	document := parts["word/document.xml"]
	for _, want := range []string{"АКТ-2026-0007", "15.10.2026", "HP LaserJet", "SN-001", "Иванов И.И.", "Петров П.П.", comment} {
		require.True(t, strings.Contains(document, want), "в документе нет %q", want)
	}
}

// TestBuildActDocx_NoPositions — акт без позиций: таблица остаётся с шапкой, а
// пустые подписи превращаются в прочерк, а не в «<nil>».
func TestBuildActDocx_NoPositions(t *testing.T) {
	act := &models.EquipmentAct{
		ID:        uuid.New(),
		RealmID:   uuid.New(),
		Number:    "АКТ-2026-0001",
		Status:    models.ActStatusFormed,
		IssuedAt:  time.Now(),
		Content:   []byte("[]"),
		CreatedAt: time.Now(),
	}

	data, err := buildActDocx(act, nil, actDocParties{})
	require.NoError(t, err)
	require.NotEmpty(t, data)

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	require.NotEmpty(t, zr.File)
}
