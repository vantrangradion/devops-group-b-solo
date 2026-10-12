# User Stories: Checkpoint 2 (Pillar 2: K8s Infrastructure & Network)

**Source:** [PROJECT_PLAN.md](PROJECT_PLAN.md), Sections 2, 3, 3.1, 4.1–4.4, 6, 7; [CP1_USER_STORIES.md](CP1_USER_STORIES.md) (app contract, official image)  
**Pillar owner / decision maker:** Engineer B  
**Window:** Sat 10 Oct (solo, on your own Pillar 1 image) → switch to the official image Mon 12, 11:30–13:00 → **★ CP2: Mon 12 Oct, 13:00** → merge by 16:00 → shared cluster up and official captures taken by Mon 12 Oct, 17:00 → captures PR merged by Mon 12 Oct, EOD  
**Rubric steps covered:** Step 2 (cluster, manual `kubectl apply` deploy into its own namespace, ConfigMap/Secret injection, `kubectl` and `curl` captures) and Step 3 (service reached through a tunnel from a device that is not the cluster machine)

---

## 1. Checkpoint Goal

> By 17:00 on Mon 12 Oct, `main` holds one set of Kubernetes manifests that the team has agreed on, and the shared `devops-shared` cluster on the demo host runs the official image from them. Two replicas sit behind a ClusterIP Service in `devops-app`, get their config from a ConfigMap and a Secret, and answer a phone on mobile data through a tunnel. That cluster is what Pillar 3 adds monitoring to and what the 16 Oct demo runs on.

**Success measures**
- All 3 engineers have a `p2` branch with a pushed `cp2` tag, a working (or documented) deploy on their own cluster, and a "Pillar 2" section in `NOTES.md`.
- Before CP2 starts, every personal Deployment runs the official image (CP1-14), so the three builds differ only in their manifests and tunnel.
- CP2 starts at 13:00 and finishes within its 75-minute timebox, with a recorded decision.
- `kubectl apply` of `k8s/base/` from `main` works on a fresh cluster in one command, with no errors.
- The four graded Step 2–3 captures are taken from the shared cluster by 17:00 on Mon 12 and merged to `main` by end of day.
- Everyone starts Pillar 3 from the team repo's `main` with the shared manifests applied on their own cluster.

---

## 2. Personas

| Persona | Who | What they need from CP2 |
| :--- | :--- | :--- |
| **Solo Builder** | Each of Eng A, B, C | Fixed names (namespace, Service, port, labels) and a clear done-check, so the weekend build needs no checkpoint to stay on track and the three builds are easy to compare. |
| **Pillar Owner** | Eng B | Comparable manifests and clear criteria, so the decision takes 5 minutes, and a PR that reviewers can approve quickly now that `main` needs 2 approvals. |
| **Downstream Builder** | Eng C (Pillar 3) | A Service with a named port and stable labels that a `ServiceMonitor` can select, and pods that stay up under k6 load. |
| **Demo Operator** | Whoever runs the 16 Oct demo on the demo host | A shared cluster and a tunnel that survive until Friday, and a documented way to bring them back if they don't. |
| **Grader / Stakeholder** | Internship Supervisor (grader); Thang Le (Track Leader) | Evidence that Steps 2–3 are real, run on the shared system, use config kept outside the image, and are reached from a separate device. |

---

## 3. Story Map

