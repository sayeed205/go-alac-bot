package bot

import (
	"log"
	"os"
	"testing"

	"go-alac-bot/config"
)

func TestStartHandlerIntegration(t *testing.T) {
	// Set up test environment variables
	os.Setenv("BOT_TOKEN", "test_token_123456789")
	os.Setenv("API_ID", "12345")
	os.Setenv("API_HASH", "test_api_hash")
	defer func() {
		os.Unsetenv("BOT_TOKEN")
		os.Unsetenv("API_ID")
		os.Unsetenv("API_HASH")
	}()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create logger
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	// Create bot instance
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Verify bot is created
	if bot == nil {
		t.Fatal("Bot instance is nil")
	}

	// Create StartHandler
	startHandler := NewStartHandler(bot, logger)
	if startHandler == nil {
		t.Fatal("StartHandler instance is nil")
	}

	// Verify handler command
	if startHandler.Command() != "start" {
		t.Errorf("Expected command 'start', got '%s'", startHandler.Command())
	}

	// Register handler with bot
	bot.RegisterCommandHandler(startHandler)

	// Verify handler is registered
	router := bot.GetRouter()
	if router == nil {
		t.Fatal("Router is nil")
	}

	if !router.HasHandler("start") {
		t.Error("StartHandler not registered with router")
	}

	// Verify registered commands include start
	commands := router.GetRegisteredCommands()
	found := false
	for _, cmd := range commands {
		if cmd == "start" {
			found = true
			break
		}
	}

	if !found {
		t.Error("'start' command not found in registered commands")
	}

	t.Log("StartHandler integration test passed successfully")
}