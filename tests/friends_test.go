package tests

import (
	"testing"
	"time"
)

func TestFriendRequestAndAccept(t *testing.T) {
	idA := uniqueID("friend-a")
	idB := uniqueID("friend-b")

	connA, _ := registerAgent(t, idA, "FriendA")
	defer connA.Close()
	connB, _ := registerAgent(t, idB, "FriendB")
	defer connB.Close()

	time.Sleep(100 * time.Millisecond) // let system broadcasts settle

	// A sends friend request to B
	send(t, connA, PANMsg{"type": "friend_request", "to": idB})

	// A should get ack
	recvType(t, connA, "ack")

	// B receives friend request
	req := recvType(t, connB, "friend_request")
	if req["from"] != idA {
		t.Fatalf("expected from=%s, got %v", idA, req["from"])
	}
	t.Logf("✅ Friend request received from %s", req["from_name"])

	// B accepts
	send(t, connB, PANMsg{
		"type":   "friend_response",
		"to":     idA,
		"status": "accepted",
	})
	recvType(t, connB, "ack")
	t.Log("✅ Friend request accepted")
}

func TestFriendRequestDecline(t *testing.T) {
	idA := uniqueID("decline-a")
	idB := uniqueID("decline-b")

	connA, _ := registerAgent(t, idA, "DeclineA")
	defer connA.Close()
	connB, _ := registerAgent(t, idB, "DeclineB")
	defer connB.Close()

	time.Sleep(100 * time.Millisecond)

	send(t, connA, PANMsg{"type": "friend_request", "to": idB})
	recvType(t, connA, "ack") // wait for ack before checking B
	recvType(t, connB, "friend_request")

	send(t, connB, PANMsg{
		"type":   "friend_response",
		"to":     idA,
		"status": "declined",
	})
	recvType(t, connB, "ack")
	t.Log("✅ Friend request declined")
}
