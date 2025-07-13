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

3.  **NATS Credentials**:
    Replace `your_nats_user` and `your_strong_nats_password` with your desired credentials.
    ```bash
    echo "your_nats_user" | docker secret create nats_user -
    echo "your_strong_nats_password" | docker secret create nats_password -
    ```

4.  **Redis Password**:
    Replace `your_strong_redis_password` with the password you want to use.
    ```bash
    echo "your_strong_redis_password" | docker secret create redis_password -
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

-   `LISTEN_PORT`: The port the application will listen on (e.g., `8080`). This is not exposed publicly but is used by Traefik to route requests to the app.
-   `GCS_STORAGE_BUCKET`: The name of your Google Cloud Storage bucket.

-   `NATS_PORT`: The port for NATS client connections (e.g., `4222`).
-   `NATS_MONITORING_PORT`: The port for NATS HTTP monitoring (e.g., `8222`).
-   `REDIS_PORT`: The port for Redis (e.g., `6379`).

## 3. Publishing a New Version

The workflow is configured to automatically build and publish a new Docker image whenever you push a Git tag that follows a semantic versioning pattern (e.g., `v1.0.0`, `v1.2.3`).

To publish a new version of your application:
1.  Commit your changes to the `main` branch.
2.  Create a new tag.
    ```bash
    git tag v1.0.1
    ```
3.  Push the tag to GitHub. This will trigger the build-and-publish workflow.
    ```bash
    git push origin v1.0.1
    ```

You can monitor the progress of your build in the "Actions" tab of your GitHub repository.

## 4. Manual Deployment to VPS

After a new image has been successfully published to the GitHub Container Registry, you can deploy it to your server by following these steps.

### a. Connect to Your VPS
Connect to your server using SSH.
```bash
ssh <your_username>@<your_vps_host>
```

### b. Prepare the Compose File
The deployment process requires the `docker-compose.prod.yml` file, which is included in this repository. For a manual deployment, ensure this file is present on the VPS.

You can either clone the entire repository to your VPS or copy the file securely.

**Option 1: Clone the Repository (Recommended)**
Cloning the repository is straightforward and ensures you have all necessary files.

```bash
# Clone your repository
git clone https://github.com/LexiconIndonesia/crawler-http-service.git

# Navigate into the project directory
cd your-repo
```
From this point, run all subsequent commands from within this directory.

**Option 2: Copy the File with `scp`**
If you prefer not to clone the repository on your server, you can copy the file from your local machine.

```bash
# Run this command from your local machine
scp docker-compose.prod.yml <your_username>@<your_vps_host>:/path/to/deployment/directory
```

### c. Set the Image Tag
Before deploying, set the `IMAGE_TAG` environment variable to the version you want to deploy. This ensures that the subsequent commands pull and deploy the correct image.

```bash
export IMAGE_TAG=<your_new_version_tag>
```
**Note**: Replace `<your_new_version_tag>` with the actual version you are deploying (e.g., `v1.0.1`).

### d. Log in to the GitHub Container Registry
To pull the private image, you must log in to `ghcr.io` on your server. It is recommended to use a [Personal Access Token (PAT)](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) with `read:packages` scope as your password.

Run the following command. When prompted for a password, paste your PAT. This is more secure because it prevents the token from being saved in your shell's history.
```bash
docker login ghcr.io -u YOUR_GITHUB_USERNAME
```

### e. Deploy the Stack
With the `IMAGE_TAG` variable set, pull the new image and redeploy the stack. Docker Swarm will perform a rolling update with zero downtime.

```bash
# Pull the new image specified by the IMAGE_TAG environment variable
docker compose -f docker-compose.prod.yml pull app

# Redeploy the stack, passing the IMAGE_TAG to the compose file.
# The --with-registry-auth flag is crucial for allowing swarm nodes to pull private images.
docker stack deploy --compose-file docker-compose.prod.yml --with-registry-auth crawler_stack
```

## 5. Verify the Deployment

Once the deployment command is complete, you can check the status of your services:

```bash
docker stack services crawler_stack
```

You should see all services (`traefik`, `app`, `postgres`, `nats`, `redis`) with `1/1` in the `REPLICAS` column. It might take a minute for all containers to download and restart.

View the logs for a specific service:

```bash
docker service logs crawler_stack_app -f
```

## 6. Managing the Application

### Updating the Application

To update your application, publish a new version by pushing a tag, then follow the manual deployment steps.

### Stopping the Stack

To tear down the entire stack, SSH into your VPS and run:

```bash
docker stack rm crawler_stack
```
This will stop and remove all containers and networks. The named volumes and secrets on the VPS will persist.
