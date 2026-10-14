GO ?= go
CORE ?= /usr/local/bin/sing-box

.PHONY: build test integration race vet audit release-dry-run clean
build:
	CGO_ENABLED=0 $(GO) build -buildvcs=false -trimpath -o bin/tunnelctl ./cmd/tunnelctl
test:
	$(GO) test ./...
integration: build
	TUNNELCTL_TEST_BIN="$(CURDIR)/bin/tunnelctl" TUNNELCTL_TEST_CORE="$(CORE)" $(GO) test -count=1 -v ./...
race:
	CGO_ENABLED=1 $(GO) build -race -o bin/tunnelctl-race ./cmd/tunnelctl
	CGO_ENABLED=1 TUNNELCTL_TEST_BIN="$(CURDIR)/bin/tunnelctl-race" TUNNELCTL_TEST_CORE="$(CORE)" $(GO) test -race -count=1 ./...
vet:
	$(GO) vet ./...
audit:
	$(GO) run ./cmd/audit
release-dry-run:
	$(GO) run ./cmd/release --tag v0.2.0-rc.1 --dry-run --out dist
clean:
	$(GO) clean