| ID | Story | Persona | Priority | Pts | Owner | Due |
| :--- | :--- | :--- | :---: | :---: | :--- | :--- |
| **Epic A: Solo Build** (each engineer, on `p2` in their personal repo) | | | | | | |
| CP2-01 | `p2` branch and local k3d cluster | Solo Builder | Must | 1 | Each | Sat 10 |
| CP2-02 | Namespace, ConfigMap, and Secret | Grader | Must | 2 | Each | Sat 10 |
| CP2-03 | Hardened Deployment (2 replicas, probes, limits) | Demo Operator | Must | 3 | Each | Sun 11 |
| CP2-04 | ClusterIP Service with a named port | Downstream Builder (Eng C) | Must | 1 | Each | Sun 11 |
| CP2-05 | One-command deploy and `curl` check | Grader | Must | 2 | Each | Sun 11 |
| CP2-06 | Self-healing, rolling update, and rollback checks | Grader | Should | 2 | Each | Sun 11 |
| CP2-07 | Remote access through a tunnel, verified on a phone | Grader | Must | 3 | Each | Sun 11 |
| CP2-08 | Pillar 2 learning notes and `cp2` tag | Grader | Must | 1 | Each | Sun 11 EOD (tag: Mon 12, 13:00) |
| *Entry criterion: CP1-14, switch to the official image (Mon 12, 11:30–13:00)* | | | | | | |
| *CP2 session (Mon 12, 13:00–14:15): demos and decision, see Section 5* | | | | | | |
| **Epic B: Merge & Shared Cluster** (Mon 12, 14:15–17:00) | | | | | | |
| CP2-09 | Checkpoint PR into `main` (with decision record) | Downstream Builder | Must | 3 | Eng B (A, C review) | Mon 12, 16:00 |
| CP2-10 | Shared cluster on the demo host | Demo Operator | Must | 2 | Demo host's owner (Eng B accountable) | Mon 12, 17:00 |
| CP2-11 | Graded Step 2–3 captures | Grader | Must | 1 | Eng B (A, C review) | Mon 12, 17:00 (taken); EOD (merged) |
| CP2-12 | Pillar 3 handoff | Downstream Builder (Eng C) | Must | 1 | Eng C briefs; each re-applies | Mon 12, 14:10 (briefing); from 16:00 (re-apply) |
| **Epic C: Research** | | | | | | |
| CP2-13 | Research seeds for 3.1b / 3.1d | Grader | Should | 2 | Eng B | Tue 13 |

**Load per person (Must only):** about 14 pts for each Solo Builder, all of it on the weekend except CP2-08's Mon 12 items (official image switch, `cp2` tag) and CP2-12. Eng B also carries about 6 pts of owner work plus running the CP2 session, all on Mon 12 afternoon, and coaches the others on Pillar 2 over the weekend (rule 6).

**Weekend has no session.** The only sync point is the async check-in in the team chat on Sun 11, 20:00 (step reached, blockers). The 2-hour stuck-rule (rule 7) is the main safety net: blocked for more than 2 hours means pairing with Eng B straight away. On Sun evening, Eng B skims the other two `p2` branches.

---

## 4. Stories & Acceptance Criteria

Criteria that describe behavior are written as **Given / When / Then** scenarios. Criteria that only check that something exists or was done are plain checklist items. `<img>` means `ghcr.io/<github-user>/devops-app:<sha>` on the weekend and `ghcr.io/<org>/devops-app:<main-sha>` from Mon 12, 11:30 (CP1-14).

### Epic A: Solo Build (each engineer, Sat 10 – Sun 11 Oct, on `p2` in their personal repo)

#### CP2-01: `p2` branch and local k3d cluster
**As a** Solo Builder, **I want** my own cluster and a branch for Pillar 2 **so that** I can build Steps 2–3 without touching my Pillar 1 state on `main`.

**Acceptance criteria**
- [ ] A `p2` branch is created from my own personal `main` and pushed. `main` keeps only Pillar 1, so `main` at its `cp1` tag still shows the Pillar 1 state.
- [ ] The cluster is created with `k3d cluster create pow-<name>` (single server, no agents needed). The exact command, including any port mapping used by CP2-07, is recorded in the `p2` `README.md`.
- [ ] **Given** the cluster was just created, **when** I run `kubectl get nodes`, **then** every node is `Ready`.
- [ ] **Given** the cluster, **when** I run `kubectl get storageclass`, **then** `local-path` (`rancher.io/local-path`) is listed as default. This is noted for 3.1d (RWO only); the app itself needs no PVC. The brief says k3d ships this provisioner as the `standard` StorageClass, so the actual name is recorded for 3.1d, which asks the team to check the brief's claims.

