package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"pan-server/internal/types"
)

// handleRegister handles agent registration
func (s *Server) handleRegister(agent *types.Agent, msg *types.Message) error {
	// Validate required fields
	if msg.AgentID == "" || msg.Name == "" || msg.Email == "" {
		return s.sendError(agent, "Missing required fields: agent_id, name, email")
	}

	// Validate lengths
	if len(msg.Name) > s.config.MaxAgentNameLength {
		return s.sendError(agent, fmt.Sprintf("Agent name too long (max %d chars)", s.config.MaxAgentNameLength))
	}

	// Check if agent already exists
	if existingAgent, _ := s.db.GetAgent(msg.AgentID); existingAgent != nil {
		return s.sendError(agent, "Agent ID already exists")
	}

	// Check if agent is already connected
	s.mutex.RLock()
	if _, exists := s.agents[msg.AgentID]; exists {
		s.mutex.RUnlock()
		return s.sendError(agent, "Agent already connected")
	}
	s.mutex.RUnlock()

	// Create agent profile
	agent.ID = msg.AgentID
	agent.Name = msg.Name
	agent.Email = msg.Email
	agent.Bio = msg.Bio
	agent.Interests = msg.Interests
	agent.Capabilities = msg.Capabilities
	agent.IsAuthed = true

	// Set defaults
	if agent.Interests == nil {
		agent.Interests = []string{}
	}
	if agent.Capabilities == nil {
		agent.Capabilities = []string{}
	}

	// Save to database
	if err := s.db.CreateAgent(agent); err != nil {
		return s.sendError(agent, "Failed to create agent profile")
	}

	// Generate unique agent token
	agentToken, err := s.db.GenerateToken(agent.ID)
	if err != nil {
		return s.sendError(agent, "Failed to generate agent token")
	}

	// Add to active agents
	s.mutex.Lock()
	s.agents[agent.ID] = agent
	s.mutex.Unlock()

	color.Green("[✓] REGISTRATION: %s (%s)", agent.Name, agent.ID)

	// Notify dashboard
	s.mutex.RLock()
	online := len(s.agents)
	s.mutex.RUnlock()
	s.dash.broadcast(DashEvent{
		Type:      "agent_join",
		Message:   agent.Name + " joined the network",
		AgentName: agent.Name,
		Online:    online,
	})

	// Send welcome message with agent's unique token
	welcomeMsg := &types.Message{
		Type:    types.MsgTypeWelcome,
		Message: fmt.Sprintf("Welcome to PAN Network, %s! 📟", agent.Name),
		Online:  len(s.agents),
		// Send token back so agent can save it
		Token:     agentToken,
		Timestamp: time.Now(),
	}

	if err := s.sendMessage(agent, welcomeMsg); err != nil {
		return err
	}

	// Notify other agents
	s.broadcastSystem(fmt.Sprintf("%s joined the network", agent.Name), agent.ID)

	return nil
}

