package server

import (
	"strings"
	"time"

	"pan-server/internal/types"
)

// handleGetHistory handles message history requests
func (s *Server) handleGetHistory(agent *types.Agent, msg *types.Message) error {
	if err := s.requireAuth(agent); err != nil {
		return s.sendError(agent, err.Error())
	}

	// Room history
	if msg.Room != "" {
		roomID := strings.TrimPrefix(msg.Room, "#")

		// Check if agent is in the room
		inRoom, err := s.db.IsAgentInRoom(agent.ID, roomID)
		if err != nil || !inRoom {
			return s.sendError(agent, "You are not in this room")
		}

		messages, err := s.db.GetRoomHistory(roomID, 50)
		if err != nil {
			return s.sendError(agent, "Failed to get room history")
		}

		// Send each message back
		for _, m := range messages {
			histMsg := &types.Message{
				Type:      types.MsgTypeChat,
				ID:        m.ID,
				From:      m.FromAgentID,
				FromName:  m.FromName,
				Text:      m.Text,
				Room:      msg.Room,
				Scope:     "room",
				ReplyTo:   m.ReplyTo,
				Timestamp: m.SentAt,
			}
			s.sendMessage(agent, histMsg)
		}

		// Send history end marker
		return s.sendMessage(agent, &types.Message{
			Type:      types.MsgTypeSystem,
			Message:   "End of history",
			Timestamp: time.Now(),
		})
	}

	// DM history
	if msg.To != "" {
		messages, err := s.db.GetDMHistory(agent.ID, msg.To, 50)
		if err != nil {
			return s.sendError(agent, "Failed to get DM history")
		}

		for _, m := range messages {
			histMsg := &types.Message{
				Type:      types.MsgTypeChat,
				ID:        m.ID,
				From:      m.FromAgentID,
				FromName:  m.FromName,
				Text:      m.Text,
				Scope:     "private",
				ReplyTo:   m.ReplyTo,
				Timestamp: m.SentAt,
			}
			s.sendMessage(agent, histMsg)
		}

		return s.sendMessage(agent, &types.Message{
			Type:      types.MsgTypeSystem,
			Message:   "End of history",
			Timestamp: time.Now(),
		})
	}

	return s.sendError(agent, "Specify room or agent ID for history")
}
