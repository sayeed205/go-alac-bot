package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// BotConfig holds all configuration values for the Telegram bot
type BotConfig struct {
	Token    string // Telegram bot token
	APIID    int    // Telegram API ID
	APIHash  string // Telegram API Hash
	LogLevel string // Logging level (INFO, WARN, ERROR, FATAL)
}

// LoadConfig loads and validates the bot configuration from environment variables
// Returns a BotConfig struct or an error if validation fails
func LoadConfig() (*BotConfig, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}
	
	// Create and use environment validator
	validator := NewEnvValidator()
	
	// Validate required environment variables
	if err := validator.ValidateRequired(); err != nil {
		return nil, fmt.Errorf("environment validation failed: %w", err)
	}
	
	// Get API credentials
	apiID, apiHash, err := validator.GetAPICredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get API credentials: %w", err)
	}
	
	// Get bot token
	token := validator.GetBotToken()
	if token == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required but not set")
	}
	
	// Get log level with default
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO" // Default log level
	}
	
	config := &BotConfig{
		Token:    token,
		APIID:    apiID,
		APIHash:  apiHash,
		LogLevel: logLevel,
	}
	
	return config, nil
}

// Validate performs additional validation on the loaded configuration
func (c *BotConfig) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("bot token cannot be empty")
	}
	
	if c.APIID <= 0 {
		return fmt.Errorf("API ID must be a positive integer, got: %d", c.APIID)
	}
	
	if c.APIHash == "" {
		return fmt.Errorf("API hash cannot be empty")
	}
	
	// Validate log level
	validLogLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
		"FATAL": true,
	}
	
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s. Valid levels are: DEBUG, INFO, WARN, ERROR, FATAL", c.LogLevel)
	}
	
	return nil
}