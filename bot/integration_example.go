package bot

import (
	"log"

	"go-alac-bot/config"
)

// ExampleBotIntegration demonstrates how to integrate all command handlers with the bot
func ExampleBotIntegration() error {
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

	// Create and register the PingHandler
	pingHandler := NewPingHandler(bot, logger)
	bot.RegisterCommandHandler(pingHandler)
	logger.Printf("PingHandler registered successfully for command: /%s", pingHandler.Command())

	// Create and register the HelpHandler
	helpHandler := NewHelpHandler(bot, logger)
	bot.RegisterCommandHandler(helpHandler)
	logger.Printf("HelpHandler registered successfully for command: /%s", helpHandler.Command())

	// At this point, the bot would be ready to handle /start, /ping, and /help commands
	// In a real application, you would call bot.Start() here

	return nil
}

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

// ExamplePingHandlerIntegration demonstrates how to integrate the PingHandler with the bot
func ExamplePingHandlerIntegration() error {
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

	// Create and register the PingHandler
	pingHandler := NewPingHandler(bot, logger)
	bot.RegisterCommandHandler(pingHandler)

	logger.Printf("PingHandler registered successfully for command: /%s", pingHandler.Command())

	// At this point, the bot would be ready to handle /ping commands
	// In a real application, you would call bot.Start() here

	return nil
}

// ExampleHelpHandlerIntegration demonstrates how to integrate the HelpHandler with the bot
func ExampleHelpHandlerIntegration() error {
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

	// Create and register the HelpHandler
	helpHandler := NewHelpHandler(bot, logger)
	bot.RegisterCommandHandler(helpHandler)

	logger.Printf("HelpHandler registered successfully for command: /%s", helpHandler.Command())

	// At this point, the bot would be ready to handle /help commands
	// In a real application, you would call bot.Start() here

	return nil
}