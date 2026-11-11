# Logging: Loki

The app's log records go through the same Collector as its traces and metrics, into Loki. Each record written inside a request carries that request's trace ID, so Grafana can jump from a trace to its logs and back:

```
devops-app ──OTLP/HTTP :4318──▶ otel-collector ──OTLP/HTTP :3100/otlp──▶ loki ◀──:3100── Grafana (Loki datasource)
 slog → otelslog                memory_limiter → batch                     trace_id ⇄ Tempo
```

The app logs through Go's `log/slog`. The default handler writes every record to stderr (for `kubectl logs`) and to the OpenTelemetry logs SDK through the `otelslog` bridge ([app/README.md](../app/README.md#logs)). A failed `/api/work` writes one `WARN` record, `work failed`, with the error and the request's context, so the bridge attaches the span's trace and span IDs.

Run commands from the repo root.

## Versions

| Component | Pin | Source |
| :--- | :--- | :--- |
| Loki | `3.7.9`, Helm chart `grafana-community/loki` `20.2.2` | [loki-values.yaml](loki-values.yaml) |
| App SDK | OpenTelemetry Go logs SDK `v1.47.0`, `otlploghttp` `v0.23.0`, `otelslog` `v0.21.0` | [app/README.md](../app/README.md#logs) |

Loki runs Monolithic: one pod, filesystem storage on the `emptyDir` the chart mounts at `/var/loki`, no gateway, caches, canary or rules sidecar. Logs are lost when the pod restarts, which is fine for a local demo. `auth_enabled: false` makes it single-tenant, so nothing needs an `X-Scope-OrgID` header.

How Loki stores OTLP logs, checked against 3.7:

- Resource attributes such as `service.name`, `k8s.namespace.name` and `k8s.pod.name` become index labels: `service_name`, `k8s_namespace_name`, `k8s_pod_name`.
- Log attributes and the record's `trace_id` and `span_id` become structured metadata. That needs schema `v13` and `allow_structured_metadata: true`, both set in the values.
- The log body is the line. Filter structured metadata after the selector, for example `{service_name="devops-app"} | trace_id="<id>"`.

## Install

After the [monitoring](../monitoring/README.md) install, and before or with [tracing/](../tracing/README.md), which holds the Collector:

```sh
helm repo add grafana-community https://grafana-community.github.io/helm-charts
helm upgrade --install loki grafana-community/loki --version 20.2.2 \
  -n monitoring -f logging/loki-values.yaml --wait --timeout 5m
kubectl apply -k tracing/
```

The Loki datasource and the trace-to-logs links are set in [monitoring/values.yaml](../monitoring/values.yaml), so rerun the `helm upgrade` for `kps` from the monitoring README if it was installed before this folder existed.

## Check

```sh
kubectl get pods -n monitoring -l app.kubernetes.io/name=loki
for i in $(seq 20); do curl -s -o /dev/null "http://localhost:8080/api/work?ms=300&fail=0.5"; done
```

Then open Grafana (see the [monitoring README](../monitoring/README.md#open-grafana-and-prometheus)):

| Check | Where | Expected |
| :--- | :--- | :--- |
| Pod | `kubectl get pods` above | `loki-0` is `1/1 Running` |
| Logs | Explore → Loki → `{service_name="devops-app"}` | `work failed` records, one per `500`. Each shows `trace_id`, `span_id`, `error` and the `work.*` attributes |
| Log → trace | Expand a record | A **View trace** link next to `trace_id` opens the failed `GET /api/work` trace in Tempo |
| Trace → logs | Explore → Tempo → `{ resource.service.name = "devops-app" && status = error }`, open a trace, span menu **Logs for this span** | The `work failed` record of that request |
| Collector | Prometheus `otelcol_exporter_sent_log_records` | Above 0. `otelcol_exporter_send_failed_log_records` is absent or 0 |

## Delete

```sh
helm uninstall loki -n monitoring
```
