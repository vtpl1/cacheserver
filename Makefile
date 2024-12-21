all: build

lintall: lintverify lint

lint: clear lint1

# Build the application
build:
	@echo "Building..."
	@go clean
	@go build -buildvcs=true -ldflags "-s -w"

# Run the application
run:
	@go run cmd/main.go

# Run the application
lintverify:
	@/home/vscode/go/bin/golangci-lint config verify

# Run the application
lint1:
	@/home/vscode/go/bin/golangci-lint run ./...

# Run the application
vul:
	@/home/vscode/go/bin/govulncheck -show verbose ./...

clear:
	@clear && printf '\e[3J'

.PHONY: all build run test clean lintverify lint lintall