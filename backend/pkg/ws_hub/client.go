package ws_hub

import (
	"fmt"
	"sync"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *Hub
	closeOnce sync.Once
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		Conn: conn,
		Send: make(chan []byte, 256), // Буфер позволяет хабу не ждать медленного клиента
		Hub:  hub,
	}
}

type WSMessage struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// Метод для внешнего мира: подписать клиента
func (c *Client) Subscribe(topic string) {
	c.Hub.Register <- &Subscription{Client: c, Topic: topic}
}

// Метод для внешнего мира: отписать клиента
func (c *Client) Unsubscribe(topic string) {
	c.Hub.Unregister <- &Subscription{Client: c, Topic: topic}
}

// Close корректно завершает работу с клиентом
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		// close(c.Send)
		c.Conn.Close()
	})
}

func (c *Client) SendJSON(msgType string, payload interface{}) error {
	message := WSMessage{
		Action: msgType,
		Data:   payload,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Используем select, чтобы избежать блокировки, если буфер полон,
	// и проверяем, не закрыт ли канал (через механизм уведомления Хаба)
	select {
	case c.Send <- data:
		return nil
	default:
		// Если буфер забит — это признак того, что клиент "мертв" или слишком медленный.
		// Хаб всё равно его отключит, так что просто возвращаем ошибку.
		return fmt.Errorf("client send buffer full")
	}
}

// версия 2 ----------------
// type Client struct {
// 	Conn *websocket.Conn
// 	Send chan []byte
// 	Hub  *Hub

// 	// Храним множество тем, на которые подписан клиент
// 	// Используем map[string]struct{} как множество (set)
// 	SubscribedTopics map[string]struct{}
// 	mu               sync.Mutex // Защищает доступ к SubscribedTopics
// 	cleanupOnce      sync.Once
// }

// func NewClient(conn *websocket.Conn, hub *Hub) *Client {
// 	return &Client{
// 		Conn:             conn,
// 		Send:             make(chan []byte, 256),
// 		Hub:              hub,
// 		SubscribedTopics: make(map[string]struct{}),
// 	}
// }

// // Метод для добавления темы в список (вызывается после успешной отправки в канал Register)
// func (c *Client) AddTopic(topic string) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	c.SubscribedTopics[topic] = struct{}{}
// }

// // Метод для удаления темы из списка
// func (c *Client) RemoveTopic(topic string) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()
// 	delete(c.SubscribedTopics, topic)
// }

// // Метод для получения копии списка тем (для безопасной итерации при очистке)
// func (c *Client) GetTopics() []string {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	topics := make([]string, 0, len(c.SubscribedTopics))
// 	for topic := range c.SubscribedTopics {
// 		topics = append(topics, topic)
// 	}
// 	return topics
// }

// // Вспомогательный метод SendJSON (как у вас было)
// func (c *Client) SendJSON(msgType string, payload interface{}) error {
// 	message := struct {
// 		Type string      `json:"type"`
// 		Data interface{} `json:"data"`
// 	}{
// 		Type: msgType,
// 		Data: payload,
// 	}

// 	data, err := json.Marshal(message)
// 	if err != nil {
// 		return err
// 	}

// 	c.Send <- data
// 	return nil
// }

// func Cleanup(hub *Hub, client *Client) {
// 	client.cleanupOnce.Do(func() {
// 		// Вся логика очистки внутри
// 		topics := client.GetTopics()
// 		for _, topic := range topics {
// 			select {
// 			case hub.Unregister <- &UnsubscriptionRequest{Client: client, Topic: topic}:
// 			default:
// 				log.Printf("Warning: Could not send unregister request for topic %s, hub busy", topic)
// 			}
// 		}
// 		close(client.Send) // Закрываем канал
// 	})
// }

// ----------------

// func Cleanup(hub *Hub, client *Client) {
// 	// Получаем список тем, пока держим лок клиента (чтобы список не изменился в процессе)
// 	topics := client.GetTopics()

// 	// Отправляем запросы на отписку в хаб для каждой темы
// 	// Важно: делаем это в отдельной горутине или неблокирующе,
// 	// чтобы не заблокировать закрытие, если каналы хаба переполнены.
// 	// Но обычно каналы Register/Unregister буферизированы или обрабатываются быстро.

// 	for _, topic := range topics {
// 		// Отправляем запрос на отписку.
// 		// Если канал полон (хаб завис), мы ничего не можем сделать, кроме как залогировать.
// 		select {
// 		case hub.Unregister <- &UnsubscriptionRequest{Client: client, Topic: topic}:
// 			// Успешно отправлено
// 		default:
// 			// Хаб перегружен, логируем и идем дальше.
// 			// Клиент все равно будет удален из мапы в Hub.Run при следующей попытке записи ему,
// 			// но явная отписка надежнее.
// 			log.Printf("Warning: Could not send unregister request for topic %s, hub busy", topic)
// 		}
// 	}

// 	// Закрываем канал отправки, чтобы горутина записи завершилась
// 	// Делаем это только один раз, проверяя, не закрыт ли он уже (хотя в нашей логике вызов однократный)
// 	select {
// 	case _, ok := <-client.Send:
// 		if ok {
// 			close(client.Send)
// 		}
// 	default:
// 		// Канал уже может быть закрыт в другом месте, игнорируем
// 	}
// }

// func (c *Client) SendJSON(msgType string, payload interface{}) error {
// 	message := struct {
// 		Type string      `json:"type"`
// 		Data interface{} `json:"data"`
// 	}{
// 		Type: msgType,
// 		Data: payload,
// 	}

// 	data, err := json.Marshal(message)
// 	if err != nil {
// 		return err
// 	}

// 	// Просто кладем в твой существующий канал
// 	c.Send <- data
// 	return nil
// }
