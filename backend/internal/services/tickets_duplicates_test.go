package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// duplicateFixtures собирает сервис и минимальный набор заявок. Исполнитель дубля
// (actor) — «рабочий» пользователь реалма, основная заявка создана им же, поэтому
// read-доступ к ней проходит по атрибутам без обращения к Casbin.
func duplicateFixtures(t *testing.T) (
	*MockTicketsRepo, *MockActivityLogService, *MockNotificationService,
	*MockGroupsRepo, *TicketService, *fakeFavoriteCleaner,
	uuid.UUID, uuid.UUID, uuid.UUID, *models.Actor,
) {
	t.Helper()
	repo, logs, _, _, notifications, groups, policies, svc, favorites := ticketServiceFixturesWithFavorites()
	actor := &models.Actor{ID: uuid.New(), Name: "Исполнитель"}
	creator := uuid.New()
	dupID, primaryID := uuid.New(), uuid.New()

	// Исполнитель не начальник области, но состоит в группе — то есть не заказчик.
	policies.On("Enforce", actor.ID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil).Maybe()
	policies.On("Enforce", actor.ID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil).Maybe()
	groups.On("GetManagedGroups", mock.Anything, actor.ID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil).Maybe()
	groups.On("GetMemberGroups", mock.Anything, actor.ID, (*uuid.UUID)(nil)).Return([]uuid.UUID{uuid.New()}, nil).Maybe()

	return repo, logs, notifications, groups, svc, favorites, dupID, primaryID, creator, actor
}

func duplicateTicket(id uuid.UUID, creator uuid.UUID, actorID uuid.UUID, status models.TicketStatus) *models.Ticket {
	return &models.Ticket{
		ID: id, Title: "Дубль", Status: status,
		Creator:  models.UserShort{ID: creator},
		Assignee: &models.UserShort{ID: actorID},
	}
}

func primaryTicket(id uuid.UUID, creator uuid.UUID, status models.TicketStatus) *models.Ticket {
	return &models.Ticket{ID: id, Title: "Основная", Status: status, Creator: models.UserShort{ID: creator}}
}

// expectDuplicateReads подменяет загрузку заявок по id: сервис читает их до и
// после блокировки, а также после коммита для уведомления.
func expectDuplicateReads(repo *MockTicketsRepo, tickets ...*models.Ticket) {
	for _, ticket := range tickets {
		id := ticket.ID
		repo.On("GetByID", mock.Anything, mock.MatchedBy(func(req *models.GetTicketByIdDTO) bool { return req.ID == id })).
			Return(ticket, nil).Maybe()
	}
}

// expectDuplicateTxFallback — широкие заглушки на вспомогательные запросы.
// Цепочка здесь намеренно отсутствует: реальный репозиторий всегда возвращает
// хотя бы стартовый узел, и сервис опирается на это (chain[len-1]). Конкретные
// ожидания GetDuplicateChain объявляются в каждом тесте до вызова хелпера.
func expectDuplicateTxFallback(repo *MockTicketsRepo) {
	repo.On("GetDuplicateDescendants", mock.Anything, mock.Anything, mock.Anything).
		Return([]uuid.UUID{}, nil).Maybe()
	repo.On("LockTickets", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
}

func TestTicketService_LinkDuplicate_ClosesAndNotifies(t *testing.T) {
	repo, logs, notifications, _, svc, favorites, dupID, primaryID, creator, actor := duplicateFixtures(t)
	dup := duplicateTicket(dupID, creator, actor.ID, models.StatusInProgress)
	primary := primaryTicket(primaryID, actor.ID, models.StatusOpen)
	expectDuplicateReads(repo, dup, primary)

	// До блокировки читаем цепочки дубля и новой основной, после — основную ещё раз.
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, dupID).
		Return([]postgres.DuplicateNode{{ID: dupID}}, nil).Once()
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, primaryID).
		Return([]postgres.DuplicateNode{{ID: primaryID}}, nil).Twice()
	expectDuplicateTxFallback(repo)

	repo.On("SetDuplicate", mock.Anything, mock.Anything, dupID, primaryID, models.StatusInProgress).Return(nil).Once()
	repo.On("AddDuplicateNotifier", mock.Anything, mock.Anything, primaryID, creator, dupID, actor.ID).Return(nil).Once()
	logs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()
	notifications.On("NotifyDuplicate", mock.Anything, dup, primary, actor.ID, true).Return(nil).Once()

	result, err := svc.LinkDuplicate(context.Background(), &models.LinkDuplicateDTO{
		ID: dupID, PrimaryID: primaryID, Actor: actor,
	})
	require.NoError(t, err)
	assert.Equal(t, primaryID, result.RootID)
	assert.Equal(t, []uuid.UUID{dupID, primaryID}, result.Chain)
	assert.Equal(t, []uuid.UUID{dupID}, favorites.ticketIDs(), "temporary-пины дубля снимаются в транзакции")
	repo.AssertExpectations(t)
	logs.AssertExpectations(t)
	notifications.AssertExpectations(t)
}