#### CP2-02: Namespace, ConfigMap, and Secret
**As a** Grader, **I want** the app's settings supplied by a ConfigMap and a Secret in their own namespace **so that** I can see config kept outside the image, as Step 2 requires.

**Acceptance criteria**
- [ ] `k8s/base/namespace.yaml` creates the namespace `devops-app` (canonical name, rule 4).
- [ ] `k8s/base/configmap.yaml` sets `APP_ENV` and `MESSAGE` in `devops-app`. `MESSAGE` holds a value that does not appear anywhere in the image's defaults (e.g. `hello from configmap`), so a `curl` proves the ConfigMap, not the default, was used.
- [ ] `k8s/base/secret.yaml` sets `API_KEY` in `devops-app` under `stringData:`, as an obviously fake placeholder (e.g. `demo-not-a-real-key`), so it is safe in a public repo. It does not use `data:`: gitleaks, run by the required `Security` check, flags any Secret with a base64 value under `data:`, even `changeme`.
- [ ] A comment in `secret.yaml` states that the API server stores Secret values only base64-encoded, which is an encoding, not encryption, and that a real secret would be created out of band (`kubectl create secret`) or stored encrypted (Sealed Secrets, SOPS). This feeds 3.1d.
- [ ] **Given** my `p2` branch, **when** I run the same gitleaks scan as CI locally (same image and arguments as the `Security` job), **then** it reports no leaks. CI runs only on `main` and on PRs, so `p2` is never scanned there, and the first CI scan would otherwise be the CP2-09 PR. There a finding cannot be fixed by a later commit: gitleaks scans the full history, and PRs are merged with merge commits.
- [ ] Every object carries the label `app.kubernetes.io/name: devops-app`.
- [ ] **Given** the objects are applied, **when** I run `kubectl get configmap,secret -n devops-app`, **then** both are listed.

#### CP2-03: Hardened Deployment (2 replicas, probes, limits)
**As a** Demo Operator, **I want** a Deployment that keeps two healthy, locked-down pods running **so that** the demo survives a pod failure and the k6 load in Pillar 3.

