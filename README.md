# devops-group-b-solo

Personal solo repo for the Group B DevOps & Observability Research assignment. It holds all three pillars:

- `main`: Pillar 1 (CI/CD & container security)
- `p2`: Pillar 2 (Kubernetes & remote access), branched from `main`
- `p3`: Pillar 3 (observability & load test), branched from the team repo's `main` after CP2

## Layout

```
app/                  Go service, Dockerfile, Makefile, lint config (Docker build context)
.github/workflows/    CI (Go and lint steps run with working-directory: app)
k8s/base/             Namespace, Deployment, Service, ConfigMap, Secret, kustomization
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

## Cluster

Pillar 2 runs on a local k3d cluster (k3d `v5.9.0`, k3s `v1.36.5`) with a single server and no agents:

```sh
k3d cluster create devops-solo -p "8080:8080@loadbalancer"
```

The port mapping sends host port `8080` to a second Service of type `LoadBalancer` on port `8080`, which the Remote access path uses. Host port `8080` is therefore taken while the cluster runs.

Check that every node is `Ready`:

```sh
kubectl get nodes
```

### Storage

`kubectl get storageclass` lists one class:

| Name | Provisioner | Default | Access mode |
| :--- | :--- | :--- | :--- |
| `local-path` | `rancher.io/local-path` | yes | `ReadWriteOnce` only |

The brief says k3d ships this provisioner as the `standard` StorageClass. On k3d it is named `local-path`. The app needs no PVC.

## Deploy

One command deploys the Namespace, ConfigMap, Secret, Deployment, ClusterIP Service, and LoadBalancer Service in `k8s/base/`:

```sh
kubectl apply -k k8s/base/
```

The deploy uses `-k` because `kubectl apply -f k8s/base/` applies files in file-name order, so `configmap.yaml` and `deployment.yaml` fail on a fresh cluster before `namespace.yaml` creates the namespace. `k8s/base/kustomization.yaml` only lists the six files, and kustomize applies the Namespace first.

### Check

Wait for both replicas, then confirm that the Service has 2 ready endpoints:

```sh
kubectl rollout status deployment/devops-app -n devops-app
kubectl get deploy -n devops-app
kubectl get endpointslices -n devops-app -l kubernetes.io/service-name=devops-app \
  -o jsonpath='{range .items[*].endpoints[*]}{.addresses[0]} {.conditions.ready}{"\n"}{end}'
```

Call the app through the Service. Local port `18080` is used because host port `8080` belongs to the k3d load balancer mapping:

```sh
kubectl port-forward -n devops-app svc/devops-app 18080:8080
```

In a second terminal:

```sh
curl -i http://localhost:18080/api/work
curl -s http://localhost:18080/metrics | grep http_requests_total
```

| Check | Expected |
| :--- | :--- |
| `/api/work` | `200`, and the body contains `hello from configmap` (the ConfigMap's `MESSAGE`) |
| `/metrics` | `http_requests_total` is present |

Confirm that the cluster matches the manifests:

```sh
kubectl diff -k k8s/base/; echo "exit $?"
```

`kubectl diff` exits `0` when nothing would change. Running `kubectl apply -k k8s/base/` again reports `unchanged` for every object except the Secret, which always shows `configured`: `stringData` is write-only and never stored, so each apply re-sends it.

### Delete

```sh
kubectl delete -k k8s/base/
```

## Remote access

A phone on mobile data reaches the app through a `cloudflared` quick tunnel. It needs no account, token, or config file, so nothing tunnel-related is committed. The path is:

```
phone → https://<random>.trycloudflare.com → cloudflared on the host → localhost:8080
      → k3d load balancer → Service devops-app-lb (LoadBalancer, port 8080) → both pods
```

1. Create the cluster with the port mapping from [Cluster](#cluster) (`-p "8080:8080@loadbalancer"`).
2. Deploy with `kubectl apply -k k8s/base/` (see [Deploy](#deploy)). It also creates `k8s/base/service-lb.yaml`, the Service `devops-app-lb` of type `LoadBalancer` on port `8080`. It sits next to the ClusterIP Service `devops-app` and selects the same pods. It does not use port 80 or 443, because the Traefik LoadBalancer that k3s installs owns those ports.
3. Check the host path. The `EXTERNAL-IP` of `devops-app-lb` must be an address, not `<pending>`:

   ```sh
   kubectl get svc -n devops-app
   curl -i http://localhost:8080/api/work
   ```

4. Start the tunnel:

   ```sh
   cloudflared tunnel --url http://localhost:8080
   ```

   The public URL is printed in the log box (`Your quick Tunnel has been created! Visit it at ...`), in the form `https://<random>.trycloudflare.com`. To run it in the background, redirect the output to a file and read the URL from it:

   ```sh
   nohup cloudflared tunnel --url http://localhost:8080 > /tmp/cloudflared.log 2>&1 &
   grep -o 'https://[a-z0-9-]*\.trycloudflare\.com' /tmp/cloudflared.log
   ```

5. Check from the host with `curl -i <url>/api/work` and `curl -i <url>/health`.

### Phone check

Turn Wi-Fi off on the phone so it uses mobile data only, then open:

| URL | Expected |
| :--- | :--- |
| `<url>/api/work` | The body contains `hello from configmap` (the ConfigMap's `MESSAGE`) |
| `<url>/health` | `200`, body `{"status":"ok"}` |

### Why a port mapping and not `kubectl port-forward`

`kubectl port-forward svc/devops-app 8080:8080` would also give the tunnel a local port, but:

- it sends every connection to a single pod, so traffic is not spread across both replicas;
- it stops when its terminal closes;
- it exits whenever that pod is deleted or restarted (a liveness restart, a rollout), which silently breaks the tunnel.

The k3d port mapping goes through the LoadBalancer Service, so it spreads traffic across both replicas and survives pod restarts. Its cost is that it is easiest to set when the cluster is created: `k3d cluster edit --port-add` can add it later, but that command is experimental.

### Demo-day notes

- A quick tunnel gets a new `trycloudflare.com` URL every time `cloudflared` starts. After any restart, read the new URL from the output and share it again.
- On macOS, `curl` on the host can fail with `Could not resolve host` for a fresh `trycloudflare.com` URL even though `dig` resolves it, because the system cached an earlier negative answer; flush it with `sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder` (the phone is not affected).
- The public URL exposes every route, including `/metrics`. That is acceptable for a demo of a toy app, but the tunnel runs only while it is needed.
- Stop the tunnel with `Ctrl+C` in its terminal, or `pkill -f "cloudflared tunnel"` if it runs in the background. The URL stops working at once.

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





