# Go HTTP Service Template

A modern, production-ready Go HTTP service template following best practices.

## Features

- **Clean Architecture**: Organized with dependency injection pattern
- **Robust Error Handling**: Contextual errors with proper propagation
- **Graceful Shutdown**: Proper signal handling for clean server shutdown
- **Structured Logging**: Using zerolog for performant structured logging
- **Database Integration**: PostgreSQL integration with connection pooling
- **Input Validation**: Request validation using go-playground/validator
- **Middleware Support**: Configurable middleware chain using Chi
- **Testing Support**: Example tests with mocking capabilities
- **Configuration Management**: Environment-based configuration
- **Messaging**: NATS and JetStream integration for pub/sub messaging
- **API Documentation**: Swagger/OpenAPI documentation with swaggo/swag

## Getting Started

### Prerequisites

- Go 1.24.1+
- PostgreSQL 14+
- SQLC (Install: `brew install sqlc`)
- NATS Server (optional, for messaging features)

### Installation

1. Clone the repository
2. Copy `.env.example` to `.env`
3. Write SQL queries in `query.sql`
4. Run `sqlc generate`, the generated files will be in `repository` folder
5. Run `make install`
6. (Optional) Install NATS server:

   ```sh
   # Using Docker
   docker run -p 4222:4222 -p 8222:8222 nats -js

   # Or using Homebrew on macOS
   brew install nats-server
   nats-server -js
   ```

### Development

For local development with hot reloading:

```sh
./run-dev.sh
```

### Usage

1. Run `make build`
2. Execute `bin/app`
3. Visit `http://localhost:8080/swagger/index.html` to access the API documentation

## Project Structure

```md
.
├── common/            # Common utilities and models
│   ├── db/            # Database access layer
│   ├── messaging/     # NATS/JetStream messaging layer
│   ├── models/        # Domain models
│   └── utils/         # Utility functions
├── docs/              # Swagger documentation
├── middlewares/       # HTTP middleware
├── migrations/        # Database migrations
├── module/            # Feature modules (resources)
├── repository/        # Database repositories (generated)
└── main.go            # Application entry point
```

## API Documentation

The API documentation is available via Swagger UI at `http://localhost:8080/swagger/index.html` when the service is running.

### Generating Documentation

The API documentation is generated using [swaggo/swag](https://github.com/swaggo/swag). To update the documentation after making changes to the API:

```bash
# Install swag CLI if you don't have it already
go install github.com/swaggo/swag/cmd/swag@latest

# Generate/update the documentation
swag init
```

## Messaging with NATS/JetStream

This template includes a fully configured NATS and JetStream client integration for pub/sub messaging. Key features include:

- Dependency-injected NATS client
- JetStream support for persistent messaging
- Automatic stream creation and management
- Asynchronous message publishing with acknowledgements
- Message subscription capabilities
- Example WebSocket endpoint for real-time messaging
- Comprehensive test mocks for the messaging layer

### Using the Messaging System

The NATS client is available via dependency injection in all modules. To publish a message:

```go
message := []byte(`{"data": "your message here"}`)
if err := module.NatsClient.Publish("some.subject", message); err != nil {
    // Handle error
}
```

For persistent messaging with JetStream:

```go
// Publishing with acknowledgement
ack, err := module.NatsClient.PublishAsync("some.subject", message)
if err != nil {
    // Handle error
}

// Wait for acknowledgement
select {
case <-ack.Ok():
    // Message was successfully stored
case err := <-ack.Err():
    // Handle error
}
```

## Best Practices

This template follows Go best practices including:

- Explicit error handling with context
- Dependency injection instead of global state
- Proper context propagation
- Structured logging
- Graceful shutdowns
- Clean separation of concerns

---

© 2024
