#!/bin/bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
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

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

validate_domain() {
    [[ "$1" =~ ^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]
}

validate_email() {
    [[ "$1" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]
}

# Function to prompt for input with validation
prompt_input() {
    local prompt="$1"
    local var_name="$2"
    local validation_func="${3:-}"
    local is_secret="${4:-false}"
    local value=""

    while true; do
        if [ "$is_secret" = "true" ]; then
            read -s -p "$prompt: " value
            echo
        else
            read -p "$prompt: " value
        fi

        if [ -n "$value" ]; then
            if [ -n "$validation_func" ]; then
                if $validation_func "$value"; then
                    break
                else
                    print_error "Invalid input. Please try again."
                fi
            else
                break
            fi
        else
            print_error "Input cannot be empty. Please try again."
        fi
    done

    eval "$var_name='$value'"
}

print_info "🚀 Starting VPS Setup for Crawler HTTP Service"
print_info "=============================================="

# Check prerequisites
print_info "Checking prerequisites..."

if ! command_exists docker; then
    print_error "Docker is not installed. Please install Docker first."
    print_info "Visit: https://docs.docker.com/engine/install/"
    exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
    print_error "Docker Compose is not available. Please make sure it's installed as a Docker plugin."
    print_info "Visit: https://docs.docker.com/compose/install/"
    exit 1
fi

print_success "Prerequisites check passed!"

# Check if Docker Swarm is already initialized
if docker info --format '{{.Swarm.LocalNodeState}}' 2>/dev/null | grep -q "active"; then
    print_warning "Docker Swarm is already initialized on this node."
    read -p "Do you want to continue with the setup? (y/N): " continue_setup
    if [[ ! "$continue_setup" =~ ^[Yy]$ ]]; then
        print_info "Setup cancelled."
        exit 0
    fi
else
    # Initialize Docker Swarm
    print_info "Initializing Docker Swarm..."
    if docker swarm init; then
        print_success "Docker Swarm initialized successfully!"
    else
        print_error "Failed to initialize Docker Swarm."
        exit 1
    fi
fi

# Collect configuration
print_info "\nCollecting configuration..."

prompt_input "Enter PostgreSQL password" POSTGRES_PASSWORD "" true
prompt_input "Enter path to GCP service account JSON file" GCP_KEY_PATH
prompt_input "Enter your domain name (e.g., crawlers.lexicon.id)" DOMAIN validate_domain
prompt_input "Enter your email address for SSL certificate" EMAIL validate_email
prompt_input "Enter NATS user" NATS_USER
prompt_input "Enter NATS password" NATS_PASSWORD "" true
prompt_input "Enter Redis password" REDIS_PASSWORD "" true

# Verify GCP key file exists
if [ ! -f "$GCP_KEY_PATH" ]; then
    print_error "GCP service account file not found at: $GCP_KEY_PATH"
    exit 1
fi

print_success "Configuration collected successfully!"

# Create Docker secrets
print_info "\nCreating Docker secrets..."

# Create PostgreSQL password secret
if docker secret ls --format "{{.Name}}" | grep -q "^postgres_password$"; then
    print_warning "postgres_password secret already exists. Skipping..."
else
    if echo "$POSTGRES_PASSWORD" | docker secret create postgres_password -; then
        print_success "PostgreSQL password secret created!"
    else
        print_error "Failed to create PostgreSQL password secret."
        exit 1
    fi
fi

# Create NATS user secret
if docker secret ls --format "{{.Name}}" | grep -q "^nats_user$"; then
    print_warning "nats_user secret already exists. Skipping..."
else
    if echo "$NATS_USER" | docker secret create nats_user -; then
        print_success "NATS user secret created!"
    else
        print_error "Failed to create NATS user secret."
        exit 1
    fi
fi

# Create NATS password secret
if docker secret ls --format "{{.Name}}" | grep -q "^nats_password$"; then
    print_warning "nats_password secret already exists. Skipping..."
else
    if echo "$NATS_PASSWORD" | docker secret create nats_password -; then
        print_success "NATS password secret created!"
    else
        print_error "Failed to create NATS password secret."
        exit 1
    fi
fi

# Create Redis password secret
if docker secret ls --format "{{.Name}}" | grep -q "^redis_password$"; then
    print_warning "redis_password secret already exists. Skipping..."
else
    if echo "$REDIS_PASSWORD" | docker secret create redis_password -; then
        print_success "Redis password secret created!"
    else
        print_error "Failed to create Redis password secret."
        exit 1
    fi
fi

# Create GCP credentials secret
if docker secret ls --format "{{.Name}}" | grep -q "^gcp_credentials$"; then
    print_warning "gcp_credentials secret already exists. Skipping..."
else
    if docker secret create gcp_credentials "$GCP_KEY_PATH"; then
        print_success "GCP credentials secret created!"
    else
        print_error "Failed to create GCP credentials secret."
        exit 1
    fi
fi

# Final summary
print_success "\n🎉 VPS Setup Complete!"
print_info "=============================="
print_info "✅ Docker Swarm initialized."
print_info "✅ The following Docker secrets have been created:"
print_info "   - postgres_password"
print_info "   - gcp_credentials"
print_info "   - nats_user"
print_info "   - nats_password"
print_info "   - redis_password"
print_info ""
print_info "Next steps:"
print_info "1. Configure your GitHub repository secrets. Make sure to set:"
print_info "   - TRAEFIK_HOST (your domain name: $DOMAIN)"
print_info "   - TRAEFIK_ACME_EMAIL (your email for Let's Encrypt: $EMAIL)"
print_info "2. Push a tag to trigger deployment."
print_info "3. Monitor deployment in GitHub Actions."
print_info ""
print_info "You can verify the setup with:"
print_info "  docker secret ls"
print_info "  docker node ls"
print_info ""
print_warning "Make sure your domain ($DOMAIN) points to this server's IP address!"
