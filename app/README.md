# App — step 1 (#47)

A small Go API, built on a distroless base by the root [CI workflow](../.github/workflows/ci.yml) and published to `ghcr.io/<owner>/devops-app`. Copied from `self-deliverable-van/app/`.

## Toolchain

| Tool | Pin | Where it applies |
| :--- | :--- | :--- |
| Go | `1.27` | `go` directive in `go.mod`, `golang:1.27` builder in `Dockerfile` |
| golangci-lint | `v2.14.0` | `Makefile` and CI, both with `.golangci.yml` |

Run from this folder: `go test ./...`, `make lint`, and `make smoke` for a container check that mirrors CI.

## HTTP

The app listens on port `8080`, and on `8081` for the [fault toggles](#fault-toggles).

| Endpoint | Behavior |
| :--- | :--- |
| `GET /health` | `200` with a small body. The livenessProbe. |
| `GET /ready` | `200`, or `503` while the `ready` toggle is `false`. The readinessProbe. Not traced and not counted in the RED metrics, so a not-ready pod's probes don't show as `5xx` on the dashboard. |
| `GET /api/work?ms=<int>&fail=<0..1>` | Sleeps for `ms` milliseconds, then returns `500` with probability `fail`, otherwise `200`. Defaults: `ms=50`, `fail=0`. Out-of-range values are clamped (`ms` to `0..2000`, `fail` to `0..1`). An `ms` that is not an integer, or a `fail` that is not a number, returns `400`. The response body includes the `MESSAGE` value and `version`, the commit the image was built from (`dev` for a local build). |

## Fault toggles

The presenter can switch failures on and off while the demo runs, without changing the traffic or any manifest. Argo CD's self-heal would revert a manifest edit. The idea comes from the OpenTelemetry Demo's feature flags.

The toggles live in each pod's memory and start at their defaults on every pod start. A second listener on `ADMIN_ADDR` (`:8081`, container port `admin`) serves them. That port is in no Service, so neither the LoadBalancer nor the tunnel reaches it. It has no authentication: only `kubectl port-forward` gets there.

| Toggle | Range, default | Effect |
| :--- | :--- | :--- |
| `error_rate` | `0..1`, `0` | Chance that `/api/work` returns `500`, on top of any `?fail=` |
| `extra_latency_ms` | `0..2000`, `0` | Added to every `/api/work` delay |
| `ready` | `true`/`false`, `true` | What `/ready` reports. `false` takes the pod out of the Service endpoints without a restart |

| Admin endpoint | Behavior |
| :--- | :--- |
| `GET /faults` | The current state as JSON. |
| `PUT /faults` | A JSON body with any of the three fields, for example `{"error_rate": 0.3}`. Fields left out keep their value. Out-of-range values are clamped like `ms` and `fail`. Unknown fields or malformed JSON return `400`. Returns the new state. |
| `DELETE /faults` | Resets to the defaults. Returns the new state. |

Each change is logged at `INFO` (`faults changed`, `faults reset`).

[`scripts/fault.sh`](../scripts/fault.sh) applies a change to every `devops-app` pod, or to one pod. It port-forwards to each pod's `admin` port in turn:

```sh
scripts/fault.sh error 0.3              # 30% of /api/work return 500
scripts/fault.sh latency 500            # +500 ms on every /api/work
scripts/fault.sh ready false <pod>      # that pod goes 0/1 Ready, no restart
scripts/fault.sh show                   # current state per pod
scripts/fault.sh reset                  # all toggles back to the defaults
```

`KUBE_CONTEXT` picks the cluster (default `k3d-devops-solo`). The demo runbook is in [load-tests/README.md](../load-tests/README.md#fault-toggles-during-the-demo).

## Environment variables

| Variable | Source | Use |
| :--- | :--- | :--- |
| `APP_ENV` | ConfigMap | Deployment environment name. |
| `MESSAGE` | ConfigMap | Returned in the `/api/work` response body, so a `curl` proves the ConfigMap was injected. |
| `API_KEY` | Secret | Never written to logs or responses. At most, the app reports whether it is set. |
| `ADMIN_ADDR` | Default | Listen address for the fault toggles. Default `:8081`. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Deployment | OTLP/HTTP base URL for traces, metrics and logs, for example `http://otel-collector.monitoring:4318`. A signal is off when this and its own `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`, `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` or `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` are unset. |
| `OTEL_METRIC_EXPORT_INTERVAL` | Deployment | Milliseconds between metric pushes. The SDK default is `60000`; the Deployment sets `5000`. |
| `OTEL_SERVICE_NAME` | Deployment | `service.name` on every span, metric and log record, for example `devops-app`. |

The app starts with safe defaults when any of these are unset. The other standard `OTEL_*` variables, such as `OTEL_TRACES_SAMPLER` and `OTEL_RESOURCE_ATTRIBUTES`, also apply.

## Metrics

The app has no `/metrics` endpoint. It records metrics with the OpenTelemetry metrics SDK and pushes them over OTLP/HTTP to the Collector, the same way it sends traces. The Collector pushes them on to Prometheus ([tracing/](../tracing/README.md)).

| OTel instrument | Prometheus name (from the Collector) | Type | Attributes / labels |
| :--- | :--- | :--- | :--- |
| `http.requests` | `http_requests_total` | counter | `method`, `route`, `status` |
| `http.request.duration` (unit `s`) | `http_request_duration_seconds` | histogram | `method`, `route`, `status` |

The `route` label is the route template (e.g. `/api/work`), and requests to unknown paths are recorded as `route="other"`. `otelhttp`'s own request metrics are turned off, because they would duplicate these without the route.

Each latency sample made inside a sampled span keeps that span's trace ID as an exemplar. The Collector exports it as `trace_id`, so Grafana can jump from a latency point to its trace.

## Traces

When an OTLP endpoint is set, every request except `/health` and `/ready` (probes) becomes one server span, sent in batches over OTLP/HTTP. The span is named by method and route template, for example `GET /api/work`, and carries the HTTP semantic-convention attributes, such as `http.route` and `http.response.status_code`. A `5xx` response sets the span status to `Error`.

`/api/work` spans also record what the caller asked for, because the HTTP attributes keep `url.path` but not the query string:

| Field | Type | Meaning |
| :--- | :--- | :--- |
| `work.ms` | attribute, int | The delay after defaults and clamping |
| `work.fail_probability` | attribute, float | The `fail` value after defaults and clamping |
| `fault.error_rate`, `fault.extra_latency_ms` | attributes, float and int | The [fault toggles](#fault-toggles) in force, set only while one of them is on |

The simulated work returns an ordinary Go error: `upstream timeout after <ms>ms` with probability `fail`, or `upstream unavailable: fault toggle error_rate=<v>` from the error toggle. The handler has one generic error path. It calls `span.RecordError(err)` and returns `500` with the error in the body's `error` field, next to the usual message. `RecordError` adds a standard `exception` event with `exception.type` and `exception.message`, so the trace shows whatever error came back, not a reason written in advance. The span status is `Error` from the `5xx` code, set by `otelhttp` after the handler returns, which is also why the handler sets no status description: `otelhttp` would clear it.

To find failed requests in Tempo: `{ status = error && event:name = "exception" }`, or `{ resource.service.name = "devops-app" && status = error }`.

An incoming W3C `traceparent` header is continued, so a trace started by a client keeps its trace ID. Buffered spans are flushed on shutdown, after in-flight requests drain.

## Logs

The app logs through `log/slog`. The default handler writes each record to stderr, for `kubectl logs`, and to the OpenTelemetry logs SDK through the `otelslog` bridge, which pushes over OTLP/HTTP to the Collector and on to Loki ([logging/](../logging/README.md)). Without an OTLP endpoint the OTel side is a no-op.

A failed `/api/work` writes one `WARN` record, `work failed`, with the attributes `route`, `work.ms`, `work.fail_probability` and `error` (the same message as the span's `exception` event). It is logged with the request's context, so the record carries the span's trace and span IDs, and Grafana links it to the trace in Tempo. Successful requests write no log record.
