# TODO

Живой список задач. Отмечай `[x]` когда сделано. Детальные планы фич — в `.opencode/plans/`.

## Сделано (текущая сессия)

- [x] **Фикс смены статуса (partial update)**: `TicketDTO`/`SubtaskDTO` получили presence-tracking (`Provided`, `UnmarshalJSON` на goccy/go-json), `GetChanges` и динамический `SET` в `postgres/{tickets,subtasks}.go` — частичное обновление не затирает незатронутые поля. Frontend: `updateTask`/`updateSubtask` принимают `Partial<DTO>`; убран `as any` в `TaskDetail.tsx`.
- [x] **Статусы `closed`/`cancelled` — только автор тикета или менеджер группы**: проверка `isCreatorOrManager` в `TicketService.Update` (Casbin `write` не спасает; строгая). `resolved` по-прежнему может ставить исполнитель.
- [x] **Авто-закрытие `resolved` → `closed`**: колонка `tickets.resolved_at` (миграция `20260813000001_...` + бэкафилл), проставляется при переходе в `resolved`; время ожидания — `tickets.resolved_to_closed_after` в `config.yaml` (168h = 7 дней, 0 — отключено).
- [x] **Планировщик на `gocron/v2`** (вместо тикера раз в минуту): `backend/internal/services/scheduler.go` (`SchedulerService`, `Start`/`Stop`), расписание — `tickets.auto_close_schedule` (`@daily`), запуск/остановка в `cmd/app/main.go`. Зависимость `github.com/go-co-op/gocron/v2 v2.16.3`.
- [x] **Заполнение `resolved_at`/`closed_at`** в `TicketService.Update`: `resolved` → `resolved_at`, `closed`/`cancelled` → `closed_at`, активные статусы очищают оба. `resolvedAt` добавлен в типы frontend.
- [x] **Валидация переходов статусов** в `TicketService.Update`: в `resolved` — нельзя, пока есть подзадачи в активном статусе (`open`/`in_progress`/`pending`/`on_hold`; `resolved`/`closed`/`cancelled` не блокируют) — `ErrSubtasksNotResolved`, подсчёт через `Subtasks.GetUnresolvedCount`; в `closed` — только из `resolved` (`ErrCloseRequiresResolved`), нерешённую задачу можно только отменить.
- [x] **Владелец: отменить / подтвердить решение / вернуть в работу** (бэкенд + UI): в `TicketService.Update` добавлен fallback на владельца в initial access check (`ownerOnly`); владельцу разрешены только три перехода (`ownerTransitionAllowed`): активный → `cancelled`, `resolved` → `closed`, `resolved` → `in_progress`; владелец пропускается в строгом gate `closed`/`cancelled`. В `InfoBar` меню «Изменить статус» показывается только создателю/исполнителю/менеджеру (`isCreator || isAssignee || canWrite`), «чистый» владелец видит три кнопки: «Отменить заявку» (XCircle), «Подтвердить решение» (CheckCircle2), «Вернуть в работу» (RotateCcw) — по `task.owner?.id` (селектор `getUserId`). В `TaskCreateForm` добавлено поле «Заказчик» (manager-only, `ownerId`); обычный пользователь owner не указывает (бэкенд ставит owner = creator).
- [x] **Сырые UUID видны только root**: хук `useIsRoot` (`frontend/src/features/access/utils/can.ts`, по наличию `*`/`*:*` в permissions). Полные id категории/площадки/группы в диалогах просмотра, `ID:` на карточке площадки, MetaRow «ID» на детальной странице — скрыты у обычных пользователей, у root показываются полностью (без обрезки). В шапке тикета и в таблице/карточках — только номер (полный id уже в блоке «Детали»), фолбэк `task.id.slice(0, 4)` убран совсем. `Role.tsx` (slug/realmId) не трогали (решение пользователя).
- [x] **Автозаполнение в форме создания заявки** (`TaskCreateForm.tsx`): каскад «категория → группа → исполнитель» для менеджера — при выборе категории проставляется `groupId` из `category.groupId`, при выборе группы — `assigneeId` из `group.defaultAssigneeId` (нет дефолта → пусто, бэкенд сам авто-назначит единственного участника на `Create`), очистка группы чистит исполнителя. Семантика — всегда перезаписывать при смене источника. Список «Исполнитель» фильтруется по участникам выбранной группы (`group.members`, бэкенд уже подгружает их в `GroupService.Get`), без группы — все пользователи.
- [x] **Редизайн формы создания заявки по макету** (`CreateTask.html` → MUI `sx`): секции-карточки с номерами «Что случилось и где» (категория и площадка — Autocomplete в 2 колонки с иконками `Layers`/`Building2` и описанием/адресом второй строкой; иконки одинаковые у всех опций, поле скроллится при любом количестве категорий), «Описание проблемы» (заголовок со счётчиком `n/150`, textarea, drag&drop вложений), «Расширенные настройки» (только менеджер, всегда раскрыт, без чекбокса: приоритет карточками с описаниями + «рекомендуется», заявитель, группа «Авто (по категории)», исполнитель «Авто (по группе)», срок). Вложения грузятся после `createTask` через существующий `POST /attachments/ticket/:id` (multipart `file`): новый `frontend/src/features/tasks/modules/attachments/attachmentsApiSlice.ts` (`uploadAttachment`, FormData, invalidates `Tasks`), в `api.ts` добавлен `attachments.upload`. Частичный сбой загрузки → тост-предупреждение. Без SLA/превью/тегов/черновика (решение пользователя).

