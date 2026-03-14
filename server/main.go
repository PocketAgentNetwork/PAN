package main

import (
	"fmt"
	"log"
	"net/http"

	"pan-server/internal/config"
	"pan-server/internal/database"
	"pan-server/internal/handlers"
	"pan-server/internal/server"

	"github.com/fatih/color"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create server instance
	panServer := server.New(cfg, db)

	// Setup HTTP mux for proper routing
	mux := http.NewServeMux()
	
	// WebSocket endpoint
	mux.HandleFunc("/ws", panServer.HandleWebSocket)
	mux.HandleFunc("/", handlers.HandleDashboard)
	mux.HandleFunc("/api/stats", panServer.HandleStats)
	mux.HandleFunc("/api/register", handlers.HandleRegister(db, cfg.SecretKey))

	// Start server
	color.Green("📟 PAN Network Server v1.0")
	color.Cyan("🚀 WebSocket Server: ws://localhost:%d/ws", cfg.Port)
	color.Cyan("🌐 Web Dashboard: http://localhost:%d", cfg.WebPort)
	color.Yellow("⏳ Waiting for agents to connect...")

	// Start web dashboard server
	go func() {
		webAddr := fmt.Sprintf(":%d", cfg.WebPort)
		log.Printf("Starting web dashboard on %s", webAddr)
		if err := http.ListenAndServe(webAddr, mux); err != nil {
			log.Fatal("Web server failed:", err)
		}
	}()

	// Start WebSocket server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Starting WebSocket server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}