package main

import (
	"encoding/json"
	"os"
)

// Config holds agent credentials and connection settings, persisted to pan.json
type Config struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"name"`
	Email     string `json:"email"`
	Bio       string `json:"bio"`
	Token     string `json:"token"`
	Server    string `json:"server"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	return &cfg, json.Unmarshal(data, &cfg)
}

func saveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
