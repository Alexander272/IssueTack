package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func checkListFixtures() (*MockChecklistsRepo, *MockSubtaskService, *MockAccessPolicies, *ChecklistService) {
	mockRepo := new(MockChecklistsRepo)
	mockSubtasks := new(MockSubtaskService)
	mockPolicies := new(MockAccessPolicies)
	svc := NewChecklistService(mockRepo, mockSubtasks, mockPolicies)
	return mockRepo, mockSubtasks, mockPolicies, svc
}

func TestChecklistService_Get_WithoutReadPerm_OnlyOwn(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actor := &models.Actor{ID: uuid.New(), Name: "test"}
	req := &models.GetChecklistTemplatesDTO{RealmID: uuid.New(), Actor: actor}
	expected := []*models.ChecklistTemplate{{ID: uuid.New(), Title: "Template 1", CreatedBy: &actor.ID}}

	mockPolicies.On("Enforce", actor.ID.String(), req.RealmID.String(), string(access.ResourceChecklist), string(access.Read)).Return(false, nil)
	// Без права checklist:read сервис подставляет фильтр по владельцу и отдаёт только свои.
	mockRepo.On("Get", mock.Anything, mock.MatchedBy(func(r *models.GetChecklistTemplatesDTO) bool {
		return r.OwnerID != nil && *r.OwnerID == actor.ID
	})).Return(expected, nil)

	got, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestChecklistService_Get_WithReadPerm_All(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actor := &models.Actor{ID: uuid.New(), Name: "test"}
	req := &models.GetChecklistTemplatesDTO{RealmID: uuid.New(), Actor: actor}
	expected := []*models.ChecklistTemplate{{ID: uuid.New(), Title: "Template 1"}}

	mockPolicies.On("Enforce", actor.ID.String(), req.RealmID.String(), string(access.ResourceChecklist), string(access.Read)).Return(true, nil)
	mockRepo.On("Get", mock.Anything, mock.MatchedBy(func(r *models.GetChecklistTemplatesDTO) bool {
		return r.OwnerID == nil
	})).Return(expected, nil)

	got, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestChecklistService_GetByID_OwnTemplate(t *testing.T) {
	mockRepo, _, _, svc := checkListFixtures()

	actorID := uuid.New()
	templateID := uuid.New()
	req := &models.GetChecklistTemplateDTO{ID: templateID}
	template := &models.ChecklistTemplate{ID: templateID, Title: "Template", RealmID: uuid.New(), CreatedBy: &actorID}
	items := []*models.ChecklistTemplateItem{{ID: uuid.New(), Title: "Item 1"}}

	mockRepo.On("GetByID", mock.Anything, req).Return(template, nil)
	mockRepo.On("GetItems", mock.Anything, templateID).Return(items, nil)

	got, err := svc.GetByID(context.Background(), req, actorID, "")
	assert.NoError(t, err)
	assert.Equal(t, items, got.Items)
}

func TestChecklistService_GetByID_Forbidden(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actorID := uuid.New()
	templateID := uuid.New()
	template := &models.ChecklistTemplate{ID: templateID, Title: "Template", RealmID: uuid.New(), CreatedBy: ptr(uuid.New())}

	mockRepo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	mockPolicies.On("Enforce", actorID.String(), mock.Anything, string(access.ResourceChecklist), string(access.Read)).Return(false, nil)

	_, err := svc.GetByID(context.Background(), &models.GetChecklistTemplateDTO{ID: templateID}, actorID, "")
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestChecklistService_GetByID_WithPerm(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actorID := uuid.New()
	templateID := uuid.New()
	template := &models.ChecklistTemplate{ID: templateID, Title: "Template", RealmID: uuid.New(), CreatedBy: ptr(uuid.New())}
	items := []*models.ChecklistTemplateItem{{ID: uuid.New(), Title: "Item 1"}}

	mockRepo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	mockPolicies.On("Enforce", actorID.String(), template.RealmID.String(), string(access.ResourceChecklist), string(access.Read)).Return(true, nil)
	mockRepo.On("GetItems", mock.Anything, templateID).Return(items, nil)

	got, err := svc.GetByID(context.Background(), &models.GetChecklistTemplateDTO{ID: templateID}, actorID, "")
	assert.NoError(t, err)
	assert.Equal(t, items, got.Items)
}

func TestChecklistService_Create_Success(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actor := &models.Actor{ID: uuid.New(), Name: "test"}
	dto := &models.ChecklistTemplateDTO{RealmID: uuid.New(), Title: "New Template", Actor: actor}

	mockPolicies.On("Enforce", actor.ID.String(), dto.RealmID.String(), string(access.ResourceChecklist), string(access.Write)).Return(false, nil)
	mockRepo.On("ExistsByTitle", mock.Anything, dto.RealmID, "New Template", &actor.ID).Return(false, nil)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(d *models.ChecklistTemplateDTO) bool {
		return d.CreatedBy != nil && *d.CreatedBy == actor.ID
	})).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
	assert.NotNil(t, dto.CreatedBy)
}

func TestChecklistService_Create_NameExists(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actor := &models.Actor{ID: uuid.New(), Name: "test"}
	dto := &models.ChecklistTemplateDTO{RealmID: uuid.New(), Title: "Duplicated", Actor: actor}

	mockPolicies.On("Enforce", actor.ID.String(), dto.RealmID.String(), string(access.ResourceChecklist), string(access.Write)).Return(false, nil)
	mockRepo.On("ExistsByTitle", mock.Anything, dto.RealmID, "Duplicated", &actor.ID).Return(true, nil)

	err := svc.Create(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrTemplateNameExists)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestChecklistService_Create_EmptyTitle(t *testing.T) {
	_, _, _, svc := checkListFixtures()

	actor := &models.Actor{ID: uuid.New(), Name: "test"}
	err := svc.Create(context.Background(), &models.ChecklistTemplateDTO{RealmID: uuid.New(), Title: "  ", Actor: actor})
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

func TestChecklistService_Create_NoActor(t *testing.T) {
	_, _, _, svc := checkListFixtures()

	err := svc.Create(context.Background(), &models.ChecklistTemplateDTO{RealmID: uuid.New(), Title: "New"})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestChecklistService_Update_Owner(t *testing.T) {
	mockRepo, _, _, svc := checkListFixtures()

	actorID := uuid.New()
	dto := &models.ChecklistTemplateDTO{ID: uuid.New(), Title: "Updated"}
	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: dto.ID}).Return(
		&models.ChecklistTemplate{ID: dto.ID, RealmID: uuid.New(), CreatedBy: &actorID}, nil)
	mockRepo.On("Update", mock.Anything, dto).Return(nil)

	err := svc.Update(context.Background(), dto, actorID, "")
	assert.NoError(t, err)
}

func TestChecklistService_Update_Forbidden(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actorID := uuid.New()
	templateID := uuid.New()
	realmID := uuid.New()
	dto := &models.ChecklistTemplateDTO{ID: templateID, Title: "Updated"}
	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: dto.ID}).Return(
		&models.ChecklistTemplate{ID: dto.ID, RealmID: realmID, CreatedBy: ptr(uuid.New())}, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceChecklist), string(access.Write)).Return(false, nil)

	err := svc.Update(context.Background(), dto, actorID, "")
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestChecklistService_Update_WithWritePerm(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	dto := &models.ChecklistTemplateDTO{ID: uuid.New(), Title: "Updated", RealmID: realmID}
	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: dto.ID}).Return(
		&models.ChecklistTemplate{ID: dto.ID, RealmID: realmID, CreatedBy: ptr(uuid.New())}, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceChecklist), string(access.Write)).Return(true, nil)
	mockRepo.On("Update", mock.Anything, dto).Return(nil)

	err := svc.Update(context.Background(), dto, actorID, "")
	assert.NoError(t, err)
}

