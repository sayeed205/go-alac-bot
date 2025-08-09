package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"go-alac-bot/config"
	"go.uber.org/zap"
)

// TelegramBot wraps the gotgproto client and provides bot lifecycle management
type TelegramBot struct {
	client *gotgproto.Client
	logger *log.Logger
	config *config.BotConfig
	router *CommandRouter
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTelegramBot creates a new TelegramBot instance with proper gotgproto client setup
func NewTelegramBot(cfg *config.BotConfig, logger *log.Logger) (*TelegramBot, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	
	// Create context for bot lifecycle management
	ctx, cancel := context.WithCancel(context.Background())
	
	bot := &TelegramBot{
		config: cfg,
		logger: logger,
		router: NewCommandRouter(logger),
		ctx:    ctx,
		cancel: cancel,
	}
	
	return bot, nil
}

// Start initializes the gotgproto client and starts the bot
func (b *TelegramBot) Start() error {
	b.logger.Printf("Starting Telegram bot...")
	
	// Create zap logger for gotgproto (it requires zap logger)
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create zap logger: %w", err)
	}
	
	// Create gotgproto client options
	clientOpts := &gotgproto.ClientOpts{
		Session: sessionMaker.SqlSession(sqlite.Open("bot_session.db")),
		Logger:  zapLogger,
	}
	
	// Initialize gotgproto client
	client, err := gotgproto.NewClient(b.config.APIID, b.config.APIHash, gotgproto.ClientTypeBot(b.config.Token), clientOpts)
	if err != nil {
		return fmt.Errorf("failed to create gotgproto client: %w", err)
	}
	
	b.client = client
	b.logger.Printf("Telegram bot client initialized successfully")
	
	// Start the client - this is a blocking call, so we run it in a goroutine
	go func() {
		b.logger.Printf("Starting gotgproto client...")
		b.client.Idle()
	}()
	
	// Wait a moment to ensure client is ready
	time.Sleep(100 * time.Millisecond)
	
	b.logger.Printf("Telegram bot started successfully")
	return nil
}

// Stop gracefully shuts down the bot
func (b *TelegramBot) Stop() error {
	b.logger.Printf("Stopping Telegram bot...")
	
	if b.cancel != nil {
		b.cancel()
	}
	
	if b.client != nil {
		// Give the client time to clean up
		time.Sleep(100 * time.Millisecond)
		b.logger.Printf("Bot client stopped")
	}
	
	b.logger.Printf("Telegram bot stopped successfully")
	return nil
}

// GetClient returns the underlying gotgproto client for advanced usage
func (b *TelegramBot) GetClient() *gotgproto.Client {
	return b.client
}

// IsRunning returns true if the bot is currently running
func (b *TelegramBot) IsRunning() bool {
	return b.client != nil && b.ctx.Err() == nil
}

// RegisterCommandHandler registers a command handler with the bot's router
func (b *TelegramBot) RegisterCommandHandler(handler CommandHandler) {
	b.router.RegisterHandler(handler)
}

// GetRouter returns the command router for advanced usage
func (b *TelegramBot) GetRouter() *CommandRouter {
	return b.router
}