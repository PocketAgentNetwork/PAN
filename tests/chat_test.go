package tests

import (
	"testing"
	"time"
)

func TestPublicBroadcast(t *testing.T) {
	idA := uniqueID("broadcast-a")
	idB := uniqueID("broadcast-b")

	connA, _ := registerAgent(t, idA, "BroadcastA")
	defer connA.Close()

	connB, _ := registerAgent(t, idB, "BroadcastB")
	defer connB.Close()

	// A sends public broadcast
	send(t, connA, PANMsg{
		"type": "chat",
		"to":   "all",
		"text": "hello network",
	})

	// B should receive it
	msg := recvType(t, connB, "chat")
	if msg["text"] != "hello network" {
		t.Fatalf("expected 'hello network', got %v", msg["text"])
	}
	t.Logf("✅ Public broadcast received by B: %s", msg["text"])
}

func TestDirectMessage(t *testing.T) {
	idA := uniqueID("dm-a")
	idB := uniqueID("dm-b")

	connA, _ := registerAgent(t, idA, "DM_A")
	defer connA.Close()

	connB, _ := registerAgent(t, idB, "DM_B")
	defer connB.Close()

	// A sends DM to B
	send(t, connA, PANMsg{
		"type": "chat",
		"to":   idB,
		"text": "hey B, private message",
	})

	// B receives DM
	msg := recvType(t, connB, "chat")
	if msg["scope"] != "private" {
		t.Fatalf("expected scope=private, got %v", msg["scope"])
	}
	if msg["text"] != "hey B, private message" {
		t.Fatalf("wrong text: %v", msg["text"])
	}
	t.Logf("✅ DM delivered: %s", msg["text"])
}

func TestRoomChat(t *testing.T) {
	idA := uniqueID("room-a")
	idB := uniqueID("room-b")

	connA, _ := registerAgent(t, idA, "RoomA")
	defer connA.Close()

	connB, _ := registerAgent(t, idB, "RoomB")
	defer connB.Close()

	// Both join #agent-square (auto-joined on register, but send join anyway)
	send(t, connA, PANMsg{"type": "join", "room": "#agent-square"})
	send(t, connB, PANMsg{"type": "join", "room": "#agent-square"})
	time.Sleep(200 * time.Millisecond)

	// A sends room message
	send(t, connA, PANMsg{
		"type": "chat",
		"to":   "#agent-square",
		"text": "hello room",
	})

	// B receives it
	msg := recvType(t, connB, "chat")
	if msg["scope"] != "room" {
		t.Fatalf("expected scope=room, got %v", msg["scope"])
	}
	t.Logf("✅ Room message received: %s", msg["text"])
}

func TestMessageReply(t *testing.T) {
	idA := uniqueID("reply-a")
	idB := uniqueID("reply-b")

	connA, _ := registerAgent(t, idA, "ReplyA")
	defer connA.Close()
	connB, _ := registerAgent(t, idB, "ReplyB")
	defer connB.Close()

	send(t, connA, PANMsg{"type": "join", "room": "#agent-square"})
	send(t, connB, PANMsg{"type": "join", "room": "#agent-square"})
	time.Sleep(200 * time.Millisecond)

	// A sends original message
	send(t, connA, PANMsg{"type": "chat", "to": "#agent-square", "text": "original"})
	orig := recvType(t, connB, "chat")
	msgID, _ := orig["id"].(string)

	// B replies
	send(t, connB, PANMsg{
		"type":     "chat",
		"to":       "#agent-square",
		"text":     "this is a reply",
		"reply_to": msgID,
	})

	reply := recvType(t, connA, "chat")
	if reply["reply_to"] != msgID {
		t.Fatalf("expected reply_to=%s, got %v", msgID, reply["reply_to"])
	}
	t.Logf("✅ Reply threading works, reply_to=%s", msgID)
}

func TestOfflineMessage(t *testing.T) {
	idA := uniqueID("offline-a")
	idB := uniqueID("offline-b")

	connA, _ := registerAgent(t, idA, "OfflineA")
	connB, tokenB := registerAgent(t, idB, "OfflineB")

	// B disconnects
	connB.Close()
	time.Sleep(300 * time.Millisecond)

	// A sends DM to offline B
	send(t, connA, PANMsg{
		"type": "chat",
		"to":   idB,
		"text": "you were offline",
	})
	// A should get ack
	ack := recvType(t, connA, "ack")
	t.Logf("✅ Offline DM ack: %s", ack["message"])
	connA.Close()

	// B reconnects — should get notification + message
	connB2 := authAgent(t, idB, tokenB)
	defer connB2.Close()

	time.Sleep(200 * time.Millisecond) // give goroutine time to fire
	notif := recvType(t, connB2, "notification")
	t.Logf("✅ Offline message delivered on reconnect: %s", notif["message"])
}
