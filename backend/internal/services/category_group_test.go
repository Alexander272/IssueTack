package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCategoryGroupService_Delete_CategoriesInSectionBlocked(t *testing.T) {
	mockRepo := new(MockCategoryGroupsRepo)
	svc := &CategoryGroupService{repo: mockRepo}

	sectionID := uuid.New()
	dto := &models.DelCategoryGroupDTO{ID: sectionID, RealmID: uuid.New()}

	mockRepo.On("CountByCategory", mock.Anything, sectionID).Return(2, nil)

	err := svc.Delete(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrCategoryGroupInUse)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestCategoryGroupService_Delete_EmptySection(t *testing.T) {
	mockRepo := new(MockCategoryGroupsRepo)
	svc := &CategoryGroupService{repo: mockRepo}

	sectionID := uuid.New()
	dto := &models.DelCategoryGroupDTO{ID: sectionID, RealmID: uuid.New()}

	mockRepo.On("CountByCategory", mock.Anything, sectionID).Return(0, nil)
	mockRepo.On("Delete", mock.Anything, dto).Return(nil)

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCategoryGroupService_Delete_CountError(t *testing.T) {
	mockRepo := new(MockCategoryGroupsRepo)
	svc := &CategoryGroupService{repo: mockRepo}

	sectionID := uuid.New()
	dto := &models.DelCategoryGroupDTO{ID: sectionID, RealmID: uuid.New()}

	mockRepo.On("CountByCategory", mock.Anything, sectionID).Return(0, assert.AnError)

	err := svc.Delete(context.Background(), dto)
	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestCategoryGroupService_Get_And_Create(t *testing.T) {
	mockRepo := new(MockCategoryGroupsRepo)
	svc := NewCategoryGroupService(mockRepo)

	realmID := uuid.New()
	req := &models.GetCategoryGroupsDTO{RealmID: realmID}
	expected := []*models.CategoryGroup{
		{ID: uuid.New(), RealmID: realmID, Name: "Информационные системы", SortOrder: 0},
		{ID: uuid.New(), RealmID: realmID, Name: "Почта", SortOrder: 1},
	}
	mockRepo.On("Get", mock.Anything, req).Return(expected, nil)

	data, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, data, 2)
	assert.Equal(t, "Почта", data[1].Name)

	dto := &models.CategoryGroupDTO{Name: "Прочее", SortOrder: 5}
	mockRepo.On("Create", mock.Anything, dto).Return(nil)
	assert.NoError(t, svc.Create(context.Background(), dto))
	mockRepo.AssertExpectations(t)
}
