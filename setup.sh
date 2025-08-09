#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        echo "Please install Go 1.21 or higher from https://golang.org/dl/"
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_success "Go version $GO_VERSION is installed"
}

# Check Go version
check_go_version() {
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.21"

    if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
        print_warning "Go version $GO_VERSION detected. Go 1.21+ is recommended"
    fi
}

# Install dependencies
install_deps() {
    print_status "Installing Go dependencies..."
    if go mod tidy; then
        print_success "Dependencies installed successfully"
    else
        print_error "Failed to install dependencies"
        exit 1
    fi
}

# Setup environment file
setup_env() {
    if [ ! -f .env ]; then
        print_status "Setting up environment file..."
        cp env.template .env
        print_success "Created .env file from template"
        print_warning "Please edit .env file with your Telegram credentials"
        echo ""
        echo "Required credentials:"
        echo "1. BOT_TOKEN - Get from @BotFather on Telegram"
        echo "2. API_ID - Get from https://my.telegram.org/apps"
        echo "3. API_HASH - Get from https://my.telegram.org/apps"
        echo ""
    else
        print_warning ".env file already exists, skipping..."
    fi
}

# Create downloads directory
create_dirs() {
    print_status "Creating necessary directories..."
    mkdir -p downloads
    print_success "Created downloads directory"
}

# Build the application
build_app() {
    print_status "Building application..."
    if go build -o go-alac-bot .; then
        print_success "Application built successfully"
    else
        print_error "Failed to build application"
        exit 1
    fi
}

# Test configuration
test_config() {
    if [ -f .env ]; then
        # Check if required env vars are set in .env
        if grep -q "BOT_TOKEN=123456789" .env || grep -q "API_ID=12345678" .env; then
            print_warning "Please update .env file with your actual Telegram credentials"
            return 1
        fi

        # Try to validate the config
        print_status "Testing configuration..."
        if timeout 5s ./go-alac-bot 2>&1 | grep -q "Configuration loaded"; then
            print_success "Configuration appears to be valid"
            return 0
        else
            print_warning "Could not validate configuration - please check your .env file"
            return 1
        fi
    else
        print_error ".env file not found"
        return 1
    fi
}

# Main setup function
main() {
    echo "🎵 Go ALAC Bot Setup Script"
    echo "=========================="
    echo ""

    # Check prerequisites
    check_go
    check_go_version

    # Setup project
    install_deps
    create_dirs
    setup_env
    build_app

    echo ""
    echo "🎉 Setup completed!"
    echo ""

    if test_config; then
        echo "✅ Ready to run with: ./go-alac-bot"
    else
        echo "⚠️  Please configure your .env file before running:"
        echo "   1. Edit .env file"
        echo "   2. Add your Telegram credentials"
        echo "   3. Run: ./go-alac-bot"
    fi

    echo ""
    echo "📚 Documentation: https://github.com/sayeed205/go-alac-bot"
    echo "🆘 Need help? Check the README.md file"
}

# Run main function
main "$@"
