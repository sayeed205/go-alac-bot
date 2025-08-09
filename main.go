package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-alac-bot/bot"
	"go-alac-bot/config"
)

func main() {
	// Set up proper logging configuration
	logger := setupLogging()
	
	logger.Printf("Starting Telegram bot application...")
	
	// Load and validate configuration using environment validation
	cfg, err := loadAndValidateConfig(logger)
	if err != nil {
		logger.Fatalf("Configuration error: %v", err)
		os.Exit(1)
	}
	
	logger.Printf("Configuration loaded and validated successfully")
	
	// Create and configure the bot
	telegramBot, err := setupBot(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to setup bot: %v", err)
		os.Exit(1)
	}
	
	// Register command handlers
	registerCommandHandlers(telegramBot, logger)
	
	// Start the bot
	if err := telegramBot.Start(); err != nil {
		logger.Fatalf("Failed to start bot: %v", err)
		os.Exit(1)
	}
	
	logger.Printf("Bot started successfully. Press Ctrl+C to stop.")
	
	// Implement graceful shutdown
	gracefulShutdown(telegramBot, logger)
}

// setupLogging configures structured logging for the application
func setupLogging() *log.Logger {
	// Create logger with timestamp and proper formatting
	logger := log.New(os.Stdout, "[TELEGRAM-BOT] ", log.LstdFlags|log.Lshortfile)
	
	// Set log output format based on environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "DEBUG" {
		logger.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	} else {
		// For non-debug levels, use cleaner format without file info
		logger.SetFlags(log.LstdFlags)
	}
	
	return logger
}

// loadAndValidateConfig loads configuration and performs environment validation
func loadAndValidateConfig(logger *log.Logger) (*config.BotConfig, error) {
	logger.Printf("Loading and validating configuration...")
	
	// Load configuration using existing config package
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Perform additional validation using existing validation
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Log configuration status (with masked sensitive data)
	logger.Printf("Configuration details:")
	logger.Printf("- API ID: %d", cfg.APIID)
	logger.Printf("- API Hash: %s", maskString(cfg.APIHash))
	logger.Printf("- Bot Token: %s", maskString(cfg.Token))
	logger.Printf("- Log Level: %s", cfg.LogLevel)
	
	return cfg, nil
}

// setupBot creates and configures the TelegramBot instance
func setupBot(cfg *config.BotConfig, logger *log.Logger) (*bot.TelegramBot, error) {
	logger.Printf("Setting up Telegram bot client...")
	
	// Create the bot instance with error handling
	telegramBot, err := bot.NewTelegramBot(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot instance: %w", err)
	}
	
	logger.Printf("Bot client created successfully")
	return telegramBot, nil
}

// registerCommandHandlers wires together all command handlers with the bot
func registerCommandHandlers(telegramBot *bot.TelegramBot, logger *log.Logger) {
	logger.Printf("Registering command handlers...")
	
	// Create and register /start command handler
	startHandler := bot.NewStartHandler(telegramBot, logger)
	telegramBot.RegisterCommandHandler(startHandler)
	
	// Create and register /ping command handler
	pingHandler := bot.NewPingHandler(telegramBot, logger)
	telegramBot.RegisterCommandHandler(pingHandler)
	
	// Create and register /help command handler
	helpHandler := bot.NewHelpHandler(telegramBot, logger)
	telegramBot.RegisterCommandHandler(helpHandler)
	
	// Create and register /id command handler
	idHandler := bot.NewIDHandler(telegramBot, logger)
	telegramBot.RegisterCommandHandler(idHandler)
	
	// Log registered commands
	registeredCommands := telegramBot.GetRouter().GetRegisteredCommands()
	logger.Printf("Registered %d command handlers: %v", len(registeredCommands), registeredCommands)
}

// gracefulShutdown implements graceful startup and shutdown handling
func gracefulShutdown(telegramBot *bot.TelegramBot, logger *log.Logger) {
	// Create channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Create context for shutdown timeout
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Wait for shutdown signal
	sig := <-sigChan
	logger.Printf("Received signal: %v. Initiating graceful shutdown...", sig)
	
	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()
	
	// Channel to signal shutdown completion
	shutdownComplete := make(chan error, 1)
	
	// Perform graceful shutdown in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Printf("Panic during shutdown: %v", r)
				shutdownComplete <- fmt.Errorf("panic during shutdown: %v", r)
			}
		}()
		
		logger.Printf("Stopping bot...")
		if err := telegramBot.Stop(); err != nil {
			logger.Printf("Error stopping bot: %v", err)
			shutdownComplete <- err
			return
		}
		
		logger.Printf("Bot stopped successfully")
		shutdownComplete <- nil
	}()
	
	// Wait for shutdown completion or timeout
	select {
	case err := <-shutdownComplete:
		if err != nil {
			logger.Printf("Shutdown completed with error: %v", err)
			os.Exit(1)
		}
		logger.Printf("Graceful shutdown completed successfully")
		
	case <-shutdownCtx.Done():
		logger.Printf("Shutdown timeout exceeded. Forcing exit...")
		os.Exit(1)
	}
}

// maskString masks sensitive information for logging
func maskString(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "***" + s[len(s)-4:]
}
