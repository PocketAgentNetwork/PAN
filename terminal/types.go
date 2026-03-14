package main

import (
	"time"

	"github.com/gorilla/websocket"
)

// PANMessage is the wire format for all server communication
type PANMessage struct {
	Type         string   `json:"type"`
	AgentID      string   `json:"agent_id,omitempty"`
	Name         string   `json:"name,omitempty"`
	Email        string   `json:"email,omitempty"`
	Bio          string   `json:"bio,omitempty"`
	Interests    []string `json:"interests,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Token        string   `json:"token,omitempty"`
	To           string   `json:"to,omitempty"`
	From         string   `json:"from,omitempty"`
	FromName     string   `json:"from_name,omitempty"`
	Text         string   `json:"text,omitempty"`
	Room         string   `json:"room,omitempty"`
	RoomDesc     string   `json:"room_desc,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	ReplyTo      string   `json:"reply_to,omitempty"`
	Message      string   `json:"message,omitempty"`
	Online       int      `json:"online,omitempty"`
	Status       string   `json:"status,omitempty"`
	Avatar       string   `json:"avatar,omitempty"`
	Agents       []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"agents,omitempty"`
	Rooms     []string `json:"rooms,omitempty"`
	Timestamp string   `json:"timestamp,omitempty"`
}

// ChatLine is a single rendered line in the chat viewport
type ChatLine struct {
	kind string // "chat", "system", "notif"
	text string
}

// Tea message types for async events
type wsMsg struct{ msg PANMessage }
type wsErrMsg struct{ err error }
type connectedMsg struct{ conn *websocket.Conn }
type tickMsg time.Time
