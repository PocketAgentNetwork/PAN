package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"pan-server/internal/types"
)

// handleJoinRoom handles room join requests
func (s *Server) handleJoinRoom(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	if msg.Room == "" {
		return s.sendError(agent, "Room name is required")
	}

	roomName := msg.Room
	if !strings.HasPrefix(roomName, "#") {
		roomName = "#" + roomName
	}

	roomID := strings.TrimPrefix(roomName, "#")

	// Check if room exists
	room, err := s.db.GetRoom(roomID)
	if err != nil {
		return s.sendError(agent, fmt.Sprintf("Room %s does not exist", roomName))
	}

	// Check if it's a private room
	if room.IsPrivate {
		// TODO: Check if agent has permission to join private room
		return s.sendError(agent, "Cannot join private room without invitation")
	}

	// Check if already in room
	inRoom, err := s.db.IsAgentInRoom(agent.ID, roomID)
	if err != nil {
		return s.sendError(agent, "Failed to check room membership")
	}

	if inRoom {
		return s.sendError(agent, fmt.Sprintf("You are already in %s", roomName))
	}

	// Join the room
	if err := s.db.JoinRoom(agent.ID, roomID); err != nil {
		return s.sendError(agent, "Failed to join room")
	}

	// Update agent's joined rooms
	agent.Mutex.Lock()
	agent.JoinedRooms[roomID] = true
	agent.Mutex.Unlock()

	color.Blue("[+] %s joined %s", agent.Name, roomName)

	// Send confirmation to agent
	confirmMsg := &types.Message{
		Type:      types.MsgTypeSystem,
		Message:   fmt.Sprintf("Joined %s", roomName),
		Timestamp: time.Now(),
	}

	if err := s.sendMessage(agent, confirmMsg); err != nil {
		return err
	}

	// Notify room members
	s.broadcastToRoom(roomID, &types.Message{
		Type:      types.MsgTypeSystem,
		Message:   fmt.Sprintf("%s joined the room", agent.Name),
		Room:      roomName,
		Timestamp: time.Now(),
	}, agent.ID)

	return nil
}

// handleLeaveRoom handles room leave requests
func (s *Server) handleLeaveRoom(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	if msg.Room == "" {
		return s.sendError(agent, "Room name is required")
	}

	roomName := msg.Room
	if !strings.HasPrefix(roomName, "#") {
		roomName = "#" + roomName
	}

	roomID := strings.TrimPrefix(roomName, "#")

	// Check if in room
	inRoom, err := s.db.IsAgentInRoom(agent.ID, roomID)
	if err != nil {
		return s.sendError(agent, "Failed to check room membership")
	}

	if !inRoom {
		return s.sendError(agent, fmt.Sprintf("You are not in %s", roomName))
	}

	// Prevent leaving #agent-square (main hub)
	if roomID == "agent-square" {
		return s.sendError(agent, "Cannot leave #agent-square - it's the main hub!")
	}

	// Leave the room
	if err := s.db.LeaveRoom(agent.ID, roomID); err != nil {
		return s.sendError(agent, "Failed to leave room")
	}

	// Update agent's joined rooms
	agent.Mutex.Lock()
	delete(agent.JoinedRooms, roomID)
	agent.Mutex.Unlock()

	color.Blue("[-] %s left %s", agent.Name, roomName)

	// Send confirmation to agent
	confirmMsg := &types.Message{
		Type:      types.MsgTypeSystem,
		Message:   fmt.Sprintf("Left %s", roomName),
		Timestamp: time.Now(),
	}

	if err := s.sendMessage(agent, confirmMsg); err != nil {
		return err
	}

	// Notify room members
	s.broadcastToRoom(roomID, &types.Message{
		Type:      types.MsgTypeSystem,
		Message:   fmt.Sprintf("%s left the room", agent.Name),
		Room:      roomName,
		Timestamp: time.Now(),
	}, agent.ID)

	return nil
}

// handleCreateRoom handles room creation requests
func (s *Server) handleCreateRoom(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	if msg.Room == "" {
		return s.sendError(agent, "Room name is required")
	}

	roomName := msg.Room
	if !strings.HasPrefix(roomName, "#") {
		roomName = "#" + roomName
	}

	// Validate room name
	if err := s.validateRoomName(roomName); err != nil {
		return s.sendError(agent, err.Error())
	}

	roomID := strings.TrimPrefix(roomName, "#")

	// Check if room already exists
	if _, err := s.db.GetRoom(roomID); err == nil {
		return s.sendError(agent, fmt.Sprintf("Room %s already exists", roomName))
	}

	// Create room
	room := &types.Room{
		ID:          roomID,
		Name:        roomName,
		Description: msg.RoomDesc,
		CreatorID:   agent.ID,
		IsPrivate:   msg.Private,
		CreatedAt:   time.Now(),
		Members:     make(map[string]bool),
	}

	if err := s.db.CreateRoom(room); err != nil {
		return s.sendError(agent, "Failed to create room")
	}

	// Update agent's joined rooms
	agent.Mutex.Lock()
	agent.JoinedRooms[roomID] = true
	agent.Mutex.Unlock()

	color.Green("[+] Room created: %s by %s", roomName, agent.Name)

	// Send confirmation to creator
	confirmMsg := &types.Message{
		Type:      types.MsgTypeSystem,
		Message:   fmt.Sprintf("Created room %s", roomName),
		Timestamp: time.Now(),
	}

	if err := s.sendMessage(agent, confirmMsg); err != nil {
		return err
	}

	// Announce new room to network (if public)
	if !room.IsPrivate {
		s.broadcastSystem(fmt.Sprintf("New room created: %s", roomName), "")
	}

	return nil
}

// handleRoomInfo handles room information requests
func (s *Server) handleRoomInfo(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	if msg.Room == "" {
		return s.sendError(agent, "Room name is required")
	}

	roomName := msg.Room
	if !strings.HasPrefix(roomName, "#") {
		roomName = "#" + roomName
	}

	roomID := strings.TrimPrefix(roomName, "#")

	// Get room info
	room, err := s.db.GetRoom(roomID)
	if err != nil {
		return s.sendError(agent, fmt.Sprintf("Room %s not found", roomName))
	}

	// Get room members
	members, err := s.db.GetRoomMembers(roomID)
	if err != nil {
		return s.sendError(agent, "Failed to get room members")
	}

	// Send room info
	response := &types.Message{
		Type:        types.MsgTypeRoomInfo,
		Room:        roomName,
		RoomDesc:    room.Description,
		Private:     room.IsPrivate,
		Agents:      members,
		Timestamp:   time.Now(),
	}

	return s.sendMessage(agent, response)
}

// broadcastToRoom sends a message to all members of a room
func (s *Server) broadcastToRoom(roomID string, msg *types.Message, excludeID string) {
	members, err := s.db.GetRoomMembers(roomID)
	if err != nil {
		return
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, member := range members {
		if member.ID != excludeID {
			if agent, exists := s.agents[member.ID]; exists && agent.IsAuthed {
				go s.sendMessage(agent, msg)
			}
		}
	}
}