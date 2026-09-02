package services

import (
	"context"
	"strings"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func commentServiceFixtures() (*MockCommentsRepo, *MockTicketAccessChecker, *MockTicketsService, *MockUserService, *MockMattermostRepo, *MockMMSender, *CommentService) {
	mockRepo := new(MockCommentsRepo)
	mockAccess := new(MockTicketAccessChecker)
	mockTickets := new(MockTicketsService)
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
		txManager:     &mockTransactionManager{},
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
	mockTickets.On("GetSummary", mock.Anything, ticket.ID).Return(ticket, nil)
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
	mockTickets.On("GetSummary", mock.Anything, ticket.ID).Return(ticket, nil)

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
	mockTickets.On("GetSummary", mock.Anything, ticket.ID).Return(ticket, nil)

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
	mockTickets.On("GetSummary", mock.Anything, ticket.ID).Return(ticket, nil)
	mockUsers.On("GetByID", mock.Anything, ownerID).Return(&models.UserData{ID: ownerID, MattermostID: nil}, nil)

	_, err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	mockSender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockMMRepo.AssertNotCalled(t, "GetByRealm", mock.Anything, mock.Anything)
}

func TestCommentService_Create_WithFiles_BindsAttachments(t *testing.T) {
	mockRepo, mockAccess, mockTickets, _, _, _, svc := commentServiceFixtures()
	mockAttachments := new(MockAttachmentService)
	svc.attachments = mockAttachments

	prismID := uuid.New()
	ticket := commentTicket(prismID, prismID)
	fileDTO := &models.UploadAttachmentDTO{
		EntityType: "ticket",
		EntityID:   ticket.ID,
		FileName:   "doc.pdf",
		FileSize:   123,
		MimeType:   "application/pdf",
		File:       strings.NewReader("data"),
		UploadedBy: prismID,
		Realm:      "realm",
	}

	dto := &models.CreateCommentDTO{
		Text:     "с файлом",
		TicketID: ticket.ID,
		UserID:   prismID,
		Realm:    "realm",
		Files:    []*models.UploadAttachmentDTO{fileDTO},
	}

	mockAccess.On("CheckWorkAccess", mock.Anything, mock.Anything).Return(nil)
	mockTickets.On("GetSummary", mock.Anything, ticket.ID).Return(ticket, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.MatchedBy(func(c *models.Comment) bool {
		return c.Text == dto.Text && c.TicketID == ticket.ID
	})).Return(nil)
	mockAttachments.On("Upload", mock.Anything, mock.Anything, mock.MatchedBy(func(f *models.UploadAttachmentDTO) bool {
		return f.FileName == "doc.pdf" && f.CommentID != nil && *f.CommentID != uuid.Nil
	})).Return(&models.Attachment{ID: uuid.New(), FileName: "doc.pdf"}, nil)

	comment, err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	assert.Equal(t, dto.Text, comment.Text)
	mockAttachments.AssertExpectations(t)
}

func TestCommentService_GetByTicket_PopulatesAttachments(t *testing.T) {
	mockRepo, mockAccess, _, _, _, _, svc := commentServiceFixtures()
	mockAttachments := new(MockAttachmentService)
	svc.attachments = mockAttachments

	ticketID := uuid.New()
	userID := uuid.New()
	commentID := uuid.New()
	att := &models.Attachment{ID: uuid.New(), FileName: "file.png", CommentID: &commentID}

	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{TicketID: ticketID, UserID: userID, Action: "read", Realm: "realm"}).Return(nil)
	mockAccess.On("CheckInternalAssigneeAccess", mock.Anything, mock.AnythingOfType("*models.AccessCheckDTO")).Return(nil)
	mockRepo.On("GetByTicket", mock.Anything, ticketID, userID, true).Return([]*models.Comment{{ID: commentID, Text: "внутр"}}, nil)
	mockAttachments.On("GetForComments", mock.Anything, ticketID, true).Return(map[uuid.UUID][]*models.Attachment{commentID: {att}}, nil)

	comments, err := svc.GetByTicket(context.Background(), ticketID, userID, "realm")
	assert.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Len(t, comments[0].Attachments, 1)
	assert.Equal(t, "file.png", comments[0].Attachments[0].FileName)
}
