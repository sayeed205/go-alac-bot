# Go ALAC Bot

A Telegram bot for downloading songs from Apple Music with queue management and real-time progress tracking.

## Features

- 🎵 Download songs from Apple Music URLs
- 📊 Real-time download and upload progress bars
- 🔄 Queue system with up to 7 concurrent requests
- 📱 Support for both private chats and groups
- 🎯 Automatic file cleanup after upload
- 📋 Queue status monitoring with `/queue` command
- 🔍 URL validation and metadata extraction

## Prerequisites

- Go 1.21 or higher
- Telegram Bot Token
- Telegram API credentials (API ID and API Hash)

## Setup Instructions

### 1. Clone the Repository

```bash
git clone https://github.com/sayeed205/go-alac-bot.git
cd go-alac-bot
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Get Telegram Credentials

#### Get Bot Token:
1. Message [@BotFather](https://t.me/botfather) on Telegram
2. Send `/newbot` and follow instructions
3. Copy your bot token (looks like: `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)

#### Get API ID and API Hash:
1. Go to [my.telegram.org/apps](https://my.telegram.org/apps)
2. Log in with your Telegram account
3. Create a new application
4. Copy your `API ID` (number) and `API Hash` (string)

### 4. Environment Configuration

Create a `.env` file in the project root:

```bash
# Required: Telegram Bot Credentials
BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
API_ID=12345678
API_HASH=abcdef1234567890abcdef1234567890

# Optional: Logging
LOG_LEVEL=INFO
```

#### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `BOT_TOKEN` | ✅ | Telegram bot token from BotFather | `123456789:ABCdefGHI...` |
| `API_ID` | ✅ | Telegram API ID from my.telegram.org | `12345678` |
| `API_HASH` | ✅ | Telegram API Hash from my.telegram.org | `abcdef1234567890...` |
| `LOG_LEVEL` | ❌ | Logging level (DEBUG, INFO, WARN, ERROR) | `INFO` |

### 5. Build and Run

#### Development Mode:
```bash
go run .
```

#### Production Build:
```bash
go build -o go-alac-bot .
./go-alac-bot
```

#### Using Docker (Optional):
```bash
# Build image
docker build -t go-alac-bot .

# Run container
docker run -d \
  --name go-alac-bot \
  --env-file .env \
  -v $(pwd)/downloads:/app/downloads \
  go-alac-bot
```

## Usage

### Available Commands

| Command | Description | Usage |
|---------|-------------|-------|
| `/start` | Welcome message | `/start` |
| `/help` | Show help and examples | `/help` |
| `/song` | Download a song (queued) | `/song https://music.apple.com/...` |
| `/queue` | Check queue status | `/queue` |
| `/id` | Get chat/user ID | `/id` or reply to message |
| `/ping` | Test bot responsiveness | `/ping` |

### Song Download Examples

```
/song https://music.apple.com/us/song/never-gonna-give-you-up/1559523359

/song https://music.apple.com/us/album/never-gonna-give-you-up/1559523357?i=1559523359
```

### Queue System

- **Maximum**: 7 requests in queue
- **Processing**: One song at a time
- **Status**: Use `/queue` to check position
- **Automatic**: Processes requests in order

#### Queue Messages:
- ✅ **Empty queue**: "🎵 Processing your request..."
- 📋 **In queue**: "🎵 Your request is in queue at position 3"
- ❌ **Full queue**: "❌ Queue is full! Current limit is 7 requests..."

## Project Structure

```
go-alac-bot/
├── bot/                    # Bot handlers and logic
│   ├── client.go          # Telegram client setup
│   ├── song_handler.go    # Song download handler
│   ├── song_queue.go      # Queue management
│   ├── queue_handler.go   # Queue status handler
│   ├── help_handler.go    # Help command
│   └── ...               # Other handlers
├── downloader/            # Download logic
│   ├── song_downloader_impl.go # Apple Music integration
│   ├── telegram_progress_reporter.go # Progress tracking
│   └── types.go          # Data structures
├── config/               # Configuration management
├── downloads/           # Downloaded files (auto-cleanup)
├── main.go             # Application entry point
├── go.mod              # Go dependencies
└── README.md          # This file
```

## Configuration Files

### `.env` File Format
```bash
# Telegram credentials
BOT_TOKEN=your_bot_token_here
API_ID=your_api_id_here
API_HASH=your_api_hash_here

# Optional settings
LOG_LEVEL=INFO
```

### Logging Levels
- `DEBUG`: Detailed debugging information
- `INFO`: General operational messages (default)
- `WARN`: Warning messages
- `ERROR`: Error messages only

## Development

### Prerequisites for Development
- Go 1.21+
- Git
- Text editor/IDE

### Building from Source
```bash
# Get dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build -o go-alac-bot .

# Run with verbose logging
LOG_LEVEL=DEBUG ./go-alac-bot
```

### Adding New Commands
1. Create handler in `bot/` directory
2. Implement `CommandHandler` interface
3. Register in `main.go` `registerCommandHandlers()` function

## Troubleshooting

### Common Issues

#### Bot doesn't respond
- ✅ Check bot token is correct
- ✅ Verify bot is not already running elsewhere
- ✅ Ensure `.env` file is in project root
- ✅ Check logs for error messages

#### Download failures
- ✅ Verify Apple Music URL format
- ✅ Check internet connectivity
- ✅ Look for rate limiting messages
- ✅ Try with different songs

#### Queue issues
- ✅ Restart bot to clear queue
- ✅ Check logs for processing errors
- ✅ Verify sufficient disk space
- ✅ Monitor memory usage

#### Permission errors
- ✅ Ensure write permissions to `downloads/` directory
- ✅ Check file system space
- ✅ Verify directory exists

### Debug Mode
Run with debug logging:
```bash
LOG_LEVEL=DEBUG go run .
```

### Log Files
Logs are output to stdout. To save logs:
```bash
./go-alac-bot 2>&1 | tee bot.log
```

## Security Notes

- ⚠️ Keep your `.env` file secure and never commit it to version control
- ⚠️ Bot tokens and API credentials should be treated as passwords
- ⚠️ Consider running in a containerized environment for isolation
- ⚠️ Monitor disk usage as downloaded files consume storage
- ⚠️ Implement rate limiting if deploying publicly

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Support

- 🐛 **Issues**: [GitHub Issues](https://github.com/yourusername/go-alac-bot/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/yourusername/go-alac-bot/discussions)
- 📧 **Email**: your-email@example.com

## Acknowledgments

- [gotgproto](https://github.com/celestix/gotgproto) - Telegram MTProto client
- [gotd](https://github.com/gotd/td) - Low-level Telegram client
- Apple Music for the music metadata API

---

**Made with ❤️ for music lovers**
