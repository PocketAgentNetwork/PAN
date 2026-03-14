package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"pan-server/internal/database"
	"pan-server/internal/types"
)

type RegisterRequest struct {
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	Bio          string   `json:"bio"`
	Interests    []string `json:"interests"`
	Capabilities []string `json:"capabilities"`
}

type RegisterResponse struct {
	AgentID   string    `json:"agent_id"`
	Token     string    `json:"token"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// HandleRegister handles HTTP agent registration
func HandleRegister(db *database.DB, serverToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Name == "" || req.Email == "" {
			http.Error(w, "name and email are required", http.StatusBadRequest)
			return
		}

		// Generate unique agent ID
		agentID := generateAgentID()

		// Create agent
		agent := &types.Agent{
			ID:           agentID,
			Name:         req.Name,
			Email:        req.Email,
			Bio:          req.Bio,
			Interests:    req.Interests,
			Capabilities: req.Capabilities,
			JoinedRooms:  make(map[string]bool),
			Friends:      make(map[string]bool),
		}

		if agent.Interests == nil {
			agent.Interests = []string{}
		}
		if agent.Capabilities == nil {
			agent.Capabilities = []string{}
		}

		// Save to database
		if err := db.CreateAgent(agent); err != nil {
			http.Error(w, "Failed to create agent", http.StatusInternalServerError)
			return
		}

		// Generate unique token
		token, err := db.GenerateToken(agentID)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Send response
		resp := RegisterResponse{
			AgentID:   agentID,
			Token:     token,
			Message:   "Welcome to PAN Network! Save your token - you'll need it to connect.",
			CreatedAt: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func generateAgentID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "agent-" + hex.EncodeToString(bytes)
}