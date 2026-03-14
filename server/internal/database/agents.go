package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"pan-server/internal/types"
)

// CreateAgent creates a new agent in the database
func (db *DB) CreateAgent(agent *types.Agent) error {
	interests, _ := json.Marshal(agent.Interests)
	capabilities, _ := json.Marshal(agent.Capabilities)

	query := `
		INSERT INTO agents (id, name, email, bio, interests, capabilities, avatar, status, created_at, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := db.conn.Exec(query,
		agent.ID, agent.Name, agent.Email, agent.Bio,
		string(interests), string(capabilities),
		agent.Avatar, agent.Status,
		time.Now(), time.Now(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Auto-join #agent-square
	if err := db.JoinRoom(agent.ID, "agent-square"); err != nil {
		// Log but don't fail - agent creation is more important
		fmt.Printf("Warning: Failed to auto-join agent-square for %s: %v\n", agent.ID, err)
	}

	return nil
}

// GetAgent retrieves an agent by ID
func (db *DB) GetAgent(agentID string) (*types.Agent, error) {
	query := `
		SELECT id, name, email, bio, interests, capabilities, avatar, status, created_at, last_seen, total_messages
		FROM agents WHERE id = ?
	`
	
	row := db.conn.QueryRow(query, agentID)
	
	var agent types.Agent
	var interests, capabilities string
	var createdAt, lastSeen time.Time
	
	err := row.Scan(
		&agent.ID, &agent.Name, &agent.Email, &agent.Bio,
		&interests, &capabilities, &agent.Avatar, &agent.Status,
		&createdAt, &lastSeen, &agent.LastSeen,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Parse JSON fields
	json.Unmarshal([]byte(interests), &agent.Interests)
	json.Unmarshal([]byte(capabilities), &agent.Capabilities)
	
	agent.ConnectedAt = createdAt
	agent.LastSeen = lastSeen

	return &agent, nil
}

// UpdateAgent updates an agent's profile
func (db *DB) UpdateAgent(agent *types.Agent) error {
	interests, _ := json.Marshal(agent.Interests)
	capabilities, _ := json.Marshal(agent.Capabilities)

	query := `
		UPDATE agents 
		SET name = ?, email = ?, bio = ?, interests = ?, capabilities = ?, avatar = ?, status = ?, last_seen = ?
		WHERE id = ?
	`
	
	_, err := db.conn.Exec(query,
		agent.Name, agent.Email, agent.Bio,
		string(interests), string(capabilities),
		agent.Avatar, agent.Status, time.Now(),
		agent.ID,
	)
	
	return err
}

// UpdateLastSeen updates the agent's last seen timestamp
func (db *DB) UpdateLastSeen(agentID string) error {
	query := `UPDATE agents SET last_seen = ? WHERE id = ?`
	_, err := db.conn.Exec(query, time.Now(), agentID)
	return err
}

// GetOnlineAgents returns list of agents that were seen recently
func (db *DB) GetOnlineAgents(withinMinutes int) ([]types.AgentInfo, error) {
	query := `
		SELECT id, name FROM agents 
		WHERE last_seen > datetime('now', '-' || ? || ' minutes')
		ORDER BY last_seen DESC
	`
	
	rows, err := db.conn.Query(query, withinMinutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []types.AgentInfo
	for rows.Next() {
		var agent types.AgentInfo
		if err := rows.Scan(&agent.ID, &agent.Name); err != nil {
			continue
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// IncrementMessageCount increments an agent's message count
func (db *DB) IncrementMessageCount(agentID string) error {
	query := `UPDATE agents SET total_messages = total_messages + 1, last_seen = ? WHERE id = ?`
	_, err := db.conn.Exec(query, time.Now(), agentID)
	return err
}