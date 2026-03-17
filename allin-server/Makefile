GO := $(HOME)/sdk/go1.24.7/bin/go

.PHONY: run build tidy test

run:
	$(GO) run ./cmd/server

build:
	$(GO) build -o bin/allin-server ./cmd/server

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...
