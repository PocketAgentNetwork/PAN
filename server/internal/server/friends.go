package server

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"pan-server/internal/types"
)

// handleFriendRequest handles friend request messages
func (s *Server) handleFriendRequest(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	if msg.To == "" {
		return s.sendError(agent, "Target agent ID is required")
	}

	targetID := msg.To

	// Can't friend yourself
	if targetID == agent.ID {
		return s.sendError(agent, "Cannot send friend request to yourself")
	}

	// Check if target agent exists
	targetAgent, err := s.db.GetAgent(targetID)
	if err != nil {
		return s.sendError(agent, "Agent not found")
	}

	// TODO: Check if friendship already exists or pending
	// For now, we'll implement basic friend request

	color.Yellow("[👥] Friend request: %s -> %s", agent.Name, targetAgent.Name)

	// If target is online, send notification
	s.mutex.RLock()
	if onlineTarget, exists := s.agents[targetID]; exists && onlineTarget.IsAuthed {
		notification := &types.Message{
			Type:      types.MsgTypeFriendRequest,
			From:      agent.ID,
			FromName:  agent.Name,
			Message:   fmt.Sprintf("%s sent you a friend request", agent.Name),
			Timestamp: time.Now(),
		}
		
		go s.sendMessage(onlineTarget, notification)
	}
	s.mutex.RUnlock()

	// Send confirmation to requester
	confirmMsg := &types.Message{
		Type:      types.MsgTypeAck,
		Message:   fmt.Sprintf("Friend request sent to %s", targetAgent.Name),
		Timestamp: time.Now(),
	}

	return s.sendMessage(agent, confirmMsg)
}
// handleFriendResponse handles friend request responses (accept/decline)
func (s *Server) handleFriendResponse(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	if msg.From == "" {
		return s.sendError(agent, "Requester agent ID is required")
	}

	requesterID := msg.From
	action := msg.Message // "accept" or "decline"

	if action != "accept" && action != "decline" {
		return s.sendError(agent, "Action must be 'accept' or 'decline'")
	}

	// Check if requester exists
	requesterAgent, err := s.db.GetAgent(requesterID)
	if err != nil {
		return s.sendError(agent, "Requester agent not found")
	}

	color.Yellow("[👥] Friend response: %s %s %s", agent.Name, action, requesterAgent.Name)

	// Update friendship status in database
	// TODO: Implement friendship database operations

	// Notify requester if online
	s.mutex.RLock()
	if onlineRequester, exists := s.agents[requesterID]; exists && onlineRequester.IsAuthed {
		var message string
		if action == "accept" {
			message = fmt.Sprintf("%s accepted your friend request", agent.Name)
		} else {
			message = fmt.Sprintf("%s declined your friend request", agent.Name)
		}

		notification := &types.Message{
			Type:      types.MsgTypeFriendResponse,
			From:      agent.ID,
			FromName:  agent.Name,
			Message:   message,
			Timestamp: time.Now(),
		}
		
		go s.sendMessage(onlineRequester, notification)
	}
	s.mutex.RUnlock()

	// Send confirmation
	confirmMsg := &types.Message{
		Type:      types.MsgTypeAck,
		Message:   fmt.Sprintf("Friend request %s", action),
		Timestamp: time.Now(),
	}

	return s.sendMessage(agent, confirmMsg)
}