package ws_hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *Hub
	closeOnce sync.Once
	UserID    uuid.UUID

	mu              sync.Mutex
	subscribedTopics map[string]struct{}
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		Conn:             conn,
		Send:             make(chan []byte, 256),
		Hub:              hub,
		subscribedTopics: make(map[string]struct{}),
	}
}

func (c *Client) addTopic(topic string) {
	c.mu.Lock()
	c.subscribedTopics[topic] = struct{}{}
	c.mu.Unlock()
}

func (c *Client) removeTopic(topic string) {
	c.mu.Lock()
	delete(c.subscribedTopics, topic)
	c.mu.Unlock()
}

func (c *Client) Topics() []string {
	c.mu.Lock()
	topics := make([]string, 0, len(c.subscribedTopics))
	for t := range c.subscribedTopics {
		topics = append(topics, t)
	}
	c.mu.Unlock()
	return topics
}

func (c *Client) Subscribe(topic string) {
	c.Hub.Register <- &Subscription{Client: c, Topic: topic}
}

func (c *Client) Unsubscribe(topic string) {
	c.Hub.Unregister <- &Subscription{Client: c, Topic: topic}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.Send)
		c.Conn.Close()
	})
}

type WSMessage struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

func (c *Client) SendJSON(msgType string, payload interface{}) error {
	data, err := json.Marshal(WSMessage{Action: msgType, Data: payload})
	if err != nil {
		return err
	}
	select {
	case c.Send <- data:
		return nil
	default:
		return fmt.Errorf("client send buffer full")
	}
}

func (c *Client) ReadPump(timeout time.Duration) {
	defer func() {
		c.Hub.Disconnect(c)
		c.Close()
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(timeout))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(timeout))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) WritePump(pingInterval, writeTimeout time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
