# devops-group-b-solo

Personal solo repo for the Group B DevOps & Observability Research assignment. It holds all three pillars:

- `main`: Pillar 1 (CI/CD & container security)
- `p2`: Pillar 2 (Kubernetes & remote access), branched from `main`
- `p3`: Pillar 3 (observability & load test), branched from the team repo's `main` after CP2

## Layout

```
app/                  Go service, Dockerfile, Makefile, lint config (Docker build context)
.github/workflows/    CI (Go and lint steps run with working-directory: app)
k8s/base/             Namespace, Deployment, Service, ConfigMap, Secret
k8s/monitoring/       kube-prometheus-stack values, ServiceMonitor
load-tests/           k6 scripts
docs/                 Research write-ups
captures/             Evidence screenshots and command output
```

## Toolchain

Pinned versions, agreed by the team at kickoff. Local builds, the Dockerfile, and CI all use the same pins.

| Tool | Pin | Where it applies |
| :--- | :--- | :--- |
| Go | `1.27` | `go` directive in `app/go.mod`, `golang:1.27` builder in `app/Dockerfile` |
| golangci-lint | `v2.14.0` | `app/Makefile` and CI, both with `app/.golangci.yml` |

golangci-lint `v2.14.0` is built with Go 1.27, so it can lint a module whose `go` directive is `1.27.x`.

## App contract

Agreed by the team at kickoff. All three personal builds expose this same interface, so Pillar 2 manifests and Pillar 3 dashboards work on any of the three images.

### HTTP

The app listens on port `8080`.

| Endpoint | Behavior |
| :--- | :--- |
| `GET /health` | `200` with a small body. |
| `GET /api/work?ms=<int>&fail=<0..1>` | Sleeps for `ms` milliseconds, then returns `500` with probability `fail`, otherwise `200`. Defaults: `ms=50`, `fail=0`. Out-of-range values are clamped (`ms` to `0..2000`, `fail` to `0..1`). An `ms` that is not an integer (e.g. `abc` or `1.5`), or a `fail` that is not a number, returns `400`. The response body includes the `MESSAGE` value. |
| `GET /metrics` | Prometheus exposition format. |

### Environment variables

| Variable | Source | Use |
| :--- | :--- | :--- |
| `APP_ENV` | ConfigMap | Deployment environment name. |
| `MESSAGE` | ConfigMap | Returned in the `/api/work` response body, so a `curl` proves the ConfigMap was injected. |
| `API_KEY` | Secret | Never written to logs or responses. At most, the app reports whether it is set. |

The app starts with safe defaults when any of these are unset.

### Metrics

| Name | Type | Labels |
| :--- | :--- | :--- |
| `http_requests_total` | counter | `method`, `route`, `status` |
| `http_request_duration_seconds` | histogram | `method`, `route`, `status` |

The `route` label is the route template (e.g. `/api/work`), and requests to unknown paths are recorded as `route="other"`.
