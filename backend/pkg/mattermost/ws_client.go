package mattermost

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/gorilla/websocket"
	"github.com/mattermost/mattermost/server/public/model"
)

type wsEventData struct {
	ChannelType   string `json:"channel_type"`
	Post          string `json:"post"`
	SenderName    string `json:"sender_name"`
	ChannelID     string `json:"channel_id"`
	TeamID        string `json:"team_id"`
	SetOnline     bool   `json:"set_online"`
	ChannelName   string `json:"channel_name"`
	ChannelDispName string `json:"channel_display_name"`
	Mentions      string `json:"mentions"`
}

type PostedEvent struct {
	ChannelID string
	Post      *model.Post
	Username  string
}

type WSEventHandler func(ctx context.Context, event PostedEvent)

type WSClient struct {
	serverURL string
	botUserID string
	handler   WSEventHandler
	conn      *websocket.Conn
	mu        sync.Mutex
	cancel    context.CancelFunc
}

func NewWSClient(serverURL, botUserID string, handler WSEventHandler) *WSClient {
	return &WSClient{
		serverURL: strings.TrimRight(serverURL, "/"),
		botUserID: botUserID,
		handler:   handler,
	}
}

func (c *WSClient) Run(ctx context.Context, botToken string) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := c.connect(ctx, botToken); err != nil {
			logger.Error("mattermost WS connect failed, retrying...", logger.ErrAttr(err))
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}
		backoff = 1 * time.Second

		logger.Info("mattermost WS connected")
		c.listen(ctx)
		logger.Warn("mattermost WS disconnected, reconnecting...")
	}
}

func (c *WSClient) connect(ctx context.Context, botToken string) error {
	wsURL, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("parse server URL: %w", err)
	}

	switch wsURL.Scheme {
	case "https":
		wsURL.Scheme = "wss"
	default:
		wsURL.Scheme = "ws"
	}
	wsURL.Path = "/api/v4/websocket"

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL.String(), nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	auth := map[string]any{
		"seq":    1,
		"action": "authentication_challenge",
		"data":   map[string]string{"token": botToken},
	}
	if err := conn.WriteJSON(auth); err != nil {
		conn.Close()
		return fmt.Errorf("send auth challenge: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	return nil
}

func (c *WSClient) listen(ctx context.Context) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				if c.conn != nil {
					if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						logger.Warn("mattermost WS ping failed", logger.ErrAttr(err))
						c.mu.Unlock()
						return
					}
				}
				c.mu.Unlock()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logger.Info("mattermost WS closed")
			} else {
				logger.Error("mattermost WS read error", logger.ErrAttr(err))
			}
			return
		}

		var msg struct {
			Event string          `json:"event"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			logger.Warn("mattermost WS unmarshal error", logger.ErrAttr(err))
			continue
		}

		if msg.Event != "posted" || len(msg.Data) == 0 {
			continue
		}

		var data wsEventData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			logger.Warn("mattermost WS unmarshal data error", logger.ErrAttr(err))
			continue
		}

		if data.Post == "" {
			continue
		}

		var post model.Post
		if err := json.Unmarshal([]byte(data.Post), &post); err != nil {
			logger.Warn("mattermost WS unmarshal post error", logger.ErrAttr(err))
			continue
		}

		if post.UserId == c.botUserID {
			continue
		}

		if post.Message == "" && len(post.FileIds) == 0 {
			continue
		}

		c.handler(ctx, PostedEvent{
			ChannelID: post.ChannelId,
			Post:      &post,
			Username:  data.SenderName,
		})
	}
}

func (c *WSClient) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
