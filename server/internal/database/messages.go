package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StoredMessage represents a message stored in the database
type StoredMessage struct {
	ID          string    `json:"id"`
	FromAgentID string    `json:"from_agent_id"`
	FromName    string    `json:"from_name"`
	ToAgentID   string    `json:"to_agent_id,omitempty"`
	RoomID      string    `json:"room_id,omitempty"`
	Text        string    `json:"text"`
	MessageType string    `json:"message_type"` // "dm", "room", "public"
	ReplyTo     string    `json:"reply_to,omitempty"`
	SentAt      time.Time `json:"sent_at"`
}

// SaveMessage saves a message to the database
func (db *DB) SaveMessage(fromID, toID, roomID, text, msgType, replyTo string) (string, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO messages (id, from_agent_id, to_agent_id, room_id, message_text, message_type, reply_to, sent_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Use nil for empty optional fields
	var toIDVal, roomIDVal, replyToVal interface{}
	if toID != "" {
		toIDVal = toID
	}
	if roomID != "" {
		roomIDVal = roomID
	}
	if replyTo != "" {
		replyToVal = replyTo
	}

	_, err := db.conn.Exec(query, id, fromID, toIDVal, roomIDVal, text, msgType, replyToVal, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to save message: %w", err)
	}

	return id, nil
}

// GetRoomHistory returns recent messages from a room
func (db *DB) GetRoomHistory(roomID string, limit int) ([]StoredMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT m.id, m.from_agent_id, a.name, m.message_text, m.message_type, 
		       COALESCE(m.reply_to, ''), m.sent_at
		FROM messages m
		JOIN agents a ON m.from_agent_id = a.id
		WHERE m.room_id = ?
		ORDER BY m.sent_at DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, roomID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get room history: %w", err)
	}
	defer rows.Close()

	var messages []StoredMessage
	for rows.Next() {
		var msg StoredMessage
		msg.RoomID = roomID
		if err := rows.Scan(&msg.ID, &msg.FromAgentID, &msg.FromName,
			&msg.Text, &msg.MessageType, &msg.ReplyTo, &msg.SentAt); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetDMHistory returns direct message history between two agents
func (db *DB) GetDMHistory(agentID1, agentID2 string, limit int) ([]StoredMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT m.id, m.from_agent_id, a.name, m.to_agent_id, m.message_text, 
		       m.message_type, COALESCE(m.reply_to, ''), m.sent_at
		FROM messages m
		JOIN agents a ON m.from_agent_id = a.id
		WHERE (m.from_agent_id = ? AND m.to_agent_id = ?)
		   OR (m.from_agent_id = ? AND m.to_agent_id = ?)
		ORDER BY m.sent_at DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, agentID1, agentID2, agentID2, agentID1, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get DM history: %w", err)
	}
	defer rows.Close()

	var messages []StoredMessage
	for rows.Next() {
		var msg StoredMessage
		if err := rows.Scan(&msg.ID, &msg.FromAgentID, &msg.FromName,
			&msg.ToAgentID, &msg.Text, &msg.MessageType, &msg.ReplyTo, &msg.SentAt); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMessageThread returns all replies to a specific message
func (db *DB) GetMessageThread(messageID string) ([]StoredMessage, error) {
	query := `
		SELECT m.id, m.from_agent_id, a.name, COALESCE(m.to_agent_id, ''),
		       COALESCE(m.room_id, ''), m.message_text, m.message_type,
		       COALESCE(m.reply_to, ''), m.sent_at
		FROM messages m
		JOIN agents a ON m.from_agent_id = a.id
		WHERE m.reply_to = ?
		ORDER BY m.sent_at ASC
	`

	rows, err := db.conn.Query(query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}
	defer rows.Close()

	var messages []StoredMessage
	for rows.Next() {
		var msg StoredMessage
		if err := rows.Scan(&msg.ID, &msg.FromAgentID, &msg.FromName,
			&msg.ToAgentID, &msg.RoomID, &msg.Text, &msg.MessageType,
			&msg.ReplyTo, &msg.SentAt); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
