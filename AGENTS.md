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
- `frontend/src/features/` — фичи (tasks, groups, categories, access, auth, realms, sites, user). Внутри: `components/`, `types/` и API-слайс `<feature>ApiSlice.ts` (напр. `tasksApiSlice.ts`). `pages/` есть не у всех фич — часть страниц лежит в общем `frontend/src/pages/`.
- API-эндпоинты собраны в `frontend/src/app/api.ts`.

## Ключевые конвенции

- **Иконки**: только `lucide-mui` (MUI SvgIcon). НЕ использовать самописные `@/components/Icons/*` в новом коде. Цвет — через `sx={{ color }}` (не `fill`), размер — `sx={{ fontSize }}`. Исключение — brand-иконки (логотипы браузеров/ОС): в `lucide-mui` их нет, поэтому для них самописные иконки допустимы (`LoginsModal.tsx`).
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

- **`CheckWorkAccess`** = `CheckAccess(Write)` ИЛИ исполнитель тикета. Используется для: создания/изменения подзадач и вложений, комментариев; **сам по себе права на смену статуса заявки уже не даёт** — смена статуса гейтится `canChangeStatus` (см. ниже).
- **Смена статуса заявки** (`TicketService.canChangeStatus`, вызывается в `Update` и влияет на `computeAllowedStatuses`): только **автор, исполнитель, владелец, менеджер группы или админ реалма (realm supervisor)**. Политика Casbin `write` права на смену статуса не даёт — участник группы с realm-wide `ticket:write` переводить статусы не может. Исключения: закрытие/отмена (`closed`/`cancelled`) — только автор/менеджер/владелец (строгая проверка, `write`-права не спасают); терминальные `closed`/`cancelled` вообще неизменяемы.
- **Срок выполнения (dueDate)**: поставить/снять срок при создании и в `Update` может только **менеджер группы тикета или админ реалма** (`CanManage`; в `Update` — кейс `ActionDueDateChanged` в «тонких правах», в `Create` — проверка после `autoAssign`). Создатель/владелец/исполнитель срок менять не могут. Во фронтенде (`InfoBar` — гейт «Установить срок», `TaskEditForm`/`AdvancedSettingsSection` — `canEditDueDate`) та же формула: `task.access?.isManager || capabilities.isRealmAdmin`, плюс активный статус.
- **Подзадачи**: роуты гейтятся только Casbin `read` тикета (`handlers/subtasks`), решения принимает сервис: **создание** (`Create`/`CreateSeveral`) — через **`CanCreateSubtask`** (создатель/исполнитель тикета, менеджер группы, админ реалма; Casbin `write` rights НЕ дают право создавать), плюс `CheckWorkAccess`-блокировка «замороженных»; **`Update`** — через `CheckWorkAccess`; правка содержимого (все поля кроме `status`) — через **`CanEditSubtask`** (автор подзадачи (`subtasks.created_by`, бэкфилл из `activity_logs`) или «управляющий» тикетом `CanManage` — менеджер группы / админ реалма); смену статуса может делать любой обладатель work-доступа. `Delete` — через `CheckWorkAccess` и `CheckAccess(Delete)`. Во фронтенде (`Subtasks`) управление по `canWork`, pencil для правки — `sub.createdBy === userId || canManage`, удаление — `canDelete`; кнопка «Добавить» (`canCreate = task.access?.isManager || task.access?.isAssignee || task.access?.isCreator || capabilities.isRealmAdmin`) и лента скрываются по этим флагам.
- **«Взять в работу»** (`TicketService.Take`, `POST /tickets/:id/take`, флаг `canTake` в `AccessFlags`): пользователь назначает себя исполнителем и переводит заявку в `in_progress`. Разрешено при read-доступе и активном статусе, если исполнителя нет совсем (любой активный статус), либо исполнитель — другой пользователь и статус `open`. Во фронтенде (`InfoBar`) кнопка «Взять в работу» показывается в этих случаях **вместо** «Изменить статус»; если забирать нельзя и у пользователя нет роли для меню статуса (не автор/не исполнитель/не менеджер/не админ), у пользователя ничего не показывается.
- `TicketService.Update`: если у пользователя только work-access, то разрешена смена только статуса (`ActionStatusChanged`, `ActionClosed`), остальные поля — запрет; сама смена статуса дополнительно требует `canChangeStatus`.
- Статусы `closed`/`cancelled` может ставить **автор, владелец (owner) или менеджер группы** (`isCreatorOrManager` + проверка владельца, строгая проверка — Casbin `write`-права не спасают).
- **Владелец (owner)** без write/work-доступа может делать в `Update` только три перехода (`ownerTransitionAllowed`): активный статус → `cancelled` (отменить заявку), `resolved` → `closed` (принять решение), `resolved` → `in_progress` (вернуть в работу). Все прочие смены статусов и изменение полей ему запрещены.
- Во фронтенде (`InfoBar`) кнопка «Изменить статус» показывается только создателю/исполнителю/менеджеру/админу реалма (`isCreator || isAssignee || task.access?.isManager || capabilities.isRealmAdmin`, без `canWrite`); «чистый» владелец видит только свои три кнопки: «Отменить заявку», «Подтвердить решение», «Вернуть в работу».
- Переход в `resolved` проставляет `resolved_at`; тикеты со статусом `resolved` автоматически закрываются через `tickets.resolved_to_closed_after` в `config.yaml` (0 — отключено). `closed_at` при `resolved` не проставляется.
- Переход в `resolved` возможен, только если нет подзадач в активном статусе (`open`/`in_progress`/`pending`/`on_hold`); подзадачи `resolved`/`closed`/`cancelled` не блокируют (`ErrSubtasksNotResolved`, подсчёт — `Subtasks.GetUnresolvedCount`).
- Статус `closed` можно поставить **только из `resolved`** (`ErrCloseRequiresResolved`); нерешённую задачу можно только отменить (`cancelled`).
- Закрытая/отменённая заявка **терминальна**: её статус изменить нельзя никому (в `Update` — `ErrTicketFrozen`; в `computeAllowedStatuses` для `closed`/`cancelled` разрешён лишь взаимообмен `closed⇄cancelled` автору/менеджеру/владельцу, исполнитель с work-доступом активные статусы не получает).
- На «замороженных» заявках (resolved/closed/cancelled) **внешние комментарии запрещены** (`CommentService.Create` → `isTicketInactive`), допустимы только внутренние — от исполнителя/менеджера группы (`CheckInternalAssigneeAccess`). Во фронтенде (`Comments`) на закрытой заявке переключатель скрыт и коммент форсится внутренним.
- **Закрепление (temporary)** нельзя оставлять на «замороженных» заявках (`TicketFavoritesService.Add` при inactive + `favoritesType=temporary` → `ErrTicketFrozen`; кнопка Pin скрыта в `Header`). **Избранное (permanent)** разрешено в любом статусе (кнопка «В избранное» всегда доступна).
- Тикеты без группы: в списке показываются только те, где пользователь создатель или исполнитель (`IncludeUngroupedAssignedTo` в `TicketFilter`).
- `realm` пробрасывается в сервисы подзадач/вложений (variadic `...string`), чтобы Casbin-проверка шла по правильному домену.

