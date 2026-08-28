package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGroupService_Delete_OpenTicketsBlocked(t *testing.T) {
	mockRepo := new(MockGroupRepo)
	mockTickets := new(MockTicketsRepo)
	svc := &GroupService{repo: mockRepo, tickets: mockTickets}

	groupID := uuid.New()
	dto := &models.DelGroupDTO{ID: groupID}

	mockTickets.On("CountNotClosedByGroup", mock.Anything, groupID).Return(1, nil)

	err := svc.Delete(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrGroupHasOpenTickets)
	mockTickets.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestGroupService_Delete_NoOpenTickets(t *testing.T) {
	mockRepo := new(MockGroupRepo)
	mockTickets := new(MockTicketsRepo)
	svc := &GroupService{repo: mockRepo, tickets: mockTickets}

	groupID := uuid.New()
	dto := &models.DelGroupDTO{ID: groupID}

	mockTickets.On("CountNotClosedByGroup", mock.Anything, groupID).Return(0, nil)
	mockRepo.On("Delete", mock.Anything, dto).Return(nil)

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
	mockTickets.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGroupService_Delete_CountError(t *testing.T) {
	mockRepo := new(MockGroupRepo)
	mockTickets := new(MockTicketsRepo)
	svc := &GroupService{repo: mockRepo, tickets: mockTickets}

	groupID := uuid.New()
	dto := &models.DelGroupDTO{ID: groupID}

	mockTickets.On("CountNotClosedByGroup", mock.Anything, groupID).Return(0, assert.AnError)

	err := svc.Delete(context.Background(), dto)
	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestCategoryService_Delete_OpenTicketsBlocked(t *testing.T) {
	mockRepo := new(MockCategoriesRepo)
	mockTickets := new(MockTicketsRepo)
	svc := &CategoryService{repo: mockRepo, tickets: mockTickets}

	categoryID := uuid.New()
	dto := &models.DelCategoryDTO{ID: categoryID}

	mockTickets.On("CountNotClosedByCategory", mock.Anything, categoryID).Return(1, nil)

	err := svc.Delete(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrCategoryHasOpenTickets)
	mockTickets.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestCategoryService_Delete_NoOpenTickets(t *testing.T) {
	mockRepo := new(MockCategoriesRepo)
	mockTickets := new(MockTicketsRepo)
	svc := &CategoryService{repo: mockRepo, tickets: mockTickets}

	categoryID := uuid.New()
	dto := &models.DelCategoryDTO{ID: categoryID}

	mockTickets.On("CountNotClosedByCategory", mock.Anything, categoryID).Return(0, nil)
	mockRepo.On("Delete", mock.Anything, dto).Return(nil)

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
	mockTickets.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
