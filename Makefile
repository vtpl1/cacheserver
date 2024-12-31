# Application name
APP_NAME := cacheserver

# Version and build info
VERSION := $(shell git describe --tags --always)
BUILD := $(shell date +%Y-%m-%dT%H:%M)

# Output directory
OUTPUT_DIR := bin

# Platforms to build for
PLATFORMS := \
    windows/amd64 \
    linux/386 \
    linux/amd64 \
    linux/arm/7 \
    linux/arm64 \
    darwin/amd64

# Build flags
LDFLAGS := -s -w -X main.GitCommit=$(VERSION) -X main.BuildTime=$(BUILD)

.PHONY: all clean build

# Default target: build for all platforms
all: clean build

# Clean up the output directory
clean:
	rm -rf $(OUTPUT_DIR)

# Build for all platforms
build:
	mkdir -p $(OUTPUT_DIR)
	$(foreach platform, $(PLATFORMS), $(call build_platform, $(platform)))

# Function to build for a specific platform
define build_platform
	$(eval OS := $(word 1, $(subst /, ,$1)))
	$(eval ARCH := $(word 2, $(subst /, ,$1)))
	$(eval ARM := $(word 3, $(subst /, ,$1)))
	$(eval OUTPUT := $(OUTPUT_DIR)/$(APP_NAME)_$(OS)_$(ARCH)$(if $(ARM),v$(ARM)))
	$(if $(filter windows, $(OS)), $(eval OUTPUT := $(OUTPUT).exe))
	@echo "Building for $(OS)/$(ARCH)$(if $(ARM),v$(ARM))..."
	GOOS=$(OS) GOARCH=$(ARCH) $(if $(ARM),GOARM=$(ARM)) go build -ldflags "$(LDFLAGS)" -o $(OUTPUT)
	@echo "Built: $(OUTPUT)"
endef
