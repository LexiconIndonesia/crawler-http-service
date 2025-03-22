FROM golang:latest
# Environment variables for development
ENV PROJECT_DIR=/app \
  GO111MODULE=on \
  CGO_ENABLED=0

# Basic setup of the container
RUN mkdir /app
COPY . /app
WORKDIR /app

# Install Air for hot-reloading with debug support
RUN go install github.com/air-verse/air@latest

# Install Delve debugger
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Copy Air configuration
COPY .air.toml .

# Run Air with the configuration we set up
ENTRYPOINT ["air"]
