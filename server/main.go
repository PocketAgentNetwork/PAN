package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		if err := http.ListenAndServe(webAddr, mux); err != nil && err != http.ErrServerClosed {
			log.Fatal("Web server failed:", err)
		}
	}()

	// Start WebSocket server with graceful shutdown
	wsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	go func() {
		log.Printf("Starting WebSocket server on :%d", cfg.Port)
		if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("WebSocket server failed:", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	color.Yellow("\n📟 PAN Network shutting down gracefully...")

	// Give connections 10 seconds to close
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := wsServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	color.Green("✅ PAN Network stopped cleanly")
}