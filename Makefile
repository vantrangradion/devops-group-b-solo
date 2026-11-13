# One command each to bring the shared system up, check it and tear it down
# (gradionhq #141). Run from the repo root. Needs only k3d, kubectl,
# helm, curl and make; `make tunnel` also needs cloudflared.
#
# Every command names the k3d context, so the targets never touch another
# cluster, whatever kubectl's current context is.

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# Pins: keep in step with the folder READMEs.
CLUSTER        ?= devops-solo
K3D_AGENTS     := 1
APP_PORT       := 8080
KPS_VERSION    := 92.10.0
TEMPO_VERSION  := 3.2.1
LOKI_VERSION   := 20.2.2
# Argo CD's version is pinned in gitops/install/kustomization.yaml.

CTX     := k3d-$(CLUSTER)
KUBECTL := kubectl --context $(CTX)
HELM    := helm --kube-context $(CTX)

# Argo CD deploys the app when gitops/ exists; USE_ARGOCD=0 forces kubectl.
ifeq ($(wildcard gitops/application.yaml),)
USE_ARGOCD ?= 0
else
USE_ARGOCD ?= 1
endif

.PHONY: help up cluster repos app monitoring tracing logging load status check down tunnel ui ui-status ui-stop

help:
	@echo "make up       create or update the cluster and install everything (safe to rerun)"
	@echo "make status   pods, app URL and Argo CD sync state"
	@echo "make ui       Grafana, Argo CD and Prometheus on localhost, kept up in the background"
	@echo "make ui-stop  stop those forwards (make ui-status shows them)"
	@echo "make check    one request, Collector counts, one trace in Tempo; non-zero on failure"
	@echo "make down     delete the k3d cluster (asks first; FORCE=1 skips the prompt)"
	@echo "make tunnel   public quick tunnel to the app (cloudflared)"
	@echo "USE_ARGOCD=$(USE_ARGOCD)  (1: Argo CD deploys the app from gitops/, 0: kubectl apply -k k8s/base/)"

# README order: cluster, app, kube-prometheus-stack, dashboard, Tempo, Loki,
# Collector. The app runs before the Collector exists; its OTLP exports retry.
up: cluster repos app monitoring tracing logging load
	@$(KUBECTL) rollout status deploy/devops-app -n devops-app --timeout 3m
	@$(MAKE) --no-print-directory status
	@echo
	@KUBE_CONTEXT=$(CTX) scripts/ui.sh start

cluster:
	@if k3d cluster get $(CLUSTER) >/dev/null 2>&1; then \
	  k3d cluster start $(CLUSTER) >/dev/null; echo "cluster $(CLUSTER): exists, started"; \
	else \
	  k3d cluster create $(CLUSTER) --agents $(K3D_AGENTS) -p "$(APP_PORT):8080@loadbalancer" --wait; \
	fi
	@$(KUBECTL) wait --for=condition=Ready nodes --all --timeout 2m >/dev/null

repos:
	@helm repo add prometheus-community https://prometheus-community.github.io/helm-charts --force-update >/dev/null
	@helm repo add grafana-community https://grafana-community.github.io/helm-charts --force-update >/dev/null
	@helm repo update prometheus-community grafana-community >/dev/null
	@echo "helm repos: prometheus-community, grafana-community"

app:
ifeq ($(USE_ARGOCD),1)
	$(KUBECTL) apply --server-side -k gitops/install
	$(KUBECTL) -n argocd rollout status deploy/argocd-server --timeout 5m
	$(KUBECTL) -n argocd rollout status deploy/argocd-repo-server --timeout 5m
	$(KUBECTL) -n argocd rollout status statefulset/argocd-application-controller --timeout 5m
	$(KUBECTL) apply -f gitops/project.yaml -f gitops/application.yaml
	$(KUBECTL) -n argocd wait application/devops-app --for=jsonpath='{.status.sync.status}'=Synced --timeout 5m
	$(KUBECTL) -n argocd wait application/devops-app --for=jsonpath='{.status.health.status}'=Healthy --timeout 5m
else
	$(KUBECTL) apply -k k8s/
endif

monitoring:
	$(HELM) upgrade --install kps prometheus-community/kube-prometheus-stack --version $(KPS_VERSION) \
	  -n monitoring --create-namespace -f monitoring/values.yaml --wait --timeout 8m --hide-notes
	$(KUBECTL) apply -k monitoring/

