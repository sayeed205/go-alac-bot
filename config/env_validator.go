package config

import (
	"fmt"
	"os"
	"strconv"
)

// EnvValidator handles validation of required environment variables
type EnvValidator struct{}

// NewEnvValidator creates a new environment validator instance
func NewEnvValidator() *EnvValidator {
	return &EnvValidator{}
}

// ValidateRequired validates that all required environment variables are present
// Returns an error if any required variables are missing
func (e *EnvValidator) ValidateRequired() error {
	requiredVars := []string{"BOT_TOKEN", "API_ID", "API_HASH"}
	
	var missingVars []string
	for _, varName := range requiredVars {
		if value := os.Getenv(varName); value == "" {
			missingVars = append(missingVars, varName)
		}
	}
	
	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v. Please set these variables in your .env file or environment", missingVars)
	}
	
	// Validate API_ID is a valid integer
	if _, _, err := e.GetAPICredentials(); err != nil {
		return fmt.Errorf("invalid API_ID: %w", err)
	}
	
	return nil
}

// GetBotToken returns the bot token from environment variables
func (e *EnvValidator) GetBotToken() string {
	return os.Getenv("BOT_TOKEN")
}

// GetAPICredentials returns the API ID and API Hash from environment variables
// Returns an error if API_ID cannot be converted to integer
func (e *EnvValidator) GetAPICredentials() (apiID int, apiHash string, err error) {
	apiIDStr := os.Getenv("API_ID")
	apiHash = os.Getenv("API_HASH")
	
	if apiIDStr == "" {
		return 0, "", fmt.Errorf("API_ID environment variable is not set")
	}
	
	if apiHash == "" {
		return 0, "", fmt.Errorf("API_HASH environment variable is not set")
	}
	
	apiID, err = strconv.Atoi(apiIDStr)
	if err != nil {
		return 0, "", fmt.Errorf("API_ID must be a valid integer, got: %s", apiIDStr)
	}
	
	return apiID, apiHash, nil
}