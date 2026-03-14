package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"pan-server/internal/types"
)

// jsonMarshal marshals a value to JSON string
func jsonMarshal(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return "pan_tok_" + hex.EncodeToString(b), nil
}

// scanAgent scans a single agent row
func scanAgent(row *sql.Row) (*types.Agent, error) {
	var agent types.Agent
	var interests, capabilities string
	var lastSeen time.Time

	err := row.Scan(
		&agent.ID, &agent.Name, &agent.Email, &agent.Bio,
		&interests, &capabilities, &agent.Avatar, &agent.Status,
		&lastSeen,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, err
	}

	json.Unmarshal([]byte(interests), &agent.Interests)
	json.Unmarshal([]byte(capabilities), &agent.Capabilities)
	agent.LastSeen = lastSeen
	return &agent, nil
}

// scanAgentInfos scans rows of (id, name)
func scanAgentInfos(rows *sql.Rows) ([]types.AgentInfo, error) {
	var agents []types.AgentInfo
	for rows.Next() {
		var a types.AgentInfo
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			continue
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// scanRoomMessages scans room history rows and reverses to chronological order
func scanRoomMessages(rows *sql.Rows, roomID string) ([]StoredMessage, error) {
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		m.RoomID = roomID
		if err := rows.Scan(&m.ID, &m.FromAgentID, &m.FromName, &m.Text, &m.MessageType, &m.ReplyTo, &m.SentAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	reverseMessages(msgs)
	return msgs, nil
}

// scanDMMessages scans DM history rows and reverses to chronological order
func scanDMMessages(rows *sql.Rows) ([]StoredMessage, error) {
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.ID, &m.FromAgentID, &m.FromName, &m.ToAgentID, &m.Text, &m.MessageType, &m.ReplyTo, &m.SentAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	reverseMessages(msgs)
	return msgs, nil
}

// scanThreadMessages scans thread reply rows
func scanThreadMessages(rows *sql.Rows) ([]StoredMessage, error) {
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.ID, &m.FromAgentID, &m.FromName, &m.ToAgentID, &m.RoomID, &m.Text, &m.MessageType, &m.ReplyTo, &m.SentAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// scanStoredMsgIDs scans (id, name) rows into StoredMessage stubs (used for friends list)
func scanStoredMsgIDs(rows *sql.Rows) ([]StoredMessage, error) {
	var results []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.ID, &m.FromName); err != nil {
			continue
		}
		results = append(results, m)
	}
	return results, nil
}

func reverseMessages(msgs []StoredMessage) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}
