# VPS Setup Scripts

This directory contains automation scripts for setting up the crawler-http-service on a VPS.

## setup-vps.sh

An automated bash script that handles the complete VPS setup process for deploying the crawler-http-service with Docker Swarm.

### What it does

The script automates all the manual steps described in `DEPLOYMENT.md`:

1. **Prerequisites Check**: Verifies Docker and Docker Compose are installed
2. **Docker Swarm Setup**: Initializes Docker Swarm (if not already done)
3. **Secret Management**: Creates Docker secrets for:
   - PostgreSQL password
   - Google Cloud Platform credentials
4. **Nginx Configuration**: Sets up reverse proxy with SSL support
5. **SSL Certificate**: Generates Let's Encrypt SSL certificates
6. **Auto-renewal**: Configures automatic certificate renewal via cron

### Prerequisites

Before running the script, ensure you have:

- A VPS with a public IP address
- Docker and Docker Compose installed
- Your domain pointing to the VPS IP address
- A GCP service account JSON key file downloaded to your VPS
- SSH access to your VPS

### Usage

1. **Copy the script to your VPS**:
   ```bash
   scp scripts/setup-vps.sh user@your-vps-ip:/path/to/deployment/
   ```

2. **SSH into your VPS**:
   ```bash
   ssh user@your-vps-ip
   ```

3. **Navigate to your deployment directory**:
   ```bash
   cd /path/to/deployment/
   ```

4. **Run the setup script**:
   ```bash
   ./setup-vps.sh
   ```

5. **Follow the interactive prompts**:
   - PostgreSQL password (hidden input)
   - Path to GCP service account JSON file
   - Your domain name (e.g., crawlers.lexicon.id)
   - Email address for SSL certificate

### What the script prompts for

- **PostgreSQL Password**: A strong password for your PostgreSQL database
- **GCP Key Path**: Full path to your GCP service account JSON file on the VPS
- **Domain Name**: Your fully qualified domain name (validates format)
- **Email Address**: Your email for Let's Encrypt certificate notifications (validates format)

### Example run

```bash
$ ./setup-vps.sh

🚀 Starting VPS Setup for Crawler HTTP Service
==============================================
[INFO] Checking prerequisites...
[SUCCESS] Prerequisites check passed!
[INFO] Initializing Docker Swarm...
[SUCCESS] Docker Swarm initialized successfully!

[INFO] Collecting configuration...
Enter PostgreSQL password: [hidden]
Enter path to GCP service account JSON file: /home/user/gcp-key.json
Enter your domain name (e.g., crawlers.lexicon.id): crawlers.lexicon.id
Enter your email address for SSL certificate: admin@example.com
[SUCCESS] Configuration collected successfully!

[INFO] Creating Docker secrets...
[SUCCESS] PostgreSQL password secret created!
[SUCCESS] GCP credentials secret created!

[INFO] Setting up Nginx configuration...
[SUCCESS] Nginx configuration created!

[INFO] Setting up SSL certificate...
[INFO] Starting temporary Nginx for certificate generation...
[SUCCESS] Temporary Nginx started!
[INFO] Generating SSL certificate for crawlers.lexicon.id...
[SUCCESS] SSL certificate generated successfully!
[INFO] Stopping temporary Nginx...
[SUCCESS] Temporary Nginx stopped!

[INFO] Setting up automatic certificate renewal...
[SUCCESS] Certificate renewal cron job added!

🎉 VPS Setup Complete!
==============================
✅ Docker Swarm initialized
✅ Docker secrets created:
   - postgres_password
   - gcp_credentials
✅ Nginx configuration created
✅ SSL certificate generated for crawlers.lexicon.id
✅ Certificate auto-renewal configured

Next steps:
1. Configure your GitHub repository secrets
2. Push a tag to trigger deployment
3. Monitor deployment in GitHub Actions

You can verify the setup with:
  docker secret ls
  docker node ls

⚠️  Make sure your domain crawlers.lexicon.id points to this server's IP address!
```

### Safety Features

- **Validation**: Input validation for email and domain formats
- **Idempotent**: Safe to run multiple times (skips existing resources)
- **Error Handling**: Exits gracefully on errors with clear messages
- **Confirmation**: Asks before overwriting existing configurations
- **Prerequisites**: Checks for required tools before starting

### Troubleshooting

**Docker not found**: Install Docker first using the official installation guide
**Domain validation fails**: Ensure your domain follows standard format (no protocols, just domain)
**SSL generation fails**: Check that your domain points to the VPS IP and port 80 is accessible
**GCP file not found**: Verify the path to your service account JSON file is correct

### What happens next

After running this script successfully:

1. Configure your GitHub repository secrets as described in `DEPLOYMENT.md`
2. Push a git tag to trigger the deployment workflow
3. Monitor the deployment in your GitHub Actions tab

The VPS is now ready to receive deployments from your GitHub Actions workflow!
