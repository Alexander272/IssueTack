package tickets

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTicketFilter_BindUUIDParams защищает биндинг uuid-полей фильтра из query-параметров.
// Gin не умеет писать uuid.UUID ([16]byte) в форму без тега parser — без него на любой
// непустой assigneeId/ownerId приходит 400 «is not valid value for uuid.UUID».
func TestTicketFilter_BindUUIDParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		require.NoError(t, v.RegisterValidation("enum", models.UniversalEnumValidator))
	}

	router := gin.New()
	router.GET("/bind", func(c *gin.Context) {
		filter := &models.TicketFilter{}
		if err := c.ShouldBindQuery(filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, filter)
	})

	const ownerUUID = "0f3d4a5b-1111-4222-8333-444455556666"
	const assigneeUUID = "1f4e5a6b-2222-4333-8444-555566667777"
	const realmUUID = "2f5e6a7b-3333-4333-8444-555566667777"

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bind?ownerId="+ownerUUID+"&assigneeId="+assigneeUUID+"&realmId="+realmUUID+"&mode=created", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.TicketFilter
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, uuid.MustParse(ownerUUID), *got.OwnerID)
	assert.Equal(t, uuid.MustParse(assigneeUUID), *got.AssigneeID)
	assert.Equal(t, uuid.MustParse(realmUUID), *got.RealmID)
	assert.NotNil(t, got.Mode)
	assert.Equal(t, "created", *got.Mode)
}

func TestNormalizeTicketFilter_StatusesCSV(t *testing.T) {
	filter := &models.TicketFilter{
		Statuses: []models.TicketStatus{"open,in_progress"},
	}

	require.NoError(t, normalizeTicketFilter(filter))
	assert.Equal(t, []models.TicketStatus{models.StatusOpen, models.StatusInProgress}, filter.Statuses)
}

func TestNormalizeTicketFilter_MultiValueFilters(t *testing.T) {
	filter := &models.TicketFilter{
		Statuses:   []models.TicketStatus{"open", "in_progress,pending"},
		Priorities: []models.Priority{"high,urgent"},
		SiteIDs:    []string{"site-a,site-b", "site-c"},
	}

	require.NoError(t, normalizeTicketFilter(filter))
	assert.Equal(t, []models.TicketStatus{models.StatusOpen, models.StatusInProgress, models.StatusPending}, filter.Statuses)
	assert.Equal(t, []models.Priority{models.PriorityHigh, models.PriorityUrgent}, filter.Priorities)
	assert.Equal(t, []string{"site-a", "site-b", "site-c"}, filter.SiteIDs)
}

func TestNormalizeTicketFilter_InvalidStatus(t *testing.T) {
	filter := &models.TicketFilter{Statuses: []models.TicketStatus{"open,bogus"}}

	err := normalizeTicketFilter(filter)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

func TestNormalizeTicketFilter_InvalidPriority(t *testing.T) {
	filter := &models.TicketFilter{Priorities: []models.Priority{"high,bogus"}}

	err := normalizeTicketFilter(filter)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

func TestTicketFilter_BindListAndScalarParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		require.NoError(t, v.RegisterValidation("enum", models.UniversalEnumValidator))
	}

	router := gin.New()
	router.GET("/bind", func(c *gin.Context) {
		filter := &models.TicketFilter{}
		if err := c.ShouldBindQuery(filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := normalizeTicketFilter(filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, filter)
	})

	equipmentID := "2f5e6a7b-3333-4333-8444-555566667777"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/bind?statuses=open%2Cin_progress&priorities=high&priorities=urgent%2Clow&siteIds=s1%2Cs2"+
			"&number=42&search=test&sort=status_asc&mode=created_or_owned&archived=true&allStatuses=1"+
			"&dueDateFrom=2026-11-01&dueDateTo=2026-11-30&closedAtFrom=2026-10-01&closedAtTo=2026-10-31"+
			"&equipmentId="+equipmentID+"&limit=20&offset=40",
		nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.TicketFilter
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))

	require.NotNil(t, got.Number)
	assert.Equal(t, 42, *got.Number)
	assert.Equal(t, []models.TicketStatus{models.StatusOpen, models.StatusInProgress}, got.Statuses)
	assert.Equal(t, []models.Priority{models.PriorityHigh, models.PriorityUrgent, models.PriorityLow}, got.Priorities)
	assert.Equal(t, []string{"s1", "s2"}, got.SiteIDs)
	require.NotNil(t, got.Search)
	assert.Equal(t, "test", *got.Search)
	require.NotNil(t, got.Sort)
	assert.Equal(t, "status_asc", *got.Sort)
	require.NotNil(t, got.Mode)
	assert.Equal(t, "created_or_owned", *got.Mode)
	require.NotNil(t, got.Archived)
	assert.True(t, *got.Archived)
	require.NotNil(t, got.DueDateFrom)
	assert.Equal(t, "2026-11-01", got.DueDateFrom.Format("2006-01-02"))
	require.NotNil(t, got.DueDateTo)
	assert.Equal(t, "2026-11-30", got.DueDateTo.Format("2006-01-02"))
	require.NotNil(t, got.ClosedAtFrom)
	assert.Equal(t, "2026-10-01", got.ClosedAtFrom.Format("2006-01-02"))
	require.NotNil(t, got.ClosedAtTo)
	assert.Equal(t, "2026-10-31", got.ClosedAtTo.Format("2006-01-02"))
	require.NotNil(t, got.EquipmentID)
	assert.Equal(t, uuid.MustParse(equipmentID), *got.EquipmentID)
	assert.Equal(t, 20, got.Limit)
	assert.Equal(t, 40, got.Offset)
}