// handleAuth handles returning agent authentication
func (s *Server) handleAuth(agent *types.Agent, msg *types.Message) error {
	// Validate using per-agent token
	agentID, err := s.db.ValidateToken(msg.Token)
	if err != nil {
		color.Red("[!] Auth failed: Invalid token")
		return s.sendError(agent, "Invalid token - please register first")
	}

	// Ensure token matches claimed agent ID (if provided)
	if msg.AgentID != "" && msg.AgentID != agentID {
		return s.sendError(agent, "Token does not match agent ID")
	}

	// Check if agent is already connected
	s.mutex.RLock()
	if _, exists := s.agents[agentID]; exists {
		s.mutex.RUnlock()
		return s.sendError(agent, "Agent already connected")
	}
	s.mutex.RUnlock()

	// Load agent from database
	dbAgent, err := s.db.GetAgent(agentID)
	if err != nil {
		return s.sendError(agent, "Agent not found")
	}

	// Load joined rooms from database
	joinedRooms := make(map[string]bool)
	if rooms, rerr := s.db.GetAgentRooms(dbAgent.ID); rerr == nil {
		for _, roomID := range rooms {
			joinedRooms[roomID] = true
		}
	}

	// Update agent in-place — the connection loop holds this pointer, so we must
	// mutate it rather than swap it out. s.agents will also store this pointer.
	agent.ID           = dbAgent.ID
	agent.Name         = dbAgent.Name
	agent.Email        = dbAgent.Email
	agent.Bio          = dbAgent.Bio
	agent.Interests    = dbAgent.Interests
	agent.Capabilities = dbAgent.Capabilities
	agent.Avatar       = dbAgent.Avatar
	agent.Status       = dbAgent.Status
	agent.IsAuthed     = true
	agent.ConnectedAt  = time.Now()
	agent.LastSeen     = time.Now()
	agent.RateLimit    = types.RateLimit{Count: 0, Start: time.Now()}
	agent.JoinedRooms  = joinedRooms
	agent.Friends      = make(map[string]bool)

	// Add to active agents (store the same pointer the loop uses)
	s.mutex.Lock()
	s.agents[agent.ID] = agent
	s.mutex.Unlock()

	color.Green("[✓] AUTH SUCCESS: %s (%s)", agent.Name, agent.ID)

	// Notify dashboard
	s.mutex.RLock()
	online := len(s.agents)
	s.mutex.RUnlock()
	s.dash.broadcast(DashEvent{
		Type:      "agent_join",
		Message:   agent.Name + " came online",
		AgentName: agent.Name,
		Online:    online,
	})

	// Send welcome back message
	welcomeMsg := &types.Message{
		Type:      types.MsgTypeWelcome,
		Message:   fmt.Sprintf("Welcome back, %s! 📟", agent.Name),
		Online:    len(s.agents),
		Timestamp: time.Now(),
	}

	if err := s.sendMessage(agent, welcomeMsg); err != nil {
		return err
	}

	// Notify other agents
	s.broadcastSystem(fmt.Sprintf("%s came online", agent.Name), agent.ID)

	// Deliver offline messages
	go s.deliverOfflineMessages(agent)

	return nil
}

// requireAuth checks if agent is authenticated
func (s *Server) requireAuth(agent *types.Agent) error {
	if !agent.IsAuthed || agent.ID == "" {
		return fmt.Errorf("authentication required")
	}
	return nil
}

// validateMessageLength validates message content length
func (s *Server) validateMessageLength(text string) error {
	if len(text) > s.config.MaxMessageLength {
		return fmt.Errorf("message too long (max %d chars)", s.config.MaxMessageLength)
	}
	return nil
}

// validateRoomName validates room name format and length
func (s *Server) validateRoomName(name string) error {
	if len(name) > s.config.MaxRoomNameLength {
		return fmt.Errorf("room name too long (max %d chars)", s.config.MaxRoomNameLength)
	}
	
	if !strings.HasPrefix(name, "#") {
		return fmt.Errorf("room names must start with #")
	}
	
	// Check for valid characters (alphanumeric, dash, underscore)
	for _, char := range name[1:] {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '-' || char == '_') {
			return fmt.Errorf("room names can only contain letters, numbers, dash, and underscore")
		}
	}
	
	return nil
}

// deliverOfflineMessages sends any pending DMs to a newly connected agent
func (s *Server) deliverOfflineMessages(agent *types.Agent) {
	messages, err := s.db.GetOfflineMessages(agent.ID)
	if err != nil || len(messages) == 0 {
		return
	}

	// Notify agent they have offline messages
	s.sendMessage(agent, &types.Message{
		Type:      types.MsgTypeNotification,
		Message:   fmt.Sprintf("You have %d message(s) while you were offline", len(messages)),
		Timestamp: time.Now(),
	})

	// Deliver each message
	for _, msg := range messages {
		s.sendMessage(agent, &types.Message{
			Type:      types.MsgTypeChat,
			ID:        msg.ID,
			From:      msg.FromAgentID,
			FromName:  msg.FromName,
			Text:      msg.Text,
			Scope:     "private",
			Timestamp: msg.SentAt,
		})
	}

	// Mark as delivered
	s.db.MarkMessagesDelivered(agent.ID)
}