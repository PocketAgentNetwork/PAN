package server

import (
	"time"

	"github.com/fatih/color"
	"pan-server/internal/types"
)

// handleUpdateProfile handles profile update requests
func (s *Server) handleUpdateProfile(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	// Update agent fields if provided
	updated := false

	if msg.Bio != "" {
		agent.Bio = msg.Bio
		updated = true
	}

	if msg.Avatar != "" {
		agent.Avatar = msg.Avatar
		updated = true
	}

	if msg.Status != "" {
		agent.Status = msg.Status
		updated = true
	}

	if msg.Interests != nil {
		agent.Interests = msg.Interests
		updated = true
	}

	if msg.Capabilities != nil {
		agent.Capabilities = msg.Capabilities
		updated = true
	}

	if !updated {
		return s.sendError(agent, "No profile fields to update")
	}

	// Update in database
	if err := s.db.UpdateAgent(agent); err != nil {
		return s.sendError(agent, "Failed to update profile")
	}

	color.Green("[📝] Profile updated: %s", agent.Name)

	// Send confirmation
	confirmMsg := &types.Message{
		Type:      types.MsgTypeAck,
		Message:   "Profile updated successfully",
		Timestamp: time.Now(),
	}

	return s.sendMessage(agent, confirmMsg)
}

// handleGetProfile handles profile retrieval requests
func (s *Server) handleGetProfile(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	targetID := msg.AgentID
	if targetID == "" {
		targetID = agent.ID // Get own profile
	}

	// Get agent profile
	targetAgent, err := s.db.GetAgent(targetID)
	if err != nil {
		return s.sendError(agent, "Agent not found")
	}

	// Send profile info
	response := &types.Message{
		Type:         types.MsgTypeGetProfile,
		AgentID:      targetAgent.ID,
		Name:         targetAgent.Name,
		Bio:          targetAgent.Bio,
		Interests:    targetAgent.Interests,
		Capabilities: targetAgent.Capabilities,
		Avatar:       targetAgent.Avatar,
		Status:       targetAgent.Status,
		Timestamp:    time.Now(),
	}

	return s.sendMessage(agent, response)
}