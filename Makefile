# Makefile for ingest project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Binary name
BINARY_NAME=ingest

# Version information
VERSION := $(shell git describe --tags --always)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
LDFLAGS := -ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Main package path
MAIN_PACKAGE=.

.PHONY: all build clean test deps

all: clean build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

lint:
	gofmt -w -s .
	golangci-lint run
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...

test:
	$(GOTEST) -v ./...

deps:
	$(GOGET) ./...

# Run the application
run: build
	./$(BINARY_NAME)

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)

# Install the binary
install: build
	mv $(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# Uninstall the binary
uninstall:
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# output the version information
version:
	@echo $(VERSION)
