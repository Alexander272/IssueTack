package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCalendarRepo struct {
	daysByYear map[int][]models.CalendarDay
	getErr     error
	upsertErr  error
	upserted   [][]models.CalendarDay
}

func (m *mockCalendarRepo) Upsert(_ context.Context, days []models.CalendarDay) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	cp := append([]models.CalendarDay(nil), days...)
	m.upserted = append(m.upserted, cp)
	for _, d := range days {
		if m.daysByYear == nil {
			m.daysByYear = map[int][]models.CalendarDay{}
		}
		m.daysByYear[d.Year] = append(m.daysByYear[d.Year], d)
	}
	return nil
}

func (m *mockCalendarRepo) GetByYears(_ context.Context, years []int) ([]models.CalendarDay, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	var out []models.CalendarDay
	for _, y := range years {
		out = append(out, m.daysByYear[y]...)
	}
	return out, nil
}

func newTestCalendar(loc *time.Location, enabled bool) *workCalendar {
	if loc == nil {
		loc = time.UTC
	}
	return &workCalendar{
		repo:       &mockCalendarRepo{},
		conf:       config.CalendarConfig{Enabled: enabled},
		client:     http.DefaultClient,
		loc:        loc,
		locName:    loc.String(),
		exceptions: make(map[dateKey]bool),
		years:      make(map[int]struct{}),
	}
}

