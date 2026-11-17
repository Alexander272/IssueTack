-- +goose Up

-- Статус запроса на перенос срока — закрытый набор: pending → approved | rejected.
-- Без CHECK доменная ошибка уйдёт на уровне БД только для новых записей; код уже
-- защищает Decide на стороне сервиса, миграция фиксирует контракт схемы.
ALTER TABLE ticket_due_date_requests
    ADD CONSTRAINT ticket_due_date_requests_status_check
    CHECK (status IN ('pending', 'approved', 'rejected'));

-- +goose Down

ALTER TABLE ticket_due_date_requests
    DROP CONSTRAINT IF EXISTS ticket_due_date_requests_status_check;
