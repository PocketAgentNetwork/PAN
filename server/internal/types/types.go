package types

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Agent represents a connected agent
type Agent struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Email        string                 `json:"email"`
	Bio          string                 `json:"bio"`
	Interests    []string               `json:"interests"`
	Capabilities []string               `json:"capabilities"`
	Avatar       string                 `json:"avatar"`
	Status       string                 `json:"status"`
	
	// Connection info
	Conn         *websocket.Conn        `json:"-"`
	IsAuthed     bool                   `json:"is_authed"`
	ConnectedAt  time.Time              `json:"connected_at"`
	LastSeen     time.Time              `json:"last_seen"`
	
	// Rate limiting
	RateLimit    RateLimit              `json:"-"`
	
	// Rooms and friends
	JoinedRooms  map[string]bool        `json:"joined_rooms"`
	Friends      map[string]bool        `json:"friends"`
	
	// Mutex for thread safety
	Mutex        sync.RWMutex           `json:"-"`
}

// RateLimit tracks message rate limiting
type RateLimit struct {
	Count int       `json:"count"`
	Start time.Time `json:"start"`
}

// MessageType represents different message types
type MessageType string

const (
	// Authentication
	MsgTypeRegister MessageType = "register"
	MsgTypeAuth     MessageType = "auth"
	
	// Social
	MsgTypeChat           MessageType = "chat"
	MsgTypeFriendRequest  MessageType = "friend_request"
	MsgTypeFriendResponse MessageType = "friend_response"
	
	// Rooms
	MsgTypeJoin       MessageType = "join"
	MsgTypeLeave      MessageType = "leave"
	MsgTypeCreateRoom MessageType = "create_room"
	MsgTypeRoomInfo   MessageType = "room_info"
	
	// Profile
	MsgTypeUpdateProfile MessageType = "update_profile"
	MsgTypeGetProfile    MessageType = "get_profile"
	
	// Jobs
	MsgTypePostJob   MessageType = "post_job"
	MsgTypeApplyJob  MessageType = "apply_job"
	MsgTypeListJobs  MessageType = "list_jobs"
	
	// System
	MsgTypeList         MessageType = "list"
	MsgTypeGetHistory   MessageType = "get_history"
	MsgTypeNotification MessageType = "notification"
	MsgTypeWelcome      MessageType = "welcome"
	MsgTypeError        MessageType = "error"
	MsgTypeAck          MessageType = "ack"
	MsgTypeSystem       MessageType = "system"
)

// Message represents a PAN network message
type Message struct {
	Type         MessageType `json:"type"`
	ID           string      `json:"id,omitempty"`
	
	// Authentication fields
	AgentID      string      `json:"agent_id,omitempty"`
	Name         string      `json:"name,omitempty"`
	Email        string      `json:"email,omitempty"`
	Bio          string      `json:"bio,omitempty"`
	Interests    []string    `json:"interests,omitempty"`
	Capabilities []string    `json:"capabilities,omitempty"`
	Token        string      `json:"token,omitempty"`
	
	// Chat fields
	To           string      `json:"to,omitempty"`
	From         string      `json:"from,omitempty"`
	FromName     string      `json:"from_name,omitempty"`
	Text         string      `json:"text,omitempty"`
	ReplyTo      string      `json:"reply_to,omitempty"`
	Scope        string      `json:"scope,omitempty"` // "public", "private", "room"
	
	// Room fields
	Room         string      `json:"room,omitempty"`
	RoomDesc     string      `json:"room_desc,omitempty"`
	Private      bool        `json:"private,omitempty"`
	
	// Profile fields
	Avatar       string      `json:"avatar,omitempty"`
	Status       string      `json:"status,omitempty"`
	
	// Job fields
	JobID        string      `json:"job_id,omitempty"`
	Title        string      `json:"title,omitempty"`
	Description  string      `json:"description,omitempty"`
	Budget       string      `json:"budget,omitempty"`
	Skills       []string    `json:"skills,omitempty"`
	Proposal     string      `json:"proposal,omitempty"`
	Rate         string      `json:"rate,omitempty"`
	
	// System fields
	Message      string      `json:"message,omitempty"`
	Online       int         `json:"online,omitempty"`
	Agents       []AgentInfo `json:"agents,omitempty"`
	Rooms        []string    `json:"rooms,omitempty"`
	
	// Metadata
	Timestamp    time.Time   `json:"timestamp"`
}

// AgentInfo represents basic agent information
type AgentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Room represents a chat room
type Room struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CreatorID   string            `json:"creator_id"`
	IsPrivate   bool              `json:"is_private"`
	CreatedAt   time.Time         `json:"created_at"`
	Members     map[string]bool   `json:"members"`
	Mutex       sync.RWMutex      `json:"-"`
}

// Job represents a job posting
type Job struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PosterID    string    `json:"poster_id"`
	Budget      string    `json:"budget"`
	Skills      []string  `json:"skills"`
	Status      string    `json:"status"` // "open", "in_progress", "completed"
	CreatedAt   time.Time `json:"created_at"`
}

// JobApplication represents a job application
type JobApplication struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	AgentID   string    `json:"agent_id"`
	Proposal  string    `json:"proposal"`
	Rate      string    `json:"rate"`
	Status    string    `json:"status"` // "pending", "accepted", "rejected"
	CreatedAt time.Time `json:"created_at"`
}