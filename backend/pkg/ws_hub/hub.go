package ws_hub

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	// Мапа подписок: "topic_name" -> map[*Client]bool
	// Пример ключей: "orders", "notifications", "chat_room_123"
	Subscribers map[string]map[*Client]struct{}

	Register   chan *Subscription
	Unregister chan *Subscription
	Broadcast  chan *BroadcastMessage
	disconnect chan *Client

	mu sync.RWMutex
}

// Запрос на подписку/отписку
type SubscriptionRequest struct {
	Client *Client
	Topic  string
}

// Запрос на отписку (можно использовать тот же, но для ясности разделим или передадим флаг)
type UnsubscriptionRequest struct {
	Client *Client
	Topic  string
}

// Сообщение для рассылки
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
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Завершаем работу: закрываем все соединения
			h.Stop()
			return

		case req := <-h.Register:
			if _, ok := h.Subscribers[req.Topic]; !ok {
				h.Subscribers[req.Topic] = make(map[*Client]struct{})
			}
			h.Subscribers[req.Topic][req.Client] = struct{}{}

		case req := <-h.Unregister:
			h.removeSpecific(req.Topic, req.Client)

		case client := <-h.disconnect:
			h.fullDisconnect(client)

		case msg := <-h.Broadcast:
			if clients, ok := h.Subscribers[msg.Topic]; ok {
				for client := range clients {
					select {
					case client.Send <- msg.Data:
					default:
						// Если буфер клиента забит — отключаем его полностью
						h.fullDisconnect(client)
					}
				}
			}
		}

		// case req := <-h.Register:
		// 	h.mu.Lock()
		// 	if _, ok := h.Subscribers[req.Topic]; !ok {
		// 		h.Subscribers[req.Topic] = make(map[*Client]bool)
		// 	}
		// 	h.Subscribers[req.Topic][req.Client] = true
		// 	h.mu.Unlock()

		// 	// Сообщаем клиенту, что он успешно добавлен в локальный список (опционально)
		// 	req.Client.AddTopic(req.Topic)

		// case req := <-h.Unregister:
		// 	h.mu.Lock()
		// 	if clients, ok := h.Subscribers[req.Topic]; ok {
		// 		if _, exists := clients[req.Client]; exists {
		// 			delete(clients, req.Client)
		// 			// Удаляем тему, если она пуста
		// 			if len(clients) == 0 {
		// 				delete(h.Subscribers, req.Topic)
		// 			}
		// 		}
		// 	}
		// 	h.mu.Unlock()

		// 	// Удаляем тему из списка клиента
		// 	req.Client.RemoveTopic(req.Topic)

		// case msg := <-h.Broadcast:
		// 	h.mu.Lock()
		// 	// Клонируем список клиентов для безопасности, если нужно,
		// 	// но обычно итерация по мапе под локом допустима, если мы не меняем мапу
		// 	if clients, ok := h.Subscribers[msg.Topic]; ok {
		// 		for client := range clients {
		// 			select {
		// 			case client.Send <- msg.Data:
		// 				// Успех
		// 			default:
		// 				// БУФЕР ПЕРЕПОЛНЕН! Клиент не успевает обрабатывать сообщения.
		// 				// Нужно срочно отключить его, чтобы не блокировать весь хаб.

		// 				// ВАЖНО: Мы не можем вызвать Cleanup прямо здесь под локом хаб-а (deadlock риск),
		// 				// поэтому просто удаляем из мапы, а полную очистку инициируем асинхронно
		// 				delete(clients, client)

		// 				go func(c *Client) {
		// 					// Асинхронная полная очистка в отдельной горутине
		// 					Cleanup(h, c)
		// 				}(client)
		// 			}
		// 		}
		// 	}
		// 	h.mu.Unlock()
		// }
	}
}

func (h *Hub) Disconnect(c *Client) {
	select {
	case h.disconnect <- c:
	default:
		// Если канал переполнен, значит хаб и так занят очисткой или завершением
	}
}

func (h *Hub) BroadcastMessage(topic string, data []byte) {
	msg := &BroadcastMessage{
		Topic: topic,
		Data:  data,
	}

	select {
	case h.Broadcast <- msg:
	default:
		log.Println("Broadcast queue is full")
	}
}

func (h *Hub) HasSubscribers(topic string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.Subscribers[topic]
	return ok && len(clients) > 0
}

// Полезный помощник: ждем подписчика с таймаутом
func (h *Hub) WaitForFirstSubscriber(ctx context.Context, topic string, timeout time.Duration) bool {
	stop := time.After(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return false // Никто не пришел за 5 секунд
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if h.HasSubscribers(topic) {
				return true
			}
		}
	}
}

// Вспомогательный метод для очистки при выключении сервера
func (h *Hub) Stop() {
	// Собираем всех уникальных клиентов
	uniqueClients := make(map[*Client]struct{})
	for _, clients := range h.Subscribers {
		for c := range clients {
			uniqueClients[c] = struct{}{}
		}
	}
	// Закрываем каждого
	for c := range uniqueClients {
		// Отправляем CloseMessage с кодом 1001 (Going Away)
		msg := websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server is shutting down")

		// Пытаемся отправить напрямую в сокет, так как цикл хаба может быть уже блокирован
		c.Conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))

		// Вызываем полную очистку через существующий механизм
		h.fullDisconnect(c)
	}
	log.Println("Hub stopped: all connections closed")
}

// Вспомогательные неэкспортируемые методы (вызываются только внутри Run)
func (h *Hub) removeSpecific(topic string, client *Client) {
	if clients, ok := h.Subscribers[topic]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.Subscribers, topic)
		}
	}
}

func (h *Hub) fullDisconnect(client *Client) {
	// Удаляем клиента из всех тем
	for topic := range h.Subscribers {
		h.removeSpecific(topic, client)
	}
	// Закрываем канал клиента только здесь, когда уверены, что Hub больше не будет туда слать
	client.Close()
}

// type Hub struct {
// 	// Сохраняем клиентов в map для эффективного удаления
// 	Clients    map[*Client]bool
// 	Register   chan *Client
// 	Unregister chan *Client
// 	broadcast  chan []byte
// 	mu         sync.Mutex
// }

// func NewWebsocketHub() *Hub {
// 	return &Hub{
// 		Clients:    make(map[*Client]bool),
// 		Register:   make(chan *Client),
// 		Unregister: make(chan *Client),
// 		broadcast:  make(chan []byte),
// 	}
// }

// // Реализация интерфейса UseCase.MessageBroadcaster
// func (h *Hub) Broadcast(data []byte) {
// 	h.broadcast <- data
// }

// func (h *Hub) Run() {
// 	for {
// 		select {
// 		case client := <-h.Register:
// 			h.mu.Lock()
// 			h.Clients[client] = true
// 			h.mu.Unlock()

// 		case client := <-h.Unregister:
// 			h.mu.Lock()
// 			if _, ok := h.Clients[client]; ok {
// 				delete(h.Clients, client)
// 				close(client.Send)
// 			}
// 			h.mu.Unlock()

// 		case message := <-h.broadcast:
// 			h.mu.Lock()
// 			for client := range h.Clients {
// 				if client.Mode == ModeOrderListener {
// 					select {
// 					case client.Send <- message:
// 					default:
// 						// Если буфер клиента переполнен, закрываем соединение
// 						close(client.Send)
// 						delete(h.Clients, client)
// 					}
// 				}
// 			}
// 			h.mu.Unlock()
// 		}
// 	}
// }
