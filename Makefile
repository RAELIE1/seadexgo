BINARY   := seadexgo
CMD      := ./cmd/seadexgo
VERSION  := $(shell grep 'version\s*=' seadex.go | head -1 | grep -o '"[^"]*"' | tr -d '"')
LDFLAGS  := -s -w

.PHONY: build install test lint clean

## build: compile the CLI binary into ./bin/
build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(CMD)

## install: install the CLI binary to GOPATH/bin (or $GOBIN)
install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

## test: run all tests
test:
	go test ./...

## lint: run go vet (add staticcheck/golangci-lint here if desired)
lint:
	go vet ./...

## clean: remove build artifacts
clean:
	rm -rf bin/

## help: print this message
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
