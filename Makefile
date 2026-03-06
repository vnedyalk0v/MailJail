GO ?= go
GOFLAGS ?=
CGO_ENABLED ?= 0
BINARY ?= mailjail
DIST_DIR ?= dist

.PHONY: fmt test race vet lint vuln build build-freebsd-amd64 build-freebsd-arm64 clean ci

fmt:
	gofmt -w cmd internal

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

lint:
	golangci-lint run --timeout=5m

vuln:
	govulncheck ./...

build:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath -o $(DIST_DIR)/$(BINARY) ./cmd/mailjail

build-freebsd-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=freebsd GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-freebsd-amd64 ./cmd/mailjail

build-freebsd-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=freebsd GOARCH=arm64 $(GO) build $(GOFLAGS) -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-freebsd-arm64 ./cmd/mailjail

ci:
	$(MAKE) fmt
	test -z "$$(gofmt -l cmd internal)"
	$(GO) mod tidy
	$(GO) test ./...
	$(GO) test -race ./...
	$(GO) vet ./...

clean:
	rm -rf $(DIST_DIR)
