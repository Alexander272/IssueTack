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
- **«Взять в работу»** (`TicketService.Take`, `POST /tickets/:id/take`, флаг `canTake` в `AccessFlags`): пользователь назначает себя исполнителем и переводит заявку в `in_progress`. Разрешено при read-доступе и активном статусе, если исполнителя нет совсем (любой активный статус), либо исполнитель — другой пользователь и статус `open`. Во фронтенде (`InfoBar`) кнопка «Взять в работу» показывается в этих случаях **вместо** «Изменить статус»; если забирать нельзя и нет `canWrite`/`canWork`, у пользователя ничего не показывается.
- `TicketService.Update`: если у пользователя только work-access, то разрешена смена только статуса (`ActionStatusChanged`, `ActionClosed`), остальные поля — запрет.
- Статусы `closed`/`cancelled` может ставить **автор, владелец (owner) или менеджер группы** (`isCreatorOrManager` + проверка владельца, строгая проверка — Casbin `write`-права не спасают).
- **Владелец (owner)** без write/work-доступа может делать в `Update` только три перехода (`ownerTransitionAllowed`): активный статус → `cancelled` (отменить заявку), `resolved` → `closed` (принять решение), `resolved` → `in_progress` (вернуть в работу). Все прочие смены статусов и изменение полей ему запрещены.
- Во фронтенде (`InfoBar`) кнопка «Изменить статус» показывается только создателю/исполнителю/менеджеру (`isCreator || isAssignee || canWrite`); «чистый» владелец видит только свои три кнопки: «Отменить заявку», «Подтвердить решение», «Вернуть в работу».
- Переход в `resolved` проставляет `resolved_at`; тикеты со статусом `resolved` автоматически закрываются через `tickets.resolved_to_closed_after` в `config.yaml` (0 — отключено). `closed_at` при `resolved` не проставляется.
- Переход в `resolved` возможен, только если нет подзадач в активном статусе (`open`/`in_progress`/`pending`/`on_hold`); подзадачи `resolved`/`closed`/`cancelled` не блокируют (`ErrSubtasksNotResolved`, подсчёт — `Subtasks.GetUnresolvedCount`).
- Статус `closed` можно поставить **только из `resolved`** (`ErrCloseRequiresResolved`); нерешённую задачу можно только отменить (`cancelled`).
- Закрытая/отменённая заявка **терминальна**: её статус изменить нельзя никому (в `Update` — `ErrTicketFrozen`; в `computeAllowedStatuses` для `closed`/`cancelled` разрешён лишь взаимообмен `closed⇄cancelled` автору/менеджеру/владельцу, исполнитель с work-доступом активные статусы не получает).
- На «замороженных» заявках (resolved/closed/cancelled) **внешние комментарии запрещены** (`CommentService.Create` → `isTicketInactive`), допустимы только внутренние — от исполнителя/менеджера группы (`CheckInternalAssigneeAccess`). Во фронтенде (`Comments`) на закрытой заявке переключатель скрыт и коммент форсится внутренним.
- **Закрепление (temporary)** нельзя оставлять на «замороженных» заявках (`TicketFavoritesService.Add` при inactive + `favoritesType=temporary` → `ErrTicketFrozen`; кнопка Pin скрыта в `Header`). **Избранное (permanent)** разрешено в любом статусе (кнопка «В избранное» всегда доступна).
- Тикеты без группы: в списке показываются только те, где пользователь создатель или исполнитель (`IncludeUngroupedAssignedTo` в `TicketFilter`).
- `realm` пробрасывается в сервисы подзадач/вложений (variadic `...string`), чтобы Casbin-проверка шла по правильному домену.

## Уведомления

Реализованы в `backend/internal/services/notifications.go` (`NotificationService`).

**Жизненный цикл события** (для каждого `Notify*`-метода): 1) собрать получателей + нейтральный `CreateNotificationDTO` (персональный тумблер `enabled` применяется только при авто-подписке в `autoSubscribeOnCreate`); 2) **deliver** — разослать по каналам (`channels []Notifier`, best-effort, ошибки логируются); канал возвращает признак реальной доставки; 3) **persist** — сохранить строку в БД (транзакция `repo.Create`, сбой одного не откатывает остальных, ошибки логируются) **только тем, кому доставлено хотя бы одним каналом**. Строка = фиксация доставки и дедупликация повторных прогонов (например, cron по просрочке); необслуженные (нет `mattermost_id`/неактивная интеграция/сбой) не персистятся и будут повторены на следующей попытке.

- **Каналы** — интерфейс `Notifier` (`services/channels.go`: `Name()` + `Notify(ctx, userID, dto, ticket) (delivered bool, err error)`). Сейчас один: `mattermostNotifier` (`services/mattermost_notifier.go`) — DM от бота реалма. Канал сам гейтит себя: тикет без `RealmID` / неактивная интеграция (`mmRepo.GetByRealm` → `IsActive`/`BotToken`) / отсутствие `mattermost_id` у пользователя — `delivered=false` без ошибки, причина логируется в `Info`; успешная отправка — `delivered=true` + Info-лог. Формат DM — в `format()`: «Новая задача / Задача обновлена / Задача удалена / Новый комментарий / Новое вложение / Задача просрочена» + `№N` (если `TicketNumber`) + заголовок; для `ticket.updated` добавляется сводка изменений из `Data.changes`; ссылка «Открыть: {baseURL}/tasks/{id}» при непустом `baseURL`. Примечание: поле `data.changes` в `CreateNotificationDTO` хранится как **строка** (JSON-массив `FieldChange`), не как `[]byte` (иначе при маршалинге получится base64). Новый канал (push/email) — отдельный `Notifier` в списке `channels` при сборке в `services.go`.
- **Актор не уведомляется о собственных действиях**: `TicketCreated` (параметр `actorID`, `delete(recipients, actorID)`), `TicketUpdated` (`delete(...)` + исполнитель при смене статуса уведомляется только если он не актор), `TicketCommented`/`AttachmentAdded` (было ранее).
- **WebSocket** не является каналом уведомлений: `NotificationService` хаб не держит. `/api/ws` (`transport/ws/handler.go`) остаётся транспортной заготовкой: через него асинхронно отдаются непрочитанные при коннекте (`SendUnread`, `repo.GetUnread`+`MarkAllRead`), push-доставка в реальном времени не реализована. WebSocket-отправку из комментариев/тикетов не добавлять — уведомления идут через `channels` в Mattermost.
- Дублирующая DM-отправка владельцу заявки о комментарии живёт отдельно (`comments.go:notifyOwnerViaMattermost`) и используется для **внутренних** context-комментариев создателю; внешние комментарии теперь покрываются `TicketCommented` через канал.

## Текущий статус

Страница деталей тикета (`/tasks/:id`) реализована: `frontend/src/features/tasks/components/Detail/*` (Header, InfoBar, Description, Subtasks, Attachments, Comments, Participants, Meta, Notifications). Часть из них — заглушки, см. `TODO.md`.

Проект использует opencode; `AGENTS.md` читается автоматически при старте сессии.
