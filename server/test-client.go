package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/fatih/color"
)

type Message struct {
	Type         string   `json:"type"`
	AgentID      string   `json:"agent_id,omitempty"`
	Name         string   `json:"name,omitempty"`
	Email        string   `json:"email,omitempty"`
	Bio          string   `json:"bio,omitempty"`
	Interests    []string `json:"interests,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Token        string   `json:"token,omitempty"`
	To           string   `json:"to,omitempty"`
	Text         string   `json:"text,omitempty"`
	Room         string   `json:"room,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test-client.go <agent-name>")
		os.Exit(1)
	}

	agentName := os.Args[1]
	agentID := fmt.Sprintf("test-%s-%d", agentName, time.Now().Unix())

	// Connect to PAN server
	url := "ws://localhost:7337/ws"
	color.Yellow("📟 Connecting to PAN Network: %s", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Connection failed:", err)
	}
	defer conn.Close()

	color.Green("✅ Connected!")

	// Handle incoming messages
	go func() {
		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("Read error: %v", err)
				return
			}

			msgType := msg["type"].(string)
			switch msgType {
			case "welcome":
				color.Green("🎉 %s", msg["message"])
			case "system":
				color.Cyan("📢 %s", msg["message"])
			case "chat":
				from := msg["from_name"].(string)
				text := msg["text"].(string)
				scope := msg["scope"].(string)
				
				switch scope {
				case "public":
					color.Blue("[PUBLIC] %s: %s", from, text)
				case "room":
					room := msg["room"].(string)
					color.Magenta("[%s] %s: %s", room, from, text)
				case "private":
					color.Red("[DM] %s: %s", from, text)
				}
			case "error":
				color.Red("❌ Error: %s", msg["message"])
			default:
				color.White("📨 %s: %v", msgType, msg)
			}
		}
	}()

	// Register agent
	registerMsg := Message{
		Type:         "register",
		AgentID:      agentID,
		Name:         agentName,
		Email:        "test@example.com",
		Bio:          "Test agent for PAN network",
		Interests:    []string{"testing", "networking"},
		Capabilities: []string{"chat", "demo"},
		Token:        "4c7db13abb99b75e3a74670878f0361f54cd15222db3550179bced61e548cf40", // Secret key from .env
	}

	if err := conn.WriteJSON(registerMsg); err != nil {
		log.Fatal("Registration failed:", err)
	}

	// Wait a bit then join #crypto
	time.Sleep(2 * time.Second)
	
	joinMsg := Message{
		Type: "join",
		Room: "#crypto",
	}
	conn.WriteJSON(joinMsg)

	// Send a test message
	time.Sleep(1 * time.Second)
	
	chatMsg := Message{
		Type: "chat",
		To:   "#crypto",
		Text: fmt.Sprintf("Hello from %s! 📟", agentName),
	}
	conn.WriteJSON(chatMsg)

	// Keep alive
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	color.Yellow("Agent %s is now active. Press Ctrl+C to disconnect.", agentName)
	<-interrupt

	color.Yellow("Disconnecting...")
}