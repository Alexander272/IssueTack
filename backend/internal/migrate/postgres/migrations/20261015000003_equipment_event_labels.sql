-- +goose Up
-- +goose StatementBegin

-- Снимок события журнала хранит читаемые подписи рядом с идентификаторами: FK
-- объявлены ON DELETE SET NULL, поэтому джойн на чтении потерял бы подпись после
-- удаления площадки или сотрудника, а переименование переписало бы прошлое.
-- События, записанные до появления подписей, знают только идентификаторы — разово
-- достраиваем названия по ещё живым ссылкам. Удалённые ссылки останутся без имени:
-- лента покажет #id, и это честнее выдуманного названия.

UPDATE public.equipment_events AS ev
SET new_value = ev.new_value || jsonb_build_object('userName', u.full_name)
FROM (
    SELECT id, COALESCE(NULLIF(TRIM(last_name || ' ' || first_name), ''), username) AS full_name
    FROM public.users
) AS u
WHERE (ev.new_value ? 'userId')
  AND NOT (ev.new_value ? 'userName')
  AND (ev.new_value ->> 'userId')::uuid = u.id;

UPDATE public.equipment_events AS ev
SET new_value = ev.new_value || jsonb_build_object('siteName', s.name)
FROM public.sites AS s
WHERE (ev.new_value ? 'siteId')
  AND NOT (ev.new_value ? 'siteName')
  AND (ev.new_value ->> 'siteId')::uuid = s.id;

UPDATE public.equipment_events AS ev
SET old_value = ev.old_value || jsonb_build_object('siteName', s.name)
FROM public.sites AS s
WHERE (ev.old_value ? 'siteId')
  AND NOT (ev.old_value ? 'siteName')
  AND (ev.old_value ->> 'siteId')::uuid = s.id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Операция необратима: снять подписи нельзя, не тронув строки, которые пришли с
-- ними изначально, а имена удалённых пользователей и площадок восстановить
-- неоткуда. Журнал иммутабелен, поэтому откат ничего не делает.
SELECT 1;

-- +goose StatementEnd
