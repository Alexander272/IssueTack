package services

import (
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/stretchr/testify/assert"
)

// Даты в таблицах фиксированы: 2026-10-26 — понедельник, 2026-10-24 — суббота,
// 2026-10-25 — воскресенье.
func TestWorkingHours_Allows(t *testing.T) {
	utcAt := func(month time.Month, day, hour, min int) time.Time {
		return time.Date(2026, month, day, hour, min, 0, 0, time.UTC)
	}
	weekdays := []string{"mon", "tue", "wed", "thu", "fri"}
	base := config.WorkingHoursConfig{Start: "08:00", End: "18:00", Timezone: "UTC", Days: weekdays}

	cases := []struct {
		name string
		cfg  config.WorkingHoursConfig
		now  time.Time
		want bool
	}{
		{"внутри окна", base, utcAt(time.October, 26, 12, 0), true},
		{"за час до начала", base, utcAt(time.October, 26, 7, 59), false},
		{"ровно начало — внутри", base, utcAt(time.October, 26, 8, 0), true},
		{"за минуту до конца — внутри", base, utcAt(time.October, 26, 17, 59), true},
		{"ровно конец — уже вне окна", base, utcAt(time.October, 26, 18, 0), false},
		{"суббота вне списка", base, utcAt(time.October, 24, 12, 0), false},
		{"воскресенье вне списка", base, utcAt(time.October, 25, 12, 0), false},
		{
			"без списка дней — все дни",
			config.WorkingHoursConfig{Start: "08:00", End: "18:00", Timezone: "UTC"},
			utcAt(time.October, 24, 12, 0), true,
		},
		{
			"полное название дня принимается",
			config.WorkingHoursConfig{Start: "08:00", End: "18:00", Timezone: "UTC", Days: []string{"monday"}},
			utcAt(time.October, 26, 12, 0), true,
		},
		{
			"полное название дня ограничивает остальные",
			config.WorkingHoursConfig{Start: "08:00", End: "18:00", Timezone: "UTC", Days: []string{"Monday"}},
			utcAt(time.October, 27, 12, 0), false,
		},
		{
			"неизвестный день отбрасывается, известные работают",
			config.WorkingHoursConfig{Start: "08:00", End: "18:00", Timezone: "UTC", Days: []string{"mon", "funday"}},
			utcAt(time.October, 26, 12, 0), true,
		},
		{"окно выключено — пустой конфиг", config.WorkingHoursConfig{}, utcAt(time.October, 24, 3, 0), true},
		{
			"окно выключено — задан только start",
			config.WorkingHoursConfig{Start: "08:00", Timezone: "UTC", Days: weekdays},
			utcAt(time.October, 24, 12, 0), true,
		},
		{
			"битое время — окно выключено, рассылка не встаёт",
			config.WorkingHoursConfig{Start: "8h00", End: "18:00", Timezone: "UTC", Days: weekdays},
			utcAt(time.October, 24, 12, 0), true,
		},
		{
			"конец раньше начала — окно выключено",
			config.WorkingHoursConfig{Start: "18:00", End: "08:00", Timezone: "UTC", Days: weekdays},
			utcAt(time.October, 24, 12, 0), true,
		},
		{
			"пояс применяется: 12:00 в +05 — это 07:00 окна UTC",
			base,
			time.Date(2026, time.October, 26, 12, 0, 0, 0, time.FixedZone("UTC+5", 5*3600)),
			false,
		},
		{
			"та же дата в UTC внутри окна",
			base,
			utcAt(time.October, 26, 12, 0),
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wh := newWorkingHours(tc.cfg)
			assert.Equal(t, tc.want, wh.allows(tc.now))
		})
	}
}

// TestWorkingHours_Allows_Nil проверяет вызов до Start (workingHours == nil) —
// существующие тесты джоб строят сервис без Start, гейт не должен их ломать.
func TestWorkingHours_Allows_Nil(t *testing.T) {
	var wh *workingHours
	assert.True(t, wh.allows(time.Now()))
}

func TestWorkingHours_ParseDayTime(t *testing.T) {
	cases := []struct {
		value   string
		want    time.Duration
		wantErr bool
	}{
		{value: "08:00", want: 8 * time.Hour},
		{value: "8:00", want: 8 * time.Hour},
		{value: "00:00", want: 0},
		{value: "23:59", want: 23*time.Hour + 59*time.Minute},
		{value: " 09:30 ", want: 9*time.Hour + 30*time.Minute},
		{value: "24:00", wantErr: true},
		{value: "8h00", wantErr: true},
		{value: "", wantErr: true},
		{value: "09:60", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			got, err := parseDayTime(tc.value)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWorkingHours_String(t *testing.T) {
	enabled := newWorkingHours(config.WorkingHoursConfig{
		Start: "08:00", End: "18:00", Timezone: "UTC", Days: []string{"mon", "tue", "wed", "thu", "fri"},
	})
	assert.Equal(t, "08:00-18:00 mon,tue,wed,thu,fri (UTC)", enabled.String())

	daily := newWorkingHours(config.WorkingHoursConfig{Start: "09:00", End: "17:00", Timezone: "UTC"})
	assert.Equal(t, "09:00-17:00 daily (UTC)", daily.String())

	assert.Equal(t, "disabled", newWorkingHours(config.WorkingHoursConfig{}).String())

	var nilWH *workingHours
	assert.Equal(t, "disabled", nilWH.String())
}
