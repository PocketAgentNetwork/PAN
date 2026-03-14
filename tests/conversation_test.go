package tests

import (
	"testing"
	"time"
)

// pace adds a small delay between sends to avoid hitting the rate limit (5 msgs/sec default)
func pace() { time.Sleep(250 * time.Millisecond) }

// TestConversation simulates two agents having a real back-and-forth exchange:
// room chat, threaded replies, DMs, friend request, profile check, offline messages.
func TestConversation(t *testing.T) {
	idAlpha := uniqueID("alpha")
	idBeta  := uniqueID("beta")

	// ── Register both agents ──────────────────────────────────────────────────
	connAlpha, tokenAlpha := registerAgent(t, idAlpha, "Alpha")
	defer connAlpha.Close()
	connBeta, tokenBeta := registerAgent(t, idBeta, "Beta")
	defer connBeta.Close()
	t.Log("✅ Both agents registered")
	time.Sleep(100 * time.Millisecond)

	// ── Alpha says hi in #agent-square ────────────────────────────────────────
	send(t, connAlpha, PANMsg{"type": "chat", "to": "#agent-square", "text": "hey Beta, you there?"})
	msg := recvType(t, connBeta, "chat")
	if msg["text"] != "hey Beta, you there?" {
		t.Fatalf("Beta didn't get Alpha's room message, got: %v", msg["text"])
	}
	t.Logf("✅ Beta received room message: %q", msg["text"])
	pace()

	// ── Beta replies with a thread ────────────────────────────────────────────
	origID, _ := msg["id"].(string)
	send(t, connBeta, PANMsg{
		"type":     "chat",
		"to":       "#agent-square",
		"text":     "yeah I'm here, what's up?",
		"reply_to": origID,
	})
	reply := recvType(t, connAlpha, "chat")
	if reply["text"] != "yeah I'm here, what's up?" {
		t.Fatalf("Alpha didn't get Beta's reply, got: %v", reply["text"])
	}
	if reply["reply_to"] != origID {
		t.Fatalf("reply_to mismatch: expected %s got %v", origID, reply["reply_to"])
	}
	t.Logf("✅ Alpha received threaded reply: %q", reply["text"])
	pace()

	// ── Alpha sends Beta a DM ─────────────────────────────────────────────────
	send(t, connAlpha, PANMsg{"type": "chat", "to": idBeta, "text": "sending you a private message"})
	dm := recvType(t, connBeta, "chat")
	if dm["scope"] != "private" {
		t.Fatalf("expected scope=private, got %v", dm["scope"])
	}
	t.Logf("✅ Beta received DM: %q", dm["text"])
	pace()

	// ── Beta DMs back ─────────────────────────────────────────────────────────
	send(t, connBeta, PANMsg{"type": "chat", "to": idAlpha, "text": "got it, replying privately"})
	dm2 := recvType(t, connAlpha, "chat")
	if dm2["scope"] != "private" {
		t.Fatalf("expected scope=private, got %v", dm2["scope"])
	}
	t.Logf("✅ Alpha received DM back: %q", dm2["text"])
	pace()

	// ── Alpha sends Beta a friend request ─────────────────────────────────────
	send(t, connAlpha, PANMsg{"type": "friend_request", "to": idBeta})
	recvType(t, connAlpha, "ack")
	req := recvType(t, connBeta, "friend_request")
	if req["from"] != idAlpha {
		t.Fatalf("friend request from wrong agent: %v", req["from"])
	}
	t.Logf("✅ Beta received friend request from %v", req["from_name"])
	pace()

	// ── Beta accepts ──────────────────────────────────────────────────────────
	send(t, connBeta, PANMsg{"type": "friend_response", "to": idAlpha, "status": "accepted"})
	recvType(t, connBeta, "ack")
	t.Log("✅ Beta accepted friend request")
	pace()

	// ── Alpha checks Beta's profile ───────────────────────────────────────────
	send(t, connAlpha, PANMsg{"type": "get_profile", "agent_id": idBeta})
	profile := recvType(t, connAlpha, "get_profile")
	if profile["name"] != "Beta" {
		t.Fatalf("wrong profile name: %v", profile["name"])
	}
	t.Logf("✅ Alpha fetched Beta's profile: name=%v", profile["name"])
	pace()

	// ── Beta updates their profile ────────────────────────────────────────────
	send(t, connBeta, PANMsg{
		"type":   "update_profile",
		"bio":    "I am Beta, a test agent",
		"status": "online and chatting",
		"avatar": "🤖",
	})
	recvType(t, connBeta, "ack")
	t.Log("✅ Beta updated profile")
	pace()

	// ── Both join #crypto and chat there ─────────────────────────────────────
	send(t, connAlpha, PANMsg{"type": "join", "room": "#crypto"})
	recvType(t, connAlpha, "system")
	pace()
	send(t, connBeta, PANMsg{"type": "join", "room": "#crypto"})
	recvType(t, connBeta, "system")
	time.Sleep(150 * time.Millisecond)

	send(t, connAlpha, PANMsg{"type": "chat", "to": "#crypto", "text": "crypto talk from Alpha"})
	cryptoMsg := recvType(t, connBeta, "chat")
	if cryptoMsg["text"] != "crypto talk from Alpha" {
		t.Fatalf("wrong crypto message: %v", cryptoMsg["text"])
	}
	t.Logf("✅ Beta received Alpha's #crypto message: %q", cryptoMsg["text"])
	pace()

	send(t, connBeta, PANMsg{"type": "chat", "to": "#crypto", "text": "crypto talk from Beta"})
	cryptoMsg2 := recvType(t, connAlpha, "chat")
	if cryptoMsg2["text"] != "crypto talk from Beta" {
		t.Fatalf("wrong crypto message: %v", cryptoMsg2["text"])
	}
	t.Logf("✅ Alpha received Beta's #crypto message: %q", cryptoMsg2["text"])
	pace()

	// ── Offline message: Alpha disconnects, Beta sends DM, Alpha reconnects ───
	connAlpha.Close()
	time.Sleep(300 * time.Millisecond)

	send(t, connBeta, PANMsg{"type": "chat", "to": idAlpha, "text": "you were offline, catch you later"})
	ack := recvType(t, connBeta, "ack")
	t.Logf("✅ Offline DM ack: %v", ack["message"])

	connAlpha2 := authAgent(t, idAlpha, tokenAlpha)
	defer connAlpha2.Close()
	time.Sleep(200 * time.Millisecond)

	notif := recvType(t, connAlpha2, "notification")
	t.Logf("✅ Alpha got offline notification: %v", notif["message"])
	pace()

	// ── Beta disconnects and reconnects ──────────────────────────────────────
	connBeta.Close()
	time.Sleep(200 * time.Millisecond)
	connBeta2 := authAgent(t, idBeta, tokenBeta)
	defer connBeta2.Close()
	time.Sleep(200 * time.Millisecond)
	// consume Beta's offline notification + the offline DM content
	recvType(t, connBeta2, "notification")
	recvType(t, connBeta2, "chat") // the offline DM itself
	t.Log("✅ Beta reconnected and drained offline messages")
	pace()

	// ── Final exchange: Alpha DMs Beta, then room bye ────────────────────────
	send(t, connAlpha2, PANMsg{"type": "chat", "to": idBeta, "text": "direct bye from Alpha"})
	directBye := recvType(t, connBeta2, "chat")
	if directBye["scope"] != "private" {
		t.Fatalf("expected private scope, got %v", directBye["scope"])
	}
	t.Logf("✅ Full conversation complete — direct bye: %q", directBye["text"])
}
