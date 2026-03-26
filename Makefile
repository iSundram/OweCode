BINARY     := owecode
MODULE     := github.com/iSundram/OweCode
VERSION    := $(shell cat VERSION 2>/dev/null || echo "0.0.0")
VERSION_INSTALLER := $(shell cat VERSION_INSTALLER 2>/dev/null || echo "0.0.0")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS_OWECODE := -s -w \
  -X '$(MODULE)/internal/version.Version=$(VERSION)' \
  -X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
  -X '$(MODULE)/internal/version.BuildDate=$(BUILD_DATE)'

LDFLAGS_INSTALLER := -s -w \
  -X '$(MODULE)/internal/installer.Version=$(VERSION_INSTALLER)' \
  -X '$(MODULE)/internal/installer.Commit=$(COMMIT)' \
  -X '$(MODULE)/internal/installer.BuildDate=$(BUILD_DATE)'

.PHONY: all build clean test lint fmt tidy install

all: build build-installer

build:
	go build -ldflags "$(LDFLAGS_OWECODE)" -o bin/$(BINARY) ./cmd/owecode

build-installer:
	go build -ldflags "$(LDFLAGS_INSTALLER)" -o bin/installer ./cmd/installer

install:
	go install -ldflags "$(LDFLAGS_OWECODE)" ./cmd/owecode

clean:
	rm -rf bin/

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

tidy:
	go mod tidy

.PHONY: release
release:
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS_OWECODE)" -o bin/$(BINARY)-linux-amd64   ./cmd/owecode
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS_OWECODE)" -o bin/$(BINARY)-darwin-amd64  ./cmd/owecode
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS_OWECODE)" -o bin/$(BINARY)-darwin-arm64  ./cmd/owecode
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS_OWECODE)" -o bin/$(BINARY)-windows-amd64.exe ./cmd/owecode
