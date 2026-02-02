package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/errors"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

type Client struct {
	conn          *websocket.Conn
	url           string
	token         string
	mu            sync.Mutex
	handlers      map[string]MessageHandler // Message type handlers
	topicHandlers map[string]TopicHandler   // Topic-specific handlers
	isRunning     bool
	done          chan struct{}
}

type MessageHandler func(msgType string, payload []byte)

// TopicHandler handles messages for a specific topic
type TopicHandler func(payload []byte)

type Message struct {
	Type    string          `json:"type"`
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

type SubscribeMessage struct {
	Action string `json:"action"`
	Topic  string `json:"topic"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		url:           cfg.DnseWsUrl,
		handlers:      make(map[string]MessageHandler),
		topicHandlers: make(map[string]TopicHandler),
		done:          make(chan struct{}),
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger.Info().Str("url", c.url).Msg("Connecting to WebSocket")

	headers := make(map[string][]string)
	if c.token != "" {
		headers["Authorization"] = []string{"Bearer " + c.token}
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(c.url, headers)
	if err != nil {
		logger.Error().Err(err).Msg("WebSocket connection failed")
		return errors.Wrap(err, errors.ErrWsConnectionFailed)
	}

	c.conn = conn
	c.isRunning = true

	logger.Info().Msg("WebSocket connected successfully")

	go c.listen()

	return nil
}

func (c *Client) Subscribe(topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return errors.ErrWsConnectionFailed
	}

	msg := SubscribeMessage{
		Action: "subscribe",
		Topic:  topic,
	}

	data, _ := json.Marshal(msg)
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Error().Err(err).Str("topic", topic).Msg("Failed to subscribe")
		return errors.Wrap(err, errors.ErrWsSubscribeFailed)
	}

	logger.Info().Str("topic", topic).Msg("Subscribed to topic")
	return nil
}

func (c *Client) Unsubscribe(topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	msg := SubscribeMessage{
		Action: "unsubscribe",
		Topic:  topic,
	}

	data, _ := json.Marshal(msg)
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) RegisterHandler(msgType string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[msgType] = handler
}

// RegisterTopicHandler registers a callback for a specific topic
func (c *Client) RegisterTopicHandler(topic string, handler TopicHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.topicHandlers[topic] = handler
}

// UnregisterTopicHandler removes a topic handler
func (c *Client) UnregisterTopicHandler(topic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.topicHandlers, topic)
}

func (c *Client) listen() {
	defer func() {
		c.isRunning = false
		close(c.done)
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				logger.Info().Msg("WebSocket closed normally")
				return
			}
			logger.Error().Err(err).Msg("WebSocket read error")
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Warn().Err(err).Msg("Failed to parse WebSocket message")
			continue
		}

		c.mu.Lock()
		// First try topic-specific handler
		topicHandler, topicExists := c.topicHandlers[msg.Topic]
		// Then try message type handler
		typeHandler, typeExists := c.handlers[msg.Type]
		c.mu.Unlock()

		// Priority: topic handler > type handler
		if topicExists {
			topicHandler(msg.Payload)
		} else if typeExists {
			typeHandler(msg.Type, msg.Payload)
		} else {
			logger.Debug().
				Str("type", msg.Type).
				Str("topic", msg.Topic).
				Msg("No handler for message")
		}
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.isRunning = false
		err := c.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			return err
		}
		return c.conn.Close()
	}
	return nil
}

func (c *Client) IsConnected() bool {
	return c.isRunning
}
