GO ?= go
BINARY ?= bin/toolbox

.PHONY: build test test-unit test-integration test-watch fmt tidy

build:
	mkdir -p bin
	$(GO) build -o $(BINARY) ./cmd/toolbox

test:
	$(GO) test ./...

test-unit:
	$(GO) test ./internal/... ./pkg/... ./tests/unit/...

test-integration:
	$(GO) test ./tests/integration/... -count=1
	$(GO) test ./internal/cli -run TestGoldenOutputs -count=1
	$(GO) test ./ -run TestScripts -count=1

test-watch:
	./scripts/test-watch.sh

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy
