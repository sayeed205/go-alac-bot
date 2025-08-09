package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"BOT_TOKEN": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				"API_ID":    "12345",
				"API_HASH":  "abcdef123456",
				"LOG_LEVEL": "INFO",
			},
			expectError: false,
		},
		{
			name: "valid configuration with default log level",
			envVars: map[string]string{
				"BOT_TOKEN": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				"API_ID":    "12345",
				"API_HASH":  "abcdef123456",
			},
			expectError: false,
		},
		{
			name: "missing BOT_TOKEN",
			envVars: map[string]string{
				"API_ID":   "12345",
				"API_HASH": "abcdef123456",
			},
			expectError: true,
			errorMsg:    "environment validation failed",
		},
		{
			name: "invalid API_ID",
			envVars: map[string]string{
				"BOT_TOKEN": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				"API_ID":    "not_a_number",
				"API_HASH":  "abcdef123456",
			},
			expectError: true,
			errorMsg:    "environment validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()
			
			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config, err := LoadConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error()[:len(tt.errorMsg)] != tt.errorMsg {
					t.Errorf("expected error message to start with %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
					return
				}
				if config == nil {
					t.Errorf("expected config but got nil")
					return
				}
				
				// Verify config values
				if config.Token != tt.envVars["BOT_TOKEN"] {
					t.Errorf("expected token %q, got %q", tt.envVars["BOT_TOKEN"], config.Token)
				}
				
				expectedLogLevel := tt.envVars["LOG_LEVEL"]
				if expectedLogLevel == "" {
					expectedLogLevel = "INFO" // default
				}
				if config.LogLevel != expectedLogLevel {
					t.Errorf("expected log level %q, got %q", expectedLogLevel, config.LogLevel)
				}
			}
		})
	}
}

func TestBotConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *BotConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: &BotConfig{
				Token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				APIID:    12345,
				APIHash:  "abcdef123456",
				LogLevel: "INFO",
			},
			expectError: false,
		},
		{
			name: "empty token",
			config: &BotConfig{
				Token:    "",
				APIID:    12345,
				APIHash:  "abcdef123456",
				LogLevel: "INFO",
			},
			expectError: true,
			errorMsg:    "bot token cannot be empty",
		},
		{
			name: "invalid API ID (zero)",
			config: &BotConfig{
				Token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				APIID:    0,
				APIHash:  "abcdef123456",
				LogLevel: "INFO",
			},
			expectError: true,
			errorMsg:    "API ID must be a positive integer",
		},
		{
			name: "invalid API ID (negative)",
			config: &BotConfig{
				Token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				APIID:    -1,
				APIHash:  "abcdef123456",
				LogLevel: "INFO",
			},
			expectError: true,
			errorMsg:    "API ID must be a positive integer",
		},
		{
			name: "empty API hash",
			config: &BotConfig{
				Token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				APIID:    12345,
				APIHash:  "",
				LogLevel: "INFO",
			},
			expectError: true,
			errorMsg:    "API hash cannot be empty",
		},
		{
			name: "invalid log level",
			config: &BotConfig{
				Token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				APIID:    12345,
				APIHash:  "abcdef123456",
				LogLevel: "INVALID",
			},
			expectError: true,
			errorMsg:    "invalid log level",
		},
		{
			name: "valid DEBUG log level",
			config: &BotConfig{
				Token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				APIID:    12345,
				APIHash:  "abcdef123456",
				LogLevel: "DEBUG",
			},
			expectError: false,
		},
		{
			name: "valid ERROR log level",
			config: &BotConfig{
				Token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				APIID:    12345,
				APIHash:  "abcdef123456",
				LogLevel: "ERROR",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error()[:len(tt.errorMsg)] != tt.errorMsg {
					t.Errorf("expected error message to start with %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}