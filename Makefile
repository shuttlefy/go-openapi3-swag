# ──────────────────────────────────────────────────────────────────────────────
# swag3 — Makefile
# Usage: make <target>
# Run `make help` to list all targets.
# ──────────────────────────────────────────────────────────────────────────────

MODULE       := github.com/shuttlefy/go-openapi3-swag
CLI          := ./cmd/swag3
EXAMPLE_DIR  := examples/bookstore
EXAMPLE_SPEC := $(EXAMPLE_DIR)/docs/openapi.json
PORT         ?= 9999

VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DIST_DIR     := dist
LDFLAGS      := -s -w -X main.version=$(VERSION)
PLATFORMS    := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.DEFAULT_GOAL := help

# ── Help ──────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n\n"} \
	     /^[a-zA-Z_\-]+:.*##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

# ── Dependencies ──────────────────────────────────────────────────────────────

.PHONY: tidy
tidy: ## Tidy and verify Go module dependencies
	go mod tidy
	go mod verify

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build the swag3 CLI binary → ./bin/swag3
	go build -o bin/swag3 $(CLI)

.PHONY: build-all
build-all: build ## Build all binaries
	go build ./...

# ── Spec generation ───────────────────────────────────────────────────────────

.PHONY: gen
gen: ## Generate openapi.json from examples/bookstore → examples/bookstore/docs/openapi.json
	go run $(CLI) -dirs $(EXAMPLE_DIR) -output $(EXAMPLE_SPEC)
	@echo "✓ spec written to $(EXAMPLE_SPEC)"

.PHONY: gen-yaml
gen-yaml: ## Generate openapi.yaml from examples/bookstore
	go run $(CLI) -dirs $(EXAMPLE_DIR) -output $(EXAMPLE_DIR)/docs/openapi.yaml
	@echo "✓ spec written to $(EXAMPLE_DIR)/docs/openapi.yaml"

.PHONY: gen-dirs
gen-dirs: ## Generate spec from custom directories  (usage: make gen-dirs DIRS=./myapp OUTPUT=./docs/openapi.json)
	go run $(CLI) -dirs $(DIRS) -output $(OUTPUT)
	@echo "✓ spec written to $(OUTPUT)"

# ── Run example ───────────────────────────────────────────────────────────────

.PHONY: example
example: ## Start the bookstore server (spec must already exist; run 'make gen' first if needed)
	@test -f $(EXAMPLE_SPEC) || (echo "✗ $(EXAMPLE_SPEC) not found, run 'make gen' first" && exit 1)
	@echo "→ starting bookstore on :$(PORT)"
	@echo "   Swagger UI : http://localhost:$(PORT)/docs"
	@echo "   Redoc      : http://localhost:$(PORT)/redoc"
	@echo "   Raw JSON   : http://localhost:$(PORT)/openapi.json"
	cd $(EXAMPLE_DIR) && go run .

.PHONY: example-fresh
example-fresh: gen ## Regenerate spec then start the bookstore server
	@echo "→ starting bookstore on :$(PORT)"
	@echo "   Swagger UI : http://localhost:$(PORT)/docs"
	@echo "   Redoc      : http://localhost:$(PORT)/redoc"
	@echo "   Raw JSON   : http://localhost:$(PORT)/openapi.json"
	cd $(EXAMPLE_DIR) && go run .

# ── Test ──────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Run all unit and integration tests
	go test ./... -count=1

.PHONY: test-verbose
test-verbose: ## Run all tests with verbose output
	go test ./... -count=1 -v

.PHONY: test-short
test-short: ## Run tests skipping long-running ones
	go test ./... -count=1 -short

.PHONY: test-pkg
test-pkg: ## Run tests for a specific package  (usage: make test-pkg PKG=./internal/builder)
	go test $(PKG) -count=1 -v

.PHONY: test-e2e
test-e2e: ## Run end-to-end CLI tests only
	go test ./cmd/swag3/ -count=1 -v

.PHONY: test-race
test-race: ## Run all tests with race detector enabled
	go test ./... -count=1 -race

.PHONY: cover
cover: ## Run tests and open HTML coverage report
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: cover-func
cover-func: ## Print per-function coverage summary
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out

# ── Code quality ──────────────────────────────────────────────────────────────

.PHONY: vet
vet: ## Run go vet on all packages
	go vet ./...

.PHONY: lint
lint: ## Run staticcheck linter (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

.PHONY: fmt
fmt: ## Format all Go source files with gofmt
	gofmt -w -s .

.PHONY: fmt-check
fmt-check: ## Check formatting without modifying files (CI-safe)
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		echo "The following files need formatting:"; echo "$$out"; exit 1; \
	fi
	@echo "✓ all files are formatted"

# ── Package / Release ─────────────────────────────────────────────────────────

.PHONY: dist
dist: ## Cross-compile swag3 for all platforms → ./dist/
	@mkdir -p $(DIST_DIR)
	@$(foreach platform,$(PLATFORMS), \
		$(eval OS   := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval OUT  := $(DIST_DIR)/swag3-$(VERSION)-$(OS)-$(ARCH)$(EXT)) \
		echo "→ building $(OS)/$(ARCH)"; \
		GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags "$(LDFLAGS)" -o $(OUT) $(CLI); \
	)
	@echo "✓ binaries written to $(DIST_DIR)/"

.PHONY: package
package: dist ## Build and archive all platform binaries → ./dist/*.tar.gz / .zip
	@cd $(DIST_DIR) && \
	for f in swag3-$(VERSION)-linux-* swag3-$(VERSION)-darwin-*; do \
		[ -f "$$f" ] || continue; \
		tar czf "$$f.tar.gz" "$$f" && rm -f "$$f"; \
		echo "  packed $$f.tar.gz"; \
	done && \
	for f in swag3-$(VERSION)-windows-*.exe; do \
		[ -f "$$f" ] || continue; \
		zip "$$f.zip" "$$f" && rm -f "$$f"; \
		echo "  packed $$f.zip"; \
	done
	@echo "✓ archives written to $(DIST_DIR)/"

.PHONY: release-dry
release-dry: ## Show what would be packaged (no files written)
	@echo "Version : $(VERSION)"
	@echo "Targets : $(PLATFORMS)"
	@echo "Output  : $(DIST_DIR)/"

# ── Clean ─────────────────────────────────────────────────────────────────────

.PHONY: clean
clean: ## Remove build artifacts and generated files
	rm -rf bin/ $(DIST_DIR)/
	rm -f coverage.out
	rm -f $(EXAMPLE_DIR)/docs/openapi.yaml

.PHONY: clean-all
clean-all: clean ## Remove all generated files including the example spec
	rm -f $(EXAMPLE_SPEC)