**Acceptance criteria**
- [ ] `k8s/base/deployment.yaml` runs `replicas: 2` in `devops-app`, with selector and pod labels `app.kubernetes.io/name: devops-app`.
- [ ] The image is pinned to a full commit SHA (`<img>`), never `:latest`. No `imagePullSecrets`: the GHCR package is public.
- [ ] The container port is `8080`, named `http`.
- [ ] `APP_ENV` and `MESSAGE` come from the ConfigMap and `API_KEY` from the Secret, through `env` (`configMapKeyRef` / `secretKeyRef`) or `envFrom`.
- [ ] Both a `readinessProbe` and a `livenessProbe` call `GET /health` on port `http`. Probe timings are the builder's choice and are compared at CP2.
- [ ] Every container sets CPU and memory `requests` and `limits`. The values, and why, go in the decision record at CP2. The memory limit leaves headroom for k6 load (Pillar 3).
- [ ] Every build meets this minimum security baseline, which carries CP1-08's non-root image into the cluster. Anything a builder adds beyond it is compared at CP2.
  - the pod `securityContext` sets `runAsNonRoot: true`, `runAsUser: 65532`, and `seccompProfile.type: RuntimeDefault`;
  - the container `securityContext` sets `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, and `capabilities.drop: [ALL]`. These three fields exist only at container level. In the pod `securityContext` they are unknown fields: `kubectl apply` rejects them (strict validation), `--validate=warn` applies the rest with only a warning, and `--validate=false` drops them silently, so the setting looks applied but isn't;
  - `automountServiceAccountToken: false`, because the app never calls the Kubernetes API.
- [ ] **Given** the Deployment is applied, **when** I run `kubectl get deploy -n devops-app`, **then** it shows `2/2` ready within 60 s.
- [ ] **Given** a running pod, **when** I run `kubectl logs` on it, **then** the `API_KEY` value does not appear.

#### CP2-04: ClusterIP Service with a named port
**As** Eng C (Pillar 3), **I want** a Service with a stable name, named port, and labels **so that** my `ServiceMonitor` can find the pods without changes to Pillar 2 (risk **R3**).

**Acceptance criteria**
- [ ] `k8s/base/service.yaml` creates a `ClusterIP` Service named `devops-app` in `devops-app`, port `8080`, `targetPort: http`.
- [ ] The Service port is named `http`. A `ServiceMonitor` endpoint selects a port by this name.
- [ ] The Service itself carries the label `app.kubernetes.io/name: devops-app` (a `ServiceMonitor` selects Services by their labels, not pod labels).
- [ ] **Given** the Deployment and Service are applied, **when** I list the ready condition of each endpoint in the Service's EndpointSlices (`kubectl get endpointslices -n devops-app -l kubernetes.io/service-name=devops-app -o jsonpath='{range .items[*].endpoints[*]}{.addresses[0]} {.conditions.ready}{"\n"}{end}'`), **then** exactly 2 endpoints are `true`. The default `kubectl get endpointslices` output also lists endpoints that are not ready.

#### CP2-05: One-command deploy and `curl` check
**As a** Grader, **I want** the whole app deployed by hand with one `kubectl apply` and a real response from it **so that** Step 2 is proven, not described.

**Acceptance criteria**
- [ ] The `p2` `README.md` has a "Deploy" section with the exact commands to deploy, check, and delete the app.
- [ ] **Given** a fresh cluster, **when** I run the single documented deploy command (e.g. `kubectl apply -f k8s/base/`), **then** every object is created with no errors on the first run. `kubectl apply -f <dir>` applies files in file-name order, so `configmap.yaml` runs before `namespace.yaml` and fails on a fresh cluster. The fix is the builder's choice (e.g. a `kustomization.yaml` and `kubectl apply -k`, or applying the namespace first) and is compared at CP2.
- [ ] **Given** the objects already exist, **when** I run `kubectl diff` on the same manifests, **then** it exits `0` (no changes: the declarative model). Running the deploy command again reports `unchanged` for every object except the Secret, which always shows `configured`: `stringData` is write-only and never stored, so each apply re-sends it.
- [ ] **Given** the app is deployed, **when** I `curl` `/api/work` through the Service (e.g. after `kubectl port-forward -n devops-app svc/devops-app 8080:8080`), **then** the response is `200` and contains the ConfigMap's `MESSAGE` value.
- [ ] **Given** the app is deployed, **when** I `curl` `/metrics` through the Service, **then** `http_requests_total` is present (what Pillar 3 will scrape).
- [ ] Personal captures of `kubectl get nodes`, `kubectl get all -n devops-app`, and the `curl` response are in `captures/` on `p2`.

#### CP2-06: Self-healing, rolling update, and rollback checks (Should)
**As a** Grader, **I want** to see the orchestrator do what a bare `docker run` cannot **so that** the 3.1d write-up and the demo are backed by real runs.

**Acceptance criteria**
- [ ] **Given** 2 running pods, **when** I `kubectl delete` one, **then** a replacement is `Ready` within 60 s and `kubectl get deploy` returns to `2/2`.
- [ ] **Given** a loop of `curl` requests to `http://devops-app:8080/api/work` run from a pod in the same namespace (`kubectl run -n devops-app` with a pinned curl image, e.g. `curlimages/curl:8.11.1`; without `-n` the pod lands in `default`, where the short name `devops-app` does not resolve), **when** I change `MESSAGE` and run `kubectl rollout restart deployment/devops-app -n devops-app`, **then** the new pods return the new value, and the number of failed requests in the loop is recorded in `NOTES.md`.
  - The loop does not use `kubectl port-forward`, which exits when its pod is deleted.
  - Env vars from a ConfigMap are read only at container start, so the restart is needed.
  - The app stops accepting connections as soon as it receives `SIGTERM`, so a few requests can fail while endpoints update. A `lifecycle.preStop.sleep` hook is one fix to try. It is built into Kubernetes, so it works on distroless, which has no `sleep` binary, but it needs Kubernetes 1.30 or later (check `kubectl version`).
