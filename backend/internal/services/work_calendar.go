package services

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/error_bot"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
)

// WorkCalendar считает сроки в рабочих днях с учётом производственного календаря.
// Исключения (праздники/переносы) хранятся в БД и кэшируются в памяти; при
// недоступности источника/пустом кэше действует фолбэк «Пн–Пт рабочие, Сб/Вс выходные».
type WorkCalendar interface {
	// AddWorkingDays возвращает конец N-го рабочего дня после base в календарной
	// таймзоне. База — точка отсчёта, первый засчитанный день — следующий рабочий.
	// days <= 0 — возвращает base без изменений.
	AddWorkingDays(base time.Time, days int) time.Time
	// IsWorkingDay сообщает, рабочий ли день (в календарной таймзоне) у момента t.
	IsWorkingDay(t time.Time) bool
	// Timezone возвращает имя календарной таймзоны.
	Timezone() string
	// Data возвращает снимок календаря для клиентов (исключения за годы).
	Data(ctx context.Context, years []int) (*models.CalendarData, error)
	// Sync загружает календарь за указанные годы из внешнего источника, сохраняет в БД
	// и обновляет кэш. Ошибка по отдельному году не затирает его сохранённые данные.
	Sync(ctx context.Context, years ...int) error
	// Ensure загружает кэш из БД и при необходимости дозагружает недостающие годы.
	Ensure(ctx context.Context) error
}

type WorkCalendarDeps struct {
	Repo repository.Calendar
	Conf config.CalendarConfig
	// WorkingHoursTZ — tickets.working_hours.timezone, используется когда calendar.timezone пуст.
	WorkingHoursTZ string
	Client         *http.Client
}

type workCalendar struct {
	repo    repository.Calendar
	conf    config.CalendarConfig
	client  *http.Client
	loc     *time.Location
	locName string

	mu         sync.RWMutex
	exceptions map[dateKey]bool // civil date (cal TZ) -> isWorking
	years      map[int]struct{}
	warnedOnce sync.Once
}

type dateKey struct {
	y int
	m time.Month
	d int
}

func NewWorkCalendar(deps *WorkCalendarDeps) *workCalendar {
	loc, locName := resolveCalendarLocation(deps.Conf.Timezone, deps.WorkingHoursTZ)

	client := deps.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	return &workCalendar{
		repo:       deps.Repo,
		conf:       deps.Conf,
		client:     client,
		loc:        loc,
		locName:    locName,
		exceptions: make(map[dateKey]bool),
		years:      make(map[int]struct{}),
	}
}

func resolveCalendarLocation(calendarTZ, workingHoursTZ string) (*time.Location, string) {
	tz := strings.TrimSpace(calendarTZ)
	if tz == "" {
		tz = strings.TrimSpace(workingHoursTZ)
	}
	if tz == "" {
		// Имя таймзоны для клиентов пустое: "Local" не является валидной IANA-зоной,
		// из-за чего dayjs.tz/Intl в браузере падают. Клиенты трактуют пустое
		// значение как локальную таймзону; расчёты на сервере идут в time.Local.
		return time.Local, ""
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		logger.Error("calendar: unknown timezone, using server time",
			logger.StringAttr("timezone", tz), logger.ErrAttr(err))
		return time.Local, ""
	}
	return loc, tz
}

func (c *workCalendar) Timezone() string {
	return c.locName
}

func (c *workCalendar) holidaysEnabled() bool {
	// Учёт праздников/переносов из внешнего календаря. Выходные (Сб/Вс) учитываются
	// всегда, независимо от этого флага.
	return c.conf.Enabled
}

func (c *workCalendar) IsWorkingDay(t time.Time) bool {
	at := t.In(c.loc)
	k := dateKey{y: at.Year(), m: at.Month(), d: at.Day()}

	c.mu.RLock()
	ex, ok := c.exceptions[k]
	empty := len(c.years) == 0
	c.mu.RUnlock()

	if ok {
		return ex
	}
	// Дефолт: Пн–Пт рабочие, Сб/Вс выходные.
	wd := at.Weekday()
	isWork := wd != time.Saturday && wd != time.Sunday

	if empty && c.holidaysEnabled() {
		c.warnedOnce.Do(func() {
			logger.Warn("production calendar has no loaded data, using Mon–Fri fallback without holidays")
		})
	}
	return isWork
}

