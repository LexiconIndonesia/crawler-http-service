.EXPORT_ALL_VARIABLES:
OUT_DIR := ./_output
BIN_DIR := ./bin

$(shell mkdir -p $(OUT_DIR) $(BIN_DIR))

# Main Test Targets (without docker)
.PHONY: test
test:
	go test -race -coverprofile=$(OUT_DIR)/coverage.out ./...

.PHONY: integration-test
integration-test:
	go test -race -tags=integration -coverprofile=$(OUT_DIR)/coverage.out ./...

.PHONY: install
install:
	go mod tidy && go mod vendor

.PHONY: build
build:
	go build -o $(BIN_DIR)/app

# Development targets
.PHONY: dev
dev:
	air

.PHONY: docker-dev
docker-dev:
	docker build -t go-app-dev -f dev.Dockerfile .
	docker run -p 2345:2345 -p 8080:8080 -v $(shell pwd):/app go-app-dev

.PHONY: docker-dev-build
docker-dev-build:
	docker build -t go-app-dev -f dev.Dockerfile .
