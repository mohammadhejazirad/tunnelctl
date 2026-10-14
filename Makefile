GO ?= go
CORE ?= /usr/local/bin/sing-box

.PHONY: build test integration vet clean
build:
	CGO_ENABLED=0 $(GO) build -buildvcs=false -trimpath -o bin/tunnelctl ./cmd/tunnelctl
test:
	$(GO) test ./...
integration: build
	TUNNELCTL_TEST_BIN="$(CURDIR)/bin/tunnelctl" TUNNELCTL_TEST_CORE="$(CORE)" $(GO) test -count=1 -v ./...
vet:
	$(GO) vet ./...
clean:
	$(GO) clean
