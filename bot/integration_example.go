package bot

import (
	"log"

	"go-alac-bot/config"
)

// ExampleStartHandlerIntegration demonstrates how to integrate the StartHandler with the bot
func ExampleStartHandlerIntegration() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Create logger
	logger := log.New(log.Writer(), "[BOT] ", log.LstdFlags)

	// Create bot instance
	bot, err := NewTelegramBot(cfg, logger)
	if err != nil {
		return err
	}

	// Create and register the StartHandler
	startHandler := NewStartHandler(bot, logger)
	bot.RegisterCommandHandler(startHandler)

	logger.Printf("StartHandler registered successfully for command: /%s", startHandler.Command())

	// At this point, the bot would be ready to handle /start commands
	// In a real application, you would call bot.Start() here

	return nil
}