- [ ] **Given** a rollout to an image tag that does not exist, **when** I watch the pods until the new one shows `ImagePullBackOff`, **then** the old pods keep serving, and `kubectl rollout undo` restores the previous ReplicaSet. With 2 replicas, the default strategy (25% / 25%) already rounds to `maxUnavailable: 0`, `maxSurge: 1`. `kubectl rollout status` would block until `progressDeadlineSeconds` (600 s by default), so it is not used to wait here. After the undo, re-run the deploy command on the committed manifests (the bad tag is never saved to the file), so the cluster matches git again (`rollout undo` does not update the `last-applied-configuration` that `kubectl apply` compares against).
- [ ] The results, including anything that did not behave as expected, are written in the "Pillar 2" section of `NOTES.md`.

#### CP2-07: Remote access through a tunnel, verified on a phone
**As a** Grader, **I want** a real request from a device that is not the cluster machine **so that** Step 3 is proven.

**Acceptance criteria**
- [ ] The tunnel uses `cloudflared` or `ngrok` on a free tier. No tokens, credentials, or tunnel config with secrets are committed.
- [ ] The `p2` `README.md` has a "Remote access" section with the exact commands, from cluster to public URL, including how the Service is reached on the host. There are two paths:
  - `kubectl port-forward`;
  - a k3d port mapping (`-p`) to a second Service of type `NodePort` or `LoadBalancer`, next to the CP2-04 `ClusterIP` Service. A port mapping reaches nothing on its own, and Ingress is out of scope. The second Service must not use port 80 or 443: k3s installs Traefik as a LoadBalancer on those ports, so a second LoadBalancer there stays `<pending>` and requests get a `404` from Traefik. For example, use a `LoadBalancer` on port 8080 with `-p 8080:8080@loadbalancer`, or a `NodePort` 30080 with `-p 30080:30080@server:0`.
- [ ] The section states the trade-off of the chosen path:
  - `kubectl port-forward svc/...` sends every connection to a single pod, so traffic through it is not spread across both replicas. It stops when the terminal closes, and it exits whenever that pod is deleted or restarted (a liveness restart, a rollout, CP3 work), which silently breaks the tunnel.
  - A k3d port mapping spreads traffic across both replicas and survives pod restarts. It is easiest to set when the cluster is created (`k3d cluster edit --port-add` exists but is experimental).
- [ ] The section states whether the public URL changes on restart (e.g. a `cloudflared` quick tunnel gets a new `trycloudflare.com` URL each time), so the Demo Operator knows what to re-share on demo day (risk **R4**).
- [ ] The section notes that the public URL also exposes `/metrics`, which is acceptable for a demo of a toy app, and that the tunnel is stopped when not in use.
- [ ] **Given** the tunnel is running, **when** I open `<public-url>/api/work` on a phone with Wi-Fi turned off (mobile data only), **then** the page shows the ConfigMap's `MESSAGE` value.
- [ ] **Given** the tunnel is running, **when** I request `<public-url>/health` from the phone, **then** it returns `200`.
- [ ] A phone screenshot showing the URL bar, the response, and the mobile-data indicator (no Wi-Fi icon) is in `captures/` on `p2`.

#### CP2-08: Pillar 2 learning notes and `cp2` tag
**As a** Grader, **I want** each engineer's Pillar 2 learning recorded **so that** individual work is visible and the "Challenges & Lessons Learned" slide has real material.

