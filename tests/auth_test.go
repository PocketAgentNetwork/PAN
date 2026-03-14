package tests

import (
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	id := uniqueID("reg-agent")
	conn, token := registerAgent(t, id, "RegBot")
	defer conn.Close()

	if token == "" {
		t.Fatal("expected token, got empty string")
	}
	t.Logf("✅ Register OK — token: %s", token[:20]+"...")
}

func TestRegisterDuplicateID(t *testing.T) {
	id := uniqueID("dup-agent")
	conn1, _ := registerAgent(t, id, "DupBot")
	defer conn1.Close()

	// Try registering same ID again on a new connection
	conn2 := dial(t)
	defer conn2.Close()
	send(t, conn2, PANMsg{
		"type":     "register",
		"agent_id": id,
		"name":     "DupBot2",
		"email":    "test@example.com",
	})
	msg := recvType(t, conn2, "error")
	t.Logf("✅ Duplicate register rejected: %s", msg["message"])
}

func TestAuthWithToken(t *testing.T) {
	id := uniqueID("auth-agent")
	conn1, token := registerAgent(t, id, "AuthBot")
	conn1.Close() // disconnect first
	time.Sleep(200 * time.Millisecond) // wait for server to process disconnect

	// Reconnect with token
	conn2 := authAgent(t, id, token)
	defer conn2.Close()
	t.Log("✅ Token auth OK")
}

func TestAuthInvalidToken(t *testing.T) {
	conn := dial(t)
	defer conn.Close()
	send(t, conn, PANMsg{
		"type":     "auth",
		"token":    "pan_tok_invalid_fake_token",
		"agent_id": "nobody",
	})
	msg := recvType(t, conn, "error")
	t.Logf("✅ Invalid token rejected: %s", msg["message"])
}

func TestAuthMissingFields(t *testing.T) {
	conn := dial(t)
	defer conn.Close()
	send(t, conn, PANMsg{
		"type": "register",
		// missing agent_id, name, email
	})
	msg := recvType(t, conn, "error")
	t.Logf("✅ Missing fields rejected: %s", msg["message"])
}
