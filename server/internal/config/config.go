package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strconv"
	
	"github.com/joho/godotenv"
)

type Config struct {
	Port         int
	WebPort      int
	SecretKey    string
	DatabasePath string
	
	// Rate limiting
	RateLimitWindow    int // milliseconds
	MaxMsgsPerWindow   int
	MaxMessageLength   int
	MaxAgentNameLength int
	MaxRoomNameLength  int
	
	// Connection limits
	MaxAgentsPerIP int
	MaxTotalAgents int
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// .env file not found, that's okay
	}
	
	// Generate or load secret key
	secretKey := getSecretKey()
	
	return &Config{
		Port:               getEnvInt("PAN_PORT", 7337),     // 1337 but with 7 for luck 😄
		WebPort:            getEnvInt("PAN_WEB_PORT", 7338), // Sequential for web dashboard
		SecretKey:          secretKey,
		DatabasePath:       getEnvString("PAN_DB_PATH", "./pan.db"),
		
		RateLimitWindow:    getEnvInt("PAN_RATE_LIMIT_WINDOW", 1000),
		MaxMsgsPerWindow:   getEnvInt("PAN_MAX_MSGS_PER_WINDOW", 5),
		MaxMessageLength:   getEnvInt("PAN_MAX_MESSAGE_LENGTH", 10000),
		MaxAgentNameLength: getEnvInt("PAN_MAX_AGENT_NAME_LENGTH", 30),
		MaxRoomNameLength:  getEnvInt("PAN_MAX_ROOM_NAME_LENGTH", 50),
		
		MaxAgentsPerIP:     getEnvInt("PAN_MAX_AGENTS_PER_IP", 10),
		MaxTotalAgents:     getEnvInt("PAN_MAX_TOTAL_AGENTS", 100000),
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getSecretKey() string {
	// 1. Check environment variable first
	if key := os.Getenv("PAN_SECRET_KEY"); key != "" {
		if len(key) < 32 {
			log.Fatal("PAN_SECRET_KEY must be at least 32 characters long")
		}
		return key
	}
	
	// 2. Check for existing key file
	keyFile := ".pan_secret"
	if data, err := os.ReadFile(keyFile); err == nil {
		key := string(data)
		if len(key) >= 32 {
			return key
		}
	}
	
	// 3. Generate new secure key
	log.Println("⚠️  No PAN_SECRET_KEY found. Generating new secure key...")
	key := generateSecureKey()
	
	// Save to file for persistence
	if err := os.WriteFile(keyFile, []byte(key), 0600); err != nil {
		log.Printf("Warning: Could not save secret key to file: %v", err)
	} else {
		log.Printf("✅ Secret key saved to %s", keyFile)
	}
	
	log.Println("🔐 For production, set PAN_SECRET_KEY environment variable")
	return key
}

func generateSecureKey() string {
	// Generate 32 random bytes (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate secure key:", err)
	}
	return hex.EncodeToString(bytes)
}