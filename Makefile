.DEFAULT_GOAL := build

BINARY_NAME ?= bondcalc
CMD_DIR ?= cmd/bondcalc
BUILD_DIR=releases
VERSION?=$(shell git describe --tags 2>/dev/null || echo "v0.0")
PLATFORMS=linux-amd64 linux-arm64 linux-386 darwin-amd64 darwin-arm64 windows-386 windows-amd64
EXT = $(if $(findstring windows,$(1)),.exe)

GREEN=\033[0;32m
BLUE=\033[0;34m
NC=\033[0m

.PHONY: build
build:
	@echo "Building application..."
	@go build -o $(BINARY_NAME) $(CMD_DIR)/*.go

.PHONY: build-all
build-all: $(PLATFORMS)
	@echo "$(GREEN)All builds completed!$(NC)"

$(PLATFORMS):
	@echo "$(BLUE)Building for $@...$(NC)"
	@mkdir -p $(BUILD_DIR)

	$(eval OS := $(word 1,$(subst -, ,$@)))
	$(eval ARCH := $(word 2,$(subst -, ,$@)))
	$(eval EXTENSION := $(call EXT,$(OS)))

	GOOS=$(OS) GOARCH=$(ARCH) go build -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION).$(OS)-$(ARCH)$(EXTENSION) $(CMD_DIR)/*.go


.PHONY: linux
linux: linux-amd64 linux-arm64 linux-386

.PHONY: darwin
darwin: darwin-amd64 darwin-arm64

.PHONY: windows
windows: windows-386 windows-amd64

.PHONY: release
release: build-all
	@echo "$(GREEN)Creating release archives...$(NC)"
	@for file in $(BUILD_DIR)/*; do \
		tar -czf $$file.tar.gz $$file config/config.yml; \
	done

.PHONY: clean
clean:
	@echo "$(GREEN)Cleaning...$(NC)"
	rm -rf $(BUILD_DIR) $(BINARY_NAME)
