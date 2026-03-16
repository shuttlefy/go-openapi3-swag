# ──────────────────────────────────────────────────────────────────────────────
# swag3 — Makefile
# Usage: make <target>
# Run `make help` to list all targets.
# ──────────────────────────────────────────────────────────────────────────────

MODULE       := github.com/shuttlefy/go-openapi3-swag
CLI          := ./cmd/swag3
EXAMPLE      := ./examples/petstore
EXAMPLE_DIR  := examples/petstore
SPEC_DIR     := testdata/petstore
SPEC_OUT     := openapi.json
EXAMPLE_SPEC := examples/petstore/openapi.json
PORT         ?= 8088

VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DIST_DIR     := dist
LDFLAGS      := -s -w -X main.version=$(VERSION)
PLATFORMS    := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Metadata used when generating a spec from examples/petstore (which carries
# operation annotations on handler methods but no global swaggerInfo function).
EXAMPLE_TITLE   ?= Pet Store API
EXAMPLE_VERSION ?= 1.0.0

.DEFAULT_GOAL := help

# ── Help ──────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n\n"} \
	     /^[a-zA-Z_\-]+:.*##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
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

.PHONY: build-example
build-example: ## Build the petstore example binary → ./bin/petstore
	go build -o bin/petstore $(EXAMPLE)

.PHONY: build-all
build-all: build build-example ## Build all binaries

# ── Spec generation ───────────────────────────────────────────────────────────

.PHONY: gen
gen: ## Generate openapi.json from testdata/petstore (canonical annotation source)
	go run $(CLI) -dir $(SPEC_DIR) -output $(SPEC_OUT)
	@echo "✓ spec written to $(SPEC_OUT)"

.PHONY: gen-yaml
gen-yaml: ## Generate openapi.yaml from testdata/petstore annotations
	go run $(CLI) -dir $(SPEC_DIR) -format yaml -output openapi.yaml
	@echo "✓ spec written to openapi.yaml"

.PHONY: gen-example
gen-example: ## Generate openapi.json from examples/petstore handler annotations
	go run $(CLI) \
		-dir $(EXAMPLE_DIR) \
		-output $(EXAMPLE_SPEC)
	@echo "✓ spec written to $(EXAMPLE_SPEC)"

.PHONY: gen-update
gen-update: ## Regenerate and update the e2e golden file
	go test ./cmd/swag3/ -run TestE2E -update
	@echo "✓ golden file updated"

# ── Run ───────────────────────────────────────────────────────────────────────

.PHONY: run
run: gen ## Generate spec (testdata) then start the petstore example server
	PORT=$(PORT) go run $(EXAMPLE)

.PHONY: example
example: gen-example ## Generate spec from example code then start the server
	@echo "→ starting server on :$(PORT)  (Swagger UI: http://localhost:$(PORT)/docs)"
	cd $(EXAMPLE_DIR) && PORT=$(PORT) go run .

.PHONY: run-example
run-example: ## Start the petstore server using the existing spec (no regen)
	PORT=$(PORT) go run $(EXAMPLE)

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
	rm -f openapi.yaml
	rm -f $(EXAMPLE_SPEC)

.PHONY: clean-all
clean-all: clean ## Remove all generated files including openapi.json
	rm -f $(SPEC_OUT)
