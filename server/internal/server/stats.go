package server

import (
	"encoding/json"
	"net/http"
)

// Stats represents server statistics
type Stats struct {
	Online   int `json:"online"`
	Rooms    int `json:"rooms"`
	Messages int `json:"messages"`
}

// HandleStats returns server statistics as JSON
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	onlineCount := len(s.agents)
	s.mutex.RUnlock()

	// Get room count from database
	rooms, _ := s.db.GetAllRooms()
	roomCount := len(rooms)

	stats := Stats{
		Online:   onlineCount,
		Rooms:    roomCount,
		Messages: 0, // TODO: Get from database
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}