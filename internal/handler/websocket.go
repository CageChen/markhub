package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/CageChen/markhub/internal/watcher"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSHandler handles WebSocket connections for hot reload
type WSHandler struct {
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

// NewWSHandler creates a new WebSocket handler
func NewWSHandler() *WSHandler {
	return &WSHandler{
		clients: make(map[*websocket.Conn]bool),
	}
}

// HandleWS handles WebSocket upgrade and connection
func (h *WSHandler) HandleWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() {
		h.removeClient(conn)
		_ = conn.Close()
	}()

	h.addClient(conn)

	// Keep connection alive and handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// OnFileChange is called when a file change is detected
func (h *WSHandler) OnFileChange(event watcher.Event) {
	var eventType string
	switch event.Type {
	case watcher.EventCreate:
		eventType = "create"
	case watcher.EventWrite:
		eventType = "update"
	case watcher.EventRemove:
		eventType = "remove"
	case watcher.EventRename:
		eventType = "rename"
	default:
		return
	}

	msg := WSMessage{
		Type: "fileChange",
		Payload: map[string]string{
			"event": eventType,
			"path":  event.Path,
		},
	}

	h.broadcast(msg)
}

func (h *WSHandler) addClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

func (h *WSHandler) removeClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
}

func (h *WSHandler) broadcast(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			h.removeClient(client)
		}
	}
}
