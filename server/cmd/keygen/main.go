package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
)

func main() {
	// Generate secure 256-bit key
	key := generateSecureKey()
	
	// Create .env file with the key
	envContent := fmt.Sprintf("# PAN Network Configuration\nPAN_SECRET_KEY=%s\n", key)
	
	if err := os.WriteFile(".env", []byte(envContent), 0600); err != nil {
		log.Fatal("Failed to create .env file:", err)
	}
	
	fmt.Println("🔐 Generated secure secret key!")
	fmt.Println("✅ Saved to .env file")
	fmt.Println("🚀 You can now run: go run main.go")
	fmt.Printf("\nYour key: %s\n", key)
}

func generateSecureKey() string {
	// Generate 32 random bytes (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate secure key:", err)
	}
	return hex.EncodeToString(bytes)
}