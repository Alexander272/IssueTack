package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func commentServiceFixtures() (*MockCommentsRepo, *MockTicketAccessChecker, *MockTicketsRepo, *MockUserService, *MockMattermostRepo, *MockMMSender, *CommentService) {
	mockRepo := new(MockCommentsRepo)
	mockAccess := new(MockTicketAccessChecker)
	mockTickets := new(MockTicketsRepo)
	mockUsers := new(MockUserService)
	mockMMRepo := new(MockMattermostRepo)
	mockSender := new(MockMMSender)

	svc := &CommentService{
		repo:          mockRepo,
		ticketAccess:  mockAccess,
		tickets:       mockTickets,
		users:         mockUsers,
		mmRepo:        mockMMRepo,
		mmSender:      mockSender,
		notifications: nil,
	}
	return mockRepo, mockAccess, mockTickets, mockUsers, mockMMRepo, mockSender, svc
}

func commentTicket(ownerID, authorID uuid.UUID) *models.Ticket {
	realmID := uuid.New()
	num := 42
	return &models.Ticket{
		ID:           uuid.New(),
		Title:        "Заголовок заявки",
		TicketNumber: &num,
		RealmID:      &realmID,
		Owner:        &models.UserShort{ID: ownerID},
		Assignee:     &models.UserShort{ID: authorID},
		Creator:      models.UserShort{ID: authorID},
	}
}

func TestCommentService_Create_SendsDMToOwner(t *testing.T) {
	mockRepo, mockAccess, mockTickets, mockUsers, mockMMRepo, mockSender, svc := commentServiceFixtures()

	prismID := uuid.New()
	ownerID := uuid.New()
	ticket := commentTicket(ownerID, prismID)
	ownerMM := "owner-mm-id"
	realmID := *ticket.RealmID

	dto := &models.CreateCommentDTO{
		Text:     "уточните, пожалуйста, сроки",
		TicketID: ticket.ID,
		UserID:   prismID,
		Realm:    realmID.String(),
	}

	mockAccess.On("CheckWorkAccess", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.MatchedBy(func(c *models.Comment) bool {
		return c.Text == dto.Text && c.UserID == prismID && c.TicketID == ticket.ID && !c.IsInternal
	})).Return(nil)
	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticket.ID}).Return(ticket, nil)
	mockUsers.On("GetByID", mock.Anything, ownerID).Return(&models.UserData{ID: ownerID, MattermostID: &ownerMM}, nil)
	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockSender.On("Send", "bt", "bb", ownerMM, mock.Anything).Return(nil)

	comment, err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	assert.Equal(t, dto.Text, comment.Text)
	mockSender.AssertCalled(t, "Send", "bt", "bb", ownerMM, mock.Anything)
	mockAccess.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockTickets.AssertExpectations(t)
	mockUsers.AssertExpectations(t)
	mockMMRepo.AssertExpectations(t)
	mockSender.AssertExpectations(t)
}

func TestCommentService_Create_AuthorIsOwner_NoDM(t *testing.T) {
	mockRepo, mockAccess, mockTickets, mockUsers, mockMMRepo, mockSender, svc := commentServiceFixtures()

	prismID := uuid.New()
	ticket := commentTicket(prismID, prismID)

	dto := &models.CreateCommentDTO{
		Text:     "моя заявка",
		TicketID: ticket.ID,
		UserID:   prismID,
		Realm:    "realm",
	}

	mockAccess.On("CheckWorkAccess", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticket.ID}).Return(ticket, nil)

	_, err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	mockSender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockUsers.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	mockMMRepo.AssertNotCalled(t, "GetByRealm", mock.Anything, mock.Anything)
}

func TestCommentService_Create_Internal_NoDM(t *testing.T) {
	mockRepo, mockAccess, mockTickets, mockUsers, _, mockSender, svc := commentServiceFixtures()

	prismID := uuid.New()
	ownerID := uuid.New()
	ticket := commentTicket(ownerID, prismID)

	dto := &models.CreateCommentDTO{
		Text:       "внутренний коммент",
		TicketID:   ticket.ID,
		UserID:     prismID,
		Realm:      "realm",
		IsInternal: true,
	}

	mockAccess.On("CheckWorkAccess", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticket.ID}).Return(ticket, nil)

	_, err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	mockSender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockUsers.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestCommentService_Create_OwnerWithoutMM_NoDM(t *testing.T) {
	mockRepo, mockAccess, mockTickets, mockUsers, mockMMRepo, mockSender, svc := commentServiceFixtures()

	prismID := uuid.New()
	ownerID := uuid.New()
	ticket := commentTicket(ownerID, prismID)

	dto := &models.CreateCommentDTO{
		Text:     "вопрос",
		TicketID: ticket.ID,
		UserID:   prismID,
		Realm:    "realm",
	}

	mockAccess.On("CheckWorkAccess", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticket.ID}).Return(ticket, nil)
	mockUsers.On("GetByID", mock.Anything, ownerID).Return(&models.UserData{ID: ownerID, MattermostID: nil}, nil)

	_, err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	mockSender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockMMRepo.AssertNotCalled(t, "GetByRealm", mock.Anything, mock.Anything)
}
