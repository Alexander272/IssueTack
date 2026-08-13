# TODO

Живой список задач. Отмечай `[x]` когда сделано. Детальные планы фич — в `.opencode/plans/`.

## Сделано (текущая сессия)

- [x] **Фикс смены статуса (partial update)**: `TicketDTO`/`SubtaskDTO` получили presence-tracking (`Provided`, `UnmarshalJSON` на goccy/go-json), `GetChanges` и динамический `SET` в `postgres/{tickets,subtasks}.go` — частичное обновление не затирает незатронутые поля. Frontend: `updateTask`/`updateSubtask` принимают `Partial<DTO>`; убран `as any` в `TaskDetail.tsx`.
- [x] **Статусы `closed`/`cancelled` — только автор тикета или менеджер группы**: проверка `isCreatorOrManager` в `TicketService.Update` (Casbin `write` не спасает; строгая). `resolved` по-прежнему может ставить исполнитель.
- [x] **Авто-закрытие `resolved` → `closed`**: колонка `tickets.resolved_at` (миграция `20260813000001_...` + бэкафилл), проставляется при переходе в `resolved`; время ожидания — `tickets.resolved_to_closed_after` в `config.yaml` (168h = 7 дней, 0 — отключено).
- [x] **Планировщик на `gocron/v2`** (вместо тикера раз в минуту): `backend/internal/services/scheduler.go` (`SchedulerService`, `Start`/`Stop`), расписание — `tickets.auto_close_schedule` (`@daily`), запуск/остановка в `cmd/app/main.go`. Зависимость `github.com/go-co-op/gocron/v2 v2.16.3`.
- [x] **Заполнение `resolved_at`/`closed_at`** в `TicketService.Update`: `resolved` → `resolved_at`, `closed`/`cancelled` → `closed_at`, активные статусы очищают оба. `resolvedAt` добавлен в типы frontend.
- [x] **Валидация переходов статусов** в `TicketService.Update`: в `resolved` — нельзя, пока есть подзадачи в активном статусе (`open`/`in_progress`/`pending`/`on_hold`; `resolved`/`closed`/`cancelled` не блокируют) — `ErrSubtasksNotResolved`, подсчёт через `Subtasks.GetUnresolvedCount`; в `closed` — только из `resolved` (`ErrCloseRequiresResolved`), нерешённую задачу можно только отменить.

## Незакоммичено (сейчас в git status)

- [ ] Закоммитить: фикс смены статуса (partial update): `backend/internal/models/{ticket,subtask}.go`, `backend/internal/repository/postgres/{tickets,subtasks}.go`, `backend/internal/services/tickets.go`, тесты, `frontend/src/features/tasks/{tasksApiSlice.ts,modules/subtasks/subtasksApiSlice.ts,pages/TaskDetail.tsx}` + ранее: `InfoBar.tsx`, `sidebarConf.tsx`, `.gitignore` (добавлен `.opencode`)
- [ ] Закоммитить: закрытие/отмена только автору/менеджеру + авто-закрытие resolved + планировщик + валидация переходов (resolved — подзадачи, closed — только из resolved): миграция `20260813000001_add_resolved_at_to_tickets.sql`, `backend/internal/{models/ticket.go,models/errors.go,repository/postgres/tickets.go,services/tickets.go,services/subtasks.go,services/scheduler.go,services/services.go,config/config.go,cmd/app/main.go}`, `backend/configs/config.yaml` (`resolved_to_closed_after`, `auto_close_schedule`), тесты (`tickets_test.go`, `scheduler_test.go`), `frontend/src/features/tasks/types/task.ts`, `go.mod`/`go.sum` (gocron/v2); рефакторинг `NewTicketService` → `TicketDeps`
- [x] Решить что делать с удалённым бинарником `backend/app` (добавить в `.gitignore`?)

## Фичи (планы в `.opencode/plans/`)

- [ ] **Архив тикетов + пагинация только для архива** — план: `.opencode/plans/archive-pagination.md`

## Заглушки / не реализовано

- [ ] **Редактирование задачи + назначение**: нет UI. `Description.tsx` — мёртвая кнопка «Редактировать», `Participants.tsx`/`Meta.tsx` — только чтение. План: вынести общую `TaskForm` из `TaskCreateForm` (принимает `initialValues`), добавить `TaskEditModal` (title, description, category, site, priority, group, assignee, dueDate), подключить «Редактировать» и пикер исполнителя. Бэкенд `Update` уже поддерживает partial update.
- [ ] **Подзадачи: добавить/удалить/редактировать**: сейчас только смена статуса; бэкенд `Create/CreateSeveral/Update/Delete` готов. UI «Добавить» нет.
- [ ] **Вложения: загрузка/удаление**: `Attachments.tsx` read-only; бэкенд загрузки/удаления готов (`CheckWorkAccess`).
- [ ] **Удаление задачи**: нет UI (бэкенд `Delete` готов, менеджер группы).
- [ ] **Комментарии**: бэкенд handler заглушен (`backend/internal/transport/http/handlers/comments/comments.go:15` — `//TODO реализовать`), сервиса/репозитория нет. Фронт передаёт пустой массив (`TaskDetail.tsx` — `comments={[]}`), сам компонент `Detail/Comments.tsx` готов.
- [ ] **Уведомления по задаче**: бэкенд handler заглушен (`backend/internal/transport/http/handlers/notifications/notifications.go:23` — `//TODO реализовать`). Фронт `Detail/Notifications.tsx` — чекбокс-заглушка без подключения к API.

## Разделение прав

- [ ] **UI по правам**: скрывать/показывать действия детальной страницы и списка по правам (`useCan` + уровень тикета): меню «Изменить статус», «Редактировать», назначение исполнителя и т.д. Сейчас кнопки видны всем.
- [ ] **Тонкие права операций (бэкенд)**: права на назначение исполнителя, смену группы/менеджера; авто-назначение исполнителя/менеджера из группы при смене группы (сейчас `autoAssign` работает только при `Create`).
- [ ] **Менеджер тикета**: `manager_id` не проставляется в `Create` (и не обновляется в `Update`) — заполнять менеджером группы.

## Тех. долг (найденные TODO в коде)

- [ ] `backend/internal/services/realms.go:54` — создать несколько системных ролей (вынести куда-нибудь)
- [ ] `backend/internal/services/groups.go:154` — проверить, все ли тикеты в группе закрыты, перед удалением
- [ ] `backend/internal/services/category.go:58` — проверить, все ли тикеты в категории закрыты, перед удалением
- [ ] `backend/internal/transport/middleware/identity.go:16` — забирать ключи из keycloak и проверять токен здесь
- [ ] `backend/internal/repository/postgres/permissions.go:257` — добавить уровни для сортировки
- [ ] `frontend/src/app/middlewares/resetStore.ts:16` — сброс состояний при logout
- [ ] Подзадачи: `closed_at` не проставляется при статусе `closed` (у тикетов теперь проставляется в `TicketService.Update`, у подзадач нет)

## Идеи / на будущее (не подтверждено)

- Пока пусто. Добавляй сюда.
