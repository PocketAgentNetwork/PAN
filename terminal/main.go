package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var (
		agentID   = flag.String("agent-id", "", "Agent ID")
		agentName = flag.String("name", "", "Agent name")
		email     = flag.String("email", "", "Owner email")
		bio       = flag.String("bio", "PAN Terminal Agent", "Agent bio")
		serverURL = flag.String("server", "ws://localhost:7337/ws", "PAN server URL")
		token     = flag.String("token", "", "Agent token (from registration)")
		cfgFile   = flag.String("config", "pan.json", "Config file path")
	)
	flag.Parse()

	cfg, err := loadConfig(*cfgFile)
	if err != nil {
		cfg = &Config{}
	}

	// CLI flags override config file
	if *agentID != "" {
		cfg.AgentID = *agentID
	}
	if *agentName != "" {
		cfg.AgentName = *agentName
	}
	if *email != "" {
		cfg.Email = *email
	}
	if *bio != "PAN Terminal Agent" || cfg.Bio == "" {
		cfg.Bio = *bio
	}
	if *token != "" {
		cfg.Token = *token
	}
	if *serverURL != "ws://localhost:7337/ws" || cfg.Server == "" {
		cfg.Server = *serverURL
	}

	if cfg.AgentID == "" || cfg.AgentName == "" {
		fmt.Println("Usage: pan-terminal --agent-id <id> --name <name> --email <email> [--token <tok>]")
		fmt.Println("       or create pan.json with your credentials")
		os.Exit(1)
	}

	saveConfig(*cfgFile, cfg) //nolint

	p := tea.NewProgram(
		initialModel(cfg.AgentID, cfg.AgentName, cfg.Email, cfg.Bio, cfg.Server, cfg.Token),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
