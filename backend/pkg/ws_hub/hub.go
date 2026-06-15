package ws_hub

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Hub struct {
	Subscribers map[string]map[*Client]struct{}

	Register   chan *Subscription
	Unregister chan *Subscription
	Broadcast  chan *BroadcastMessage
	disconnect chan *Client
	stopped    chan struct{}

	mu       sync.RWMutex
	stopOnce sync.Once
}

type BroadcastMessage struct {
	Topic string
	Data  []byte
}

type Subscription struct {
	Client *Client
	Topic  string
}

func NewWebsocketHub() *Hub {
	return &Hub{
		Subscribers: make(map[string]map[*Client]struct{}),
		Register:    make(chan *Subscription, 256),
		Unregister:  make(chan *Subscription, 256),
		Broadcast:   make(chan *BroadcastMessage, 512),
		disconnect:  make(chan *Client, 256),
		stopped:     make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.Stop()
			return
		case <-h.stopped:
			return

		case req := <-h.Register:
			h.mu.Lock()
			if _, ok := h.Subscribers[req.Topic]; !ok {
				h.Subscribers[req.Topic] = make(map[*Client]struct{})
			}
			h.Subscribers[req.Topic][req.Client] = struct{}{}
			req.Client.addTopic(req.Topic)
			h.mu.Unlock()

		case req := <-h.Unregister:
			h.mu.Lock()
			h.removeSpecific(req.Topic, req.Client)
			h.mu.Unlock()

		case client := <-h.disconnect:
			h.mu.Lock()
			h.fullDisconnect(client)
			h.mu.Unlock()
			client.Close()

		case msg := <-h.Broadcast:
			h.mu.RLock()
			clients, ok := h.Subscribers[msg.Topic]
			if !ok {
				h.mu.RUnlock()
				continue
			}
			snapshot := make([]*Client, 0, len(clients))
			for c := range clients {
				snapshot = append(snapshot, c)
			}
			h.mu.RUnlock()

			for _, client := range snapshot {
				select {
				case client.Send <- msg.Data:
				default:
					h.Disconnect(client)
				}
			}
		}
	}
}

func (h *Hub) Disconnect(c *Client) {
	select {
	case h.disconnect <- c:
	default:
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, data []byte) {
	h.BroadcastMessage("user:"+userID.String(), data)
}

func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	return h.HasSubscribers("user:" + userID.String())
}

func (h *Hub) BroadcastMessage(topic string, data []byte) {
	msg := &BroadcastMessage{Topic: topic, Data: data}
	select {
	case h.Broadcast <- msg:
	case <-time.After(time.Second):
		log.Println("broadcast queue is full, message dropped after 1s timeout")
	}
}

func (h *Hub) HasSubscribers(topic string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients, ok := h.Subscribers[topic]
	return ok && len(clients) > 0
}

func (h *Hub) WaitForFirstSubscriber(ctx context.Context, topic string, timeout time.Duration) bool {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return false
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if h.HasSubscribers(topic) {
				return true
			}
		}
	}
}

func (h *Hub) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopped)

		h.mu.RLock()
		var all []*Client
		seen := make(map[*Client]struct{})
		for _, clients := range h.Subscribers {
			for c := range clients {
				if _, ok := seen[c]; !ok {
					seen[c] = struct{}{}
					all = append(all, c)
				}
			}
		}
		h.mu.RUnlock()

		for _, c := range all {
			c.Conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server is shutting down"),
				time.Now().Add(time.Second))
			h.mu.Lock()
			h.fullDisconnect(c)
			h.mu.Unlock()
			c.Close()
		}
		log.Println("hub stopped: all connections closed")
	})
}

func (h *Hub) removeSpecific(topic string, client *Client) {
	if clients, ok := h.Subscribers[topic]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.Subscribers, topic)
		}
	}
	client.removeTopic(topic)
}

func (h *Hub) fullDisconnect(client *Client) {
	for _, topic := range client.Topics() {
		h.removeSpecific(topic, client)
	}
}