func (c *workCalendar) AddWorkingDays(base time.Time, days int) time.Time {
	if days <= 0 {
		return base
	}

	// База — точка отсчёта: первый засчитанный день — следующий рабочий.
	at := base.In(c.loc)
	// Нормализуем к началу дня в календарной TZ для итерации по civil-датам.
	cur := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, c.loc)
	added := 0
	for added < days {
		cur = cur.AddDate(0, 0, 1)
		if c.IsWorkingDay(cur) {
			added++
		}
	}
	// Конец целевого рабочего дня в календарной TZ.
	return endOfDay(cur, c.loc)
}

func endOfDay(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 23, 59, 59, 999000000, loc)
}

func (c *workCalendar) Data(ctx context.Context, years []int) (*models.CalendarData, error) {
	if len(years) == 0 {
		now := time.Now().In(c.loc)
		years = []int{now.Year(), now.Year() + 1}
	}
	days, err := c.repo.GetByYears(ctx, years)
	if err != nil {
		return nil, err
	}
	exceptions := make([]models.CalendarException, 0, len(days))
	for _, d := range days {
		// Дата хранится как DATE; нормализуем в календарной TZ.
		t := d.Day.In(c.loc)
		exceptions = append(exceptions, models.CalendarException{
			Date:      fmt.Sprintf("%04d-%02d-%02d", t.Year(), int(t.Month()), t.Day()),
			IsWorking: d.IsWorking,
		})
	}
	return &models.CalendarData{
		Timezone:   c.locName,
		Years:      years,
		Exceptions: exceptions,
	}, nil
}

func (c *workCalendar) Ensure(ctx context.Context) error {
	if !c.holidaysEnabled() {
		return nil
	}
	if err := c.Load(ctx); err != nil {
		return err
	}

	now := time.Now().In(c.loc)
	want := []int{now.Year(), now.Year() + 1}
	missing := c.missingYears(want)
	if len(missing) == 0 {
		return nil
	}
	return c.Sync(ctx, missing...)
}

func (c *workCalendar) Load(ctx context.Context) error {
	now := time.Now().In(c.loc)
	years := []int{now.Year() - 1, now.Year(), now.Year() + 1}
	days, err := c.repo.GetByYears(ctx, years)
	if err != nil {
		return fmt.Errorf("failed to load production calendar: %w", err)
	}
	c.applyDays(days)
	return nil
}

func (c *workCalendar) missingYears(want []int) []int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var missing []int
	for _, y := range want {
		if _, ok := c.years[y]; !ok {
			missing = append(missing, y)
		}
	}
	return missing
}

func (c *workCalendar) applyDays(days []models.CalendarDay) {
	ex := make(map[dateKey]bool, len(days))
	years := make(map[int]struct{})
	for _, d := range days {
		t := d.Day.In(c.loc)
		k := dateKey{y: t.Year(), m: t.Month(), d: t.Day()}
		ex[k] = d.IsWorking
		years[d.Year] = struct{}{}
	}
	c.mu.Lock()
	// Сливаем с существующим кэшем, чтобы догрузка одного года не стирала другие.
	for k, v := range ex {
		c.exceptions[k] = v
	}
	for y := range years {
		c.years[y] = struct{}{}
	}
	c.mu.Unlock()
}

