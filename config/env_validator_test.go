package config

import (
	"os"
	"testing"
)

func TestEnvValidator_ValidateRequired(t *testing.T) {
	validator := NewEnvValidator()

	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "all required variables present",
			envVars: map[string]string{
				"BOT_TOKEN": "test_token",
				"API_ID":    "12345",
				"API_HASH":  "test_hash",
			},
			expectError: false,
		},
		{
			name: "missing BOT_TOKEN",
			envVars: map[string]string{
				"API_ID":   "12345",
				"API_HASH": "test_hash",
			},
			expectError: true,
			errorMsg:    "missing required environment variables: [BOT_TOKEN]",
		},
		{
			name: "missing API_ID",
			envVars: map[string]string{
				"BOT_TOKEN": "test_token",
				"API_HASH":  "test_hash",
			},
			expectError: true,
			errorMsg:    "missing required environment variables: [API_ID]",
		},
		{
			name: "missing API_HASH",
			envVars: map[string]string{
				"BOT_TOKEN": "test_token",
				"API_ID":    "12345",
			},
			expectError: true,
			errorMsg:    "missing required environment variables: [API_HASH]",
		},
		{
			name:        "all variables missing",
			envVars:     map[string]string{},
			expectError: true,
			errorMsg:    "missing required environment variables: [BOT_TOKEN API_ID API_HASH]",
		},
		{
			name: "invalid API_ID format",
			envVars: map[string]string{
				"BOT_TOKEN": "test_token",
				"API_ID":    "not_a_number",
				"API_HASH":  "test_hash",
			},
			expectError: true,
			errorMsg:    "invalid API_ID",
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

			err := validator.ValidateRequired()

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

func TestEnvValidator_GetBotToken(t *testing.T) {
	validator := NewEnvValidator()

	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "valid token",
			envValue: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			expected: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
		},
		{
			name:     "empty token",
			envValue: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				os.Setenv("BOT_TOKEN", tt.envValue)
			}

			result := validator.GetBotToken()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEnvValidator_GetAPICredentials(t *testing.T) {
	validator := NewEnvValidator()

	tests := []struct {
		name          string
		apiID         string
		apiHash       string
		expectedID    int
		expectedHash  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid credentials",
			apiID:        "12345",
			apiHash:      "abcdef123456",
			expectedID:   12345,
			expectedHash: "abcdef123456",
			expectError:  false,
		},
		{
			name:          "missing API_ID",
			apiID:         "",
			apiHash:       "abcdef123456",
			expectedID:    0,
			expectedHash:  "",
			expectError:   true,
			errorContains: "API_ID environment variable is not set",
		},
		{
			name:          "missing API_HASH",
			apiID:         "12345",
			apiHash:       "",
			expectedID:    0,
			expectedHash:  "",
			expectError:   true,
			errorContains: "API_HASH environment variable is not set",
		},
		{
			name:          "invalid API_ID format",
			apiID:         "not_a_number",
			apiHash:       "abcdef123456",
			expectedID:    0,
			expectedHash:  "",
			expectError:   true,
			errorContains: "API_ID must be a valid integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.apiID != "" {
				os.Setenv("API_ID", tt.apiID)
			}
			if tt.apiHash != "" {
				os.Setenv("API_HASH", tt.apiHash)
			}

			apiID, apiHash, err := validator.GetAPICredentials()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && err.Error()[:len(tt.errorContains)] != tt.errorContains {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
					return
				}
				if apiID != tt.expectedID {
					t.Errorf("expected API ID %d, got %d", tt.expectedID, apiID)
				}
				if apiHash != tt.expectedHash {
					t.Errorf("expected API Hash %q, got %q", tt.expectedHash, apiHash)
				}
			}
		})
	}
}