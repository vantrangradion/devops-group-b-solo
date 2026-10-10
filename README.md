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