**Acceptance criteria**
- [ ] By Sun 11 EOD, `p2` is pushed and `NOTES.md` on it has a "Pillar 2" section with 3–5 bullets: what broke, how it was fixed, and what you would keep.
- [ ] Anything unfinished is written down as "tried X, failed at Y" (rule 8).
- [ ] Each engineer posts their status in the team chat at the Sun 11, 20:00 async check-in.
- [ ] *(Due Mon 12, 13:00, after CP1-14.)* My `p2` Deployment runs the official image `ghcr.io/<org>/devops-app:<main-sha>` and still passes the CP2-05 `curl` check; that change is committed to `p2`.
- [ ] *(Due Mon 12, 13:00.)* At 13:00 on Mon 12 Oct, my `p2` branch is tagged `cp2` and the tag is pushed. It marks the Pillar 2 state that gets demoed and is kept as evidence of individual work (risks **R6**, **R8**).

---

### Epic B: Merge & Shared Cluster (Mon 12 Oct, 14:15–17:00)

#### CP2-09: Checkpoint PR into `main` (with decision record)
**As a** Downstream Builder, **I want** the chosen manifests merged into the team repo as one focused PR, with the reasons written down **so that** Pillar 3 builds on a single shared baseline and the team can defend its choices.

**Acceptance criteria**
- [ ] Eng B copies the chosen files from the personal repos (at their `cp2` tags) onto a branch in the team repo and opens one PR containing only:
  - `k8s/base/` (`namespace.yaml`, `configmap.yaml`, `secret.yaml`, `deployment.yaml`, `service.yaml`, plus any file the chosen deploy command or access path needs, such as `kustomization.yaml` or a second `NodePort` or `LoadBalancer` Service);
  - `README.md` updated with the "Deploy" and "Remote access" sections from CP2-05 and CP2-07;
  - the `## CP2` section of `docs/pow-merge-decisions.md`.
