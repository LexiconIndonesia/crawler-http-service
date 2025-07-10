# Production Deployment on a VPS with Docker Swarm

This guide provides step-by-step instructions for deploying the crawler-http-service to a Docker Swarm on a Virtual Private Server (VPS) using a fully automated GitHub Actions workflow.

## Prerequisites

Before you begin, ensure you have the following:
- **A VPS** with a public IP address.
- **Docker** installed on your VPS: [Install Docker Engine](https://docs.docker.com/engine/install/)
- **A GitHub Repository** for your application code.

## 1. One-Time VPS Setup

You only need to perform these steps once on your VPS to prepare it for deployments.

### Automated Setup (Recommended)

For a fully automated setup, use the provided bash script:

```bash
# Copy the script to your VPS
scp scripts/setup-vps.sh user@your-vps-ip:/path/to/deployment/

# SSH into your VPS and run the script
ssh user@your-vps-ip
cd /path/to/deployment/
chmod +x setup-vps.sh
./setup-vps.sh
```

The script will guide you through the setup process with interactive prompts. See `scripts/README.md` for detailed usage instructions.

### Manual Setup

If you prefer to set up manually or need to troubleshoot, follow the steps below:

### a. Initialize Docker Swarm

You must initialize Docker Swarm on your VPS. This turns your Docker instance into a swarm manager, allowing you to deploy stacks.

```bash
docker swarm init
```

### b. Create Production Secrets

For production, sensitive data like passwords and credentials must be managed using **Docker Secrets**. Our application is configured to read these from the swarm.

1.  **PostgreSQL Password**:
    Replace `your_strong_postgres_password` with the actual password you want to use.
    ```bash
    echo "your_strong_postgres_password" | docker secret create postgres_password -
    ```

2.  **Google Cloud Storage (GCS) Credentials**:
    Create a secret from your downloaded GCP service account JSON key file. You must first copy your key file to the VPS.
    ```bash
    docker secret create gcp_credentials /path/on/vps/to/your-gcp-key.json
    ```

## 2. GitHub Repository Configuration

The repository contains a pre-configured GitHub Actions workflow at `.github/workflows/docker-build-push.yml`. This workflow automates testing, building, and deploying the application. For it to function correctly, you need to configure a few settings in your repository.

### a. Enable GitHub Packages

The workflow publishes the application's Docker image to GitHub Packages (which uses GitHub Container Registry). For private repositories, you must ensure that GitHub Packages is enabled. You can find this setting under your repository's **Settings > Packages**. It is enabled by default for public repositories.

### b. Configure Repository Secrets

The workflow requires several secrets to connect to your VPS and configure the application services. You must add these to your GitHub repository by navigating to **Settings > Secrets and variables > Actions**.

**Note**: The workflow uses a `GITHUB_TOKEN` to log in to the GitHub Container Registry. You do **not** need to create this secret, as it is automatically provided by GitHub Actions at the start of each workflow run.

#### How to Generate an SSH Key
To generate the `VPS_SSH_KEY`, you can run the following command on your local machine. This will create a private key (e.g., `id_ed25519`) and a public key (`id_ed25519.pub`). You will add the content of the **private key** file as a repository secret and copy the content of the **public key** to your VPS's `~/.ssh/authorized_keys` file.
```bash
ssh-keygen -t ed25519 -C "your_email@example.com"
```

#### Required Secrets
-   `VPS_HOST`: The hostname or IP address of your VPS.
-   `VPS_USERNAME`: The username for SSH login (e.g., `root`, `ubuntu`).
-   `VPS_SSH_KEY`: The **private** SSH key used to connect to your VPS.
-   `VPS_PORT`: The SSH port for your VPS (usually `22`).

-   `TRAEFIK_HOST`: Your domain name (e.g., `crawlers.lexicon.id`). Traefik will use this to route traffic and obtain SSL certificates.
-   `TRAEFIK_ACME_EMAIL`: The email address for Let's Encrypt to send certificate notifications.

-   `POSTGRES_DB_NAME`: The name for your PostgreSQL database (e.g., `crawlerdb`).
-   `POSTGRES_USERNAME`: The username for the PostgreSQL database.
-   `POSTGRES_PORT`: The port for PostgreSQL (e.g., `5432`).

-   `LISTEN_PORT`: The port the application will listen on (e.g., `8080`). This is not exposed publicly but is used for Nginx to route requests to the app.
-   `GCS_STORAGE_BUCKET`: The name of your Google Cloud Storage bucket.

-   `NATS_USER`: The username for NATS (if any).
-   `NATS_PASSWORD`: The password for NATS (if any).
-   `NATS_PORT`: The port for NATS client connections (e.g., `4222`).
-   `NATS_MONITORING_PORT`: The port for NATS HTTP monitoring (e.g., `8222`).
-   `REDIS_PORT`: The port for Redis (e.g., `6379`).

## 3. Triggering a Deployment

The workflow is configured to run automatically whenever you push a new Git tag that follows a semantic versioning pattern (e.g., `v1.0.0`, `v1.2.3`).

To deploy a new version of your application:
1.  Commit your changes to the `main` branch.
2.  Create a new tag.
    ```bash
    git tag v1.0.1
    ```
3.  Push the tag to GitHub. This will trigger the deployment workflow.
    ```bash
    git push origin v1.0.1
    ```

You can monitor the progress of your deployment in the "Actions" tab of your GitHub repository.

## 4. Verify the Deployment

Once the workflow is complete, you can check the status of your deployed services:

```bash
docker stack services crawler_stack
```

You should see all services (`nginx`, `app`, `postgres`, `nats`, `redis`, `certbot`) with `1/1` in the `REPLICAS` column. It might take a minute for all containers to download and start.

View the logs for a specific service:

```bash
docker service logs crawler_stack_app -f
```

## 5. Managing the Application

### Updating the Application

To update your application, simply push a new tag as described in the "Triggering a Deployment" section. The workflow will handle building the new image and updating the services.

### Stopping the Stack

To tear down the entire stack, SSH into your VPS and run:

```bash
docker stack rm crawler_stack
```
This will stop and remove all containers and networks. The named volumes and secrets on the VPS will persist.