func TestParseCalendarXML(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<calendar year="2026" lang="ru" date="2025.09.30">
	<days>
		<day d="01.01" t="1" h="1"/>
		<day d="02.01" t="1"/>
		<day d="04.03" t="2"/>
		<day d="03.07" t="3"/>
		<day d="13.99" t="1"/>
		<day d="bad" t="1"/>
	</days>
</calendar>`

	days, err := parseCalendarXML([]byte(xml), 0)
	require.NoError(t, err)
	require.Len(t, days, 4, "malformed days must be skipped")

	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), days[0].Day)
	assert.False(t, days[0].IsWorking, "t=1 is non-working")
	require.NotNil(t, days[0].HolidayID)
	assert.Equal(t, 1, *days[0].HolidayID)
	require.NotNil(t, days[0].SourceDate)
	assert.Equal(t, "2025-09-30", days[0].SourceDate.Format("2006-01-02"))

	assert.True(t, days[2].IsWorking, "t=2 shortened day is working")
	assert.True(t, days[3].IsWorking, "t=3 transferred working day is working")

	// fallback year when attribute missing
	days, err = parseCalendarXML([]byte(`<calendar><days><day d="05.05" t="1"/></days></calendar>`), 2027)
	require.NoError(t, err)
	require.Len(t, days, 1)
	assert.Equal(t, 2027, days[0].Year)
}

func TestAddWorkingDaysMonFri(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	// Friday 2026-11-20 10:00 MSK, 1 working day -> Monday 2026-11-23 end of day (weekend skipped)
	base := time.Date(2026, 11, 20, 10, 0, 0, 0, loc) // Friday
	got := addWorkingDaysMonFri(base, 1, loc)
	assert.Equal(t, endOfDay(time.Date(2026, 11, 23, 0, 0, 0, 0, loc), loc), got)

	// Base on Saturday (non-working) is still just a reference point: 1st working day is Monday.
	base = time.Date(2026, 11, 21, 12, 0, 0, 0, loc) // Saturday
	got = addWorkingDaysMonFri(base, 1, loc)
	assert.Equal(t, endOfDay(time.Date(2026, 11, 23, 0, 0, 0, 0, loc), loc), got)

	// Base on a working day is excluded: Monday + 1 working day -> Tuesday.
	base = time.Date(2026, 11, 23, 9, 0, 0, 0, loc) // Monday
	got = addWorkingDaysMonFri(base, 1, loc)
	assert.Equal(t, endOfDay(time.Date(2026, 11, 24, 0, 0, 0, 0, loc), loc), got)

	// days <= 0 returns base unchanged
	assert.Equal(t, base, addWorkingDaysMonFri(base, 0, loc))
}

func TestWorkCalendar_AddWorkingDays_WithExceptions(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)
	cal := newTestCalendar(loc, true)

	// 2026-11-23 Monday is a holiday (exception non-working), 2026-11-28 Saturday
	// is a transferred working day (exception working).
	cal.applyDays([]models.CalendarDay{
		{Day: time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC), Year: 2026, IsWorking: false},
		{Day: time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC), Year: 2026, IsWorking: true},
	})

	// Friday 2026-11-20 + 1 working day: Mon holiday, Tue 24th is 1st.
	base := time.Date(2026, 11, 20, 10, 0, 0, 0, loc)
	got := cal.AddWorkingDays(base, 1)
	assert.Equal(t, endOfDay(time.Date(2026, 11, 24, 0, 0, 0, 0, loc), loc), got)

	// Friday 2026-11-27 + 1 working day: Sat 28th is a transferred working day.
	base = time.Date(2026, 11, 27, 10, 0, 0, 0, loc)
	got = cal.AddWorkingDays(base, 1)
	assert.Equal(t, endOfDay(time.Date(2026, 11, 28, 0, 0, 0, 0, loc), loc), got)

	// IsWorkingDay reflects exceptions and default weekends.
	assert.False(t, cal.IsWorkingDay(time.Date(2026, 11, 23, 12, 0, 0, 0, loc)), "holiday")
	assert.True(t, cal.IsWorkingDay(time.Date(2026, 11, 28, 12, 0, 0, 0, loc)), "transferred working Saturday")
	assert.False(t, cal.IsWorkingDay(time.Date(2026, 11, 21, 12, 0, 0, 0, loc)), "regular Saturday")
	assert.True(t, cal.IsWorkingDay(time.Date(2026, 11, 24, 12, 0, 0, 0, loc)), "regular Tuesday")
}

func TestWorkCalendar_DisabledIgnoresSync(t *testing.T) {
	repo := &mockCalendarRepo{}
	cal := &workCalendar{
		repo:       repo,
		conf:       config.CalendarConfig{Enabled: false},
		client:     http.DefaultClient,
		loc:        time.UTC,
		locName:    "UTC",
		exceptions: make(map[dateKey]bool),
		years:      make(map[int]struct{}),
	}

	require.NoError(t, cal.Ensure(context.Background()))
	require.NoError(t, cal.Sync(context.Background(), 2026))
	assert.Empty(t, repo.upserted, "disabled calendar must not fetch/sync")
}

func TestWorkCalendar_SyncAndEnsure(t *testing.T) {
	xml2027 := `<?xml version="1.0"?><calendar year="2027"><days><day d="01.01" t="1"/></days></calendar>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, xml2027)
	}))
	defer srv.Close()

	repo := &mockCalendarRepo{daysByYear: map[int][]models.CalendarDay{}}
	cal := &workCalendar{
		repo: repo,
		conf: config.CalendarConfig{
			Enabled:   true,
			SourceURL: srv.URL + "/%d/calendar.xml",
		},
		client:     srv.Client(),
		loc:        time.UTC,
		locName:    "UTC",
		exceptions: make(map[dateKey]bool),
		years:      make(map[int]struct{}),
	}

	require.NoError(t, cal.Sync(context.Background(), 2027))
	require.Len(t, repo.upserted, 1)
	require.Len(t, repo.upserted[0], 1)
	assert.False(t, repo.upserted[0][0].IsWorking)
	assert.False(t, cal.IsWorkingDay(time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)))

	// Ensure loads cached years from repo and syncs only missing current/next years.
	now := time.Now().In(time.UTC)
	repo.daysByYear[now.Year()] = []models.CalendarDay{{Day: time.Date(now.Year(), 5, 1, 0, 0, 0, 0, time.UTC), Year: now.Year(), IsWorking: false}}
	repo.daysByYear[now.Year()+1] = []models.CalendarDay{{Day: time.Date(now.Year()+1, 5, 1, 0, 0, 0, 0, time.UTC), Year: now.Year() + 1, IsWorking: false}}
	upsertsBefore := len(repo.upserted)
	require.NoError(t, cal.Ensure(context.Background()))
	assert.Equal(t, upsertsBefore, len(repo.upserted), "no missing years -> no sync")
	assert.False(t, cal.IsWorkingDay(time.Date(now.Year(), 5, 1, 12, 0, 0, 0, time.UTC)))
}

func TestWorkCalendar_SyncFailureKeepsExisting(t *testing.T) {
	repo := &mockCalendarRepo{}
	cal := &workCalendar{
		repo: repo,
		conf: config.CalendarConfig{
			Enabled:   true,
			SourceURL: "http://127.0.0.1:1/%d/calendar.xml",
		},
		client:     &http.Client{Timeout: 200 * time.Millisecond},
		loc:        time.UTC,
		locName:    "UTC",
		exceptions: make(map[dateKey]bool),
		years:      make(map[int]struct{}),
	}
	cal.applyDays([]models.CalendarDay{{Day: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Year: 2026, IsWorking: false}})

	err := cal.Sync(context.Background(), 2026)
	require.Error(t, err)
	assert.False(t, cal.IsWorkingDay(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)), "existing cache must be preserved")
}