func TestChecklistService_Delete_Forbidden(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	actorID := uuid.New()
	dto := &models.DelChecklistTemplateDTO{ID: uuid.New()}
	realmID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: dto.ID}).Return(
		&models.ChecklistTemplate{ID: dto.ID, RealmID: realmID, CreatedBy: ptr(uuid.New())}, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceChecklist), string(access.Delete)).Return(false, nil)

	err := svc.Delete(context.Background(), dto, actorID, "")
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestChecklistService_SetItems_Owner(t *testing.T) {
	mockRepo, _, _, svc := checkListFixtures()

	actorID := uuid.New()
	templateID := uuid.New()
	items := []*models.ChecklistTemplateItemDTO{{Title: "Item"}}
	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: templateID}).Return(
		&models.ChecklistTemplate{ID: templateID, RealmID: uuid.New(), CreatedBy: &actorID}, nil)
	mockRepo.On("SetItems", mock.Anything, nil, templateID, items).Return(nil)

	err := svc.SetItems(context.Background(), nil, templateID, items, actorID, "")
	assert.NoError(t, err)
}

func TestChecklistService_GetItems_Own(t *testing.T) {
	mockRepo, _, _, svc := checkListFixtures()

	actorID := uuid.New()
	templateID := uuid.New()
	expected := []*models.ChecklistTemplateItem{{ID: uuid.New(), Title: "Item"}}
	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: templateID}).Return(
		&models.ChecklistTemplate{ID: templateID, RealmID: uuid.New(), CreatedBy: &actorID}, nil)
	mockRepo.On("GetItems", mock.Anything, templateID).Return(expected, nil)

	got, err := svc.GetItems(context.Background(), templateID, actorID, "")
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestChecklistService_ApplyTemplate_Success(t *testing.T) {
	mockRepo, mockSubtasks, _, svc := checkListFixtures()

	ticketID := uuid.New()
	templateID := uuid.New()
	actor := &models.Actor{ID: uuid.New(), Name: "test"}
	items := []*models.ChecklistTemplateItem{
		{ID: uuid.New(), Title: "Task 1", SortOrder: 1},
		{ID: uuid.New(), Title: "Task 2", SortOrder: 2},
	}

	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: templateID}).Return(
		&models.ChecklistTemplate{ID: templateID, RealmID: uuid.New(), CreatedBy: &actor.ID}, nil)
	mockRepo.On("GetItems", mock.Anything, templateID).Return(items, nil)
	mockSubtasks.On("CreateSeveral", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.ApplyTemplate(context.Background(), nil, &models.ApplyTemplateDTO{TicketID: ticketID, TemplateID: templateID, Actor: actor})
	assert.NoError(t, err)
}