// Регресс: связывающий дубль пользователь часто является его автором. Исключать
// актора из списка наблюдателей основной было неверно — он терял read-доступ к
// ней после снятия своей же прямой связи. Уведомление о собственном действии
// актору по-прежнему не уходит (это задача NotifyDuplicate).
func TestTicketService_LinkDuplicate_ActorIsAuthorStillObserver(t *testing.T) {
	repo, logs, notifications, _, svc, _, dupID, primaryID, _, actor := duplicateFixtures(t)
	ownerID := uuid.New()
	dup := duplicateTicket(dupID, actor.ID, actor.ID, models.StatusInProgress)
	dup.Owner = &models.UserShort{ID: ownerID}
	primary := primaryTicket(primaryID, actor.ID, models.StatusOpen)
	expectDuplicateReads(repo, dup, primary)

	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, dupID).
		Return([]postgres.DuplicateNode{{ID: dupID}}, nil).Once()
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, primaryID).
		Return([]postgres.DuplicateNode{{ID: primaryID}}, nil).Twice()
	expectDuplicateTxFallback(repo)

	repo.On("SetDuplicate", mock.Anything, mock.Anything, dupID, primaryID, models.StatusInProgress).Return(nil).Once()
	repo.On("AddDuplicateNotifier", mock.Anything, mock.Anything, primaryID, actor.ID, dupID, actor.ID).Return(nil).Once()
	repo.On("AddDuplicateNotifier", mock.Anything, mock.Anything, primaryID, ownerID, dupID, actor.ID).Return(nil).Once()
	logs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()
	notifications.On("NotifyDuplicate", mock.Anything, dup, primary, actor.ID, true).Return(nil).Once()

	_, err := svc.LinkDuplicate(context.Background(), &models.LinkDuplicateDTO{
		ID: dupID, PrimaryID: primaryID, Actor: actor,
	})
	require.NoError(t, err)
	repo.AssertExpectations(t)
	logs.AssertExpectations(t)
	notifications.AssertExpectations(t)
}

func TestTicketService_LinkDuplicate_RepointKeepsOtherDuplicates(t *testing.T) {
	repo, logs, notifications, _, svc, _, dupID, primaryID, creator, actor := duplicateFixtures(t)
	oldPrimaryID := uuid.New()
	dup := duplicateTicket(dupID, creator, actor.ID, models.StatusClosed)
	dup.DuplicateOf = &oldPrimaryID
	status := models.StatusInProgress
	dup.PreviousStatus = &status
	oldPrimary := primaryTicket(oldPrimaryID, creator, models.StatusClosed)
	primary := primaryTicket(primaryID, actor.ID, models.StatusOpen)
	expectDuplicateReads(repo, dup, oldPrimary, primary)

	chain := []postgres.DuplicateNode{{ID: dupID, DuplicateOf: &oldPrimaryID}, {ID: oldPrimaryID}}
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, dupID).Return(chain, nil).Once()
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, primaryID).
		Return([]postgres.DuplicateNode{{ID: primaryID}}, nil).Twice()
	expectDuplicateTxFallback(repo)

	// Перенаправляется только этот дубль: RepointDuplicates (вся ветка старой
	// основной) вызываться не должен.
	repo.On("RepointOne", mock.Anything, mock.Anything, dupID, primaryID).Return(nil).Once()
	repo.On("MoveDuplicateNotifier", mock.Anything, mock.Anything, oldPrimaryID, primaryID, dupID).Return(nil).Once()
	repo.On("AddDuplicateNotifier", mock.Anything, mock.Anything, primaryID, creator, dupID, actor.ID).Return(nil).Once()
	logs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	notifications.On("NotifyDuplicate", mock.Anything, dup, primary, actor.ID, true).Return(nil).Once()

	_, err := svc.LinkDuplicate(context.Background(), &models.LinkDuplicateDTO{
		ID: dupID, PrimaryID: primaryID, Actor: actor,
	})
	require.NoError(t, err)
	repo.AssertCalled(t, "RepointOne", mock.Anything, mock.Anything, dupID, primaryID)
	repo.AssertNotCalled(t, "RepointDuplicates", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "SetDuplicate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
	logs.AssertExpectations(t)
	notifications.AssertExpectations(t)
}

