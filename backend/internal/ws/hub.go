package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]struct{})}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) BroadcastToSession(sessionID string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.SessionID == sessionID {
			select {
			case c.send <- payload:
			default:
			}
		}
	}
}

func (h *Hub) BroadcastPrice(symbol string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.Watches(symbol) {
			select {
			case c.send <- payload:
			default:
			}
		}
	}
}

type Client struct {
	SessionID string
	Symbols   map[string]struct{}
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	mu        sync.RWMutex
}

func NewClient(sessionID string, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		SessionID: sessionID,
		Symbols:   make(map[string]struct{}),
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 64),
	}
}

func (c *Client) SetSymbols(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Symbols = make(map[string]struct{}, len(symbols))
	for _, s := range symbols {
		c.Symbols[s] = struct{}{}
	}
}

func (c *Client) Watches(symbol string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.Symbols[symbol]
	return ok
}

func (c *Client) WritePump() {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) ReadPump(onClose func()) {
	defer onClose()
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

type PriceMessage struct {
	Type     string  `json:"type"`
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	Open     float64 `json:"open"`
	Stale    bool    `json:"stale"`
	Currency string  `json:"currency"`
	Ts       string  `json:"ts"`
}

func MarshalPrice(pm PriceMessage) []byte {
	pm.Type = "price"
	b, _ := json.Marshal(pm)
	return b
}
