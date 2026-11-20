-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS production_calendar (
    day DATE PRIMARY KEY,
    year INT NOT NULL,
    is_working BOOLEAN NOT NULL,
    holiday_id INT,
    source_date DATE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_production_calendar_year ON production_calendar(year);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS production_calendar;
-- +goose StatementEnd
