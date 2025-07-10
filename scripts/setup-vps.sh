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

# Function to validate email format
validate_email() {
    if [[ $1 =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]]; then
        return 0
    else
        return 1
    fi
}

# Function to validate domain format
validate_domain() {
    if [[ $1 =~ ^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$ ]]; then
        return 0
    else
        return 1
    fi
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

# Create nginx directory and configuration
print_info "\nSetting up Nginx configuration..."

mkdir -p nginx

# Check if nginx.conf already exists
if [ -f "nginx/nginx.conf" ]; then
    print_warning "nginx/nginx.conf already exists."
    read -p "Do you want to overwrite it? (y/N): " overwrite_nginx
    if [[ ! "$overwrite_nginx" =~ ^[Yy]$ ]]; then
        print_info "Keeping existing nginx.conf"
    else
        create_nginx_config=true
    fi
else
    create_nginx_config=true
fi

if [ "${create_nginx_config:-false}" = true ]; then
    cat > nginx/nginx.conf << EOF
events {
    worker_connections 1024;
}

http {
    upstream app {
        server app:8080;
    }

    # HTTP server - redirects to HTTPS
    server {
        listen 80;
        server_name ${DOMAIN};

        # Let's Encrypt challenge location
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }

        # Redirect all other HTTP traffic to HTTPS
        location / {
            return 301 https://\$server_name\$request_uri;
        }
    }

    # HTTPS server
    server {
        listen 443 ssl http2;
        server_name ${DOMAIN};

        ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

        # SSL configuration
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;

        # Security headers
        add_header X-Frame-Options DENY;
        add_header X-Content-Type-Options nosniff;
        add_header X-XSS-Protection "1; mode=block";

        location / {
            proxy_pass http://app;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;

            # Timeouts
            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }
    }
}
EOF
    print_success "Nginx configuration created!"
fi

# Create temporary docker-compose file for Certbot
print_info "\nSetting up SSL certificate..."

cat > docker-compose.certbot.yml << EOF
version: '3.8'
services:
  nginx:
    image: "nginx:alpine"
    ports:
      - "80:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - certbot-etc:/etc/letsencrypt
      - certbot-var:/var/lib/letsencrypt
      - web-root:/var/www/certbot
volumes:
  certbot-etc:
  certbot-var:
  web-root:
EOF

# Check if certificates already exist
if docker volume ls --format "{{.Name}}" | grep -q "certbot-etc" && \
   docker run --rm -v certbot-etc:/etc/letsencrypt alpine ls /etc/letsencrypt/live/${DOMAIN} >/dev/null 2>&1; then
    print_warning "SSL certificates for $DOMAIN already exist. Skipping certificate generation..."
else
    print_info "Starting temporary Nginx for certificate generation..."

    # Start temporary Nginx
    if docker compose -f docker-compose.certbot.yml up -d; then
        print_success "Temporary Nginx started!"

        # Wait a moment for Nginx to be ready
        sleep 5

        # Generate SSL certificate
        print_info "Generating SSL certificate for $DOMAIN..."

        if docker compose -f docker-compose.prod.yml run --rm certbot certonly \
            --webroot \
            --webroot-path /var/www/certbot \
            --email "$EMAIL" \
            --agree-tos \
            --no-eff-email \
            -d "$DOMAIN"; then
            print_success "SSL certificate generated successfully!"
        else
            print_error "Failed to generate SSL certificate."
            docker compose -f docker-compose.certbot.yml down
            rm -f docker-compose.certbot.yml
            exit 1
        fi

        # Stop temporary Nginx
        print_info "Stopping temporary Nginx..."
        docker compose -f docker-compose.certbot.yml down
        print_success "Temporary Nginx stopped!"
    else
        print_error "Failed to start temporary Nginx."
        rm -f docker-compose.certbot.yml
        exit 1
    fi
fi

# Clean up temporary file
rm -f docker-compose.certbot.yml

# Create a cron job for certificate renewal
print_info "\nSetting up automatic certificate renewal..."

CRON_JOB="0 3 * * 0 cd $(pwd) && docker compose -f docker-compose.prod.yml run --rm certbot renew --quiet && docker service update --force crawler_stack_nginx"

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -F "$CRON_JOB" >/dev/null; then
    print_warning "Certificate renewal cron job already exists."
else
    # Add cron job
    (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
    print_success "Certificate renewal cron job added!"
fi

# Final summary
print_success "\n🎉 VPS Setup Complete!"
print_info "=============================="
print_info "✅ Docker Swarm initialized"
print_info "✅ Docker secrets created:"
print_info "   - postgres_password"
print_info "   - gcp_credentials"
print_info "✅ Nginx configuration created"
print_info "✅ SSL certificate generated for $DOMAIN"
print_info "✅ Certificate auto-renewal configured"
print_info ""
print_info "Next steps:"
print_info "1. Configure your GitHub repository secrets"
print_info "2. Push a tag to trigger deployment"
print_info "3. Monitor deployment in GitHub Actions"
print_info ""
print_info "You can verify the setup with:"
print_info "  docker secret ls"
print_info "  docker node ls"
print_info ""
print_warning "Make sure your domain $DOMAIN points to this server's IP address!"