func TestTicketService_LinkDuplicate_FlattensChain(t *testing.T) {
	repo, logs, notifications, _, svc, _, dupID, primaryID, creator, actor := duplicateFixtures(t)
	rootID, childID := uuid.New(), uuid.New()
	dup := duplicateTicket(dupID, creator, actor.ID, models.StatusOpen)
	primary := primaryTicket(primaryID, actor.ID, models.StatusClosed)
	root := primaryTicket(rootID, creator, models.StatusOpen)
	primary.DuplicateOf = &rootID
	expectDuplicateReads(repo, dup, primary, root)

	chain := []postgres.DuplicateNode{{ID: primaryID, DuplicateOf: &rootID}, {ID: rootID}}
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, dupID).
		Return([]postgres.DuplicateNode{{ID: dupID}}, nil).Once()
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, primaryID).Return(chain, nil).Twice()
	// Второй обход потомков — только по старой основной, после переcalc цепочки.
	repo.On("GetDuplicateDescendants", mock.Anything, mock.Anything, []uuid.UUID{primaryID}).
		Return([]uuid.UUID{primaryID, childID}, nil).Once()
	expectDuplicateTxFallback(repo)

	repo.On("RepointDuplicates", mock.Anything, mock.Anything, primaryID, rootID).Return(nil).Once()
	repo.On("MoveDuplicateNotifiers", mock.Anything, mock.Anything, primaryID, rootID).Return(nil).Once()
	// новую связь пишем прямо на корень, минуя промежуточную основную
	repo.On("SetDuplicate", mock.Anything, mock.Anything, dupID, rootID, models.StatusOpen).Return(nil).Once()
	repo.On("AddDuplicateNotifier", mock.Anything, mock.Anything, rootID, creator, dupID, actor.ID).Return(nil).Once()
	// repointed + dup_linked + status_changed
	logs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(3)
	notifications.On("NotifyDuplicate", mock.Anything, dup, root, actor.ID, true).Return(nil).Once()

	result, err := svc.LinkDuplicate(context.Background(), &models.LinkDuplicateDTO{
		ID: dupID, PrimaryID: primaryID, Actor: actor,
	})
	require.NoError(t, err)
	assert.Equal(t, rootID, result.RootID)
	repo.AssertExpectations(t)
	logs.AssertExpectations(t)
	notifications.AssertExpectations(t)
}

