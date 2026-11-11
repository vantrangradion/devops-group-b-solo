# Load test — step 5 (#51)

`test.js` is a k6 script, tested with k6 `v2.3.0` (`k6 version`). It ramps 0 → 20 VUs (1 m), 20 → 100 (2 m), holds 100 (2 m) and ramps down (1 m), about 6 minutes in total. Each VU loops over a mix that moves every panel of the [step 4 dashboard](../monitoring/README.md):

| Share | Request | Moves |
| :--- | :--- | :--- |
| 75% | `/api/work?ms=50` | Request rate |
| 15% | `/api/work?ms=300` | p95 and p99 latency |
| 10% | `/api/work?ms=50&fail=0.2` | 5xx ratio (about 2% of all requests) |

Run from the repo root:

```sh
k6 run load-tests/test.js
```

The target is `http://localhost:8080`, the k3d load balancer, which spreads load across both pods. `kubectl port-forward` would send every request to a single pod. Set `BASE_URL` to point elsewhere, but never at the quick tunnel: it is rate-limited, so the numbers would measure Cloudflare, not the app. For a 30-second check, override the stages: `k6 run --stage 20s:10 --stage 10s:0 load-tests/test.js`.

The deliberate 500s from `fail=0.2` are marked as expected responses, so the thresholds only fail on real errors:

| Threshold | Limit |
| :--- | :--- |
| `http_req_failed` | `< 1%` |
| `http_req_duration{expected_response:true}` p95 | `< 500ms` |
| `checks` (status, body contains `hello from configmap`) | `> 99%` |

k6 exits non-zero when a threshold fails.

## Steady background load

[`k8s/`](k8s/) runs a small k6 inside the cluster, so the dashboard is never empty and a fault flips against a flat baseline. The idea comes from the OpenTelemetry Demo's and Grafana's `intro-to-mltp` always-on load generators.

| | |
| :--- | :--- |
| Script | [`k8s/steady.js`](k8s/steady.js), mounted from the ConfigMap `k6-steady` |
| Load | 2 VUs (`constant-vus`, `24h`), about 5 req/s in total: 80% `/api/work?ms=50`, 20% `/api/work?ms=300` |
| Errors | No `fail=`, so the error ratio rests at 0%. Every `5xx` during the demo comes from a fault toggle or a real problem |
| Runs as | Deployment `k6-steady` in `devops-app`, `grafana/k6:2.3.0` pinned by digest. Non-root, read-only root filesystem, all capabilities dropped. cpu 50m / memory 64Mi requested |
| Target | `http://devops-app:8080`, the ClusterIP Service. `noConnectionReuse` opens a new connection per request, because kube-proxy balances per connection: with kept-alive connections both VUs can stick to one pod |
| Thresholds | None, so a fault doesn't stop the run. After 24 h k6 exits and the pod restarts it |
| Tagged | `User-Agent: k6-steady`. Find it in Tempo with `{ span.user_agent.original = "k6-steady" }`. The dashboard panels count it with the rest |

It lives outside `k8s/base/` (the app's folder, which Argo CD manages with self-heal), so it can be scaled by hand. `make up` applies it.

```sh
kubectl apply -k load-tests/k8s/                                       # install, starts at 1 replica
kubectl logs -n devops-app deploy/k6-steady                            # one line per VU about every 2 minutes
kubectl scale deploy/k6-steady -n devops-app --replicas=0              # stop
kubectl scale deploy/k6-steady -n devops-app --replicas=1              # start
kubectl delete -k load-tests/k8s/                                      # remove
```

## Demo runbook

1. Cluster up, app deployed and unchanged:

   ```sh
   k3d cluster list
   kubectl rollout status deployment/devops-app -n devops-app
   ```

2. Monitoring up, with 2 targets (see [monitoring › Check](../monitoring/README.md#check)).
3. Grafana port-forward, sign in, open <http://localhost:3000/d/devops-app>. Check the time range is last 15 minutes and refresh is 5 s.
4. The steady load is running, so the panels already show a flat baseline: about 5 req/s, 0% errors.
5. Put the terminal and the browser side by side, then run `k6 run load-tests/test.js`.
6. What to point at, in order:
   - **Request rate:** climbs from the 5 req/s baseline with the VUs, to about 1.1k req/s at 100 VUs, and falls back to the baseline after the run.
   - **Error ratio:** settles near 2%, the injected share.
   - **Latency:** p50 stays near 50 ms; p95/p99 sit higher because of the 300 ms share. They are bucket estimates; k6's own p95 in the summary is the exact one.
   - **CPU and memory per pod:** both pods rise together, far below the dashed limits, and ready replicas stay at 2.
7. Read the k6 summary: every threshold has a ✓.

Fallback if the live run fails: show the captures in [captures/step-5](https://github.com/gradionhq/2026-devops-assignment-intern/tree/main/common-deliverable/shared-system/captures/step-5/README.md).

## Fault toggles during the demo

The load stays the same while the presenter flips a fault, and the audience watches Grafana and Kubernetes react. The toggles and the script are described in [app › Fault toggles](../app/README.md#fault-toggles). The [steady background load](#steady-background-load) is the traffic the faults act on, so no terminal run is needed. With its 0% baseline, every error on screen comes from the toggle.

Without the steady load, run `k6 run --stage 20s:20 --stage 5m:20 --stage 10s:0 load-tests/test.js` in its own terminal instead. Its `fail=0.2` share puts the baseline near 2%, and k6 exits non-zero because it counts the toggle's `500`s as failures.

| Flip | Look at | Reset |
| :--- | :--- | :--- |
| `scripts/fault.sh error 0.3` | **Error ratio** climbs from 0% toward 30% on both pods within a minute (`[1m]` rate). In Tempo, `{ status = error && event:name = "exception" }` finds the failed requests, with `exception.message` `upstream unavailable: fault toggle error_rate=0.3`; open one, then **Logs for this span** shows its `work failed` record with the same `error` | `scripts/fault.sh reset` |
| `scripts/fault.sh latency 500` | **Latency /api/work** p50 and p95 rise by about 500 ms. Hover an exemplar dot, then **View trace**: the span carries `fault.extra_latency_ms=500` | `scripts/fault.sh reset` |
| `scripts/fault.sh ready false <pod>` (pick a pod from `kubectl get pods -n devops-app -l app.kubernetes.io/name=devops-app`) | `kubectl get pods -n devops-app -l app.kubernetes.io/name=devops-app -w`: that pod goes `0/1` with `RESTARTS` unchanged. `kubectl get endpointslices -n devops-app` lists only the other pod's address. **Ready replicas** drops to 1, and the request rate moves to the remaining pod while `/api/work` keeps answering | `scripts/fault.sh ready true <pod>`, or `scripts/fault.sh reset` |

Reset before the next scene. `scripts/fault.sh show` prints the state of each pod. A pod restart also resets its toggles.