# tracing/ holds the Collector, which also exports to Loki, so Loki goes first.
tracing: logging
	$(HELM) upgrade --install tempo grafana-community/tempo --version $(TEMPO_VERSION) \
	  -n monitoring -f tracing/tempo-values.yaml --wait --timeout 5m --hide-notes
	$(KUBECTL) apply -k tracing/
	$(KUBECTL) -n monitoring rollout status deploy/otel-collector --timeout 3m

logging:
	$(HELM) upgrade --install loki grafana-community/loki --version $(LOKI_VERSION) \
	  -n monitoring -f logging/loki-values.yaml --wait --timeout 5m --hide-notes

# In-cluster background load (#140), once that folder exists.
load:
	@if [ -d load-tests/k8s ]; then $(KUBECTL) apply -k load-tests/k8s/; else echo "load-tests/k8s: not present, skipped"; fi

status:
	@for ns in devops-app monitoring argocd; do \
	  if $(KUBECTL) get ns $$ns >/dev/null 2>&1; then echo "--- $$ns"; $(KUBECTL) get pods -n $$ns; fi; \
	done
	@echo
	@echo "App:        http://localhost:$(APP_PORT)/api/work"
	@if $(KUBECTL) get ns argocd >/dev/null 2>&1; then \
	  echo "Argo CD app: $$($(KUBECTL) -n argocd get application devops-app -o jsonpath='{.status.sync.status}/{.status.health.status}' 2>/dev/null || echo 'no Application')"; \
	fi
	@echo "Web UIs:    make ui (Grafana, Argo CD, Prometheus on localhost)"

# Reads the Collector and Tempo through the API server proxy, so no
# port-forward is needed.
COLLECTOR_METRICS := /api/v1/namespaces/monitoring/services/otel-collector:metrics/proxy/metrics
TEMPO_SEARCH      := /api/v1/namespaces/monitoring/services/tempo:3200/proxy/api/search?limit=1&q=%7Bresource.service.name%3D%22devops-app%22%7D

check:
	@body=$$(curl -fsS --max-time 5 "http://localhost:$(APP_PORT)/api/work?ms=10") \
	  || { echo "FAIL app: no answer on localhost:$(APP_PORT)"; exit 1; }; \
	echo "$$body" | grep -q '"status":200' || { echo "FAIL app: $$body"; exit 1; }; \
	echo "ok   app: $$body"
	@metrics=$$($(KUBECTL) get --raw '$(COLLECTOR_METRICS)' 2>/dev/null) \
	  || { echo "FAIL collector: metrics endpoint unreachable (is otel-collector running?)"; exit 1; }; \
	sent=$$(echo "$$metrics" | grep '^otelcol_exporter_sent_' || true); \
	[ -n "$$sent" ] || { echo "FAIL collector: no otelcol_exporter_sent_* series yet"; exit 1; }; \
	echo "$$sent" | awk '{ match($$0, /exporter="[^"]*"/); split($$1, n, "{"); printf "ok   collector: %-45s %-28s %s\n", n[1], substr($$0, RSTART+10, RLENGTH-11), $$2 }'; \
	failed=$$(echo "$$metrics" | grep '^otelcol_exporter_send_failed_' | awk '$$2 > 0' || true); \
	[ -z "$$failed" ] || { echo "FAIL collector: send failures"; echo "$$failed"; exit 1; }
	@for i in $$(seq 15); do \
	  trace=$$($(KUBECTL) get --raw '$(TEMPO_SEARCH)' 2>/dev/null | grep -o '"traceID":"[0-9a-f]*"' | head -1 | cut -d'"' -f4 || true); \
	  [ -n "$$trace" ] && { echo "ok   tempo: devops-app trace $$trace"; exit 0; }; sleep 2; \
	done; echo "FAIL tempo: no devops-app trace found in 30 s"; exit 1

down:
	@if [ "$${FORCE:-0}" != 1 ]; then \
	  read -r -p "Delete k3d cluster $(CLUSTER) and everything in it? [y/N] " ans; \
	  [ "$$ans" = y ] || [ "$$ans" = Y ] || { echo "kept $(CLUSTER)"; exit 0; }; \
	fi; \
	k3d cluster delete $(CLUSTER)

# Grafana, Argo CD and Prometheus live in the cluster; scripts/ui.sh keeps a
# port-forward to each running in the background and restarts it if it drops.
ui:
	@KUBE_CONTEXT=$(CTX) scripts/ui.sh start

ui-status:
	@KUBE_CONTEXT=$(CTX) scripts/ui.sh status

ui-stop:
	@KUBE_CONTEXT=$(CTX) scripts/ui.sh stop

# Public URL for the app; rate-limited, so never point k6 at it.
tunnel:
	cloudflared tunnel --url http://localhost:$(APP_PORT)
