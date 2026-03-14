package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DashEvent is a live event pushed to dashboard watchers
type DashEvent struct {
	Type      string `json:"type"`
	Message   string `json:"message,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	Room      string `json:"room,omitempty"`
	Online    int    `json:"online"`
	Rooms     int    `json:"rooms,omitempty"`
	Messages  int    `json:"messages,omitempty"`
	Timestamp string `json:"timestamp"`
}

// dashboardHub manages read-only dashboard WebSocket connections
type dashboardHub struct {
	clients  map[*websocket.Conn]bool
	mu       sync.Mutex
	upgrader websocket.Upgrader
}

func newDashboardHub() *dashboardHub {
	return &dashboardHub{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// ServeHTTP upgrades dashboard browser connections to WebSocket
func (h *dashboardHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Dashboard WS upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Read loop — just drain pings, dashboard is read-only
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
}

// broadcast sends an event to all connected dashboard clients
func (h *dashboardHub) broadcast(event DashEvent) {
	event.Timestamp = time.Now().Format("15:04:05")
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
