package tests

import (
	"testing"
	"time"
)

func TestJoinAndLeaveRoom(t *testing.T) {
	id := uniqueID("room-join")
	conn, _ := registerAgent(t, id, "RoomJoiner")
	defer conn.Close()

	send(t, conn, PANMsg{"type": "join", "room": "#crypto"})
	time.Sleep(200 * time.Millisecond)

	send(t, conn, PANMsg{"type": "leave", "room": "#crypto"})
	time.Sleep(200 * time.Millisecond)
	t.Log("✅ Join and leave room OK")
}

func TestCreateRoom(t *testing.T) {
	id := uniqueID("room-creator")
	conn, _ := registerAgent(t, id, "RoomCreator")
	defer conn.Close()

	roomName := "#test-" + uniqueID("r")
	send(t, conn, PANMsg{
		"type":      "create_room",
		"room":      roomName,
		"room_desc": "a test room",
		"private":   false,
	})

	msg := recvType(t, conn, "ack")
	t.Logf("✅ Room created: %s — %s", roomName, msg["message"])
}

func TestRoomInfo(t *testing.T) {
	id := uniqueID("room-info")
	conn, _ := registerAgent(t, id, "RoomInfo")
	defer conn.Close()

	send(t, conn, PANMsg{"type": "join", "room": "#agent-square"})
	time.Sleep(100 * time.Millisecond)

	send(t, conn, PANMsg{"type": "room_info", "room": "#agent-square"})
	msg := recvType(t, conn, "room_info")
	t.Logf("✅ Room info: %v members", len(msg["agents"].([]interface{})))
}

func TestSendToUnjoinedRoom(t *testing.T) {
	id := uniqueID("unjoined")
	conn, _ := registerAgent(t, id, "UnjoinedBot")
	defer conn.Close()

	// Try to send to a room without joining
	send(t, conn, PANMsg{
		"type": "chat",
		"to":   "#research",
		"text": "should fail",
	})
	msg := recvType(t, conn, "error")
	t.Logf("✅ Unjoined room rejected: %s", msg["message"])
}

func TestListAgentsAndRooms(t *testing.T) {
	id := uniqueID("list-agent")
	conn, _ := registerAgent(t, id, "ListBot")
	defer conn.Close()

	send(t, conn, PANMsg{"type": "list"})
	msg := recvType(t, conn, "list")

	rooms, _ := msg["rooms"].([]interface{})
	agents, _ := msg["agents"].([]interface{})
	t.Logf("✅ List OK — %d agents online, %d rooms", len(agents), len(rooms))
}