func TestTicketService_LinkDuplicate_CycleRejected(t *testing.T) {
	repo, _, _, _, svc, _, dupID, primaryID, creator, actor := duplicateFixtures(t)
	dup := duplicateTicket(dupID, creator, actor.ID, models.StatusOpen)
	primary := primaryTicket(primaryID, actor.ID, models.StatusOpen)
	expectDuplicateReads(repo, dup, primary)

	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, dupID).
		Return([]postgres.DuplicateNode{{ID: dupID}}, nil).Once()
	// Основная уже ведёт цепочку через дубль → замыкать нельзя.
	cycle := []postgres.DuplicateNode{{ID: primaryID, DuplicateOf: &dupID}, {ID: dupID}}
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, primaryID).Return(cycle, nil).Twice()
	expectDuplicateTxFallback(repo)

	_, err := svc.LinkDuplicate(context.Background(), &models.LinkDuplicateDTO{
		ID: dupID, PrimaryID: primaryID, Actor: actor,
	})
	assert.ErrorIs(t, err, models.ErrDuplicateCycle)
	repo.AssertNotCalled(t, "SetDuplicate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketService_LinkDuplicate_Rejections(t *testing.T) {
	tests := []struct {
		name      string
		status    models.TicketStatus
		primary   *models.Ticket
		primaryID func(dupID uuid.UUID) uuid.UUID
		wantErr   error
	}{
		{
			name:      "self",
			status:    models.StatusOpen,
			primaryID: func(dupID uuid.UUID) uuid.UUID { return dupID },
			wantErr:   models.ErrDuplicateOwnTicket,
		},
		{
			name:      "duplicate status not allowed",
			status:    models.StatusCancelled,
			primaryID: func(uuid.UUID) uuid.UUID { return uuid.New() },
			wantErr:   models.ErrDuplicateStatusNotAllowed,
		},
		{
			name:      "primary cancelled",
			status:    models.StatusOpen,
			primary:   &models.Ticket{ID: uuid.New(), Status: models.StatusCancelled},
			primaryID: func(uuid.UUID) uuid.UUID { return uuid.New() },
			wantErr:   models.ErrDuplicatePrimaryInactive,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, _, _, _, svc, _, dupID, _, creator, actor := duplicateFixtures(t)
			dup := duplicateTicket(dupID, creator, actor.ID, tc.status)
			expectDuplicateReads(repo, dup)
			primaryID := tc.primaryID(dupID)
			if tc.primary == nil {
				tc.primary = primaryTicket(primaryID, actor.ID, models.StatusOpen)
			}
			primaryID = tc.primary.ID
			primary := *tc.primary
			primary.Creator = models.UserShort{ID: actor.ID}
			expectDuplicateReads(repo, &primary)
			expectDuplicateTxFallback(repo)

			_, err := svc.LinkDuplicate(context.Background(), &models.LinkDuplicateDTO{
				ID: dupID, PrimaryID: primaryID, Actor: actor,
			})
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// Закрытая и решённая заявки теперь допустимы в качестве основной: дубль
// закрывается своей операцией, а unlink восстанавливает его previous_status.
func TestTicketService_LinkDuplicate_ClosedOrResolvedPrimaryAllowed(t *testing.T) {
	for _, status := range []models.TicketStatus{models.StatusClosed, models.StatusResolved} {
		t.Run(string(status), func(t *testing.T) {
			repo, logs, notifications, _, svc, _, dupID, primaryID, creator, actor := duplicateFixtures(t)
			dup := duplicateTicket(dupID, creator, actor.ID, models.StatusInProgress)
			primary := primaryTicket(primaryID, actor.ID, status)
			expectDuplicateReads(repo, dup, primary)

			repo.On("GetDuplicateChain", mock.Anything, mock.Anything, dupID).
				Return([]postgres.DuplicateNode{{ID: dupID}}, nil).Once()
			repo.On("GetDuplicateChain", mock.Anything, mock.Anything, primaryID).
				Return([]postgres.DuplicateNode{{ID: primaryID}}, nil).Twice()
			expectDuplicateTxFallback(repo)

			repo.On("SetDuplicate", mock.Anything, mock.Anything, dupID, primaryID, models.StatusInProgress).Return(nil).Once()
			repo.On("AddDuplicateNotifier", mock.Anything, mock.Anything, primaryID, creator, dupID, actor.ID).Return(nil).Once()
			logs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()
			notifications.On("NotifyDuplicate", mock.Anything, dup, primary, actor.ID, true).Return(nil).Once()

			_, err := svc.LinkDuplicate(context.Background(), &models.LinkDuplicateDTO{
				ID: dupID, PrimaryID: primaryID, Actor: actor,
			})
			require.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}

// Повторный выбор той же основной не должен переписывать связь и писать
// dup_repointed — но наблюдатели остаются привязанными к корню.
func TestTicketService_LinkDuplicate_NoOpRepoint(t *testing.T) {
	repo, logs, notifications, _, svc, _, dupID, primaryID, creator, actor := duplicateFixtures(t)
	previous := models.StatusInProgress
	dup := duplicateTicket(dupID, creator, actor.ID, models.StatusClosed)
	dup.DuplicateOf = &primaryID
	dup.PreviousStatus = &previous
	primary := primaryTicket(primaryID, actor.ID, models.StatusClosed)
	expectDuplicateReads(repo, dup, primary)

	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, dupID).
		Return([]postgres.DuplicateNode{{ID: dupID, DuplicateOf: &primaryID}, {ID: primaryID}}, nil).Once()
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, primaryID).
		Return([]postgres.DuplicateNode{{ID: primaryID}}, nil).Twice()
	expectDuplicateTxFallback(repo)

	repo.On("AddDuplicateNotifier", mock.Anything, mock.Anything, primaryID, creator, dupID, actor.ID).Return(nil).Once()
	notifications.On("NotifyDuplicate", mock.Anything, dup, primary, actor.ID, true).Return(nil).Once()

	_, err := svc.LinkDuplicate(context.Background(), &models.LinkDuplicateDTO{
		ID: dupID, PrimaryID: primaryID, Actor: actor,
	})
	require.NoError(t, err)
	repo.AssertNotCalled(t, "RepointOne", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "SetDuplicate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	logs.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
	notifications.AssertExpectations(t)
}

func TestTicketService_UnlinkDuplicate_RestoresStatus(t *testing.T) {
	repo, logs, notifications, _, svc, _, dupID, primaryID, creator, actor := duplicateFixtures(t)
	previous := models.StatusInProgress
	resolvedAt := time.Now().Add(-time.Hour)
	dup := duplicateTicket(dupID, creator, actor.ID, models.StatusClosed)
	dup.DuplicateOf = &primaryID
	dup.PreviousStatus = &previous
	dup.ResolvedAt = &resolvedAt
	primary := primaryTicket(primaryID, actor.ID, models.StatusOpen)
	expectDuplicateReads(repo, dup, primary)
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, primaryID).
		Return([]postgres.DuplicateNode{{ID: primaryID}}, nil).Once()
	expectDuplicateTxFallback(repo)

	repo.On("ClearDuplicate", mock.Anything, mock.Anything, dupID).Return(nil).Once()
	repo.On("DeleteDuplicateNotifiers", mock.Anything, mock.Anything, dupID).Return(nil).Once()
	logs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()
	notifications.On("NotifyDuplicate", mock.Anything, dup, primary, actor.ID, false).Return(nil).Once()

	result, err := svc.UnlinkDuplicate(context.Background(), &models.UnlinkDuplicateDTO{ID: dupID, Actor: actor})
	require.NoError(t, err)
	assert.Equal(t, primaryID, result.RootID)
	repo.AssertExpectations(t)
	logs.AssertExpectations(t)
	notifications.AssertExpectations(t)
}

// Дочерний дубль после flatten-link указывает прямо на корень. Отвязка такого
// дубля не должна переписывать связи остальных дублей корня.
func TestTicketService_UnlinkDuplicate_ChildAfterFlatten(t *testing.T) {
	repo, logs, notifications, _, svc, _, dupID, rootID, creator, actor := duplicateFixtures(t)
	previous := models.StatusOpen
	dup := duplicateTicket(dupID, creator, actor.ID, models.StatusClosed)
	dup.DuplicateOf = &rootID // после flatten промежуточная основная исчезла
	dup.PreviousStatus = &previous
	root := primaryTicket(rootID, creator, models.StatusOpen)
	expectDuplicateReads(repo, dup, root)
	repo.On("GetDuplicateChain", mock.Anything, mock.Anything, rootID).
		Return([]postgres.DuplicateNode{{ID: rootID}}, nil).Once()
	expectDuplicateTxFallback(repo)

	repo.On("ClearDuplicate", mock.Anything, mock.Anything, dupID).Return(nil).Once()
	repo.On("DeleteDuplicateNotifiers", mock.Anything, mock.Anything, dupID).Return(nil).Once()
	logs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()
	notifications.On("NotifyDuplicate", mock.Anything, dup, root, actor.ID, false).Return(nil).Maybe()

	_, err := svc.UnlinkDuplicate(context.Background(), &models.UnlinkDuplicateDTO{ID: dupID, Actor: actor})
	require.NoError(t, err)
	repo.AssertNotCalled(t, "RepointDuplicates", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "RepointOne", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestTicketService_UnlinkDuplicate_NotLinked(t *testing.T) {
	repo, _, _, _, svc, _, dupID, _, creator, actor := duplicateFixtures(t)
	dup := duplicateTicket(dupID, creator, actor.ID, models.StatusClosed)
	expectDuplicateReads(repo, dup)

	_, err := svc.UnlinkDuplicate(context.Background(), &models.UnlinkDuplicateDTO{ID: dupID, Actor: actor})
	assert.ErrorIs(t, err, models.ErrDuplicateNotLinked)
}

func TestTicketService_GetDuplicates_FiltersByReadAccess(t *testing.T) {
	repo, _, _, _, _, groups, policies, svc, _ := ticketServiceFixturesWithFavorites()
	actor := &models.Actor{ID: uuid.New(), Name: "Исполнитель"}
	primaryID, dupID, hiddenID := uuid.New(), uuid.New(), uuid.New()
	number := 42
	created := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	previous := models.StatusClosed

	primary := primaryTicket(primaryID, actor.ID, models.StatusOpen)
	dup := duplicateTicket(dupID, actor.ID, actor.ID, models.StatusClosed)
	dup.TicketNumber = &number
	dup.CreatedAt = created
	dup.PreviousStatus = &previous
	dup.Creator = models.UserShort{ID: actor.ID, FirstName: "Пётр", LastName: "Иванов"}
	dup.Assignee = &models.UserShort{ID: actor.ID, FirstName: "Илья", LastName: "Сидоров"}
	hidden := primaryTicket(hiddenID, uuid.New(), models.StatusClosed)

	policies.On("Enforce", actor.ID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil).Maybe()
	policies.On("Enforce", actor.ID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil).Maybe()
	markUserAsEmployee(groups, actor.ID)

	expectDuplicateReads(repo, primary, dup, hidden)
	repo.On("GetDuplicateDescendants", mock.Anything, mock.Anything, []uuid.UUID{primaryID}).
		Return([]uuid.UUID{primaryID, dupID, hiddenID}, nil).Once()
	repo.On("CountDuplicateActivity", mock.Anything, mock.Anything).
		Return(map[uuid.UUID]postgres.DuplicateActivityCounts{
			dupID: {Comments: 3, Attachments: 2},
		}, nil).Once()

	refs, err := svc.GetDuplicates(context.Background(), &models.GetDuplicatesDTO{ID: primaryID, Actor: actor})
	require.NoError(t, err)
	require.Len(t, refs, 1, "дубли без read-доступа не попадают в список")
	assert.Equal(t, dupID, refs[0].ID)
	assert.Equal(t, number, *refs[0].TicketNumber)
	assert.Equal(t, "Дубль", refs[0].Title)
	assert.Equal(t, "Иванов Пётр", refs[0].CreatorName)
	assert.Equal(t, "Сидоров Илья", refs[0].AssigneeName)
	assert.Equal(t, 3, refs[0].CommentsCount)
	assert.Equal(t, 2, refs[0].AttachmentsCount)
	assert.Equal(t, created, refs[0].CreatedAt)
	assert.True(t, refs[0].CanUnlink, "исполнитель дубля видит кнопку отвязки")
	repo.AssertExpectations(t)
}

// Право отвязки считается для каждого дубля отдельно: автор основной без
// рабочего доступа к чужому дублю кнопку не видит.
func TestTicketService_GetDuplicates_CanUnlinkPerChild(t *testing.T) {
	repo, _, _, _, _, groups, policies, svc, _ := ticketServiceFixturesWithFavorites()
	actor := &models.Actor{ID: uuid.New(), Name: "Наблюдатель"}
	primaryID, dupID := uuid.New(), uuid.New()

	primary := primaryTicket(primaryID, actor.ID, models.StatusOpen)
	dup := duplicateTicket(dupID, uuid.New(), uuid.New(), models.StatusClosed)
	dup.Notifiers = []models.UserShort{{ID: actor.ID}}

	policies.On("Enforce", actor.ID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil).Maybe()
	policies.On("Enforce", actor.ID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil).Maybe()
	// Наблюдатель не является сотрудником и не управляет группой основной.
	groups.On("GetManagedGroups", mock.Anything, actor.ID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil).Maybe()
	groups.On("GetMemberGroups", mock.Anything, actor.ID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil).Maybe()

	expectDuplicateReads(repo, primary, dup)
	repo.On("GetDuplicateDescendants", mock.Anything, mock.Anything, []uuid.UUID{primaryID}).
		Return([]uuid.UUID{primaryID, dupID}, nil).Once()
	repo.On("CountDuplicateActivity", mock.Anything, mock.Anything).
		Return(map[uuid.UUID]postgres.DuplicateActivityCounts{}, nil).Once()

	refs, err := svc.GetDuplicates(context.Background(), &models.GetDuplicatesDTO{ID: primaryID, Actor: actor})
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.False(t, refs[0].CanUnlink)
	repo.AssertExpectations(t)
}

func TestTicketService_GetDuplicates_PrimaryAccessDenied(t *testing.T) {
	repo, _, _, _, _, _, policies, svc, _ := ticketServiceFixturesWithFavorites()
	actor := &models.Actor{ID: uuid.New(), Name: "Чужой"}
	primaryID := uuid.New()

	policies.On("Enforce", actor.ID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil).Once()
	policies.On("Enforce", actor.ID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil).Once()
	expectDuplicateReads(repo, primaryTicket(primaryID, uuid.New(), models.StatusOpen))

	_, err := svc.GetDuplicates(context.Background(), &models.GetDuplicatesDTO{ID: primaryID, Actor: actor})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	repo.AssertNotCalled(t, "GetDuplicateDescendants", mock.Anything, mock.Anything, mock.Anything)
}

// Менеджер основной видит кнопку «Отвязать» и в карточке самого дубля —
// AccessFlags должен совпадать с проверкой в UnlinkDuplicate.
func TestTicketService_GetAccessFlags_PrimaryManagerCanUnlinkChild(t *testing.T) {
	repo, _, _, _, _, groups, policies, svc, _ := ticketServiceFixturesWithFavorites()
	userID := uuid.New()
	realmID := uuid.New()
	primaryID := uuid.New()
	primaryGroupID := uuid.New()
	dupGroupID := uuid.New()

	dup := &models.Ticket{
		ID: uuid.New(), Title: "Дубль", Status: models.StatusClosed,
		RealmID: &realmID, Group: &models.GroupShort{ID: dupGroupID},
		Creator:     models.UserShort{ID: uuid.New()},
		Notifiers:   []models.UserShort{{ID: userID}},
		DuplicateOf: &primaryID,
	}
	primary := primaryTicket(primaryID, uuid.New(), models.StatusOpen)
	primary.RealmID = &realmID
	primary.Group = &models.GroupShort{ID: primaryGroupID}
	expectDuplicateReads(repo, primary)

	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil).Maybe()
	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil).Maybe()
	// Управляет группой основной, но не группой дубля и не состоит в ней.
	groups.On("GetManagedGroups", mock.Anything, userID, (*uuid.UUID)(nil)).Return([]uuid.UUID{primaryGroupID}, nil).Maybe()
	groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{primaryGroupID}, nil).Maybe()
	groups.On("GetMemberGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{}, nil).Maybe()
	groups.On("IsMember", mock.Anything, dupGroupID, userID).Return(false, nil).Maybe()

	flags, err := svc.GetAccessFlags(context.Background(), dup, userID)
	require.NoError(t, err)
	assert.False(t, flags.CanLinkDuplicate)
	assert.True(t, flags.CanUnlinkDuplicate)
}

func TestUserService_SkipDuplicatePrimaries(t *testing.T) {
	repo := new(MockTicketsRepo)
	svc := &userService{tickets: repo}
	keepID, skipID := uuid.New(), uuid.New()
	skipName := "ivanov"
	repo.On("GetIDsWithDuplicatePrimaries", mock.Anything, []uuid.UUID{keepID, skipID}).
		Return([]uuid.UUID{skipID}, nil).Once()

	kept, skipped, err := svc.skipDuplicatePrimaries(context.Background(),
		[]uuid.UUID{keepID, skipID}, map[uuid.UUID]string{skipID: skipName})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{keepID}, kept)
	assert.Equal(t, []string{skipName}, skipped)
	repo.AssertExpectations(t)

	// Сбой проверки не должен молча удалять пользователей.
	repo.On("GetIDsWithDuplicatePrimaries", mock.Anything, []uuid.UUID{keepID}).
		Return(nil, errors.New("db is down")).Once()
	_, _, err = svc.skipDuplicatePrimaries(context.Background(), []uuid.UUID{keepID}, nil)
	assert.Error(t, err)
}

// fixedTxManager отдаёт одну и ту же заглушку транзакции в callback: так видно,
// что и блокировка, и проверка ссылок идут внутри одной транзакции.
type fixedTxManager struct{ tx postgres.Tx }

func (m fixedTxManager) WithinTransaction(_ context.Context, fn func(postgres.Tx) error) error {
	return fn(m.tx)
}

func TestTicketService_Delete_PrimaryWithDuplicatesRejected(t *testing.T) {
	repo, _, _, _, _, _, policies, svc := ticketServiceFixtures()
	actorID, ticketID := uuid.New(), uuid.New()
	dto := &models.DeleteTicketDTO{ID: ticketID, Actor: &models.Actor{ID: actorID}}
	tx := fakeTx{}
	svc.tx = fixedTxManager{tx: tx}

	policies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(true, nil)
	policies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	repo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).
		Return(&models.Ticket{ID: ticketID, Title: "Основная"}, nil).Once()

	var order []string
	repo.On("LockTickets", mock.Anything, tx, []uuid.UUID{ticketID}).
		Run(func(mock.Arguments) { order = append(order, "lock") }).Return(nil).Once()
	repo.On("HasDuplicateReferences", mock.Anything, tx, ticketID).
		Run(func(mock.Arguments) { order = append(order, "check") }).Return(true, nil).Once()

	err := svc.Delete(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrDuplicatePrimaryHasDuplicates)
	assert.Equal(t, []string{"lock", "check"}, order, "ссылки проверяются в транзакции после блокировки родителя")
	repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

// Номер основной заявки приходит вместе с дублем из SQL-джойна, минуя проверку
// доступа. SERV-гейт обязан убрать и номер, и саму ссылку, если основную нельзя
// прочитать; при ошибке загрузки/проверки — тоже убрать (fail closed).
func TestTicketService_HideInaccessibleDuplicateRef(t *testing.T) {
	ctx := context.Background()
	actorID, primaryID := uuid.New(), uuid.New()
	number := 42
	dup := func() *models.Ticket {
		return &models.Ticket{ID: uuid.New(), DuplicateOf: &primaryID, DuplicateOfNumber: &number}
	}

	t.Run("readable primary keeps the reference", func(t *testing.T) {
		repo, _, _, _, _, _, _, svc := ticketServiceFixtures()
		primary := primaryTicket(primaryID, actorID, models.StatusOpen)
		repo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: primaryID}).Return(primary, nil).Once()

		ticket := dup()
		svc.hideInaccessibleDuplicateRef(ctx, ticket, actorID)
		assert.Equal(t, &primaryID, ticket.DuplicateOf)
		assert.Equal(t, &number, ticket.DuplicateOfNumber)
		repo.AssertExpectations(t)
	})

	t.Run("foreign primary is hidden", func(t *testing.T) {
		repo, _, _, _, _, _, policies, svc := ticketServiceFixtures()
		primary := primaryTicket(primaryID, uuid.New(), models.StatusClosed)
		repo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: primaryID}).Return(primary, nil).Once()
		policies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil).Once()
		policies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil).Once()

		ticket := dup()
		svc.hideInaccessibleDuplicateRef(ctx, ticket, actorID)
		assert.Nil(t, ticket.DuplicateOf)
		assert.Nil(t, ticket.DuplicateOfNumber)
		repo.AssertExpectations(t)
		policies.AssertExpectations(t)
	})

	t.Run("load failure fails closed", func(t *testing.T) {
		repo, _, _, _, _, _, _, svc := ticketServiceFixtures()
		repo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: primaryID}).
			Return(nil, errors.New("db is down")).Once()

		ticket := dup()
		svc.hideInaccessibleDuplicateRef(ctx, ticket, actorID)
		assert.Nil(t, ticket.DuplicateOf)
		assert.Nil(t, ticket.DuplicateOfNumber)
		repo.AssertExpectations(t)
	})

	t.Run("list checks each primary once", func(t *testing.T) {
		repo, _, _, _, _, _, policies, svc := ticketServiceFixtures()
		primary := primaryTicket(primaryID, uuid.New(), models.StatusClosed)
		repo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: primaryID}).Return(primary, nil).Once()
		policies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil).Once()
		policies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil).Once()

		tickets := []*models.Ticket{dup(), dup()}
		svc.hideInaccessibleDuplicateRefs(ctx, tickets, actorID)
		for _, ticket := range tickets {
			assert.Nil(t, ticket.DuplicateOf)
			assert.Nil(t, ticket.DuplicateOfNumber)
		}
		repo.AssertExpectations(t)
		policies.AssertExpectations(t)
	})
}

