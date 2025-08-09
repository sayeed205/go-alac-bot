package main

import (
	"fmt"
	"log"
	"os"

	"go-alac-bot/config"
)

func main() {
	// Load and validate configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
		os.Exit(1)
	}
	
	// Additional validation
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
		os.Exit(1)
	}
	
	// Configuration loaded successfully
	fmt.Printf("Bot configuration loaded successfully:\n")
	fmt.Printf("- API ID: %d\n", cfg.APIID)
	fmt.Printf("- API Hash: %s\n", maskString(cfg.APIHash))
	fmt.Printf("- Bot Token: %s\n", maskString(cfg.Token))
	fmt.Printf("- Log Level: %s\n", cfg.LogLevel)
}

// maskString masks sensitive information for logging
func maskString(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "***" + s[len(s)-4:]
}
