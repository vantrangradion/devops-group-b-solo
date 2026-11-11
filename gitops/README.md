# GitOps: Argo CD (#136)

Argo CD deploys the app to the k3d cluster from Git. A commit that changes the image reference in Git is the deploy; nobody runs `kubectl apply` for the app.

```
git push (k8s/base/deployment.yaml) ──▶ GitHub ◀──poll / Refresh── argocd-repo-server ──▶ argocd-application-controller ──▶ Deployment devops-app
```

Run commands from the repo root.

## Which repo Argo CD watches

The Application watches this repo, **`vantrangradion/devops-group-b-solo`**, path `k8s/base`, branch `main`. The whole live demo runs from here. The team's deliverable stays in [gradionhq/2026-devops-assignment-intern](https://github.com/gradionhq/2026-devops-assignment-intern/tree/main/common-deliverable/shared-system), which holds the same system:

- The gradionhq Actions budget is used up, so the team repo's CI builds no images. This repo's CI builds, tests, scans and publishes `ghcr.io/vantrangradion/devops-app:<commit SHA>`.
- This repo and its image are public, so Argo CD needs no repo credential and the cluster no image pull secret. Nothing secret exists to commit.
- `main` is not protected, so the presenter can push live.

CI deploys every `main` build: after publishing the image, a **Deploy** job commits the new tag to `k8s/base`, and Argo CD rolls it out ([Live demo](#live-demo)). Only pushes that touch `app/` or the workflow rebuild the image.

## Versions

| Component | Pin | Source |
| :--- | :--- | :--- |
| Argo CD | `v3.5.6`, the same as Sang's lab (#102) | [install/kustomization.yaml](install/kustomization.yaml) |

The install runs only the server, repo-server, Redis and application-controller. Dex, notifications and the ApplicationSet controller are scaled to 0: one Application, local `admin` login.

## Files

| File | What it does |
| :--- | :--- |
| [install/](install/) | Namespace `argocd` and the upstream `install.yaml` for v3.5.6 |
| [project.yaml](project.yaml) | AppProject `devops-demo`: only the personal repo, only namespace `devops-app`, only the kinds `k8s/` creates (Namespace, ConfigMap, Secret, Service, Deployment) |
| [application.yaml](application.yaml) | Application `devops-app`: repo, path and branch above, automatic sync, self-heal on, prune off |

Self-heal undoes hand edits on the cluster, such as `kubectl scale`. Prune is off, so a manifest removed from Git leaves its resource on the cluster until it is deleted by hand.

## Install

After the [cluster](../k8s/README.md#cluster) exists:

```sh
kubectl apply --server-side -k gitops/install
kubectl -n argocd rollout status deploy/argocd-server --timeout 5m
kubectl apply -f gitops/project.yaml -f gitops/application.yaml
kubectl -n argocd get application devops-app
```

`--server-side` because the Argo CD CRDs are too large for the client-side `last-applied` annotation. The Application takes over the resources in `devops-app` if `kubectl apply -k k8s/base/` created them earlier; it does not recreate them.

## Open the Argo CD screen

```sh
kubectl -n argocd port-forward svc/argocd-server 8443:443
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 --decode; echo
```

Open <https://localhost:8443> (self-signed certificate) and log in as `admin` with that password. Port `8443`, because the k3d mapping holds `8080`. Do not paste the password into chat or captures.

## Live demo

One push deploys. The personal repo's CI tests, builds and publishes `ghcr.io/vantrangradion/devops-app:<commit SHA>`, then its **Deploy** job commits that tag to `k8s/base/deployment.yaml` as `github-actions[bot]`. Argo CD sees the new commit and rolls the pods. The image carries its commit, so `/api/work` answers with `"version": "<commit SHA>"`: `curl` shows the old version before the push and the new one after.

```
git push (app/) ──▶ CI: test, build, publish ──▶ Deploy job: k8s/base tag ──▶ Argo CD: Refresh ──▶ pods roll ──▶ curl shows new version
                     ~3.5 min                      (in the same run)           ~30 s
```

Measured on 11 November 2026: push to Deploy job done in 3 min 26 s, then Refresh to `Synced`/`Healthy` with both pods rolled in 29 s ([captures/one-push-deploy](https://github.com/gradionhq/2026-devops-assignment-intern/tree/main/common-deliverable/shared-system/captures/one-push-deploy/README.md)).

Before the talk:

1. Open the Argo CD screen and Grafana's `devops-app` dashboard. The Application should be **Synced** and **Healthy**.
2. Have a small code change ready in a clone of the personal repo, for example a comment in `app/main.go`.

During the talk:

1. `curl -s "http://localhost:8080/api/work?ms=0"` and point at `version`.
2. In the personal repo, commit the change and `git push` to `main`. Open the repo's **Actions** tab to show the run starting.
3. While CI runs (about 3 minutes), carry on with the talk: the app, Argo CD, the Grafana baseline.
4. When the run's **Deploy** job is green, a `deploy: run devops-app <sha>` commit is on `main`. In Argo CD, press **Refresh** on `devops-app`. Without it, Argo CD polls Git about every 3 minutes.
5. Watch the new ReplicaSet appear and the pods roll; Grafana's **Ready replicas** stays at 2.
6. `curl` again: `version` is now the commit you pushed.

If CI is slow or fails, fall back to the hand edit: change the tag in `k8s/base/deployment.yaml` to an image that already exists, push, Refresh.

Afterwards, copy the same image pin into the team repo's [`common-deliverable/shared-system/k8s/deployment.yaml`](https://github.com/gradionhq/2026-devops-assignment-intern/blob/main/common-deliverable/shared-system/k8s/deployment.yaml) in a PR there, so the deliverable matches what ran.

## Check

| Check | Command | Expected |
| :--- | :--- | :--- |
| Argo CD | `kubectl -n argocd get pods` | 4 pods `Running` |
| Application | `kubectl -n argocd get application devops-app` | `Synced`, `Healthy` |
| Self-heal | `kubectl scale deploy/devops-app -n devops-app --replicas=1`, then read `.spec.replicas` | Back to `2` within seconds |
| Deploy | Push an image change, press Refresh | Both pods on the new image, `/api/work` returns `200` |

The rehearsal on 10 November 2026 is in [captures/gitops](https://github.com/gradionhq/2026-devops-assignment-intern/tree/main/common-deliverable/shared-system/captures/gitops/README.md), and the first one-push deploy on 11 November in [captures/one-push-deploy](https://github.com/gradionhq/2026-devops-assignment-intern/tree/main/common-deliverable/shared-system/captures/one-push-deploy/README.md).

## Fallback: deploy without Argo CD

If Argo CD fails during the talk, delete the Application, which leaves the app running because the Application has no `resources-finalizer`, then apply the manifests by hand:

```sh
kubectl -n argocd delete application devops-app
kubectl apply -k k8s/base/
```

Leave the Application in place otherwise: with self-heal on, Argo CD reverts anything `kubectl apply -k k8s/base/` changes that differs from the personal repo.

## Delete

```sh
kubectl -n argocd delete application devops-app
kubectl delete -f gitops/project.yaml
kubectl delete -k gitops/install
```

The Application has no `resources-finalizer`, so deleting it leaves the app running.
