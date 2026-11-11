# Workload monitoring — step 4 (#50)

Prometheus and Grafana run in their own namespace, `monitoring`, from the Helm chart `prometheus-community/kube-prometheus-stack`, pinned to `92.10.0`. The app stays in `devops-app`. The dashboard shows the app's own RED metrics next to pod CPU and memory from kubelet/cAdvisor. This is a local demo: no persistence, no ingress, alerting disabled.

Run commands from the repo root.

## Target app

The monitors and the dashboard expect the [app](../app/README.md) deployed by [k8s/](../k8s/README.md):

| Item | Value |
| :--- | :--- |
| Namespace, Deployment | `devops-app`, `devops-app` (2 replicas) |
| Endpoints | `GET /health`, `GET /api/work?ms=<int>&fail=<0..1>` |
| Metrics | Pushed over OTLP to the Collector ([tracing/](../tracing/README.md)), which pushes them to Prometheus's OTLP receiver as `http_requests_total` (counter) and `http_request_duration_seconds` (histogram), labels `method`, `route`, `status`, `namespace`, `pod` |

## Install

Give Docker Desktop at least 6 GB RAM and 4 CPUs.

```sh
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm upgrade --install kps prometheus-community/kube-prometheus-stack --version 92.10.0 \
  -n monitoring --create-namespace -f monitoring/values.yaml --wait --timeout 8m
kubectl apply -k monitoring/
```

`kubectl apply -k` needs the chart's CRDs, so it runs after the Helm install. It creates:

| Object | Purpose |
| :--- | :--- |
| ConfigMap `devops-app-dashboard` | Built from `monitoring/dashboards/devops-app.json` and labelled `grafana_dashboard: "1"`, so Grafana's sidecar loads it with no manual import |

There is no ServiceMonitor for the app: the Collector pushes its metrics to Prometheus. `values.yaml` turns on the OTLP receiver (`enableOTLPReceiver`, path `/api/v1/otlp/v1/metrics`, no authentication, so cluster-internal only), pins the name translation to `UnderscoreEscapingWithSuffixes` so names keep underscores, and allows 30 minutes of out-of-order samples for Collector retries. If you upgrade from the scrape setup, delete the old monitor with `kubectl delete servicemonitor devops-app -n monitoring`; `kubectl apply -k` does not remove it.

`values.yaml` also sets `serviceMonitorSelectorNilUsesHelmValues: false`, for the Collector's own ServiceMonitor; without it Prometheus only picks up ServiceMonitors carrying the chart's `release` label. Alertmanager, the default alert rules, node-exporter and the control-plane monitors are disabled to save memory. kubelet/cAdvisor (CPU and memory) and kube-state-metrics (replicas, limits) stay on. Grafana keeps a 1 GiB limit with `GOMEMLIMIT=768MiB`: a 384 MiB limit OOM-killed it while it loaded its frontend (PR #68).

## Open Grafana and Prometheus

Both stay private: port-forward only, never tunnelled. In separate terminals:

```sh
kubectl port-forward -n monitoring svc/kps-grafana 3000:80
kubectl port-forward -n monitoring svc/kps-kube-prometheus-stack-prometheus 9090:9090
```

Open <http://localhost:3000/d/devops-app> and sign in as `admin`. Read the generated password with the command below, and keep it out of captures:

```sh
kubectl get secret -n monitoring -l app.kubernetes.io/component=admin-secret \
  -o jsonpath='{.items[0].data.admin-password}' | base64 --decode; echo
```

## Check

```sh
curl -s localhost:9090/api/v1/query --data-urlencode 'query=count(up{namespace="devops-app"})' \
  | jq -r '.data.result[0].value[1]'
```

| Check | Expected |
| :--- | :--- |
| `count(up{namespace="devops-app"})` | `2`, one target per pod, all from Service `devops-app` |
| Dashboard `devops-app` | Refresh 5 s, last 15 minutes, every panel has data once requests reach `/api/work` |

## Dashboard

| Panel | Source | Query (simplified) |
| :--- | :--- | :--- |
| Request rate by route | App | `sum by (route) (rate(http_requests_total[1m]))` |
| Error ratio (5xx) | App | 5xx rate / total rate |
| Latency `/api/work` p50/p95/p99 | App | `histogram_quantile(q, sum by (le) (rate(http_request_duration_seconds_bucket[1m])))` |
| Ready replicas | kube-state-metrics | `kube_deployment_status_replicas_available` |
| CPU per pod | cAdvisor | `rate(container_cpu_usage_seconds_total[1m])`, with the container limit as a dashed line |
| Memory per pod | cAdvisor | `container_memory_working_set_bytes`, with the container limit as a dashed line |

Every query is filtered to `namespace="devops-app"` and the `pod` variable. The variable lists only pods that are `Running` now (`kube_pod_status_phase`), and `All` expands to that list, so pods replaced by a rollout drop out of the legend at once instead of lingering for the 15-minute window. Latency quantiles are estimates within histogram buckets: a 300 ms request falls in the `0.25`–`0.5` bucket, so p95 can read about 440 ms when the real p95 is 300 ms.

The p50 query has exemplars turned on. Prometheus stores the `trace_id` exemplars the Collector exports on latency samples (`exemplar-storage` in `values.yaml`), and the Prometheus datasource links each one to the Tempo datasource, so a dot on the panel opens its trace. Both datasources come from `values.yaml`; Tempo itself is installed from [tracing/](../tracing/README.md).

`values.yaml` also adds a Loki datasource (uid `loki`) for the app's logs, installed from [logging/](../logging/README.md). Its `TraceID` derived field turns each record's `trace_id` into a **View trace** link to Tempo, and the Tempo datasource's `tracesToLogsV2` adds **Logs for this span**, which queries `{service_name="devops-app"} | trace_id="<id>"`.

Real results are recorded in [captures/step-4](https://github.com/gradionhq/2026-devops-assignment-intern/tree/main/common-deliverable/shared-system/captures/step-4/README.md). The live load test that drives this dashboard is in [load-tests](../load-tests/README.md).

## Delete

```sh
kubectl delete -k monitoring/
helm uninstall kps -n monitoring
```

Helm leaves the CRDs behind; delete the cluster to remove them.
