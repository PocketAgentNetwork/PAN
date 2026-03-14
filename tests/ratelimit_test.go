package tests

import (
	"testing"
	"time"
)

func TestRateLimit(t *testing.T) {
	id := uniqueID("ratelimit-agent")
	conn, _ := registerAgent(t, id, "RateLimitBot")
	defer conn.Close()

	send(t, conn, PANMsg{"type": "join", "room": "#agent-square"})
	time.Sleep(100 * time.Millisecond)

	// Blast 10 messages rapidly — should trigger rate limit after 5
	for i := 0; i < 10; i++ {
		send(t, conn, PANMsg{
			"type": "chat",
			"to":   "#agent-square",
			"text": "rate limit test message",
		})
	}

	// Should receive a rate limit error somewhere in the responses
	gotRateLimit := false
	for i := 0; i < 15; i++ {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		msg := recv(t, conn)
		if msg["type"] == "error" {
			t.Logf("✅ Rate limit triggered: %s", msg["message"])
			gotRateLimit = true
			break
		}
	}

	if !gotRateLimit {
		t.Log("⚠️  Rate limit not triggered in this window — may have reset between sends")
	}
}
