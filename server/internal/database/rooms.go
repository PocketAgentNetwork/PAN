package database

import (
	"fmt"
	"time"

	"pan-server/internal/types"
)

// CreateRoom creates a new room
func (db *DB) CreateRoom(room *types.Room) error {
	query := `
		INSERT INTO rooms (id, name, description, creator_id, is_private, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	_, err := db.conn.Exec(query,
		room.ID, room.Name, room.Description,
		room.CreatorID, room.IsPrivate, time.Now(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to create room: %w", err)
	}

	// Auto-join creator to the room
	return db.JoinRoom(room.CreatorID, room.ID)
}

// GetRoom retrieves a room by ID
func (db *DB) GetRoom(roomID string) (*types.Room, error) {
	query := `
		SELECT id, name, description, creator_id, is_private, created_at
		FROM rooms WHERE id = ?
	`
	
	row := db.conn.QueryRow(query, roomID)
	
	var room types.Room
	err := row.Scan(
		&room.ID, &room.Name, &room.Description,
		&room.CreatorID, &room.IsPrivate, &room.CreatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}

	// Load members
	members, err := db.GetRoomMembers(roomID)
	if err != nil {
		return nil, err
	}
	
	room.Members = make(map[string]bool)
	for _, member := range members {
		room.Members[member.ID] = true
	}

	return &room, nil
}

// JoinRoom adds an agent to a room
func (db *DB) JoinRoom(agentID, roomID string) error {
	query := `
		INSERT OR IGNORE INTO room_members (room_id, agent_id, joined_at)
		VALUES (?, ?, ?)
	`
	
	_, err := db.conn.Exec(query, roomID, agentID, time.Now())
	return err
}

// LeaveRoom removes an agent from a room
func (db *DB) LeaveRoom(agentID, roomID string) error {
	query := `DELETE FROM room_members WHERE room_id = ? AND agent_id = ?`
	_, err := db.conn.Exec(query, roomID, agentID)
	return err
}

// GetRoomMembers returns all members of a room
func (db *DB) GetRoomMembers(roomID string) ([]types.AgentInfo, error) {
	query := `
		SELECT a.id, a.name 
		FROM agents a
		JOIN room_members rm ON a.id = rm.agent_id
		WHERE rm.room_id = ?
		ORDER BY rm.joined_at
	`
	
	rows, err := db.conn.Query(query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []types.AgentInfo
	for rows.Next() {
		var member types.AgentInfo
		if err := rows.Scan(&member.ID, &member.Name); err != nil {
			continue
		}
		members = append(members, member)
	}

	return members, nil
}

// GetAgentRooms returns all rooms an agent has joined
func (db *DB) GetAgentRooms(agentID string) ([]string, error) {
	query := `
		SELECT room_id FROM room_members 
		WHERE agent_id = ?
	`
	
	rows, err := db.conn.Query(query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []string
	for rows.Next() {
		var roomID string
		if err := rows.Scan(&roomID); err != nil {
			continue
		}
		rooms = append(rooms, roomID)
	}

	return rooms, nil
}

// GetAllRooms returns all public rooms
func (db *DB) GetAllRooms() ([]string, error) {
	query := `SELECT name FROM rooms WHERE is_private = FALSE ORDER BY name`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []string
	for rows.Next() {
		var roomName string
		if err := rows.Scan(&roomName); err != nil {
			continue
		}
		rooms = append(rooms, roomName)
	}

	return rooms, nil
}

// IsAgentInRoom checks if an agent is a member of a room
func (db *DB) IsAgentInRoom(agentID, roomID string) (bool, error) {
	query := `SELECT 1 FROM room_members WHERE room_id = ? AND agent_id = ?`
	
	var exists int
	err := db.conn.QueryRow(query, roomID, agentID).Scan(&exists)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, err
	}
	
	return true, nil
}