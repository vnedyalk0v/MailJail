GO ?= go
GOFLAGS ?=
CGO_ENABLED ?= 0
BINARY ?= mailjail
DIST_DIR ?= dist
TOOLS_BIN_DIR ?= $(CURDIR)/bin/tools
GOLANGCI_LINT_VERSION ?= v2.11.1
GOVULNCHECK_VERSION ?= latest
GOLANGCI_LINT ?= $(TOOLS_BIN_DIR)/golangci-lint
GOVULNCHECK ?= $(TOOLS_BIN_DIR)/govulncheck

.PHONY: tools install-hooks fmt fmt-check mod-check test race vet lint vuln shell-check build build-freebsd-amd64 build-freebsd-arm64 clean ci check

tools:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$(TOOLS_BIN_DIR) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	GOBIN=$(TOOLS_BIN_DIR) $(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

install-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-push

fmt:
	gofmt -w cmd internal

fmt-check:
	test -z "$$(gofmt -l cmd internal)"

mod-check:
	$(GO) mod tidy
	git diff --exit-code go.mod go.sum

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

lint: tools
	$(GOLANGCI_LINT) run --timeout=5m

vuln: tools
	$(GOVULNCHECK) ./...

shell-check:
	sh -n scripts/install.sh

build:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath -o $(DIST_DIR)/$(BINARY) ./cmd/mailjail

build-freebsd-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=freebsd GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-freebsd-amd64 ./cmd/mailjail

build-freebsd-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=freebsd GOARCH=arm64 $(GO) build $(GOFLAGS) -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-freebsd-arm64 ./cmd/mailjail

ci: fmt-check mod-check test race vet shell-check

check: ci lint vuln

clean:
	rm -rf $(DIST_DIR)
