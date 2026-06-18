package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func checkListFixtures() (*MockChecklistsRepo, *MockSubtaskService, *ChecklistService) {
	mockRepo := new(MockChecklistsRepo)
	mockSubtasks := new(MockSubtaskService)
	svc := NewChecklistService(mockRepo, mockSubtasks)
	return mockRepo, mockSubtasks, svc
}

func TestChecklistService_Get(t *testing.T) {
	mockRepo, _, svc := checkListFixtures()

	req := &models.GetChecklistTemplatesDTO{}
	expected := []*models.ChecklistTemplate{{ID: uuid.New(), Title: "Template 1"}}
	mockRepo.On("Get", mock.Anything, req).Return(expected, nil)

	got, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestChecklistService_GetByID(t *testing.T) {
	mockRepo, _, svc := checkListFixtures()

	templateID := uuid.New()
	req := &models.GetChecklistTemplateDTO{ID: templateID}
	template := &models.ChecklistTemplate{ID: templateID, Title: "Template"}
	items := []*models.ChecklistTemplateItem{{ID: uuid.New(), Title: "Item 1"}}

	mockRepo.On("GetByID", mock.Anything, req).Return(template, nil)
	mockRepo.On("GetItems", mock.Anything, templateID).Return(items, nil)

	got, err := svc.GetByID(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, items, got.Items)
}

func TestChecklistService_Create(t *testing.T) {
	mockRepo, _, svc := checkListFixtures()

	dto := &models.ChecklistTemplateDTO{Title: "New Template"}
	mockRepo.On("Create", mock.Anything, dto).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
}

func TestChecklistService_Update(t *testing.T) {
	mockRepo, _, svc := checkListFixtures()

	dto := &models.ChecklistTemplateDTO{ID: uuid.New(), Title: "Updated"}
	mockRepo.On("Update", mock.Anything, dto).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
}

func TestChecklistService_Delete(t *testing.T) {
	mockRepo, _, svc := checkListFixtures()

	dto := &models.DelChecklistTemplateDTO{ID: uuid.New()}
	mockRepo.On("Delete", mock.Anything, dto).Return(nil)

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
}

func TestChecklistService_SetItems(t *testing.T) {
	mockRepo, _, svc := checkListFixtures()

	templateID := uuid.New()
	items := []*models.ChecklistTemplateItemDTO{{Title: "Item"}}
	mockRepo.On("SetItems", mock.Anything, nil, templateID, items).Return(nil)

	err := svc.SetItems(context.Background(), nil, templateID, items)
	assert.NoError(t, err)
}

func TestChecklistService_GetItems(t *testing.T) {
	mockRepo, _, svc := checkListFixtures()

	templateID := uuid.New()
	expected := []*models.ChecklistTemplateItem{{ID: uuid.New(), Title: "Item"}}
	mockRepo.On("GetItems", mock.Anything, templateID).Return(expected, nil)

	got, err := svc.GetItems(context.Background(), templateID)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestChecklistService_ApplyTemplate_Success(t *testing.T) {
	mockRepo, mockSubtasks, svc := checkListFixtures()

	ticketID := uuid.New()
	templateID := uuid.New()
	actor := &models.Actor{ID: uuid.New(), Name: "test"}
	items := []*models.ChecklistTemplateItem{
		{ID: uuid.New(), Title: "Task 1", SortOrder: 1},
		{ID: uuid.New(), Title: "Task 2", SortOrder: 2},
	}

	mockRepo.On("GetItems", mock.Anything, templateID).Return(items, nil)
	mockSubtasks.On("CreateSeveral", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.ApplyTemplate(context.Background(), nil, ticketID, templateID, actor)
	assert.NoError(t, err)
}

func TestChecklistService_ApplyTemplate_EmptyItems(t *testing.T) {
	mockRepo, mockSubtasks, svc := checkListFixtures()

	ticketID := uuid.New()
	templateID := uuid.New()
	actor := &models.Actor{ID: uuid.New(), Name: "test"}

	mockRepo.On("GetItems", mock.Anything, templateID).Return([]*models.ChecklistTemplateItem{}, nil)

	err := svc.ApplyTemplate(context.Background(), nil, ticketID, templateID, actor)
	assert.NoError(t, err)
	mockSubtasks.AssertNotCalled(t, "CreateSeveral")
}
