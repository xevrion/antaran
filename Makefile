.PHONY: build run test fmt lint clean install

BINARY := bin/antaran
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/antaran

run:
	go run ./cmd/antaran $(ARGS)

test:
	go test -race ./...

fmt:
	gofmt -w .

lint:
	go vet ./...

clean:
	rm -rf bin/ dist/

install: build
	install -Dm755 $(BINARY) ~/.local/bin/antaran
