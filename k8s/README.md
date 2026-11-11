# Cluster, manual deploy and remote access — steps 2 and 3 (#48, #49)

Run commands from the repo root.

## Cluster

A local k3d cluster (k3d `v5.9.0`, k3s `v1.36.5`) with one server and one agent, so the two app replicas run on separate nodes:

```sh
k3d cluster create devops-solo --agents 1 -p "8080:8080@loadbalancer"
kubectl get nodes
```

The port mapping sends host port `8080` to the Service `devops-app-lb` (type `LoadBalancer`, port `8080`), which the tunnel and the k6 load test use. Host port `8080` is therefore taken while the cluster runs.

`kubectl get storageclass` lists one class, `local-path` (`rancher.io/local-path`, default, `ReadWriteOnce` only). The brief says k3d ships this provisioner as `standard`; on k3d it is named `local-path`. The app needs no PVC.

## Deploy

One command deploys the Namespace, ConfigMap, Secret, Deployment, ClusterIP Service and LoadBalancer Service:

```sh
kubectl apply -k k8s/base/
```

`-k` rather than `-f`: `kubectl apply -f k8s/` applies files in file-name order, so `configmap.yaml` and `deployment.yaml` fail on a fresh cluster before `namespace.yaml` creates the namespace. kustomize applies the Namespace first.

The Deployment pins `ghcr.io/vantrangradion/devops-app:70ba364…`, which pushes traces, metrics and logs over OTLP to the Collector, no longer serves `/metrics` (#132, #142), has the fault toggles and `/ready` (#137), records failures as `exception` events, and reports its commit as `version` in `/api/work`. The step 1–5 captures were taken with the earlier `64bf198e…` image, which served `/metrics` for Prometheus to scrape.

### Check

```sh
kubectl rollout status deployment/devops-app -n devops-app
kubectl get all -n devops-app
kubectl port-forward -n devops-app svc/devops-app 18080:8080
```

In a second terminal (local port `18080`, because `8080` belongs to the k3d mapping):

```sh
curl -i http://localhost:18080/api/work
```

| Check | Expected |
| :--- | :--- |
| `/api/work` | `200`, and the body contains `hello from configmap` (the ConfigMap's `MESSAGE`) |
| `kubectl diff -k k8s/; echo "exit $?"` | `exit 0` |

Reapplying reports `unchanged` for every object except the Secret, which always shows `configured`: `stringData` is write-only and never stored, so each apply re-sends it.

### Delete

```sh
kubectl delete -k k8s/
```

## Remote access

A phone on mobile data reaches the app through a `cloudflared` quick tunnel. It needs no account, token or config file, so nothing tunnel-related is committed.

```
phone → https://<random>.trycloudflare.com → cloudflared on the host → localhost:8080
      → k3d load balancer → Service devops-app-lb → both pods
```

1. Check the host path. The `EXTERNAL-IP` of `devops-app-lb` must be an address, not `<pending>`:

   ```sh
   kubectl get svc -n devops-app
   curl -i http://localhost:8080/api/work
   ```

2. Start the tunnel and read the URL from the log box (`Your quick Tunnel has been created! Visit it at ...`):

   ```sh
   cloudflared tunnel --url http://localhost:8080
   ```

3. Turn Wi-Fi off on the phone and open `<url>/api/work` (body contains `hello from configmap`) and `<url>/health` (`{"status":"ok"}`).

`devops-app-lb` does not use port 80 or 443, because k3s's Traefik LoadBalancer owns them. It carries `app.kubernetes.io/component: remote-access`, which marks it as the remote-access Service.

### Why a port mapping and not `kubectl port-forward`

`kubectl port-forward` sends every connection to a single pod, stops when its terminal closes, and exits when that pod restarts, which silently breaks the tunnel. The k3d port mapping goes through the LoadBalancer Service, so it spreads traffic across both replicas and survives pod restarts.

### Demo-day notes

- A quick tunnel gets a new `trycloudflare.com` URL every time `cloudflared` starts. Share the new one after any restart.
- On macOS, `curl` on the host can fail with `Could not resolve host` for a fresh URL even though `dig` resolves it; flush the cache with `sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder`.
- The public URL exposes every route. Run the tunnel only while it is needed, and stop it with `Ctrl+C`.
