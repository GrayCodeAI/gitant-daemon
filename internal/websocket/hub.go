package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var allowedOrigins = []string{
	"http://localhost:3303",
	"http://localhost:3456",
	"http://localhost:3000",
	"https://gitant.dev",
	"https://*.gitant.dev",
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		for _, allowed := range allowedOrigins {
			if strings.HasSuffix(allowed, "*") {
				// Wildcard subdomain matching
				domain := strings.TrimPrefix(allowed, "https://*.")
				if strings.HasSuffix(origin, "."+domain) || origin == "https://"+domain {
					return true
				}
			} else if origin == allowed {
				return true
			}
		}
		return false
	},
}

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client represents a WebSocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
	repos  map[string]bool
	mu     sync.RWMutex
}

// Hub manages WebSocket clients
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Info("websocket client connected", "user_id", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			slog.Info("websocket client disconnected", "user_id", client.userID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToRepo sends a message to all clients watching a repo
func (h *Hub) BroadcastToRepo(repoID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal websocket message", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.mu.RLock()
		watching := client.repos[repoID]
		client.mu.RUnlock()

		if watching {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// BroadcastToUser sends a message to a specific user
func (h *Hub) BroadcastToUser(userID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal websocket message", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// BroadcastAll sends a message to all clients
func (h *Hub) BroadcastAll(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal websocket message", "error", err)
		return
	}

	h.broadcast <- data
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(hub *Hub, userID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			return
		}

		client := &Client{
			hub:    hub,
			conn:   conn,
			send:   make(chan []byte, 256),
			userID: userID,
			repos:  make(map[string]bool),
		}

		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("websocket read error", "error", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case "subscribe":
		if repoID, ok := msg.Payload.(string); ok {
			c.mu.Lock()
			c.repos[repoID] = true
			c.mu.Unlock()
		}

	case "unsubscribe":
		if repoID, ok := msg.Payload.(string); ok {
			c.mu.Lock()
			delete(c.repos, repoID)
			c.mu.Unlock()
		}

	case "ping":
		c.send <- []byte(`{"type":"pong"}`)
	}
}

// Event types
const (
	EventIssueCreated = "issue.created"
	EventIssueUpdated = "issue.updated"
	EventIssueClosed  = "issue.closed"
	EventPROpened     = "pr.opened"
	EventPRMerged     = "pr.merged"
	EventPRReviewed   = "pr.reviewed"
	EventPush         = "push"
	EventComment      = "comment"
	EventRelease      = "release"
	EventNotification = "notification"
)

// NotifyIssueCreated notifies clients about a new issue
func (h *Hub) NotifyIssueCreated(repoID string, issue interface{}) {
	h.BroadcastToRepo(repoID, Message{
		Type:    EventIssueCreated,
		Payload: issue,
	})
}

// NotifyPROpened notifies clients about a new PR
func (h *Hub) NotifyPROpened(repoID string, pr interface{}) {
	h.BroadcastToRepo(repoID, Message{
		Type:    EventPROpened,
		Payload: pr,
	})
}

// NotifyPush notifies clients about a push event
func (h *Hub) NotifyPush(repoID string, push interface{}) {
	h.BroadcastToRepo(repoID, Message{
		Type:    EventPush,
		Payload: push,
	})
}

// NotifyUser sends a notification to a specific user
func (h *Hub) NotifyUser(userID string, notification interface{}) {
	h.BroadcastToUser(userID, Message{
		Type:    EventNotification,
		Payload: notification,
	})
}