func TestChecklistService_ApplyTemplate_Forbidden(t *testing.T) {
	mockRepo, _, mockPolicies, svc := checkListFixtures()

	templateID := uuid.New()
	actor := &models.Actor{ID: uuid.New(), Name: "test"}
	tpl := &models.ChecklistTemplate{ID: templateID, RealmID: uuid.New(), CreatedBy: ptr(uuid.New())}

	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: templateID}).Return(tpl, nil)
	mockPolicies.On("Enforce", actor.ID.String(), tpl.RealmID.String(), string(access.ResourceChecklist), string(access.Read)).Return(false, nil)

	err := svc.ApplyTemplate(context.Background(), nil, &models.ApplyTemplateDTO{TicketID: uuid.New(), TemplateID: templateID, Actor: actor})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestChecklistService_ApplyTemplate_EmptyItems(t *testing.T) {
	mockRepo, mockSubtasks, _, svc := checkListFixtures()

	ticketID := uuid.New()
	templateID := uuid.New()
	actor := &models.Actor{ID: uuid.New(), Name: "test"}

	mockRepo.On("GetByID", mock.Anything, &models.GetChecklistTemplateDTO{ID: templateID}).Return(
		&models.ChecklistTemplate{ID: templateID, RealmID: uuid.New(), CreatedBy: &actor.ID}, nil)
	mockRepo.On("GetItems", mock.Anything, templateID).Return([]*models.ChecklistTemplateItem{}, nil)

	err := svc.ApplyTemplate(context.Background(), nil, &models.ApplyTemplateDTO{TicketID: ticketID, TemplateID: templateID, Actor: actor})
	assert.NoError(t, err)
	mockSubtasks.AssertNotCalled(t, "CreateSeveral")
}

func ptr[T any](v T) *T {
	return &v
}
