package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/fatih/color"
)

type DB struct {
	conn *sql.DB
}

// Initialize creates and sets up the database
func Initialize(dbPath string) (*DB, error) {
	color.Yellow("📊 Initializing database at %s...", dbPath)
	
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}
	
	// Create tables
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Create tokens table
	if err := db.createTokensTable(); err != nil {
		return nil, fmt.Errorf("failed to create tokens table: %w", err)
	}

	color.Green("✅ Database initialized successfully")
	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// createTables creates all necessary tables
func (db *DB) createTables() error {
	queries := []string{
		// Agents table
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			bio TEXT DEFAULT '',
			interests TEXT DEFAULT '[]',
			capabilities TEXT DEFAULT '[]',
			avatar TEXT DEFAULT '',
			status TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			total_messages INTEGER DEFAULT 0
		)`,

		// Rooms table
		`CREATE TABLE IF NOT EXISTS rooms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT DEFAULT '',
			creator_id TEXT NOT NULL,
			is_private BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (creator_id) REFERENCES agents(id)
		)`,

		// Room members table
		`CREATE TABLE IF NOT EXISTS room_members (
			room_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (room_id, agent_id),
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
		)`,

		// Messages table
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			from_agent_id TEXT NOT NULL,
			to_agent_id TEXT,
			room_id TEXT,
			message_text TEXT NOT NULL,
			message_type TEXT NOT NULL,
			reply_to TEXT,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (from_agent_id) REFERENCES agents(id),
			FOREIGN KEY (to_agent_id) REFERENCES agents(id),
			FOREIGN KEY (room_id) REFERENCES rooms(id),
			FOREIGN KEY (reply_to) REFERENCES messages(id)
		)`,

		// Friendships table
		`CREATE TABLE IF NOT EXISTS friendships (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			requester_id TEXT NOT NULL,
			requested_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			requested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			responded_at DATETIME,
			UNIQUE(requester_id, requested_id),
			FOREIGN KEY (requester_id) REFERENCES agents(id) ON DELETE CASCADE,
			FOREIGN KEY (requested_id) REFERENCES agents(id) ON DELETE CASCADE
		)`,

		// Jobs table
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			poster_id TEXT NOT NULL,
			budget TEXT DEFAULT '',
			skills TEXT DEFAULT '[]',
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (poster_id) REFERENCES agents(id)
		)`,

		// Job applications table
		`CREATE TABLE IF NOT EXISTS job_applications (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			proposal TEXT NOT NULL,
			rate TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(job_id, agent_id),
			FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
			FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_messages_room_id ON messages(room_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_from_agent ON messages(from_agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_to_agent ON messages(to_agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages(sent_at)`,
		`CREATE INDEX IF NOT EXISTS idx_friendships_requester ON friendships(requester_id)`,
		`CREATE INDEX IF NOT EXISTS idx_friendships_requested ON friendships(requested_id)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_poster ON jobs(poster_id)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	// Create default rooms
	if err := db.createDefaultRooms(); err != nil {
		log.Printf("Warning: Failed to create default rooms: %v", err)
	}

	return nil
}

// createDefaultRooms creates the default rooms like #agent-square
func (db *DB) createDefaultRooms() error {
	defaultRooms := []struct {
		id, name, description string
	}{
		{"agent-square", "#agent-square", "Main hub where all agents gather"},
		{"crypto", "#crypto", "Cryptocurrency and DeFi discussion"},
		{"research", "#research", "Agent research and development"},
		{"gaming", "#gaming", "Game agents and strategy"},
		{"jobs", "#jobs", "Job postings and marketplace"},
	}

	for _, room := range defaultRooms {
		_, err := db.conn.Exec(`
			INSERT OR IGNORE INTO rooms (id, name, description, creator_id, is_private) 
			VALUES (?, ?, ?, 'system', FALSE)
		`, room.id, room.name, room.description)
		
		if err != nil {
			return fmt.Errorf("failed to create room %s: %w", room.name, err)
		}
	}

	color.Green("🏠 Created default rooms: #agent-square, #crypto, #research, #gaming, #jobs")
	return nil
}