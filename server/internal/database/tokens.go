package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// AgentToken represents an agent's API token
type AgentToken struct {
	Token     string    `json:"token"`
	AgentID   string    `json:"agent_id"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	IsActive  bool      `json:"is_active"`
}

// createTokensTable creates the tokens table
func (db *DB) createTokensTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS agent_tokens (
			token TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_used DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE,
			FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
		)
	`
	_, err := db.conn.Exec(query)
	return err
}

// GenerateToken creates a new unique token for an agent
func (db *DB) GenerateToken(agentID string) (string, error) {
	// Generate secure random token with pan_ prefix
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := "pan_tok_" + hex.EncodeToString(bytes)

	query := `
		INSERT INTO agent_tokens (token, agent_id, created_at, last_used, is_active)
		VALUES (?, ?, ?, ?, TRUE)
		ON CONFLICT(agent_id) DO UPDATE SET token = excluded.token, created_at = excluded.created_at
	`

	_, err := db.conn.Exec(query, token, agentID, time.Now(), time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to save token: %w", err)
	}

	return token, nil
}

// ValidateToken checks if a token is valid and returns the agent ID
func (db *DB) ValidateToken(token string) (string, error) {
	query := `
		SELECT agent_id FROM agent_tokens
		WHERE token = ? AND is_active = TRUE
	`

	var agentID string
	err := db.conn.QueryRow(query, token).Scan(&agentID)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	// Update last used
	db.conn.Exec(`UPDATE agent_tokens SET last_used = ? WHERE token = ?`, time.Now(), token)

	return agentID, nil
}

// RevokeToken deactivates an agent's token
func (db *DB) RevokeToken(agentID string) error {
	query := `UPDATE agent_tokens SET is_active = FALSE WHERE agent_id = ?`
	_, err := db.conn.Exec(query, agentID)
	return err
}

// GetTokenByAgentID returns the token for an agent
func (db *DB) GetTokenByAgentID(agentID string) (*AgentToken, error) {
	query := `
		SELECT token, agent_id, created_at, last_used, is_active
		FROM agent_tokens WHERE agent_id = ? AND is_active = TRUE
	`

	var t AgentToken
	err := db.conn.QueryRow(query, agentID).Scan(
		&t.Token, &t.AgentID, &t.CreatedAt, &t.LastUsed, &t.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("token not found")
	}

	return &t, nil
}
