package database

import "pan-server/internal/types"

// Store is the database interface. Both SQLite and Postgres implement this.
// Switch backends by setting DATABASE_URL in your environment:
//   - unset / sqlite:pan.db  → SQLite (default, zero ops)
//   - postgres://...         → PostgreSQL (production scale)
type Store interface {
	// Agents
	CreateAgent(agent *types.Agent) error
	GetAgent(agentID string) (*types.Agent, error)
	UpdateAgent(agent *types.Agent) error
	UpdateLastSeen(agentID string) error
	IncrementMessageCount(agentID string) error
	GetOnlineAgents(withinMinutes int) ([]types.AgentInfo, error)

	// Tokens
	GenerateToken(agentID string) (string, error)
	ValidateToken(token string) (string, error)
	RevokeToken(agentID string) error
	GetTokenByAgentID(agentID string) (*AgentToken, error)

	// Rooms
	CreateRoom(room *types.Room) error
	GetRoom(roomID string) (*types.Room, error)
	GetAllRooms() ([]string, error)
	GetRoomMembers(roomID string) ([]types.AgentInfo, error)
	GetAgentRooms(agentID string) ([]string, error)
	JoinRoom(agentID, roomID string) error
	LeaveRoom(agentID, roomID string) error
	IsAgentInRoom(agentID, roomID string) (bool, error)

	// Messages
	SaveMessage(fromID, toID, roomID, text, msgType, replyTo string) (string, error)
	GetRoomHistory(roomID string, limit int) ([]StoredMessage, error)
	GetDMHistory(agentID1, agentID2 string, limit int) ([]StoredMessage, error)
	GetMessageThread(messageID string) ([]StoredMessage, error)

	// Offline messages
	SaveOfflineMessage(msgID, fromID, toID, text string) error
	GetOfflineMessages(agentID string) ([]OfflineMessage, error)
	MarkMessagesDelivered(agentID string) error

	// Friends
	SendFriendRequest(requesterID, requestedID string) error
	RespondToFriendRequest(requesterID, requestedID, action string) error
	GetFriends(agentID string) ([]StoredMessage, error)
	GetPendingRequests(agentID string) ([]StoredMessage, error)
	AreFriends(agentID1, agentID2 string) (bool, error)

	// Lifecycle
	Close() error
}
