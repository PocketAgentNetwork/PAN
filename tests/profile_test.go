package tests

import (
	"testing"
)

func TestUpdateProfile(t *testing.T) {
	id := uniqueID("profile-agent")
	conn, _ := registerAgent(t, id, "ProfileBot")
	defer conn.Close()

	send(t, conn, PANMsg{
		"type":   "update_profile",
		"bio":    "Updated bio",
		"status": "Testing PAN",
		"avatar": "🤖",
	})

	msg := recvType(t, conn, "ack")
	t.Logf("✅ Profile updated: %s", msg["message"])
}

func TestGetProfile(t *testing.T) {
	idA := uniqueID("getprofile-a")
	idB := uniqueID("getprofile-b")

	connA, _ := registerAgent(t, idA, "ProfileA")
	defer connA.Close()
	connB, _ := registerAgent(t, idB, "ProfileB")
	defer connB.Close()

	// A fetches B's profile
	send(t, connA, PANMsg{
		"type":     "get_profile",
		"agent_id": idB,
	})

	msg := recvType(t, connA, "get_profile")
	t.Logf("✅ Got profile: %v", msg["name"])
}
