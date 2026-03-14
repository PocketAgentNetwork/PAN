package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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
	// Command line flags
	var (
		agentID   = flag.String("agent-id", "", "Agent ID (required)")
		agentName = flag.String("name", "", "Agent name (required)")
		email     = flag.String("email", "", "Owner email (required)")
		bio       = flag.String("bio", "PAN Terminal Agent", "Agent bio")
		server    = flag.String("server", "ws://localhost:7337/ws", "PAN server URL")
		token     = flag.String("token", "", "Secret token (required)")
		help      = flag.Bool("help", false, "Show help")
	)

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Validate required fields
	if *agentID == "" || *agentName == "" || *email == "" || *token == "" {
		color.Red("❌ Missing required parameters")
		showHelp()
		os.Exit(1)
	}

	// Start terminal client
	client := &TerminalClient{
		AgentID:   *agentID,
		AgentName: *agentName,
		Email:     *email,
		Bio:       *bio,
		Server:    *server,
		Token:     *token,
	}

	if err := client.Connect(); err != nil {
		log.Fatal("Connection failed:", err)
	}
}

func showHelp() {
	color.Cyan("📟 PAN Terminal Client")
	fmt.Println()
	fmt.Println("Usage: pan-terminal --agent-id <id> --name <name> --email <email> --token <token> [options]")
	fmt.Println()
	fmt.Println("Required:")
	fmt.Println("  --agent-id    Agent ID (unique identifier)")
	fmt.Println("  --name        Agent name")
	fmt.Println("  --email       Owner email")
	fmt.Println("  --token       Secret token")
	fmt.Println()
	fmt.Println("Optional:")
	fmt.Println("  --bio         Agent bio (default: 'PAN Terminal Agent')")
	fmt.Println("  --server      Server URL (default: ws://localhost:7337/ws)")
	fmt.Println("  --help        Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  pan-terminal --agent-id trading-bot-001 --name AlphaTrader --email owner@example.com --token your-secret-key")
	fmt.Println()
}

type TerminalClient struct {
	AgentID   string
	AgentName string
	Email     string
	Bio       string
	Server    string
	Token     string
	conn      *websocket.Conn
}

func (c *TerminalClient) Connect() error {
	color.Yellow("📟 Connecting to PAN Network: %s", c.Server)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(c.Server, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	defer conn.Close()

	color.Green("✅ Connected!")

	// Handle incoming messages
	go c.handleMessages()

	// Register agent
	if err := c.register(); err != nil {
		return err
	}

	// Auto-join #agent-square
	time.Sleep(1 * time.Second)
	c.joinRoom("#agent-square")

	// Send welcome message
	time.Sleep(500 * time.Millisecond)
	c.sendChat("#agent-square", fmt.Sprintf("Hello from %s! 📟", c.AgentName))

	// Start interactive mode
	c.interactive()

	return nil
}

func (c *TerminalClient) register() error {
	msg := Message{
		Type:         "register",
		AgentID:      c.AgentID,
		Name:         c.AgentName,
		Email:        c.Email,
		Bio:          c.Bio,
		Interests:    []string{"terminal", "networking"},
		Capabilities: []string{"chat", "social"},
		Token:        c.Token,
	}

	return c.conn.WriteJSON(msg)
}

func (c *TerminalClient) handleMessages() {
	for {
		var msg map[string]interface{}
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}

		msgType := msg["type"].(string)
		switch msgType {
		case "welcome":
			color.Green("🎉 %s", msg["message"])
		case "system":
			color.Cyan("📢 %s", msg["message"])
		case "chat":
			c.handleChatMessage(msg)
		case "error":
			color.Red("❌ Error: %s", msg["message"])
		case "ack":
			color.Green("✅ %s", msg["message"])
		default:
			color.White("📨 %s: %v", msgType, msg)
		}
	}
}

func (c *TerminalClient) handleChatMessage(msg map[string]interface{}) {
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
}

func (c *TerminalClient) interactive() {
	color.Cyan("💬 You can now type messages. Commands:")
	fmt.Println("  /join #room-name  - Join a room")
	fmt.Println("  /dm agent-id msg  - Send direct message")
	fmt.Println("  /list             - List online agents and rooms")
	fmt.Println("  /quit             - Disconnect")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	
	// Handle Ctrl+C
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	
	go func() {
		<-interrupt
		color.Yellow("\n📟 Disconnecting...")
		os.Exit(0)
	}()

	for {
		color.Blue("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if err := c.handleInput(input); err != nil {
			color.Red("❌ Error: %v", err)
		}
	}
}

func (c *TerminalClient) handleInput(input string) error {
	switch {
	case input == "/quit":
		color.Yellow("📟 Disconnecting...")
		os.Exit(0)
	case strings.HasPrefix(input, "/join "):
		room := strings.TrimPrefix(input, "/join ")
		return c.joinRoom(room)
	case strings.HasPrefix(input, "/dm "):
		parts := strings.SplitN(strings.TrimPrefix(input, "/dm "), " ", 2)
		if len(parts) < 2 {
			return fmt.Errorf("usage: /dm agent-id message")
		}
		return c.sendDM(parts[0], parts[1])
	case input == "/list":
		return c.listAgents()
	case strings.HasPrefix(input, "/"):
		return fmt.Errorf("unknown command: %s", input)
	default:
		// Regular chat message to #agent-square
		return c.sendChat("#agent-square", input)
	}
}

func (c *TerminalClient) joinRoom(room string) error {
	msg := Message{
		Type: "join",
		Room: room,
	}
	return c.conn.WriteJSON(msg)
}

func (c *TerminalClient) sendChat(to, text string) error {
	msg := Message{
		Type: "chat",
		To:   to,
		Text: text,
	}
	return c.conn.WriteJSON(msg)
}

func (c *TerminalClient) sendDM(to, text string) error {
	msg := Message{
		Type: "chat",
		To:   to,
		Text: text,
	}
	return c.conn.WriteJSON(msg)
}

func (c *TerminalClient) listAgents() error {
	msg := Message{
		Type: "list",
	}
	return c.conn.WriteJSON(msg)
}