// Потомок, связанный параллельно между первым обходом и захватом локов,
// должен быть найден и заблокирован на следующем витке, иначе repoint/журнал
// считались бы по устаревшему снимку графа.
func TestTicketService_lockDuplicateTree_LocksFreshDescendants(t *testing.T) {
	repo, _, _, _, _, _, _, svc := ticketServiceFixtures()
	rootID, childID, grandchildID := uuid.New(), uuid.New(), uuid.New()

	var order []string
	repo.On("LockTickets", mock.Anything, mock.Anything, []uuid.UUID{rootID}).
		Run(func(mock.Arguments) { order = append(order, "lock:root") }).Return(nil).Once()
	repo.On("GetDuplicateDescendants", mock.Anything, mock.Anything, []uuid.UUID{rootID}).
		Run(func(mock.Arguments) { order = append(order, "scan:1") }).
		Return([]uuid.UUID{rootID, childID}, nil).Once()
	repo.On("LockTickets", mock.Anything, mock.Anything, []uuid.UUID{childID}).
		Run(func(mock.Arguments) { order = append(order, "lock:child") }).Return(nil).Once()
	repo.On("GetDuplicateDescendants", mock.Anything, mock.Anything, []uuid.UUID{rootID, childID}).
		Run(func(mock.Arguments) { order = append(order, "scan:2") }).
		Return([]uuid.UUID{rootID, childID, grandchildID}, nil).Once()
	repo.On("LockTickets", mock.Anything, mock.Anything, []uuid.UUID{grandchildID}).Return(nil).Once()
	repo.On("GetDuplicateDescendants", mock.Anything, mock.Anything, []uuid.UUID{rootID, childID, grandchildID}).
		Return([]uuid.UUID{rootID, childID, grandchildID}, nil).Once()

	require.NoError(t, svc.lockDuplicateTree(context.Background(), nil, []uuid.UUID{rootID}))
	assert.Equal(t, []string{"lock:root", "scan:1", "lock:child", "scan:2"}, order)
	repo.AssertExpectations(t)
}