func (c *workCalendar) Sync(ctx context.Context, years ...int) error {
	if !c.holidaysEnabled() {
		return nil
	}
	if len(years) == 0 {
		now := time.Now().In(c.loc)
		years = []int{now.Year(), now.Year() + 1}
	}

	source := strings.TrimSpace(c.conf.SourceURL)
	if source == "" {
		source = "https://xmlcalendar.ru/data/ru/%d/calendar.xml"
	}

	var firstErr error
	for _, year := range years {
		days, err := c.fetchYear(ctx, source, year)
		if err != nil {
			logger.Error("failed to sync production calendar year",
				logger.IntAttr("year", year), logger.ErrAttr(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := c.repo.Upsert(ctx, days); err != nil {
			logger.Error("failed to store production calendar year",
				logger.IntAttr("year", year), logger.ErrAttr(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		c.applyDays(days)
		logger.Info("production calendar synced", logger.IntAttr("year", year), logger.IntAttr("days", len(days)))
	}
	return firstErr
}

// xmlCalendar/xmlDay — формат xmlcalendar.ru.
type xmlCalendar struct {
	XMLName    xml.Name `xml:"calendar"`
	Year       int      `xml:"year,attr"`
	SourceDate string   `xml:"date,attr"`
	Days       []xmlDay `xml:"days>day"`
}

type xmlDay struct {
	D string `xml:"d,attr"` // MM.DD
	T string `xml:"t,attr"` // 1=выходной, 2=сокращённый рабочий, 3=рабочий (перенос)
	H string `xml:"h,attr"` // id праздника
}

func (c *workCalendar) fetchYear(ctx context.Context, source string, year int) ([]models.CalendarDay, error) {
	url := fmt.Sprintf(source, year)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build calendar request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		error_bot.Send(nil, err.Error(), req)
		return nil, fmt.Errorf("fetch calendar %d: %w", year, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		error_bot.Send(nil, fmt.Sprintf("Not expected status code: %d", resp.StatusCode), req)
		return nil, fmt.Errorf("fetch calendar %d: status %d", year, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read calendar %d: %w", year, err)
	}

	return parseCalendarXML(body, year)
}

func parseCalendarXML(data []byte, fallbackYear int) ([]models.CalendarDay, error) {
	var cal xmlCalendar
	if err := xml.Unmarshal(data, &cal); err != nil {
		return nil, fmt.Errorf("parse calendar xml: %w", err)
	}
	year := cal.Year
	if year == 0 {
		year = fallbackYear
	}

	var sourceDate *time.Time
	if cal.SourceDate != "" {
		if t, err := time.Parse("2006.01.02", cal.SourceDate); err == nil {
			sourceDate = &t
		}
	}

	days := make([]models.CalendarDay, 0, len(cal.Days))
	for _, d := range cal.Days {
		day, ok := parseXMLDay(d, year, sourceDate)
		if !ok {
			continue
		}
		days = append(days, day)
	}
	return days, nil
}

func parseXMLDay(d xmlDay, year int, sourceDate *time.Time) (models.CalendarDay, bool) {
	parts := strings.Split(strings.TrimSpace(d.D), ".")
	if len(parts) != 2 {
		return models.CalendarDay{}, false
	}
	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return models.CalendarDay{}, false
	}
	dayNum, err := strconv.Atoi(parts[1])
	if err != nil || dayNum < 1 || dayNum > 31 {
		return models.CalendarDay{}, false
	}

	t, err := strconv.Atoi(strings.TrimSpace(d.T))
	if err != nil {
		return models.CalendarDay{}, false
	}
	// t=1 — нерабочий; t=2 (сокращённый) и t=3 (перенесённый рабочий) — рабочие.
	// Любой другой тип трактуем как рабочее исключение (безопасный дефолт).
	isWorking := t != 1

	var holidayID *int
	if h := strings.TrimSpace(d.H); h != "" {
		if id, err := strconv.Atoi(h); err == nil {
			holidayID = &id
		}
	}

	return models.CalendarDay{
		Day:        time.Date(year, time.Month(month), dayNum, 0, 0, 0, 0, time.UTC),
		Year:       year,
		IsWorking:  isWorking,
		HolidayID:  holidayID,
		SourceDate: sourceDate,
	}, true
}

// addWorkingDaysMonFri возвращает конец N-го рабочего дня после base по правилу Mon–Fri
// (без праздников) в указанной таймзоне. Используется как фолбэк, когда сервис
// производственного календаря не внедрён (например, в части тестов).
func addWorkingDaysMonFri(base time.Time, days int, loc *time.Location) time.Time {
	if days <= 0 {
		return base
	}
	if loc == nil {
		loc = time.Local
	}
	at := base.In(loc)
	cur := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, loc)
	added := 0
	for added < days {
		cur = cur.AddDate(0, 0, 1)
		wd := cur.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			added++
		}
	}
	return endOfDay(cur, loc)
}