- [ ] The Deployment's image is `ghcr.io/<org>/devops-app:<main-sha>`, where `<main-sha>` is the latest image-building commit on `main` when the PR opens (CP1-14).
- [ ] The `## CP2` section lists: whose version was chosen for each artifact (manifests, deploy command, tunnel tool and path), what was taken from the others, why, which alternatives were rejected and why, the resource values, probe settings, and update strategy chosen, the demo host, and any objections noted in the CP2 session.
- [ ] No personal repo is merged wholesale (`NOTES.md` and personal `captures/` stay in the personal repos).
- [ ] The PR description links the source personal repo and `cp2` commit for each artifact.
- [ ] The required checks `Lint`, `Test`, `Security`, and `Build` pass on the PR. Eng A and Eng C each approve it (branch protection requires 2 approvals, and Eng B cannot approve their own PR).
- [ ] The PR is merged with a merge commit. Because the PR touches `k8s/` (not ignored by CI's `paths-ignore`), the merge to `main` runs the full pipeline and publishes a new `:<sha>` and `:latest`. The manifests keep the SHA from the PR, which is still valid; nobody needs to redeploy.
- [ ] **Given** a checkout of the PR branch, **when** a reviewer runs the README's deploy command on a fresh `k3d` cluster before approving, **then** CP2-05's checks pass. Checking before the merge avoids a fix PR in a tight afternoon.

#### CP2-10: Shared cluster on the demo host
**As a** Demo Operator, **I want** the shared cluster built from `main` on the machine we demo from **so that** the graded captures and the 16 Oct demo come from the team system, not one person's sandbox.

**Acceptance criteria**
- [ ] The demo host is the laptop with the most free RAM. Eng B picks it in the CP2 "Decide" step, so it is named in the `## CP2` decision record that merges with CP2-09. Its owner builds the shared cluster; Eng B is accountable.
- [ ] On the demo host, `k3d cluster create devops-shared` (with any port mapping the chosen remote-access path needs) runs, then the README's deploy command from a clean checkout of `main`.
- [ ] **Given** the shared cluster, **when** I run the CP2-05 and CP2-07 checks against it, **then** they all pass, including the phone on mobile data.
- [ ] The cluster stays up until the 16 Oct demo. The demo host's sleep setting is turned off while it is plugged in, and the README has the commands to bring the cluster and tunnel back if the laptop restarts.
- [ ] **Given** any pod restart or rollout on the shared cluster (including CP3 work), **when** it completes, **then** the tunnel is re-checked from a phone, because a `port-forward`-based tunnel exits with its pod. The README's recovery commands include this re-check.

#### CP2-11: Graded Step 2–3 captures
**As a** Grader, **I want** Step 2–3 evidence taken from the shared cluster **so that** the submission reflects the team system.

**Acceptance criteria**
- [ ] `captures/step-2-k8s-nodes.txt` holds `kubectl get nodes -o wide` from `devops-shared`.
- [ ] `captures/step-2-k8s-resources.txt` holds `kubectl get all -n devops-app` and `kubectl get configmap,secret -n devops-app` (`get all` does not list ConfigMaps or Secrets). No Secret values are printed.
- [ ] `captures/step-2-curl-response.png` shows the `curl` command and a `200` response containing the ConfigMap's `MESSAGE`.
- [ ] `captures/step-3-tunnel-phone-access.png` is the CP2-07 phone screenshot, taken against the shared cluster.
- [ ] Branch protection is on, so the captures reach `main` through a PR with 2 approvals. A PR that only touches `captures/` still runs the required checks (the PR trigger has no path filter), but its merge does not build a new image.
- [ ] The captures are taken by 17:00 and merged to `main` by end of day Mon 12. The Step 2 and Step 3 slides in the Midway Deck use them.

#### CP2-12: Pillar 3 handoff
**As** Eng C, **I want** everyone to start Pillar 3 from the same manifests and know the common monitoring pitfalls **so that** CP3 compares like for like and nobody loses Tuesday to "No Data" (risk **R3**).

**Acceptance criteria**
- [ ] In the last 5 minutes of the CP2 session (14:10–14:15), Eng C shares the known Pillar 3 pitfalls: the `release` label the `kube-prometheus-stack` Prometheus uses to select a `ServiceMonitor`, the named Service port (`http`), and `namespaceSelector.matchNames: [devops-app]`.
- [ ] From 16:00, each engineer creates `p3` from the team repo's `main` (`git fetch upstream`, branch from `upstream/main`), or rebases work-ahead onto it (rule 1).
- [ ] **Given** my own `pow-<name>` cluster, **when** I re-apply `k8s/base/` from `main`, **then** CP2-05's checks pass on it (rule 4).
- [ ] Each engineer posts "p3 ready" in the team chat, or their blocker (2-hour stuck-rule applies).

---

### Epic C: Research

#### CP2-13: Research seeds for 3.1b / 3.1d (Should)
**As a** Grader, **I want** CP2 experience turned into research notes right away **so that** the 40% research score draws on hands-on evidence, not only theory.

**Acceptance criteria**
- [ ] `docs/02-gitops-remote-access.md` and `docs/04-kubernetes-orchestration.md` each have an outline and at least 3 official-doc citations (kubernetes.io, k3d.io, Cloudflare or ngrok docs, Argo CD or Flux docs).
- [ ] Each document includes at least one concrete CP2 finding, e.g. the `kubectl apply` ordering failure, `port-forward` pinning traffic to one pod, gitleaks flagging base64 `data:` in `secret.yaml`, the Secret showing `configured` on every apply because `stringData` is write-only, or the CP2-06 rollout results.
- [ ] `docs/02` compares the three tunnel tools (cloudflared, ngrok, Tailscale) and states which one fits a public demo and which a team-internal service. cloudflared and ngrok draw on what the solo builds found; Tailscale is covered from its docs only, since no build uses it.
- [ ] The outlines also seed the brief's other asks: `docs/04` lists the object kinds the app needed beyond the brief's list (e.g. EndpointSlice, ServiceAccount) and what a namespace does and does not separate; `docs/02` explains how the image tag is updated after a CI build, contrasting the CP1-14 manual SHA bump with the push model (the CI job holds cluster credentials and runs `kubectl apply`) and the pull model (Argo CD or Flux syncs from git).
- [ ] Both land in the team repo through a docs PR after CP2-09.

---

## 5. CP2 Session Agenda (Mon 12 Oct, 13:00–14:15)

The session follows Plan Section 4.1 and is run by Eng B as Pillar Owner. Its output is the decision that CP2-09 merges and records.

**Demos (3 × 12 min)**
- Each engineer shows, live on their own cluster: `kubectl get all -n devops-app`, the one-command deploy, the `curl` check with the ConfigMap `MESSAGE`, the official image tag in the Deployment, the tunnel answering on a phone over mobile data, and one key choice or one failure.
- What gets demoed is the `p2` branch at its `cp2` tag (CP2-08). No live fixes.
- Unfinished work is presented as-is (rule 9: no catch-up debt).

**Compare (15 min)**
- Builds are compared against the 5 criteria in Plan Section 4.1: works end to end, simplicity, fit with Section 3.1, security (non-root, pinned image, no real secrets in git), and readiness for Pillar 3 and the demo.
- Specific points to settle: the deploy command (fresh-cluster ordering), resource values, probe settings, the rolling update strategy and image pull policy, the host access path (`port-forward` vs. k3d port mapping), and the tunnel tool.
- Objections from Eng A or Eng C are noted for the decision record, not debated beyond this slot.

**Decide (5 min)**
- Eng B decides for each artifact and may combine parts from different builds. It is not a vote.
- Eng B also picks the demo host (CP2-10), so it goes into the decision record.

**Record (5 min)**
- Eng B drafts the `## CP2` section of `docs/pow-merge-decisions.md`, which goes into the CP2-09 PR.

**Pillar 3 pitfalls (5 min, 14:10–14:15)**
- Eng C shares the pitfalls in CP2-12. The agenda adds up to 66 of the 75 minutes; the rest is buffer before this slot.

---

## 6. CP2 Definition of Done

- [ ] All **Must** stories accepted by Eng B. Eng A and Eng C approve the CP2-09 and CP2-11 PRs.
- [ ] `main` holds `k8s/base/` and README deploy and remote-access instructions that work on a fresh cluster in one command.
- [ ] The shared `devops-shared` cluster on the demo host runs 2 ready pods of the official image in `devops-app`, as numeric UID `65532`, with config from a ConfigMap and a Secret.
- [ ] The app answers through the tunnel on a phone over mobile data.
- [ ] Graded Step 2–3 captures and the CP2 decision record are on `main`.
- [ ] Three personal public repos have a `p2` branch with progressive commits, a pushed `cp2` tag, and a "Pillar 2" section in `NOTES.md`.
- [ ] Every engineer has a `p3` branch from the team repo's `main` and the shared manifests applied on their own cluster.
- [ ] No real secrets or tunnel credentials in git.

## 7. Out of Scope for CP2

- Prometheus, Grafana, the `ServiceMonitor`, and k6 (CP3). CP2 only guarantees the Service name, the named port, and the labels they rely on.
- GitOps with Argo CD or Flux (Bonus 1, Sprint 2). CP2 deploys by hand with `kubectl apply`, as the brief requires.
- Bonus 2 (multi-machine cluster). The brief asks for nodes on separate machines, so extra k3d agents on one laptop do not count.
- Ingress controllers, TLS, and custom domains. A free tunnel is enough (brief Section 7).
- PersistentVolumeClaims. The app is stateless; storage limits (RWO) are covered in the 3.1d write-up only.
- App changes. If the official image blocks Pillar 2, Eng A ships a fix PR (rule 8 exception), as in CP1-14.