## Незакоммичено (сейчас в git status)

- [x] Закоммитить: фикс смены статуса (partial update): `backend/internal/models/{ticket,subtask}.go`, `backend/internal/repository/postgres/{tickets,subtasks}.go`, `backend/internal/services/tickets.go`, тесты, `frontend/src/features/tasks/{tasksApiSlice.ts,modules/subtasks/subtasksApiSlice.ts,pages/TaskDetail.tsx}` + ранее: `InfoBar.tsx`, `sidebarConf.tsx`, `.gitignore` (добавлен `.opencode`)
- [x] Закоммитить: закрытие/отмена только автору/менеджеру + авто-закрытие resolved + планировщик + валидация переходов (resolved — подзадачи, closed — только из resolved): миграция `20260813000001_add_resolved_at_to_tickets.sql`, `backend/internal/{models/ticket.go,models/errors.go,repository/postgres/tickets.go,services/tickets.go,services/subtasks.go,services/scheduler.go,services/services.go,config/config.go,cmd/app/main.go}`, `backend/configs/config.yaml` (`resolved_to_closed_after`, `auto_close_schedule`), тесты (`tickets_test.go`, `scheduler_test.go`), `frontend/src/features/tasks/types/task.ts`, `go.mod`/`go.sum` (gocron/v2); рефакторинг `NewTicketService` → `TicketDeps`
- [x] Решить что делать с удалённым бинарником `backend/app` (добавить в `.gitignore`?)
- [x] Закоммитить: права владельца + UI: `backend/internal/services/tickets.go` (`isOwner`, `ownerTransitionAllowed`, owner-fallback в access check), `backend/internal/services/tickets_test.go` (owner-тесты), `frontend/src/features/tasks/components/Detail/InfoBar.tsx` (3 кнопки владельца + скрытие меню «Изменить статус»), `frontend/src/features/tasks/components/TaskCreateForm.tsx` (поле «Заказчик», manager-only), `frontend/src/features/user/userSlice.ts` (селектор `getUserId`), `AGENTS.md`, `TODO.md`
- [x] Закоммитить: сырые UUID только для root: `frontend/src/features/access/utils/can.ts` (`useIsRoot`), `CategoryViewDialog.tsx`, `SiteViewDialog.tsx`, `SiteCardList.tsx`, `GroupViewDialog.tsx`, `tasks/.../Detail/{Meta,Header}.tsx` (в шапке только номер), `tasks/.../Table/{TaskCard,TaskRow}.tsx` (только номер), `TODO.md`
- [x] Закоммитить: автозаполнение в форме создания: `frontend/src/features/tasks/components/TaskCreateForm.tsx` (каскад категория→группа→исполнитель + фильтр исполнителя по участникам группы), `TODO.md`
- [x] Закоммитить: редизайн формы создания + вложения: `frontend/src/features/tasks/components/TaskCreateForm/` (новая папка вместо одного файла: `TaskCreateForm.tsx`, `index.ts`, `types.ts`, `styles.ts`, `SectionCard.tsx`, `CategoryAndSiteSection.tsx`, `DescriptionSection.tsx`, `AdvancedSettingsSection.tsx`, `PriorityCard.tsx`, `priorityDescriptions.ts`, `FileDropZone.tsx`; секции-карточки, категория/площадка Autocomplete с иконками `Layers`/`Building2`, drag&drop вложений, расширенные настройки всегда раскрыты; фикс типа `assigneeOptions` — нормализация `IUserData[]`→`IUserShort[]`), `frontend/src/features/tasks/modules/attachments/attachmentsApiSlice.ts` (новый, `uploadAttachment`), `frontend/src/app/api.ts` (`attachments.upload`), `TODO.md`

