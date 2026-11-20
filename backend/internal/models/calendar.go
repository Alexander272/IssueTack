package models

import "time"

// CalendarDay — исключение производственного календаря: день, статус которого
// отличается от дефолта «Пн–Пт рабочие, Сб/Вс выходные». IsWorking=false —
// нерабочий (праздник/перенесённый выходной, t=1), true — рабочий
// (сокращённый t=2 или перенесённая рабочая суббота t=3).
type CalendarDay struct {
	Day        time.Time  `json:"day" db:"day"`
	Year       int        `json:"year" db:"year"`
	IsWorking  bool       `json:"isWorking" db:"is_working"`
	HolidayID  *int       `json:"holidayId,omitempty" db:"holiday_id"`
	SourceDate *time.Time `json:"sourceDate,omitempty" db:"source_date"`
}

// CalendarException — исключение календаря в ответе API (дата — строка YYYY-MM-DD,
// чтобы не зависеть от часового пояса при передаче на клиент).
type CalendarException struct {
	Date      string `json:"date"`
	IsWorking bool   `json:"isWorking"`
}

// CalendarData — данные производственного календаря для клиентов (web/плагин),
// чтобы расчёт автосрока на фронте совпадал с бэкендом.
type CalendarData struct {
	// Timezone — IANA-таймзона, в которой определяется рабочий день и конец дня дедлайна.
	Timezone   string              `json:"timezone"`
	Years      []int               `json:"years"`
	Exceptions []CalendarException `json:"exceptions"`
}