## Mattermost-бот (DM)

Реализован в `backend/internal/services/mattermost.go` + `mattermost_dialog.go` (сервис `MattermostService`). Мобильные клиенты MM не грузят webapp плагина, поэтому всё общение — через **нативные кнопки, диалоги и absolute-ссылки**.

- Меню из кнопок «Создать заявку»/«Мои заявки» вешается на **ответы бота** (справка на любое нераспознанное сообщение и команду «помощь»; приглашение на команду «заявка»), а не на открытие диалога — `sendMenu`/`menuButtons`.
- **«Мои заявки»** (`my_tickets`): выборка `myActiveTickets` (активные статусы + `resolved`; пользователь — автор или владелец, `Mode: created_or_owned`), разбивается на сообщения по `myTicketsChunkSize = 5` (предохранитель `myTicketsMaxCount = 200`, пауза `myTicketsPostDelay` между постами). Заголовок первого — «**Ваши активные заявки** (N):», следующих — «Заявки 6–10 из N:»; карточка несёт заголовок, **описание** заявки (`cardSummary` — переносы схлопнуты в одну строку, обрезка на 200 символов с «…»), статус/дату/площадку и действия. пусто — «У вас нет активных заявок.» Ответ приходит **отдельным постом** (не `update`), чтобы не потерять пост с меню.
- **Действия владельца** — per-card кнопки (не `Links`), контекст `{action, ticket_id, status, from}`; `from` — смещение среза, в котором нажали кнопку. Правила как в плагине (`PluginGetTicket` → `CanCancel/CanConfirm/CanReopen`): `ticket_cancel` — владелец и `open → cancelled`; `ticket_confirm` — владелец и `resolved → closed`; `ticket_reopen` — владелец и `resolved → in_progress`. Владелец = `ticket.Owner != nil && ticket.Owner.ID == user.ID` (у заявок, созданных ботом, owner = создатель). Переходы гейтятся существующим `TicketService.Update`.
- **Все три действия идут через диалог подтверждения** — `cancelled`/`closed` терминальные, промах по кнопке необратим, поэтому `handleTicketStatusAction` только открывает диалог и статус не меняет: `openCancelDialog` (без полей; «Заявка №N. <заголовок>. Отменить заявку? Действие необратимо.» / «… Подтвердить решение и закрыть заявку? …», `CallbackId = ticket_cancel:<id>` / `ticket_confirm:<id>`) либо `openReopenDialog` (поле «Причина»). Текст заявки — `ticketSubject` (`GetSummary`, заголовок через `cardSummary`). Целевой статус берётся из действия (`actionTargetStatus`), **не** из `context["status"]` — контекст приходит от клиента.
- **Что происходит с сообщением списка** (`applyListEdit` + `myTicketsChunkAfterEdit`) — отмена и подтверждение решения **уводят заявку из активного списка**, поэтому карточка просто убирается из своего сообщения: срез берётся на элемент короче (`tickets[from : from+size-1]`), так что соседи остаются на местах, а лишняя карточка из следующего сообщения не подтягивается. Если карточка была единственной — сообщение **удаляется** (`Post.Delete`). Возврат в работу заявку в списке оставляет, поэтому срез перерисовывается целиком. **Заголовок всегда считается по состоянию до действия**, чтобы после удаления карточки «из N» не поехал; остальные сообщения списка до следующего «Мои заявки» остаются со старым тоталом. Откат к началу списка при исчезновении среза делать нельзя — это и был баг. Пост удаляется только когда смещение **ровно** совпало с новым концом списка (`from == len`), то есть карточка была единственной; на битом смещении (`from > len`) показываем начало списка, а не удаляем чужое сообщение.
- **`HandleInteractiveAction` → `ActionResult{Ephemeral}`** (`mattermost_dialog.go`): ответ на нажатие — только `ephemeral_text` (нажавшему; текст ошибки из `DomainError.Message()`), поля `UpdatePost`/`Post.ReplyCards` больше нет — все действия уходят в диалоги, а `update` там недоступен.
- **Сабмит диалогов** (`parseTicketActionCallback` → `handleCancelSubmission` / `handleReopenSubmission`): домен → пользователь → `changeTicketStatus` (`actionTargetStatus`) → `applyListEdit` (общий хеллер правки списка, `removed` = ушла ли заявка из активных) → пост-подтверждение в `submission.ChannelId` («Заявка отменена.» / «Заявка закрыта.» / «Заявка возвращена в работу. Причина: …»). У возврата в работу между сменой статуса и подтверждением идёт внешний комментарий «Заявка возвращена в работу: …» (порядок как в `TicketDetail.tsx`, т.к. на `resolved` внешние комментарии запрещены). Пустая причина — `models.ErrReasonRequired`. Ответ диалога не умеет обновлять посты (`SubmitDialogResponse` знает только `error`) и не несёт `post_id`, поэтому смещение и id сообщения кладутся в `State` диалога (`{"from":N,"post_id":"..."}`), а карточка правится прямым `Post.UpdateCards` (`PUT /posts/{id}/patch`) или удаляется `Post.Delete`. Хендлер сабмита (`handlers/mattermost`) на доменную ошибку отвечает `200 {"error": message}` — Mattermost держит диалог открытым и показывает текст.
- `pkg/mattermost`: `postProps`/`cardActions` вешают `Buttons`/`Links` на **свою** карточку (не склеивают в ведущий ряд), `Post.Create` строит props одним вызовом с ведущей карточкой кнопок, `OpenRequest.CallbackID`, `CreatePost` ретраит 429 (`rateLimited`/`retryAfter`).

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
