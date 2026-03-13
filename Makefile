VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_SHA  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  := -X main.version=$(VERSION) -X main.gitSHA=$(GIT_SHA)
BUILD    := go build -ldflags "$(LDFLAGS)"
BINARY   := bin/gobird

.PHONY: build test test-race lint vet fmt clean coverage ci

build:
	@mkdir -p bin
	$(BUILD) -o $(BINARY) ./cmd/gobird

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin coverage.out coverage.html

ci: vet test test-race lint build