## Фичи (планы в `.opencode/plans/`)

- [ ] **Архив тикетов + пагинация только для архива** — план: `.opencode/plans/archive-pagination.md`

## Заглушки / не реализовано

- [x] **Редактирование задачи**: UI готов. Кнопка «Редактировать» в `Description.tsx` (только `open` + creator/manager), `TaskEditModal` → `TaskEditForm` (секции: категория/площадка, заголовок/описание, расширенные настройки — менеджер), без каскадов (в отличие от создания), `updateTask` с `Partial<ITaskDTO>`. Бэкенд `Update` уже поддерживает partial update.
- [ ] **Подзадачи: добавить/удалить/редактировать**: сейчас только смена статуса; бэкенд `Create/CreateSeveral/Update/Delete` готов. UI «Добавить» нет.
- [ ] **Скачивание/просмотр вложений**: файлы уже лежат на диске (`upload_dir/ticket/<ticketId>/<uuid>_<name>`) и в БД, но содержимое никуда не отдаётся — эндпоинта нет, `initStatic` раздаёт только фронтенд-сборку, `file_path` в JSON скрыт. Клик по карточке в `Detail/Attachments.tsx` ничего не делает. План: бэкенд `GET /attachments/:id/content` (проверка доступа Read по тикету + `Content-Disposition` / `c.File`), фронт — ссылка на скачивание в `Attachments.tsx`.
- [ ] **Вложения: удаление + загрузка с детальной страницы**: загрузка реализована только в форме создания (`POST /attachments/ticket/:id`, `CheckWorkAccess`); удаления (`DELETE /attachments/:id`) и пикера файлов на странице тикета нет.
- [ ] **Удаление задачи**: нет UI (бэкенд `Delete` готов, менеджер группы).
- [ ] **Комментарии**: бэкенд handler заглушен (`backend/internal/transport/http/handlers/comments/comments.go:15` — `//TODO реализовать`), сервиса/репозитория нет. Фронт передаёт пустой массив (`TaskDetail.tsx` — `comments={[]}`), сам компонент `Detail/Comments.tsx` готов.
- [ ] **Уведомления по задаче**: бэкенд handler заглушен (`backend/internal/transport/http/handlers/notifications/notifications.go:23` — `//TODO реализовать`). Фронт `Detail/Notifications.tsx` — чекбокс-заглушка без подключения к API.

## Разделение прав

- [x] **UI по правам**: `AccessFlags` в ответе `GetByID` (`canRead`/`canWrite`/`canDelete`/`canWork`/`allowedStatuses`). Фронт: гейт «Создать заявку» (`useCan(Tasks.Write)`), subtask select `disabled={!canWork}`, статус-меню фильтруется по `allowedStatuses`, edit button по `canWrite`. `PermRules` — добавлены `Tasks.Read`, `Tasks.Delete`.
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
