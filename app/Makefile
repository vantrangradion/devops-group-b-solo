# Local checks that mirror CI. Run from app/.

# Must match the golangci-lint version pinned in CI.
GOLANGCI_LINT_VERSION := v2.14.0

IMAGE ?= devops-app
TAG   ?= dev

.PHONY: lint test build

lint:
	@v=$$(golangci-lint version --short 2>/dev/null); \
	if [ "$$v" != "$(GOLANGCI_LINT_VERSION)" ]; then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) required, found '$${v:-none}'"; exit 1; \
	fi
	golangci-lint run --config .golangci.yml

test:
	go test ./...

build:
	docker build --platform linux/amd64 -t $(IMAGE):$(TAG) .
