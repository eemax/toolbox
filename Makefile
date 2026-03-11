GO ?= go
BINARY ?= bin/toolbox
PREFIX ?= $(HOME)/.local
INSTALL_DIR ?= $(PREFIX)/bin
ZSH_COMPLETION_DIR ?= $(HOME)/.zsh/completions

.PHONY: build install install-zsh-completion test test-unit test-integration test-watch quality bench-smoke fmt tidy

build:
	mkdir -p bin
	$(GO) build -o $(BINARY) ./cmd/toolbox

install:
	mkdir -p $(INSTALL_DIR)
	$(GO) build -o $(INSTALL_DIR)/toolbox ./cmd/toolbox
	$(MAKE) install-zsh-completion

install-zsh-completion:
	mkdir -p $(ZSH_COMPLETION_DIR)
	$(GO) run ./cmd/toolbox completion zsh > $(ZSH_COMPLETION_DIR)/_toolbox

test:
	$(GO) test ./...

test-unit:
	$(GO) test ./internal/... ./pkg/... ./tests/unit/...

test-integration:
	$(GO) test ./tests/integration/... -count=1
	$(GO) test ./internal/cli -run TestGoldenOutputs -count=1

test-watch:
	./scripts/test-watch.sh

quality:
	$(GO) vet ./...
	./scripts/check-coverage.sh

bench-smoke:
	$(GO) test -run '^$$' -bench 'Benchmark(LoadCatalog|ExecuteDryRun|ResolveTemplate|ResolveEnvTemplates)$$' -benchmem -count=1 ./internal/manifest ./internal/runner

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy
