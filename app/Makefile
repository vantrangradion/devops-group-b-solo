# Local checks that mirror CI. Run from app/.

# Must match the golangci-lint version pinned in CI.
GOLANGCI_LINT_VERSION := v2.14.0

IMAGE ?= devops-app
TAG   ?= dev

# Host port for the smoke container; 18080 avoids clashing with a local run.
SMOKE_PORT ?= 18080
SMOKE_NAME := devops-app-smoke

.PHONY: lint test build smoke

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

# Starts the built image and checks it serves /health and /metrics. Distroless
# has no shell or curl, so the checks run from the host.
smoke:
	@docker rm -f $(SMOKE_NAME) >/dev/null 2>&1 || true
	docker run -d --name $(SMOKE_NAME) -p $(SMOKE_PORT):8080 $(IMAGE):$(TAG)
	@status=0; \
	for i in $$(seq 1 20); do \
		curl -fs -o /dev/null http://localhost:$(SMOKE_PORT)/health && break; \
		sleep 0.5; \
	done; \
	curl -fsS -o /dev/null http://localhost:$(SMOKE_PORT)/health || { echo "smoke: /health failed"; status=1; }; \
	curl -fsS http://localhost:$(SMOKE_PORT)/metrics | grep -q '^# TYPE' || { echo "smoke: /metrics failed"; status=1; }; \
	if [ $$status -ne 0 ]; then docker logs $(SMOKE_NAME); fi; \
	docker rm -f $(SMOKE_NAME) >/dev/null; \
	if [ $$status -eq 0 ]; then echo "smoke: ok"; fi; \
	exit $$status
