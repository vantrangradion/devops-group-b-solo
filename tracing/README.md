# Collector and tracing: bonus 3 (#119)

The app sends its traces, metrics and logs to the OpenTelemetry Collector. Logs go on to Loki, see [logging/](../logging/README.md). Every `/api/work` request becomes a trace, which you can open in the same Grafana as the dashboard, and the RED metrics reach Prometheus through the same Collector:

```
                                    ┌─ traces ──OTLP/gRPC :4317──▶ tempo ◀──:3200── Grafana (Tempo datasource)
devops-app ──OTLP/HTTP :4318──▶ otel-collector
                         memory_limiter → batch
                                    └─ metrics ──OTLP/HTTP push :9090/api/v1/otlp──▶ Prometheus ──▶ Grafana dashboard
                                         latency histogram with trace_id exemplars ──▶ "View trace"
```

The app only knows the Collector's address (`OTEL_EXPORTER_OTLP_ENDPOINT` in [k8s/deployment.yaml](../k8s/base/deployment.yaml)). The backends are chosen in [collector-config.yaml](collector-config.yaml), so swapping Tempo for Jaeger, or Prometheus or Loki for a vendor, is a change to that file, not to the app. Everything runs in `monitoring`, next to Prometheus and Grafana.

Run commands from the repo root.

## Versions

| Component | Pin | Source |
| :--- | :--- | :--- |
| OpenTelemetry Collector | `otel/opentelemetry-collector-contrib:0.164.0` | Plain manifests here ([collector.yaml](collector.yaml)) |
| Tempo | `3.1.1`, Helm chart `grafana-community/tempo` `3.2.1` | [tempo-values.yaml](tempo-values.yaml) |
| App SDK | OpenTelemetry Go `v1.47.0`, `otelhttp` `v0.72.0` | [app/README.md](../app/README.md#traces) |

The single-binary Tempo chart moved from `grafana/tempo`, which is deprecated at `1.24.4` (Tempo 2.9), to `grafana-community/tempo`. It runs as one pod with local disk on an `emptyDir`, keeps traces for 24 h, and loses them when the pod restarts, which is fine for a local demo. The Collector uses plain manifests because its whole config fits in one short file worth reading.

Collector config notes, checked against 0.164.0:

- The OTLP exporters are named `otlp_grpc` and `otlp_http`. Older examples that use `otlp` as an exporter name no longer match.
- `batch` is still beta and still recommended. It sits after `memory_limiter`, which comes first so the Collector refuses data before its pod reaches the memory limit.
- The Collector's own metrics listen on `localhost` by default, so the config sets `0.0.0.0:8888` for Prometheus to scrape.
- The app's metrics are pushed by `otlp_http/prometheus` to Prometheus's OTLP receiver (`enableOTLPReceiver` in [monitoring/values.yaml](../monitoring/values.yaml)). Nothing scrapes the app or the Collector for them. Prometheus names them with `UnderscoreEscapingWithSuffixes`: `http.requests` becomes `http_requests_total` and `http.request.duration` (unit `s`) becomes `http_request_duration_seconds`. Exemplars arrive as `trace_id` and `span_id` labels.
- Prometheus stores pushed series with only `job` (from `service.name`) and `instance` (from `service.instance.id`, the pod name). The `transform/k8s-labels` processor copies the `k8s.namespace.name` and `k8s.pod.name` resource attributes into `namespace` and `pod` labels, so the dashboard queries are unchanged.

## Install

After the [k8s](../k8s/README.md) and [monitoring](../monitoring/README.md) installs:

```sh
helm repo add grafana-community https://grafana-community.github.io/helm-charts
helm upgrade --install tempo grafana-community/tempo --version 3.2.1 \
  -n monitoring -f tracing/tempo-values.yaml --wait --timeout 5m
kubectl apply -k tracing/
```

`kubectl apply -k tracing/` creates the Collector Deployment, its Service, a ConfigMap built from `collector-config.yaml` (its hash suffix rolls the pod when the config changes), and a ServiceMonitor for the Collector's own metrics. The app's metrics reach Prometheus only through this Collector, so without this folder the dashboard's RED panels stay empty. The Tempo datasource, the exemplar link, Prometheus exemplar storage and its OTLP receiver are set in [monitoring/values.yaml](../monitoring/values.yaml), so rerun the `helm upgrade` for `kps` from the monitoring README if it was installed before this folder existed.

## Check

```sh
kubectl get pods -n monitoring -l 'app.kubernetes.io/name in (tempo,otel-collector)'
for i in $(seq 20); do curl -s -o /dev/null "http://localhost:8080/api/work?ms=300&fail=0.5"; done
```

Then open Grafana (see the [monitoring README](../monitoring/README.md#open-grafana-and-prometheus)):

| Check | Where | Expected |
| :--- | :--- | :--- |
| Pods | `kubectl get pods` above | `tempo-0` and `otel-collector-…` are `1/1 Running` |
| A trace | Explore → Tempo → TraceQL `{ resource.service.name = "devops-app" }` | `GET /api/work` traces of about 300 ms. The span has `http.route`, `http.response.status_code` and `k8s.pod.name` |
| Errors only | TraceQL `{ resource.service.name = "devops-app" && status = error }` | Only the requests that returned `500` |
| Exemplars | Dashboard `devops-app` → Latency `/api/work` | Dots on the p50 line; hovering one shows **View trace**, which opens it in Tempo |
| Collector | Prometheus `otelcol_receiver_accepted_spans`, `otelcol_exporter_sent_spans` | Both above 0 and equal. `otelcol_exporter_send_failed_spans` is absent or 0 |

New spans take a few seconds to become searchable. A trace ID from an exemplar opens straight away.

Tempo logs `error calling scheduler ... no jobs found` every few seconds. That is the backend worker polling an idle compaction scheduler, and is harmless.

## Delete

```sh
kubectl delete -k tracing/
helm uninstall tempo -n monitoring
```
