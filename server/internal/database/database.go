package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DB is the SQLite backend (fallback / local dev). Implements Store.
type DB struct {
	conn *sql.DB
}

// Open returns the right Store based on environment:
//   - DATABASE_URL=postgres://...  → PostgreSQL (recommended for production)
//   - unset / PAN_DB_PATH          → SQLite (local dev / small deployments)
func Open(dbPath string) (Store, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		color.Cyan("🐘 DATABASE_URL detected — using PostgreSQL")
		return InitPostgres(dsn)
	}
	color.Yellow("⚠️  No DATABASE_URL set — falling back to SQLite (%s)", dbPath)
	color.Yellow("   Set DATABASE_URL=postgres://... for production use")
	return Initialize(dbPath)
}

// Initialize creates and sets up the SQLite database
func Initialize(dbPath string) (*DB, error) {
	color.Yellow("📊 Initializing SQLite at %s...", dbPath)

	// WAL mode: concurrent reads + single writer, much better throughput
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	conn.SetMaxOpenConns(1) // SQLite: single writer

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}
	if err := db.createTokensTable(); err != nil {
		return nil, fmt.Errorf("failed to create tokens table: %w", err)
	}

	color.Green("✅ SQLite ready (WAL mode)")
	return db, nil
}

func (db *DB) Close() error { return db.conn.Close() }

func (db *DB) createTables() error {
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			total_messages INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS rooms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT DEFAULT '',
			creator_id TEXT NOT NULL,
			is_private BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS room_members (
			room_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (room_id, agent_id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			from_agent_id TEXT NOT NULL,
			to_agent_id TEXT,
			room_id TEXT,
			message_text TEXT NOT NULL,
			message_type TEXT NOT NULL,
			reply_to TEXT,
			delivered BOOLEAN DEFAULT FALSE,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS friendships (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			requester_id TEXT NOT NULL,
			requested_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			requested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			responded_at DATETIME,
			UNIQUE(requester_id, requested_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_room     ON messages(room_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_from     ON messages(from_agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_to       ON messages(to_agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sent     ON messages(sent_at)`,
		`CREATE INDEX IF NOT EXISTS idx_friends_requester ON friendships(requester_id)`,
		`CREATE INDEX IF NOT EXISTS idx_friends_requested ON friendships(requested_id)`,
	}

	for _, q := range queries {
		if _, err := db.conn.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if err := db.createDefaultRooms(); err != nil {
		log.Printf("Warning: default rooms: %v", err)
	}
	return nil
}

func (db *DB) createDefaultRooms() error {
	defaults := []struct{ id, name, desc string }{
		{"agent-square", "#agent-square", "Main hub — all agents gather here"},
		{"crypto", "#crypto", "Cryptocurrency and DeFi"},
		{"research", "#research", "Research and development"},
		{"gaming", "#gaming", "Game agents and strategy"},
		{"jobs", "#jobs", "Job postings and marketplace"},
	}
	for _, r := range defaults {
		db.conn.Exec(`INSERT OR IGNORE INTO rooms (id,name,description,creator_id) VALUES (?,?,?,'system')`,
			r.id, r.name, r.desc)
	}
	color.Green("🏠 Default rooms ready")
	return nil
}

// ensure DB implements Store at compile time
var _ Store = (*DB)(nil)
var _ Store = (*PostgresStore)(nil)

// isPostgresDSN checks if a string looks like a postgres connection string
func isPostgresDSN(s string) bool {
	return strings.HasPrefix(s, "postgres://") || strings.HasPrefix(s, "postgresql://")
}
