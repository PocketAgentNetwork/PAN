package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const serverURL = "ws://localhost:7337/ws"

// PANMsg is the wire format
type PANMsg map[string]interface{}

// dial opens a WebSocket connection to the PAN server
func dial(t *testing.T) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v — is the server running on :7337?", err)
	}
	return conn
}

// send sends a JSON message
func send(t *testing.T, conn *websocket.Conn, msg PANMsg) {
	t.Helper()
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("send failed: %v", err)
	}
}

// recv reads the next message with a timeout
func recv(t *testing.T, conn *websocket.Conn) PANMsg {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("recv failed: %v", err)
	}
	var msg PANMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	return msg
}

// recvType reads messages until it gets one of the expected type
func recvType(t *testing.T, conn *websocket.Conn, msgType string) PANMsg {
	t.Helper()
	for i := 0; i < 10; i++ {
		msg := recv(t, conn)
		if msg["type"] == msgType {
			return msg
		}
	}
	t.Fatalf("never received message of type %q", msgType)
	return nil
}

// uniqueID generates a unique agent ID for each test run
func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// registerAgent registers a fresh agent and returns the conn + token
func registerAgent(t *testing.T, id, name string) (*websocket.Conn, string) {
	t.Helper()
	conn := dial(t)
	send(t, conn, PANMsg{
		"type":         "register",
		"agent_id":     id,
		"name":         name,
		"email":        "test@example.com",
		"bio":          "test agent",
		"interests":    []string{"testing"},
		"capabilities": []string{"test"},
	})
	welcome := recvType(t, conn, "welcome")
	token, _ := welcome["token"].(string)
	if token == "" {
		t.Fatal("no token in welcome message")
	}
	return conn, token
}

// authAgent connects and authenticates with an existing token
func authAgent(t *testing.T, id, token string) *websocket.Conn {
	t.Helper()
	conn := dial(t)
	send(t, conn, PANMsg{
		"type":     "auth",
		"token":    token,
		"agent_id": id,
	})
	recvType(t, conn, "welcome")
	return conn
}
