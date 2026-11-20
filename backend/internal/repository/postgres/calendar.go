package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CalendarRepo interface {
	Upsert(ctx context.Context, days []models.CalendarDay) error
	GetByYears(ctx context.Context, years []int) ([]models.CalendarDay, error)
}

type calendarRepo struct {
	db *pgxpool.Pool
}

func NewCalendarRepo(db *pgxpool.Pool) CalendarRepo {
	return &calendarRepo{db: db}
}

func (r *calendarRepo) Upsert(ctx context.Context, days []models.CalendarDay) error {
	if len(days) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := fmt.Sprintf(`INSERT INTO %s (day, year, is_working, holiday_id, source_date, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (day) DO UPDATE SET
			year = EXCLUDED.year,
			is_working = EXCLUDED.is_working,
			holiday_id = EXCLUDED.holiday_id,
			source_date = EXCLUDED.source_date,
			updated_at = now()`, Tables.ProductionCalendar)

	for _, d := range days {
		batch.Queue(query, d.Day, d.Year, d.IsWorking, d.HolidayID, d.SourceDate)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range days {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to upsert production calendar: %w", err)
		}
	}
	return nil
}

func (r *calendarRepo) GetByYears(ctx context.Context, years []int) ([]models.CalendarDay, error) {
	if len(years) == 0 {
		return []models.CalendarDay{}, nil
	}

	query := fmt.Sprintf(`SELECT day, year, is_working, holiday_id, source_date
		FROM %s WHERE year = ANY($1) ORDER BY day`, Tables.ProductionCalendar)

	rows, err := r.db.Query(ctx, query, years)
	if err != nil {
		return nil, fmt.Errorf("failed to get production calendar: %w", err)
	}
	defer rows.Close()

	days := make([]models.CalendarDay, 0)
	for rows.Next() {
		var d models.CalendarDay
		var day time.Time
		if err := rows.Scan(&day, &d.Year, &d.IsWorking, &d.HolidayID, &d.SourceDate); err != nil {
			return nil, fmt.Errorf("failed to scan production calendar day: %w", err)
		}
		d.Day = day
		days = append(days, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate production calendar: %w", err)
	}
	return days, nil
}
