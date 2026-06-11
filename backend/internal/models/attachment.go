package models

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID          uuid.UUID `json:"id" db:"id"`
	EntityType  string    `json:"entityType" db:"entity_type"`
	EntityID    uuid.UUID `json:"entityId" db:"entity_id"`
	FileName    string    `json:"fileName" db:"file_name"`
	FilePath    string    `json:"-" db:"file_path"`
	FileSize    int64     `json:"fileSize" db:"file_size"`
	MimeType    string    `json:"mimeType" db:"mime_type"`
	UploadedBy  uuid.UUID `json:"uploadedBy" db:"uploaded_by"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}
