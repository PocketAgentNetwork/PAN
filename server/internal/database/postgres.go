package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/fatih/color"
	_ "github.com/lib/pq"
	"pan-server/internal/types"
)

// PostgresStore is the Postgres backend. Implements Store.
type PostgresStore struct {
	conn *sql.DB
}

// InitPostgres connects to Postgres and runs migrations
func InitPostgres(dsn string) (*PostgresStore, error) {
	color.Yellow("📊 Connecting to Postgres...")

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	pg := &PostgresStore{conn: conn}
	if err := pg.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	color.Green("✅ Postgres ready")
	return pg, nil
}

func (pg *PostgresStore) Close() error { return pg.conn.Close() }

func (pg *PostgresStore) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			bio TEXT DEFAULT '',
			interests TEXT DEFAULT '[]',
			capabilities TEXT DEFAULT '[]',
			avatar TEXT DEFAULT '',
			status TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			last_seen TIMESTAMPTZ DEFAULT NOW(),
			total_messages INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS agent_tokens (
			token TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL UNIQUE REFERENCES agents(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			last_used TIMESTAMPTZ DEFAULT NOW(),
			is_active BOOLEAN DEFAULT TRUE
		)`,
		`CREATE TABLE IF NOT EXISTS rooms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT DEFAULT '',
			creator_id TEXT NOT NULL,
			is_private BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS room_members (
			room_id TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			joined_at TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (room_id, agent_id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			from_agent_id TEXT NOT NULL REFERENCES agents(id),
			to_agent_id TEXT,
			room_id TEXT,
			message_text TEXT NOT NULL,
			message_type TEXT NOT NULL,
			reply_to TEXT,
			delivered BOOLEAN DEFAULT FALSE,
			sent_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS friendships (
			id SERIAL PRIMARY KEY,
			requester_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			requested_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			status TEXT NOT NULL DEFAULT 'pending',
			requested_at TIMESTAMPTZ DEFAULT NOW(),
			responded_at TIMESTAMPTZ,
			UNIQUE(requester_id, requested_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_room     ON messages(room_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_from     ON messages(from_agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_to       ON messages(to_agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sent     ON messages(sent_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_friends_requester ON friendships(requester_id)`,
		`CREATE INDEX IF NOT EXISTS idx_friends_requested ON friendships(requested_id)`,
	}

	for _, q := range queries {
		if _, err := pg.conn.Exec(q); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	pg.seedDefaultRooms()
	return nil
}

func (pg *PostgresStore) seedDefaultRooms() {
	defaults := []struct{ id, name, desc string }{
		{"agent-square", "#agent-square", "Main hub — all agents gather here"},
		{"crypto", "#crypto", "Cryptocurrency and DeFi"},
		{"research", "#research", "Research and development"},
		{"gaming", "#gaming", "Game agents and strategy"},
		{"jobs", "#jobs", "Job postings and marketplace"},
	}
	for _, r := range defaults {
		_, err := pg.conn.Exec(
			`INSERT INTO rooms (id, name, description, creator_id) VALUES ($1,$2,$3,'system') ON CONFLICT DO NOTHING`,
			r.id, r.name, r.desc,
		)
		if err != nil {
			log.Printf("seed room %s: %v", r.name, err)
		}
	}
}

// ── Agents ────────────────────────────────────────────────────────────────────

func (pg *PostgresStore) CreateAgent(agent *types.Agent) error {
	interests, _ := jsonMarshal(agent.Interests)
	capabilities, _ := jsonMarshal(agent.Capabilities)
	_, err := pg.conn.Exec(`
		INSERT INTO agents (id,name,email,bio,interests,capabilities,avatar,status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		agent.ID, agent.Name, agent.Email, agent.Bio,
		interests, capabilities, agent.Avatar, agent.Status,
	)
	if err != nil {
		return err
	}
	return pg.JoinRoom(agent.ID, "agent-square")
}

func (pg *PostgresStore) GetAgent(agentID string) (*types.Agent, error) {
	row := pg.conn.QueryRow(`
		SELECT id,name,email,bio,interests,capabilities,avatar,status,last_seen
		FROM agents WHERE id=$1`, agentID)
	return scanAgent(row)
}

func (pg *PostgresStore) UpdateAgent(agent *types.Agent) error {
	interests, _ := jsonMarshal(agent.Interests)
	capabilities, _ := jsonMarshal(agent.Capabilities)
	_, err := pg.conn.Exec(`
		UPDATE agents SET name=$1,email=$2,bio=$3,interests=$4,capabilities=$5,avatar=$6,status=$7,last_seen=NOW()
		WHERE id=$8`,
		agent.Name, agent.Email, agent.Bio, interests, capabilities,
		agent.Avatar, agent.Status, agent.ID,
	)
	return err
}

func (pg *PostgresStore) UpdateLastSeen(agentID string) error {
	_, err := pg.conn.Exec(`UPDATE agents SET last_seen=NOW() WHERE id=$1`, agentID)
	return err
}

func (pg *PostgresStore) IncrementMessageCount(agentID string) error {
	_, err := pg.conn.Exec(`UPDATE agents SET total_messages=total_messages+1,last_seen=NOW() WHERE id=$1`, agentID)
	return err
}

func (pg *PostgresStore) GetOnlineAgents(withinMinutes int) ([]types.AgentInfo, error) {
	rows, err := pg.conn.Query(`
		SELECT id,name FROM agents WHERE last_seen > NOW() - ($1 || ' minutes')::interval ORDER BY last_seen DESC`,
		withinMinutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgentInfos(rows)
}

// ── Tokens ────────────────────────────────────────────────────────────────────

func (pg *PostgresStore) GenerateToken(agentID string) (string, error) {
	token, err := generateSecureToken()
	if err != nil {
		return "", err
	}
	_, err = pg.conn.Exec(`
		INSERT INTO agent_tokens (token,agent_id) VALUES ($1,$2)
		ON CONFLICT (agent_id) DO UPDATE SET token=$1, created_at=NOW()`,
		token, agentID,
	)
	return token, err
}

func (pg *PostgresStore) ValidateToken(token string) (string, error) {
	var agentID string
	err := pg.conn.QueryRow(`SELECT agent_id FROM agent_tokens WHERE token=$1 AND is_active=TRUE`, token).Scan(&agentID)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}
	pg.conn.Exec(`UPDATE agent_tokens SET last_used=NOW() WHERE token=$1`, token)
	return agentID, nil
}

func (pg *PostgresStore) RevokeToken(agentID string) error {
	_, err := pg.conn.Exec(`UPDATE agent_tokens SET is_active=FALSE WHERE agent_id=$1`, agentID)
	return err
}

func (pg *PostgresStore) GetTokenByAgentID(agentID string) (*AgentToken, error) {
	var t AgentToken
	err := pg.conn.QueryRow(`
		SELECT token,agent_id,created_at,last_used,is_active FROM agent_tokens WHERE agent_id=$1 AND is_active=TRUE`,
		agentID).Scan(&t.Token, &t.AgentID, &t.CreatedAt, &t.LastUsed, &t.IsActive)
	if err != nil {
		return nil, fmt.Errorf("token not found")
	}
	return &t, nil
}

// ── Rooms ─────────────────────────────────────────────────────────────────────

func (pg *PostgresStore) CreateRoom(room *types.Room) error {
	_, err := pg.conn.Exec(`
		INSERT INTO rooms (id,name,description,creator_id,is_private) VALUES ($1,$2,$3,$4,$5)`,
		room.ID, room.Name, room.Description, room.CreatorID, room.IsPrivate,
	)
	if err != nil {
		return err
	}
	return pg.JoinRoom(room.CreatorID, room.ID)
}

func (pg *PostgresStore) GetRoom(roomID string) (*types.Room, error) {
	var room types.Room
	err := pg.conn.QueryRow(`
		SELECT id,name,description,creator_id,is_private,created_at FROM rooms WHERE id=$1`, roomID).
		Scan(&room.ID, &room.Name, &room.Description, &room.CreatorID, &room.IsPrivate, &room.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("room not found")
	}
	members, _ := pg.GetRoomMembers(roomID)
	room.Members = make(map[string]bool)
	for _, m := range members {
		room.Members[m.ID] = true
	}
	return &room, nil
}

func (pg *PostgresStore) GetAllRooms() ([]string, error) {
	rows, err := pg.conn.Query(`SELECT name FROM rooms WHERE is_private=FALSE ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rooms []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		rooms = append(rooms, name)
	}
	return rooms, nil
}

func (pg *PostgresStore) GetRoomMembers(roomID string) ([]types.AgentInfo, error) {
	rows, err := pg.conn.Query(`
		SELECT a.id,a.name FROM agents a
		JOIN room_members rm ON a.id=rm.agent_id WHERE rm.room_id=$1 ORDER BY rm.joined_at`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAgentInfos(rows)
}

func (pg *PostgresStore) GetAgentRooms(agentID string) ([]string, error) {
	rows, err := pg.conn.Query(`SELECT room_id FROM room_members WHERE agent_id=$1`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rooms []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		rooms = append(rooms, id)
	}
	return rooms, nil
}

func (pg *PostgresStore) JoinRoom(agentID, roomID string) error {
	_, err := pg.conn.Exec(`INSERT INTO room_members (room_id,agent_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, roomID, agentID)
	return err
}

func (pg *PostgresStore) LeaveRoom(agentID, roomID string) error {
	_, err := pg.conn.Exec(`DELETE FROM room_members WHERE room_id=$1 AND agent_id=$2`, roomID, agentID)
	return err
}

func (pg *PostgresStore) IsAgentInRoom(agentID, roomID string) (bool, error) {
	var exists bool
	err := pg.conn.QueryRow(`SELECT EXISTS(SELECT 1 FROM room_members WHERE room_id=$1 AND agent_id=$2)`, roomID, agentID).Scan(&exists)
	return exists, err
}

// ── Messages ──────────────────────────────────────────────────────────────────

func (pg *PostgresStore) SaveMessage(fromID, toID, roomID, text, msgType, replyTo string) (string, error) {
	id, _ := generateSecureToken()
	id = "msg_" + id[:16]
	var toVal, roomVal, replyVal interface{}
	if toID != "" { toVal = toID }
	if roomID != "" { roomVal = roomID }
	if replyTo != "" { replyVal = replyTo }
	_, err := pg.conn.Exec(`
		INSERT INTO messages (id,from_agent_id,to_agent_id,room_id,message_text,message_type,reply_to)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, fromID, toVal, roomVal, text, msgType, replyVal,
	)
	return id, err
}

func (pg *PostgresStore) GetRoomHistory(roomID string, limit int) ([]StoredMessage, error) {
	if limit <= 0 || limit > 100 { limit = 50 }
	rows, err := pg.conn.Query(`
		SELECT m.id,m.from_agent_id,a.name,m.message_text,m.message_type,COALESCE(m.reply_to,''),m.sent_at
		FROM messages m JOIN agents a ON m.from_agent_id=a.id
		WHERE m.room_id=$1 ORDER BY m.sent_at DESC LIMIT $2`, roomID, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanRoomMessages(rows, roomID)
}

func (pg *PostgresStore) GetDMHistory(agentID1, agentID2 string, limit int) ([]StoredMessage, error) {
	if limit <= 0 || limit > 100 { limit = 50 }
	rows, err := pg.conn.Query(`
		SELECT m.id,m.from_agent_id,a.name,m.to_agent_id,m.message_text,m.message_type,COALESCE(m.reply_to,''),m.sent_at
		FROM messages m JOIN agents a ON m.from_agent_id=a.id
		WHERE (m.from_agent_id=$1 AND m.to_agent_id=$2) OR (m.from_agent_id=$2 AND m.to_agent_id=$1)
		ORDER BY m.sent_at DESC LIMIT $3`, agentID1, agentID2, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanDMMessages(rows)
}

func (pg *PostgresStore) GetMessageThread(messageID string) ([]StoredMessage, error) {
	rows, err := pg.conn.Query(`
		SELECT m.id,m.from_agent_id,a.name,COALESCE(m.to_agent_id,''),COALESCE(m.room_id,''),
		       m.message_text,m.message_type,COALESCE(m.reply_to,''),m.sent_at
		FROM messages m JOIN agents a ON m.from_agent_id=a.id
		WHERE m.reply_to=$1 ORDER BY m.sent_at ASC`, messageID)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanThreadMessages(rows)
}

// ── Offline messages ──────────────────────────────────────────────────────────

func (pg *PostgresStore) SaveOfflineMessage(msgID, fromID, toID, text string) error {
	_, err := pg.conn.Exec(`
		INSERT INTO messages (id,from_agent_id,to_agent_id,message_text,message_type)
		VALUES ($1,$2,$3,$4,'dm') ON CONFLICT DO NOTHING`,
		msgID, fromID, toID, text,
	)
	return err
}

func (pg *PostgresStore) GetOfflineMessages(agentID string) ([]OfflineMessage, error) {
	rows, err := pg.conn.Query(`
		SELECT m.id,m.from_agent_id,a.name,m.to_agent_id,m.message_text,m.sent_at
		FROM messages m JOIN agents a ON m.from_agent_id=a.id
		WHERE m.to_agent_id=$1 AND m.message_type='dm' AND m.delivered=FALSE
		ORDER BY m.sent_at ASC`, agentID)
	if err != nil { return nil, err }
	defer rows.Close()
	var msgs []OfflineMessage
	for rows.Next() {
		var m OfflineMessage
		rows.Scan(&m.ID, &m.FromAgentID, &m.FromName, &m.ToAgentID, &m.Text, &m.SentAt)
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (pg *PostgresStore) MarkMessagesDelivered(agentID string) error {
	_, err := pg.conn.Exec(`UPDATE messages SET delivered=TRUE WHERE to_agent_id=$1 AND delivered=FALSE`, agentID)
	return err
}

// ── Friends ───────────────────────────────────────────────────────────────────

func (pg *PostgresStore) SendFriendRequest(requesterID, requestedID string) error {
	var exists int
	pg.conn.QueryRow(`
		SELECT COUNT(*) FROM friendships
		WHERE (requester_id=$1 AND requested_id=$2) OR (requester_id=$2 AND requested_id=$1)`,
		requesterID, requestedID).Scan(&exists)
	if exists > 0 {
		return fmt.Errorf("friendship already exists or pending")
	}
	_, err := pg.conn.Exec(`INSERT INTO friendships (requester_id,requested_id) VALUES ($1,$2)`, requesterID, requestedID)
	return err
}

func (pg *PostgresStore) RespondToFriendRequest(requesterID, requestedID, action string) error {
	status := "accepted"
	if action == "decline" { status = "declined" }
	_, err := pg.conn.Exec(`
		UPDATE friendships SET status=$1,responded_at=NOW()
		WHERE requester_id=$2 AND requested_id=$3 AND status='pending'`,
		status, requesterID, requestedID)
	return err
}

func (pg *PostgresStore) GetFriends(agentID string) ([]StoredMessage, error) {
	rows, err := pg.conn.Query(`
		SELECT a.id,a.name FROM agents a
		JOIN friendships f ON (f.requester_id=$1 AND f.requested_id=a.id) OR (f.requested_id=$1 AND f.requester_id=a.id)
		WHERE f.status='accepted'`, agentID)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanStoredMsgIDs(rows)
}

func (pg *PostgresStore) GetPendingRequests(agentID string) ([]StoredMessage, error) {
	rows, err := pg.conn.Query(`
		SELECT a.id,a.name FROM agents a
		JOIN friendships f ON f.requester_id=a.id
		WHERE f.requested_id=$1 AND f.status='pending'`, agentID)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanStoredMsgIDs(rows)
}

func (pg *PostgresStore) AreFriends(agentID1, agentID2 string) (bool, error) {
	var exists bool
	err := pg.conn.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM friendships
		WHERE ((requester_id=$1 AND requested_id=$2) OR (requester_id=$2 AND requested_id=$1))
		AND status='accepted')`, agentID1, agentID2).Scan(&exists)
	return exists, err
}
