package models

import (
	"io"
	"time"

	"github.com/google/uuid"
)

// PluginRoutePrefix — базовый путь кастомного маршрута плагина. Mattermost
// монтирует маршруты, зарегистрированные через registerCustomRoute, по
// адресу /plug/<plugin_id>/<route>, поэтому site-relative ссылка открывает
// заявку внутри Mattermost без перехода в веб-приложение.
const PluginRoutePrefix = "/plug/issuetrack"

// PluginContextResult — данные контекста канала для webapp-плагина MM:
// реалм, связанный с каналом, резолвнутый системный пользователь и
// справочники (категории, площадки) для формы создания заявки.
// Bound=false означает, что канал не привязан к реалму (или привязка
// неактивна): нормальное состояние, а не ошибка.
type PluginContextResult struct {
	Bound      bool        `json:"bound"`
	RealmID    uuid.UUID   `json:"realmId"`
	RealmName  string      `json:"realmName"`
	User       PluginUser  `json:"user"`
	Categories []*Category `json:"categories"`
	Sites      []*Site     `json:"sites"`
}

// PluginUser — краткое представление системного пользователя для плагина.
type PluginUser struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	SiteID   *string   `json:"siteId,omitempty"`
}

// PluginFile — парснутый обработчиком файл из multipart-запроса плагина.
type PluginFile struct {
	FileName string
	FileSize int64
	MimeType string
	Content  io.Reader
}

// PluginTicketLinkContext — данные для страницы заявки, открытой по deep-link.
// На этой странице нет текущего канала, поэтому реалм определяется по самой
// заявке, а ChannelID возвращается для последующих запросов (комментарии,
// смена статуса), которые принимают его параметром.
type PluginTicketLinkContext struct {
	ChannelID string              `json:"channelId"`
	Detail    *PluginTicketDetail `json:"detail"`
}

// PluginCreateTicketInput — данные для создания заявки из плагина MM.
// CategoryID/SiteID равны uuid.Nil, если не выбраны.
type PluginCreateTicketInput struct {
	ChannelID   string
	MmUserID    string
	Title       string
	Description string
	CategoryID  uuid.UUID
	SiteID      uuid.UUID
	Files       []PluginFile
}

// PluginCreateTicketResult — результат создания заявки из плагина.
type PluginCreateTicketResult struct {
	ID       uuid.UUID `json:"id"`
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	Link     string    `json:"link"`
	DeepLink string    `json:"deepLink"`
}

// PluginTicketShort — строка списка «Мои активные заявки» для плагина.
type PluginTicketShort struct {
	ID          uuid.UUID      `json:"id"`
	Number      int            `json:"number"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      TicketStatus   `json:"status"`
	Priority    Priority       `json:"priority"`
	CreatedAt   time.Time      `json:"createdAt"`
	Category    *CategoryShort `json:"category,omitempty"`
	Site        *SiteShort     `json:"site,omitempty"`
	Link        string         `json:"link"`
	DeepLink    string         `json:"deepLink"`
}

// PluginTicketDetail — компактная карточка заявки для показа в модалке
// плагина MM (без подзадач/вложений/комментариев).
type PluginTicketDetail struct {
	ID          uuid.UUID      `json:"id"`
	Number      int            `json:"number"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      TicketStatus   `json:"status"`
	Priority    Priority       `json:"priority"`
	CreatedAt   time.Time      `json:"createdAt"`
	DueDate     *time.Time     `json:"dueDate,omitempty"`
	Category    *CategoryShort `json:"category,omitempty"`
	Site        *SiteShort     `json:"site,omitempty"`
	Creator     UserShort      `json:"creator"`
	Owner       *UserShort     `json:"owner,omitempty"`
	Assignee    *UserShort     `json:"assignee,omitempty"`
	Link        string         `json:"link"`
	DeepLink    string         `json:"deepLink"`
	// Attachments — вложенные к заявке файлы (не к комментариям).
	Attachments []*PluginAttachment `json:"attachments,omitempty"`
	// Флаги действий текущего пользователя (формулы те же, что в InfoBar
	// веб-приложения): CanConfirm/CanReopen — владелец и статус resolved,
	// CanCancel — владелец и статус open (отменять можно только новые заявки).
	CanConfirm bool `json:"canConfirm,omitempty"`
	CanReopen  bool `json:"canReopen,omitempty"`
	CanCancel  bool `json:"canCancel,omitempty"`
}

// PluginAttachment — вложение заявки для плагина. Ссылка на скачивание строится
// в webapp-плагине из id (через plugin-proxy, минуя авторизацию программы).
type PluginAttachment struct {
	ID       uuid.UUID `json:"id"`
	FileName string    `json:"fileName"`
	FileSize int64     `json:"fileSize"`
	MimeType string    `json:"mimeType"`
}

// PluginComment — общедоступный комментарий к заявке для плагина.
type PluginComment struct {
	ID          uuid.UUID                  `json:"id"`
	Text        string                     `json:"text"`
	CreatedAt   time.Time                  `json:"createdAt"`
	User        *PluginCommentUser         `json:"user"`
	Attachments []*PluginCommentAttachment `json:"attachments,omitempty"`
}

type PluginCommentUser struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type PluginCommentAttachment struct {
	ID       uuid.UUID `json:"id"`
	FileName string    `json:"fileName"`
	FileSize int64     `json:"fileSize"`
	MimeType string    `json:"mimeType"`
}

// PluginCreateCommentInput — входящие данные комментария из плагина.
type PluginCreateCommentInput struct {
	ChannelID string
	MmUserID  string
	TicketID  string
	Text      string
	Files     []PluginFile
}
