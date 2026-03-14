package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/fatih/color"
	"github.com/google/uuid"

	"pan-server/internal/config"
	"pan-server/internal/database"
	"pan-server/internal/types"
)

type Server struct {
	config    *config.Config
	db        database.Store
	agents    map[string]*types.Agent
	rooms     map[string]*types.Room
	ipCounts  map[string]int
	dash      *dashboardHub
	upgrader  websocket.Upgrader
	mutex     sync.RWMutex
}

// New creates a new PAN server instance
func New(cfg *config.Config, db database.Store) *Server {
	return &Server{
		config:   cfg,
		db:       db,
		agents:   make(map[string]*types.Agent),
		rooms:    make(map[string]*types.Room),
		ipCounts: make(map[string]int),
		dash:     newDashboardHub(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// HandleDashboardWS exposes the dashboard hub as an HTTP handler
func (s *Server) HandleDashboardWS(w http.ResponseWriter, r *http.Request) {
	s.dash.ServeHTTP(w, r)
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check total capacity
	s.mutex.RLock()
	totalAgents := len(s.agents)
	s.mutex.RUnlock()

	if totalAgents >= s.config.MaxTotalAgents {
		http.Error(w, "Server at capacity", http.StatusServiceUnavailable)
		return
	}

	// Check per-IP connection limit
	clientIP := getClientIP(r)
	s.mutex.Lock()
	if s.ipCounts[clientIP] >= s.config.MaxAgentsPerIP {
		s.mutex.Unlock()
		http.Error(w, "Too many connections from your IP", http.StatusTooManyRequests)
		color.Yellow("[!] IP limit hit: %s", clientIP)
		return
	}
	s.ipCounts[clientIP]++
	s.mutex.Unlock()

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.mutex.Lock()
		s.ipCounts[clientIP]--
		s.mutex.Unlock()
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create temporary agent for connection
	tempAgent := &types.Agent{
		ID:          uuid.New().String(),
		IP:          clientIP,
		Conn:        conn,
		IsAuthed:    false,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		RateLimit:   types.RateLimit{Count: 0, Start: time.Now()},
		JoinedRooms: make(map[string]bool),
		Friends:     make(map[string]bool),
	}

	color.Yellow("[+] New connection: %s from %s (waiting for auth...)", tempAgent.ID, clientIP)

	// Handle the connection
	s.handleConnection(tempAgent)
}
// handleConnection manages a WebSocket connection
func (s *Server) handleConnection(agent *types.Agent) {
	defer func() {
		s.handleDisconnect(agent)
		agent.Conn.Close()
	}()

	// Set up message handling
	for {
		_, messageData, err := agent.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle the message
		if err := s.handleMessage(agent, messageData); err != nil {
			log.Printf("Message handling error: %v", err)
			s.sendError(agent, err.Error())
		}
	}
}

// handleMessage processes incoming messages
func (s *Server) handleMessage(agent *types.Agent, messageData []byte) error {
	// Parse message
	var msg types.Message
	if err := json.Unmarshal(messageData, &msg); err != nil {
		return err
	}

	// Add timestamp
	msg.Timestamp = time.Now()

	// Rate limiting (skip for auth messages)
	if msg.Type != types.MsgTypeRegister && msg.Type != types.MsgTypeAuth {
		if !s.checkRateLimit(agent) {
			return nil // Silently drop message
		}
	}

	// Route message based on type
	switch msg.Type {
	case types.MsgTypeRegister:
		return s.handleRegister(agent, &msg)
	case types.MsgTypeAuth:
		return s.handleAuth(agent, &msg)
	case types.MsgTypeChat:
		return s.handleChat(agent, &msg)
	case types.MsgTypeJoin:
		return s.handleJoinRoom(agent, &msg)
	case types.MsgTypeLeave:
		return s.handleLeaveRoom(agent, &msg)
	case types.MsgTypeCreateRoom:
		return s.handleCreateRoom(agent, &msg)
	case types.MsgTypeRoomInfo:
		return s.handleRoomInfo(agent, &msg)
	case types.MsgTypeFriendRequest:
		return s.handleFriendRequest(agent, &msg)
	case types.MsgTypeFriendResponse:
		return s.handleFriendResponse(agent, &msg)
	case types.MsgTypeUpdateProfile:
		return s.handleUpdateProfile(agent, &msg)
	case types.MsgTypeGetProfile:
		return s.handleGetProfile(agent, &msg)
	case types.MsgTypeList:
		return s.handleList(agent, &msg)
	case types.MsgTypeGetHistory:
		return s.handleGetHistory(agent, &msg)
	default:
		return s.sendError(agent, "Unknown message type")
	}
}

// checkRateLimit checks if agent is within rate limits
func (s *Server) checkRateLimit(agent *types.Agent) bool {
	now := time.Now()
	
	// Reset window if needed
	if now.Sub(agent.RateLimit.Start) > time.Duration(s.config.RateLimitWindow)*time.Millisecond {
		agent.RateLimit = types.RateLimit{Count: 0, Start: now}
	}

	agent.RateLimit.Count++

	if agent.RateLimit.Count > s.config.MaxMsgsPerWindow {
		if agent.RateLimit.Count == s.config.MaxMsgsPerWindow+1 {
			s.sendError(agent, "Rate limit exceeded")
			color.Yellow("[!] Rate limit: %s", agent.Name)
		}
		return false
	}

	return true
}

// handleDisconnect cleans up when agent disconnects
func (s *Server) handleDisconnect(agent *types.Agent) {
	// Decrement IP count
	if agent.IP != "" {
		s.mutex.Lock()
		if s.ipCounts[agent.IP] > 0 {
			s.ipCounts[agent.IP]--
		}
		if s.ipCounts[agent.IP] == 0 {
			delete(s.ipCounts, agent.IP)
		}
		s.mutex.Unlock()
	}

	if !agent.IsAuthed {
		return
	}

	s.mutex.Lock()
	delete(s.agents, agent.ID)
	online := len(s.agents)
	s.mutex.Unlock()

	// Update last seen in database
	s.db.UpdateLastSeen(agent.ID)

	color.Red("[-] Disconnect: %s", agent.Name)

	// Notify other agents
	s.broadcastSystem(fmt.Sprintf("%s left the network", agent.Name), agent.ID)

	// Notify dashboard
	s.dash.broadcast(DashEvent{
		Type:      "agent_leave",
		Message:   agent.Name + " left the network",
		AgentName: agent.Name,
		Online:    online,
	})
}

// sendMessage sends a message to an agent
func (s *Server) sendMessage(agent *types.Agent, msg *types.Message) error {
	agent.Mutex.Lock()
	defer agent.Mutex.Unlock()

	return agent.Conn.WriteJSON(msg)
}

// sendError sends an error message to an agent
func (s *Server) sendError(agent *types.Agent, message string) error {
	return s.sendMessage(agent, &types.Message{
		Type:      types.MsgTypeError,
		Message:   message,
		Timestamp: time.Now(),
	})
}

// broadcastSystem sends a system message to all agents except excluded
func (s *Server) broadcastSystem(message string, excludeID string) {
	msg := &types.Message{
		Type:      types.MsgTypeSystem,
		Message:   message,
		Timestamp: time.Now(),
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, agent := range s.agents {
		if agent.ID != excludeID && agent.IsAuthed {
			go s.sendMessage(agent, msg)
		}
	}
}