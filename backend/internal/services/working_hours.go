package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
)

// workingHours — разобранный интервал рабочего времени периодических рассылок.
// Пустые start/end в конфиге означают «ограничение отключено»: allows всегда true.
// Ошибки конфигурации (битое время, start >= end, неизвестный пояс или день) не
// должны останавливать рассылки — окно выключается или значение пропускается,
// в лог уходит Error, приложение стартует как раньше.
type workingHours struct {
	enabled bool
	start   time.Duration // от полуночи в поясе окна
	end     time.Duration
	days    map[time.Weekday]struct{} // пусто — все дни
	loc     *time.Location
}

// weekdayNames — английские сокращения дней недели, индекс = time.Weekday.
var weekdayNames = [...]string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// newWorkingHours разбирает конфиг рабочего времени. Ошибки не возвращаются
// наружу: планировщику нужен всегда готовый гейт, а не повод не стартовать.
func newWorkingHours(cfg config.WorkingHoursConfig) *workingHours {
	wh := &workingHours{days: make(map[time.Weekday]struct{}, len(weekdayNames)), loc: time.Local}

	if strings.TrimSpace(cfg.Start) == "" || strings.TrimSpace(cfg.End) == "" {
		return wh
	}

	start, err := parseDayTime(cfg.Start)
	if err != nil {
		logger.Error("working hours disabled: invalid start", logger.StringAttr("value", cfg.Start), logger.ErrAttr(err))
		return wh
	}
	end, err := parseDayTime(cfg.End)
	if err != nil {
		logger.Error("working hours disabled: invalid end", logger.StringAttr("value", cfg.End), logger.ErrAttr(err))
		return wh
	}
	if start >= end {
		logger.Error("working hours disabled: start must be earlier than end",
			logger.StringAttr("start", cfg.Start), logger.StringAttr("end", cfg.End))
		return wh
	}

	if tz := strings.TrimSpace(cfg.Timezone); tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			logger.Error("working hours: unknown timezone, using server time",
				logger.StringAttr("timezone", tz), logger.ErrAttr(err))
		} else {
			wh.loc = loc
		}
	}

	for _, name := range cfg.Days {
		day, ok := parseWeekday(name)
		if !ok {
			logger.Error("working hours: unknown day ignored", logger.StringAttr("day", name))
			continue
		}
		wh.days[day] = struct{}{}
	}

	wh.start, wh.end, wh.enabled = start, end, true
	return wh
}

// allows сообщает, попадает ли момент в рабочее окно. Выключенное окно и nil
// получателя (джоба вызвана до Start) означают «можно всегда».
// Проверка выполняется до дедупликации напоминаний: пропущенный прогон не должен
// занимать слот в reminder_last_sent, иначе напоминание сдвинулось бы на пол интервала.
func (w *workingHours) allows(now time.Time) bool {
	if w == nil || !w.enabled {
		return true
	}
	at := now.In(w.loc)
	if len(w.days) > 0 {
		if _, ok := w.days[at.Weekday()]; !ok {
			return false
		}
	}
	t := time.Duration(at.Hour())*time.Hour + time.Duration(at.Minute())*time.Minute + time.Duration(at.Second())*time.Second
	return t >= w.start && t < w.end
}

// String кратко описывает окно для стартового лога планировщика.
func (w *workingHours) String() string {
	if w == nil || !w.enabled {
		return "disabled"
	}
	days := make([]string, 0, len(w.days)+1)
	if len(w.days) == 0 {
		days = append(days, "daily")
	} else {
		for i, name := range weekdayNames {
			if _, ok := w.days[time.Weekday(i)]; ok {
				days = append(days, name)
			}
		}
	}
	return fmt.Sprintf("%s-%s %s (%s)", formatDayTime(w.start), formatDayTime(w.end), strings.Join(days, ","), w.loc)
}

// parseDayTime разбирает "HH:MM" в длительность от полуночи.
func parseDayTime(value string) (time.Duration, error) {
	t, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid time %q: %w", value, err)
	}
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute, nil
}

func formatDayTime(d time.Duration) string {
	return fmt.Sprintf("%02d:%02d", int(d.Hours()), int(d.Minutes())%60)
}

// parseWeekday принимает и короткие (mon), и полные (monday) названия дней.
func parseWeekday(name string) (time.Weekday, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	if len(n) > 3 {
		n = n[:3]
	}
	for i, abbr := range weekdayNames {
		if n == abbr {
			return time.Weekday(i), true
		}
	}
	return 0, false
}
