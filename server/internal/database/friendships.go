package database

import (
	"fmt"
	"time"
)

type Friendship struct {
	ID          int       `json:"id"`
	RequesterID string    `json:"requester_id"`
	RequestedID string    `json:"requested_id"`
	Status      string    `json:"status"` // pending, accepted, blocked
	RequestedAt time.Time `json:"requested_at"`
	RespondedAt time.Time `json:"responded_at"`
}

// SendFriendRequest creates a pending friendship
func (db *DB) SendFriendRequest(requesterID, requestedID string) error {
	// Check if friendship already exists
	var exists int
	db.conn.QueryRow(`
		SELECT COUNT(*) FROM friendships 
		WHERE (requester_id = ? AND requested_id = ?) 
		   OR (requester_id = ? AND requested_id = ?)
	`, requesterID, requestedID, requestedID, requesterID).Scan(&exists)

	if exists > 0 {
		return fmt.Errorf("friendship already exists or pending")
	}

	_, err := db.conn.Exec(`
		INSERT INTO friendships (requester_id, requested_id, status, requested_at)
		VALUES (?, ?, 'pending', ?)
	`, requesterID, requestedID, time.Now())

	return err
}

// RespondToFriendRequest accepts or declines a friend request
func (db *DB) RespondToFriendRequest(requesterID, requestedID, action string) error {
	status := "accepted"
	if action == "decline" {
		status = "declined"
	}

	_, err := db.conn.Exec(`
		UPDATE friendships SET status = ?, responded_at = ?
		WHERE requester_id = ? AND requested_id = ? AND status = 'pending'
	`, status, time.Now(), requesterID, requestedID)

	return err
}

// GetFriends returns all accepted friends for an agent
func (db *DB) GetFriends(agentID string) ([]StoredMessage, error) {
	query := `
		SELECT a.id, a.name FROM agents a
		JOIN friendships f ON (
			(f.requester_id = ? AND f.requested_id = a.id) OR
			(f.requested_id = ? AND f.requester_id = a.id)
		)
		WHERE f.status = 'accepted'
	`

	rows, err := db.conn.Query(query, agentID, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []StoredMessage
	for rows.Next() {
		var f StoredMessage
		if err := rows.Scan(&f.ID, &f.FromName); err != nil {
			continue
		}
		friends = append(friends, f)
	}

	return friends, nil
}

// GetPendingRequests returns pending friend requests for an agent
func (db *DB) GetPendingRequests(agentID string) ([]StoredMessage, error) {
	query := `
		SELECT a.id, a.name FROM agents a
		JOIN friendships f ON f.requester_id = a.id
		WHERE f.requested_id = ? AND f.status = 'pending'
	`

	rows, err := db.conn.Query(query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requesters []StoredMessage
	for rows.Next() {
		var a StoredMessage
		if err := rows.Scan(&a.ID, &a.FromName); err != nil {
			continue
		}
		requesters = append(requesters, a)
	}

	return requesters, nil
}

// AreFriends checks if two agents are friends
func (db *DB) AreFriends(agentID1, agentID2 string) (bool, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM friendships
		WHERE ((requester_id = ? AND requested_id = ?) OR (requester_id = ? AND requested_id = ?))
		AND status = 'accepted'
	`, agentID1, agentID2, agentID2, agentID1).Scan(&count)

	return count > 0, err
}
