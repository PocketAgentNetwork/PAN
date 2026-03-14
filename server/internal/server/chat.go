package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"pan-server/internal/types"
)

// handleChat handles chat messages
func (s *Server) handleChat(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	if err := s.validateMessageLength(msg.Text); err != nil {
		return s.sendError(agent, err.Error())
	}

	if msg.Text == "" {
		return s.sendError(agent, "Message text cannot be empty")
	}

	// Generate message ID
	msg.ID = uuid.New().String()
	msg.From = agent.ID
	msg.FromName = agent.Name

	// Route message based on destination
	if msg.To == "" || msg.To == "all" {
		// Public broadcast
		return s.handlePublicChat(agent, msg)
	} else if strings.HasPrefix(msg.To, "#") {
		// Room chat
		return s.handleRoomChat(agent, msg)
	} else {
		// Direct message
		return s.handleDirectMessage(agent, msg)
	}
}

// handlePublicChat handles public broadcast messages
func (s *Server) handlePublicChat(agent *types.Agent, msg *types.Message) error {
	msg.Scope = "public"
	
	color.Cyan("[MSG] %s -> ALL: %s", agent.Name, msg.Text)

	// public broadcasts are not shown in the room dashboard (no room = DM/public, skip)
	// Save to database
	s.db.SaveMessage(agent.ID, "", "", msg.Text, "public", msg.ReplyTo)

	// Broadcast to all authenticated agents
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, targetAgent := range s.agents {
		if targetAgent.ID != agent.ID && targetAgent.IsAuthed {
			go s.sendMessage(targetAgent, msg)
		}
	}

	// Increment message count
	s.db.IncrementMessageCount(agent.ID)

	return nil
}

// handleRoomChat handles room-specific messages
func (s *Server) handleRoomChat(agent *types.Agent, msg *types.Message) error {
	roomName := msg.To
	roomID := strings.TrimPrefix(roomName, "#")
	
	// Check if agent is in the room
	inRoom, err := s.db.IsAgentInRoom(agent.ID, roomID)
	if err != nil {
		return s.sendError(agent, "Failed to check room membership")
	}
	
	if !inRoom {
		return s.sendError(agent, fmt.Sprintf("You are not a member of %s", roomName))
	}

	msg.Room = roomName
	msg.Scope = "room"

	color.Blue("[ROOM] %s -> %s: %s", agent.Name, roomName, msg.Text)

	// Notify dashboard — room messages only
	s.dash.broadcast(DashEvent{
		Type:      "room_message",
		Message:   msg.Text,
		AgentName: agent.Name,
		AgentID:   agent.ID,
		Room:      roomName,
		Online:    len(s.agents),
	})

	// Save to database
	s.db.SaveMessage(agent.ID, "", roomID, msg.Text, "room", msg.ReplyTo)

	// Get room members
	members, err := s.db.GetRoomMembers(roomID)
	if err != nil {
		return s.sendError(agent, "Failed to get room members")
	}

	// Send to all room members who are online
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, member := range members {
		if member.ID != agent.ID {
			if targetAgent, exists := s.agents[member.ID]; exists && targetAgent.IsAuthed {
				go s.sendMessage(targetAgent, msg)
			}
		}
	}

	// Save to database
	// TODO: Implement message storage

	// Increment message count
	s.db.IncrementMessageCount(agent.ID)

	return nil
}

// handleDirectMessage handles private messages between agents
func (s *Server) handleDirectMessage(agent *types.Agent, msg *types.Message) error {
	targetID := msg.To
	
	// Check if target agent exists
	targetAgent, exists := s.agents[targetID]
	if !exists {
		// Check if agent exists in database (might be offline)
		if _, err := s.db.GetAgent(targetID); err != nil {
			return s.sendError(agent, "Agent not found")
		}
		// Save as offline message
		s.db.SaveOfflineMessage(msg.ID, agent.ID, targetID, msg.Text)
		return s.sendMessage(agent, &types.Message{
			Type:      types.MsgTypeAck,
			Message:   "Agent is offline - message will be delivered when they reconnect",
			Timestamp: time.Now(),
		})
	}

	msg.Scope = "private"

	color.Magenta("[DM] %s -> %s: %s", agent.Name, targetAgent.Name, msg.Text)

	// Save to database
	s.db.SaveMessage(agent.ID, targetID, "", msg.Text, "dm", msg.ReplyTo)

	// Send to target agent
	if err := s.sendMessage(targetAgent, msg); err != nil {
		return s.sendError(agent, "Failed to deliver message")
	}

	// Send acknowledgment to sender
	ackMsg := &types.Message{
		Type:      types.MsgTypeAck,
		Message:   fmt.Sprintf("Message sent to %s", targetAgent.Name),
		Timestamp: time.Now(),
	}
	
	if err := s.sendMessage(agent, ackMsg); err != nil {
		return err
	}

	// Save to database
	// TODO: Implement message storage

	// Increment message count
	s.db.IncrementMessageCount(agent.ID)

	return nil
}

// handleList handles list requests (agents and rooms)
func (s *Server) handleList(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	// Get online agents
	var onlineAgents []types.AgentInfo
	s.mutex.RLock()
	for _, a := range s.agents {
		if a.IsAuthed {
			onlineAgents = append(onlineAgents, types.AgentInfo{
				ID:   a.ID,
				Name: a.Name,
			})
		}
	}
	s.mutex.RUnlock()

	// Get all public rooms
	rooms, err := s.db.GetAllRooms()
	if err != nil {
		return s.sendError(agent, "Failed to get rooms list")
	}

	// Send response
	response := &types.Message{
		Type:      types.MsgTypeList,
		Agents:    onlineAgents,
		Rooms:     rooms,
		Timestamp: time.Now(),
	}

	return s.sendMessage(agent, response)
}