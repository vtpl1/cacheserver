all: build

install:
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install mvdan.cc/gofumpt@latest

lintall: lintverify lint

lint: clear lint1

# Build the application
build:
	@echo "Building..."
	@go clean
	@go build -buildvcs=true -ldflags "-s -w"

fumpt:
	@gofumpt -l -w .

# Run the application
run:
	@go run cmd/main.go

# Run the application
lintverify:
	@golangci-lint config verify

# Run the application
lint1:
	@golangci-lint run ./...

# Run the application
vul:
	@govulncheck -show verbose ./...

clear:
	@clear && printf '\e[3J'

.PHONY: all build run test clean lintverify lint lintall