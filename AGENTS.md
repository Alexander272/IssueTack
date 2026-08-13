# IssueTrack

Система учёта заявок (тикетов). Frontend + Backend в одном репозитории.

## Стек

**Backend** (`backend/`): Go 1.25, Gin, Casbin (RBAC + realms), PostgreSQL (pgx), Redis, WebSocket (ws_hub), goose-миграции.

**Frontend** (`frontend/`): React 19 + Vite, MUI v9, Redux Toolkit + RTK Query, react-router v7, react-hook-form, `lucide-mui` (иконки), react-datasheet-grid, PWA.

## Запуск

- **Backend**: `cd backend && go run ./cmd/app`. Конфиг — `backend/configs/config.yaml`. Секреты — `backend/.env` (не коммитится). Миграции применяются автоматически при старте (`goose.Up`). HTTP на порту 9000.
- **Frontend**: `cd frontend && npm start`. Vite-proxy на `http://localhost:9000`. Билд: `npm run build`. Линт: `npm run lint`.
- **Миграции (вручную, из `backend/`)**: `goose -dir internal/migrate/postgres/migrations postgres "<dsn>" up/down`. Мемо по командам — `backend/notes.md`.

## Архитектура

- `backend/internal/repository/postgres/` — слой данных (SQL), структуры DTO в `backend/internal/models/`.
- `backend/internal/services/` — бизнес-логика. Внутри пакета интерфейсы сервисов (см. `services.go`, а также интерфейсы в самих файлах сервисов).
- `backend/internal/transport/http/handlers/` — HTTP-обработчики. Роуты регистрируются через `access.Reg.R(Resource).Read()/Write()/Delete()` + Casbin-мидлвар.
- `backend/internal/transport/middleware/` — проверка прав, извлечение actor/user из контекста.
- `frontend/src/features/` — фичи (tasks, groups, categories, access, auth, realms, sites, user). Внутри: `components/`, `pages/`, `apiSlice.ts`, `types/`.
- API-эндпоинты собраны в `frontend/src/app/api.ts`.

## Ключевые конвенции

- **Иконки**: только `lucide-mui` (MUI SvgIcon). НЕ использовать самописные `@/components/Icons/*` в новом коде. Цвет — через `sx={{ color }}` (не `fill`), размер — `sx={{ fontSize }}`.
- **Стили**: MUI `sx`, не Tailwind.
- **Go**: следуй существующим паттернам сервисов/репозиториев; ошибки доступа — `models.ErrPermissionDenied` (код AU002).
- Планы фич лежат в `.opencode/plans/*.md`, живой список задач — в `TODO.md`.

## Модель доступа к тикетам

Реализована в `backend/internal/services/ticket_access.go` (`TicketAccessChecker`, `CheckAccess`, `CheckWorkAccess`).

Порядок проверки `CheckAccess`:

1. Сначала Casbin `policies.Enforce(user, realm, ticket, action)` — если да, доступ разрешён.
2. Иначе логика по атрибутам тикета:

| Действие | Разрешено                                                                                                                   |
| -------- | --------------------------------------------------------------------------------------------------------------------------- |
| Read     | участник группы тикета, менеджер группы, **исполнитель (assignee)** тикета; для тикета **без группы** — также **создатель** |
| Write    | создатель тикета, менеджер группы                                                                                           |
| Delete   | только менеджер группы (создатель НЕ может удалять)                                                                         |

- **`CheckWorkAccess`** = `CheckAccess(Write)` ИЛИ исполнитель тикета. Используется для: смены статуса (через `Update`), создания/изменения подзадач, загрузки/удаления вложений.
- `TicketService.Update`: если у пользователя только work-access, то разрешена смена только статуса (`ActionStatusChanged`, `ActionClosed`), остальные поля — запрет.
- Статусы `closed`/`cancelled` может ставить **только автор тикета или менеджер группы** (`isCreatorOrManager`, строгая проверка — Casbin `write`-права не спасают).
- Переход в `resolved` проставляет `resolved_at`; тикеты со статусом `resolved` автоматически закрываются через `tickets.resolved_to_closed_after` в `config.yaml` (0 — отключено). `closed_at` при `resolved` не проставляется.
- Переход в `resolved` возможен, только если нет подзадач в активном статусе (`open`/`in_progress`/`pending`/`on_hold`); подзадачи `resolved`/`closed`/`cancelled` не блокируют (`ErrSubtasksNotResolved`, подсчёт — `Subtasks.GetUnresolvedCount`).
- Статус `closed` можно поставить **только из `resolved`** (`ErrCloseRequiresResolved`); нерешённую задачу можно только отменить (`cancelled`).
- Тикеты без группы: в списке показываются только те, где пользователь создатель или исполнитель (`IncludeUngroupedAssignedTo` в `TicketFilter`).
- `realm` пробрасывается в сервисы подзадач/вложений (variadic `...string`), чтобы Casbin-проверка шла по правильному домену.

## Текущий статус

Страница деталей тикета (`/tasks/:id`) реализована: `frontend/src/features/tasks/components/Detail/*` (Header, InfoBar, Description, Subtasks, Attachments, Comments, Participants, Meta, Notifications). Часть из них — заглушки, см. `TODO.md`.

Проект использует opencode; `AGENTS.md` читается автоматически при старте сессии.
