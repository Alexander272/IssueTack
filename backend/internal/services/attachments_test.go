package services

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func attachmentFixtures(t *testing.T) (*MockAttachmentsRepo, *MockSubtasksRepo, *MockTicketAccessChecker, *AttachmentService, string) {
	t.Helper()
	mockRepo := new(MockAttachmentsRepo)
	mockSubtasks := new(MockSubtasksRepo)
	mockAccess := new(MockTicketAccessChecker)
	uploadDir := t.TempDir()

	svc := &AttachmentService{
		repo:         mockRepo,
		conf:         &config.FileServerConfig{UploadDir: uploadDir},
		ticketAccess: mockAccess,
		subtaskRepo:  mockSubtasks,
	}
	return mockRepo, mockSubtasks, mockAccess, svc, uploadDir
}

func TestAttachmentService_GetByEntity_Success(t *testing.T) {
	mockRepo, _, mockAccess, svc, _ := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	expected := []*models.Attachment{{ID: uuid.New(), FileName: "file.pdf"}}

	dto := &models.EntityAccessDTO{EntityType: "ticket", EntityID: entityID, ActorID: actorID, Realm: ""}
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Action: string(access.Read), Realm: ""}).Return(nil)
	mockRepo.On("GetByEntity", mock.Anything, "ticket", entityID).Return(expected, nil)

	got, err := svc.GetByEntity(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestAttachmentService_GetByEntity_NoAccess(t *testing.T) {
	_, _, _, svc, _ := attachmentFixtures(t)
	svc.ticketAccess = nil

	dto := &models.EntityAccessDTO{EntityType: "ticket", EntityID: uuid.New(), ActorID: uuid.New()}
	_, err := svc.GetByEntity(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestAttachmentService_GetByEntity_AccessDenied(t *testing.T) {
	mockRepo, _, mockAccess, svc, _ := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	dto := &models.EntityAccessDTO{EntityType: "ticket", EntityID: entityID, ActorID: actorID, Realm: ""}
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Action: string(access.Read), Realm: ""}).Return(models.ErrPermissionDenied)

	_, err := svc.GetByEntity(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "GetByEntity")
}

func TestAttachmentService_GetByEntity_Subtask(t *testing.T) {
	mockRepo, mockSubtasks, mockAccess, svc, _ := attachmentFixtures(t)

	subtaskID := uuid.New()
	ticketID := uuid.New()
	actorID := uuid.New()
	expected := []*models.Attachment{{ID: uuid.New(), FileName: "file.pdf"}}

	mockSubtasks.On("GetByID", mock.Anything, &models.GetSubtaskDTO{ID: subtaskID}).Return(&models.Subtask{
		ID: subtaskID, TicketID: ticketID,
	}, nil)
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: ticketID, UserID: actorID, Action: string(access.Read), Realm: ""}).Return(nil)
	mockRepo.On("GetByEntity", mock.Anything, "subtask", subtaskID).Return(expected, nil)

	dto := &models.EntityAccessDTO{EntityType: "subtask", EntityID: subtaskID, ActorID: actorID, Realm: ""}
	got, err := svc.GetByEntity(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestAttachmentService_Upload_InvalidEntityType(t *testing.T) {
	_, _, _, svc, _ := attachmentFixtures(t)

	dto := &models.UploadAttachmentDTO{EntityType: "invalid", EntityID: uuid.New(), FileName: "test.txt", UploadedBy: uuid.New()}
	_, err := svc.Upload(context.Background(), nil, dto)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid entity type")
}

func TestAttachmentService_Upload_Success(t *testing.T) {
	mockRepo, _, mockAccess, svc, _ := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	content := "test file content"

	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Realm: ""}).Return(nil)
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
	mockRepo, _, mockAccess, svc, _ := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()

	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Realm: ""}).Return(nil)
	mockRepo.On("Create", mock.Anything, nil, mock.AnythingOfType("*models.Attachment")).Return(assert.AnError)

	dto := &models.UploadAttachmentDTO{
		EntityType: "ticket", EntityID: entityID, FileName: "test.txt",
		File: strings.NewReader("content"), UploadedBy: actorID, Realm: "",
	}
	_, err := svc.Upload(context.Background(), nil, dto)
	assert.Error(t, err)
}

func TestAttachmentService_Delete_Success(t *testing.T) {
	mockRepo, _, mockAccess, svc, _ := attachmentFixtures(t)

	attID := uuid.New()
	entityID := uuid.New()
	actorID := uuid.New()

	att := &models.Attachment{
		ID: attID, EntityType: "ticket", EntityID: entityID,
		FilePath: "/tmp/test-file-to-delete.txt",
	}

	mockRepo.On("GetByID", mock.Anything, attID).Return(att, nil)
	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Realm: ""}).Return(nil)
	mockRepo.On("Delete", mock.Anything, nil, attID).Return(nil)

	dto := &models.DeleteAttachmentDTO{ID: attID, ActorID: actorID, Realm: ""}
	err := svc.Delete(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestAttachmentService_Delete_FileNotFound(t *testing.T) {
	mockRepo, _, mockAccess, svc, _ := attachmentFixtures(t)

	attID := uuid.New()
	entityID := uuid.New()
	actorID := uuid.New()

	att := &models.Attachment{
		ID: attID, EntityType: "ticket", EntityID: entityID,
		FilePath: "/nonexistent/path/file.txt",
	}

	mockRepo.On("GetByID", mock.Anything, attID).Return(att, nil)
	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Realm: ""}).Return(nil)
	mockRepo.On("Delete", mock.Anything, nil, attID).Return(nil)

	dto := &models.DeleteAttachmentDTO{ID: attID, ActorID: actorID, Realm: ""}
	err := svc.Delete(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestAttachmentService_Upload_ReadFileContents(t *testing.T) {
	mockRepo, _, mockAccess, svc, _ := attachmentFixtures(t)

	entityID := uuid.New()
	actorID := uuid.New()
	content := "read check content"

	mockAccess.On("CheckWorkAccess", mock.Anything, &models.AccessCheckDTO{TicketID: entityID, UserID: actorID, Realm: ""}).Return(nil)
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
