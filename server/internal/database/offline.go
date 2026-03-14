package database

import (
	"time"
)

type OfflineMessage struct {
	ID          string    `json:"id"`
	FromAgentID string    `json:"from_agent_id"`
	FromName    string    `json:"from_name"`
	ToAgentID   string    `json:"to_agent_id"`
	Text        string    `json:"text"`
	SentAt      time.Time `json:"sent_at"`
}

// SaveOfflineMessage saves a DM for an offline agent
func (db *DB) SaveOfflineMessage(msgID, fromID, toID, text string) error {
	_, err := db.conn.Exec(`
		INSERT OR IGNORE INTO messages (id, from_agent_id, to_agent_id, message_text, message_type, sent_at)
		VALUES (?, ?, ?, ?, 'dm', ?)
	`, msgID, fromID, toID, text, time.Now())
	return err
}

// GetOfflineMessages returns undelivered messages for an agent
func (db *DB) GetOfflineMessages(agentID string) ([]OfflineMessage, error) {
	query := `
		SELECT m.id, m.from_agent_id, a.name, m.to_agent_id, m.message_text, m.sent_at
		FROM messages m
		JOIN agents a ON m.from_agent_id = a.id
		WHERE m.to_agent_id = ? 
		  AND m.message_type = 'dm'
		  AND m.delivered = FALSE
		ORDER BY m.sent_at ASC
	`

	rows, err := db.conn.Query(query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []OfflineMessage
	for rows.Next() {
		var msg OfflineMessage
		if err := rows.Scan(&msg.ID, &msg.FromAgentID, &msg.FromName,
			&msg.ToAgentID, &msg.Text, &msg.SentAt); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// MarkMessagesDelivered marks messages as delivered
func (db *DB) MarkMessagesDelivered(agentID string) error {
	_, err := db.conn.Exec(`
		UPDATE messages SET delivered = TRUE 
		WHERE to_agent_id = ? AND delivered = FALSE
	`, agentID)
	return err
}