// Снапшот dup_* в «Истории» должен показывать № заявки, а не сырой uuid:
// activityMeta.ts рендерит эти значения как plain. Уведомления при этом
// продолжают получать id в Data — меняется только журнал.
func TestTicketService_writeDuplicateLogs_HumanizesTicketIDs(t *testing.T) {
	repo, logs, _, _, _, _, _, svc := ticketServiceFixtures()
	primaryID, duplicateID := uuid.New(), uuid.New()
	number := 789
	actor := &models.Actor{ID: uuid.New(), Name: "Исполнитель"}
	primary := primaryTicket(primaryID, uuid.New(), models.StatusOpen)
	primary.TicketNumber = &number
	expectDuplicateReads(repo, primary)

	var oldValues, newValues map[string]string
	logs.On("Create", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			entries := args.Get(2).([]*models.ActivityLogDTO)
			require.Len(t, entries, 1)
			require.NoError(t, json.Unmarshal(*entries[0].OldValues, &oldValues))
			require.NoError(t, json.Unmarshal(*entries[0].NewValues, &newValues))
		}).Return(nil).Once()

	require.NoError(t, svc.writeDuplicateLogs(context.Background(), nil, actor,
		models.ActionDuplicateUnlinked, []uuid.UUID{duplicateID}, primaryID.String(), "none"))
	assert.Equal(t, map[string]string{"dup_unlinked": "№789"}, oldValues)
	assert.Equal(t, map[string]string{"dup_unlinked": "none"}, newValues)
	repo.AssertExpectations(t)
	logs.AssertExpectations(t)
}
