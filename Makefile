BINARY   := bmouse
MODULE   := github.com/jashort/bmouse
GOFLAGS  := -trimpath

.PHONY: all build release test lint vet clean

default: help

all: lint test release

## build: compile the binary
build:
	go build $(GOFLAGS) -o $(BINARY) .

## release: compile the binary with stripped debug info
release:
	go build $(GOFLAGS) -ldflags "-s -w" -o $(BINARY) .

## test: run all tests
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (install: https://golangci-lint.run/usage/install/)
lint: vet
	golangci-lint run ./...

## clean: remove the compiled binary
clean:
	rm -f $(BINARY)

## help: list available targets
help:
	@grep -E '^##' Makefile | sed 's/## /  /'

