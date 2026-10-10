# Assignment 3 — Group B: DevOps & Observability Research

**For:** Group B interns (team assignment, 3 people)
**Supervisor reviews this document too.** Use it to check progress and to run the Friday reviews.

**Duration:** 1 month (Mon 5 Oct to Fri 30 Oct)  
**Half-way presentation:** Fri 16 Oct. **Final presentation:** Fri 30 Oct.  
**Workspace:** a public GitHub repo the team sets up, plus your own machines for the cluster  
**Real stakeholder:** Thang Le

---

## Teaching Intent

- **Fill the school-to-industry gap.** School teaches you to write code that works on your machine. It rarely teaches CI/CD, standard deployment, monitoring, or how a service is operated after it ships (live ops). This assignment is where you close that gap — the part of the job that starts *after* the feature is written.
- **Ship the whole loop, not just the code.** With AI support, writing the feature is no longer the hard part. A developer now owns the full loop: **build the product → deploy it in a managed, repeatable way → prove the quality is good → keep improving it from real metrics.** This loop is Gradion's standard today. It is not an extra step for a DevOps person to handle later.
- **Keep up with the trend.** CI/CD and observability keep changing. Three areas are moving in 2025–2026: supply-chain provenance (attestations, SBOMs); GitOps becoming the default deploy model; and the same trace and metric model now being used for browser monitoring (RUM). Track what is current, not only the textbook version.
- **Hands-on first, lessons learned, then ready to apply.** Build and run the pipeline yourself. Write down what broke and what surprised you, and turn each one into a lesson. The goal: later you can decide when a CI, orchestration, or observability tool fits a new project, not only describe the concepts.

When your Supervisor grades this work, a real pipeline run you captured yourself is worth at least as much as a well-designed slide deck.

---

## 1. Background

Every engineering team has to answer one question: how does code get from a commit to a running, monitored service? This assignment is general DevOps and observability research: CI/CD, GitOps, containers, orchestration, and monitoring. You research all of them, and you build and run the path commit → CI → image → deploy → cluster → dashboards yourself, on your own machines. Your team must be able to explain that path to the other interns and show working proof of it.

## 2. Objective

**Motivation.** Writing the feature is no longer the hard part. The hard part is getting a change safely to real users and knowing how it behaves once it is running. Your team's job is to build that path yourselves, from commit to a monitored service, so you can judge when each tool fits and prove it works.

Produce a findings write-up and a presentation. They must cover:

- how a real CI/CD pipeline builds a change, and which checks gate it;
- what makes a container image production-ready;
- how GitOps deploys that image to a cluster;
- the core concepts of Kubernetes orchestration;
- how OpenTelemetry, Prometheus and Grafana turn a running workload into metrics you can act on;
- how Real User Monitoring and Core Web Vitals measure the same system from the user's browser.

You must also build and operate a real pipeline yourself: commit → CI → image → deploy (by hand with `kubectl apply`) → running cluster → tunnel → dashboards.

## 3. Requirements

### 3.1 Research topics

Ground truth: `https://docs.github.com/en/actions`, `https://docs.docker.com/`, `https://kubernetes.io/docs/`, `https://opentelemetry.io/docs/` (signal status: `https://opentelemetry.io/status/`). The version numbers, "stable since" dates, and vendor claims below were correct when we wrote this document. **Check each one against the linked docs, and cite the page you checked.** State in your write-up whether anything has changed by the Fri 16 Oct half-way session, and again by the Fri 30 Oct final.

**a) CI/CD fundamentals & one real system**
What a CI/CD pipeline is; the standard stages (build, test, lint, deploy) and what each stage is actually gating. Go deep on **one** real system — GitHub Actions recommended: workflow YAML anatomy (triggers, jobs, steps, runners). Cover the 2025–2026 supply-chain additions, and say which problem each one solves:

- **Immutable Actions** — actions are published as OCI packages to `ghcr.io`, so a pinned version cannot be changed later. This shipped recently, so check the current rollout status at `https://github.blog/changelog/`.
- **Artifact Attestations** — build provenance in the SLSA and in-toto formats, signed with Sigstore. Anyone can verify it after the build has finished.

Give GitLab CI and Jenkins a short comparison paragraph. Note that the proof of work in 3.2 uses GitHub Actions.

**b) GitOps & continuous delivery**
CI stops at a tested image. CD is what gets that image onto a cluster. Cover the two models:

- **push** — the CI job holds cluster credentials and runs `kubectl apply`.
- **pull / GitOps** — an agent inside the cluster watches a git repo and keeps changing the cluster until it matches that repo.

Name **Argo CD** and **Flux** as the two standard GitOps agents. Pick one and go deep. Core ideas:

- git is the single source of truth;
- the desired state lives in a repo you can review and roll back;
- if the cluster stops matching the repo (this is called drift), the agent detects it and corrects it;
- sync can be **manual** (someone approves each change) or **automatic** (every merge deploys).

Explain how the image tag is updated after a CI build. This is the connection point between topic 3.1a and this one, and many teams make mistakes here.

Also cover how you reach a service running on a laptop cluster from outside that laptop: **cloudflared**, **ngrok**, and **Tailscale**. Say what each one does (a public tunnel, or a private mesh network), and which one fits a public demo and which fits a team-internal service. Free tiers only.

**c) Containers & production-ready images**
What a container is and why it standardizes deployment. How a `Dockerfile` builds an image (layers, caching). What makes an image production-ready, not only small:

- a **distroless** or minimal base image — no shell and no package manager inside;
- no secrets stored inside the image layers;
- health checks;
- an **SBOM** (Software Bill of Materials — the list of everything inside the image) generated at build time;
- a vulnerability-scan gate in CI: Trivy, Grype, or Docker Scout.

The SBOM flag has changed over time: older docs use `docker build --sbom=true`, newer ones use a `docker buildx` subcommand. Confirm which one applies to your installed version.

Name **Docker Hardened Images** as a vendor-backed version of these practices. Check what it really includes (provenance, SBOM, and VEX — Vulnerability Exploitability eXchange) in Docker's own docs, not in vendor advertising pages. You do not need to use it. Know what it claims and why it exists.

**d) Orchestration — Kubernetes core concepts**
Why you need an orchestrator once a container runs somewhere real: self-healing, scheduling, scaling, and rolling updates. A bare `docker run` solves none of these. Core objects and how they relate:

- **Pod** — the smallest deployable unit.
- **ReplicaSet** — keeps N pods running.
- **Deployment** — manages ReplicaSets, and drives rolling updates and rollbacks.
- **Service** — a stable network address with load-balancing. You need it because pod IPs change.
- **Namespace** — a logical partition inside one cluster.
- **ConfigMap** and **Secret** — configuration and credentials kept outside the image, injected as environment variables or files. Explain why a Secret is only base64-encoded, not encrypted, by default, and what that means for putting one in a git repo.
- **Volume** and **PersistentVolumeClaim (PVC)** — storage that outlives a single pod. Explain the difference between a pod-lifetime volume (e.g. `emptyDir`) and a PVC bound to a PersistentVolume, and why a pod's own filesystem is not a safe place for data.

You will run your app and your monitoring stack in two different namespaces, so explain what a namespace separates and what it does not.

Storage works on every local cluster tool in 3.2, but with limits you should know before you design anything:

- **minikube** provisions `hostPath` volumes through its storage-provisioner addon. Only some host directories survive a restart (`/data`, `/tmp/hostpath-provisioner` and a few others).
- **k3s** ships the `local-path` StorageClass (`rancher.io/local-path`), stores data under `/var/lib/rancher/k3s/storage`, and binds a volume to the node that first used it.
- **kind** and **k3d** ship the same provisioner as the `standard` StorageClass.

All of them are **ReadWriteOnce only** — a `ReadWriteMany` PVC will not bind. Say in your write-up what that rules out, and what a managed cloud cluster would give you instead.

The declarative model: you describe the desired state in YAML, and a control loop keeps changing the actual state until it matches. That model is the core idea, not the YAML syntax. Use `https://kubernetes.io/docs/concepts/` for the current object model — it has grown over time. List any object kinds your own app needed beyond the ones above, and say why.

Also cover cluster shape: control plane vs worker nodes, and what changes between a **single-node** cluster and a **multi-node** one (real scheduling across nodes, node failure, pod networking between machines). For your proof of work, pick one tool and explain your choice. **kind** runs nodes as containers and is easy to rebuild the same way each time. **minikube** gives a fuller VM and dashboard. **k3s** is lightweight, and the easiest tool for the multi-machine bonus. **k3d** is an acceptable substitute.

**e) Observability with OpenTelemetry, Prometheus & Grafana**
What OpenTelemetry is and the three signals it standardizes: traces, metrics, logs. How that differs from just reading application logs: structured, correlated by trace/span ID, vendor-neutral, propagated across service boundaries. Report signal maturity honestly. In the past, logs were less mature than traces and metrics in most language SDKs. Do not assume this is still true: check the current status of each signal at `https://opentelemetry.io/status/`. Name the **OpenTelemetry Collector** and explain what it does that a raw SDK export does not: batching, routing, and sending the same data to several backends. Check its current major version and note any recently stabilized feature (e.g. declarative/file-based config) against the Collector changelog.

Cover how Semantic Conventions are versioned and why a shared attribute vocabulary is what makes signals portable between backends.

Then the stack you will actually run in the cluster: **Prometheus** (pull-based scraping, time series, PromQL) and **Grafana** (dashboards on top). How Prometheus finds pods to scrape in Kubernetes, and what a useful workload dashboard shows: request rate, error rate, latency, plus CPU and memory per pod. Name **kube-prometheus-stack** as the standard one-command install of both. Explain the difference between metrics your app emits itself (through OTel or a Prometheus client library) and metrics the cluster already provides without extra code (kubelet/cAdvisor). A good dashboard uses both. Also name **k6**, Grafana's load-testing tool. It is the standard way to generate realistic traffic. One manual request only proves that the pipeline works. A load test sends many requests at once, so the system is under real stress. Your final demo uses a load test.

**f) Frontend & browser observability — the same signals, from the user's tab**
3.1e was server-side; this is the client-side half. Cover **Real User Monitoring (RUM)**: measuring real users in their own browsers. Compare it with synthetic (lab) testing, and say what each one finds that the other misses. The browser **Performance APIs** everything else is built on: `PerformanceObserver`, Navigation Timing, Resource Timing. **Core Web Vitals** — LCP, INP, CLS — what each measures and why Chrome made them the standard; the `web-vitals` JS library as the easiest way to read them. The **Chrome UX Report (CrUX)** gives field data from real users. The Chrome DevTools Performance panel and Trace Viewer give lab data, measured on your own machine. Explain the difference, and state clearly which tool gives which kind of data. Many teams confuse the two.

Then connect back to 3.1e. The **OpenTelemetry Web SDK** uses `@opentelemetry/sdk-trace-web` plus auto-instrumentation for `fetch`, `XMLHttpRequest`, and document-load. It emits the same trace and span model as your backend. **W3C Trace Context** defines the `traceparent` header, and that header lets a single trace cover all three parts: the browser click, your backend, and the response. Verify current package and API names against the OTel JS docs, not an old blog post — this area has moved. Also cover the **Server-Timing API**: a response header your backend sets, shown in the DevTools Network panel and readable via `PerformanceServerTiming`. It is the simpler way to get backend timing into browser performance data, with no SDK needed, and a good first option before the full Web SDK.

### 3.2 Deliverables

- Findings write-up (markdown, in git) organized by 3.1's topics.
- Slide deck for the Friday sessions — required.
- **Required proof of work.** Steps 1–5 build one running system that the team operates. Every step needs a real capture — a screenshot, a log, or a recording. A drawing or a description does not count.

  Numbering note: "3.2.1" to "3.2.5" always mean the required steps below. Bonus items are referred to as "bonus 1", "bonus 2", and so on.
  1. **App, image, CI.** A toy app (any language; a small API is easiest) with a `Dockerfile` on a minimal/distroless base, in a **public GitHub repo**. A **GitHub Actions** workflow builds it, tests it, and pushes the image to a registry (`ghcr.io` is free and closest to 3.1a). Capture the real run: a passing run, or a failing run that you then fixed. Note: your laptops are arm64 (Apple M1) but GitHub Actions runners build `linux/amd64` by default, so an image that builds fine in CI may refuse to start in your local cluster. Work out how to handle this and write down what you did.
  2. **Cluster and manual deploy.** Run a Kubernetes cluster yourself — one machine is the minimum (kind, minikube, k3s, or k3d). Deploy the app with `kubectl apply`. Create a Deployment and a Service in **their own namespace**. Pass at least one setting through a **ConfigMap** or a **Secret**, instead of writing it into the image. Capture `kubectl get nodes`, `kubectl get all -n <your-app-namespace>`, and a real response from the service.
  3. **Expose the service outside the machine** with **cloudflared**, **ngrok**, or **Tailscale** (free tier). Capture a real request from a device that is not the cluster machine — a phone is fine.
  4. **Your workload's metrics on a dashboard.** Instrument the app with OpenTelemetry (or a Prometheus client library) so it emits request rate, latency, and errors. Run Prometheus + Grafana in **their own namespace** (kube-prometheus-stack is fine) and build one dashboard showing those metrics plus pod CPU and memory. Screenshot with real data from requests you sent — not the stock demo dashboard.
  5. **Live demo, Fri 16 Oct (Week 2).** Run a **k6** load test against the deployed service and show the Grafana dashboard moving in real time.
- **Bonus — pick what you have time for, none of these are required:**
  1. **Automate the deploy with GitOps.** Install the agent you researched in 3.1b (**Argo CD** or **Flux**). Point it at the repo that holds your Kubernetes manifests. Then show one full loop: commit a change (image tag or replica count) → agent syncs → new pods roll out. Capture before and after, plus the sync in the agent's UI or CLI. Do it once with manual sync and once with automatic sync, and say which one you would pick for a real team.
  2. **Multi-machine cluster.** 2+ nodes on separate machines (k3s is the easiest path), with pods scheduled across them — capture `kubectl get pods -o wide`.
  3. **A real trace.** Run an OpenTelemetry Collector, export at least one trace from your app to a viewer (e.g. Jaeger), and capture it from a request you sent.
  4. **Supply chain:** an SBOM and/or a build attestation captured for your image.
  5. **A log correlated to a trace** via OTel context.
  6. **Browser-to-backend tracing.** The **OpenTelemetry Web SDK** in your frontend, so a browser request and the backend span it triggers share one trace via W3C Trace Context.
  7. **Diagram your own pipeline:** commit → CI → image registry → deploy → cluster namespace → tunnel → user, with the monitoring namespace alongside. It must match what you actually built.
  8. **One real Core Web Vital in Chrome.** Load a real page: your app's frontend if it has one, otherwise the plain HTTP response page of your own service, or any public page. Capture LCP, INP, or CLS with the `web-vitals` library in the console, or with the DevTools Performance panel. It must be a real number from a real page load.

## 4. Constraints

- Research Wednesdays, present Fridays, across the whole period.
- Use git; commit progressively, not one final paste.
- Team of three. Split the six research topics and the five proof-of-work steps among yourselves, and record the split in the write-up. One combined write-up, one shared cluster, one combined demo. Every member presents part of it.
- Your own machines and free tiers only — no paid cloud, no managed Kubernetes, no production access.
- Commit every capture (screenshots, logs, recordings) into the repo, next to the write-up.

## 5. Timeline

| When | Milestone |
| --- | --- |
| Wed 7 Oct | Kickoff. Research starts; team records its topic split. |
| Fri 16 Oct | **Half-way presentation**, including the live demo. |
| Fri 30 Oct | **Final presentation**, including the live demo. |

## 6. Definition of Done

**Scoring (identical for Group A and Group B):** research quality with citations 40%, proof of work 40%, presentation 20%. Bonus items only break ties — they never replace a missing required item. A required item that is genuinely blocked costs nothing, as long as you document what you tried and where it failed.

- [ ] Write-up covers all of 3.1 (a–f), in the team's own words, each claim tied to a cited doc page.
- [ ] Where a topic is still moving (OTel signal maturity, Immutable Actions rollout), the write-up says so instead of stating it as settled.
- [ ] Real CI run captured (screenshot/log) from a public GitHub repo: it runs the tests and pushes an image built on a minimal/distroless base.
- [ ] Cluster running and the app deployed manually into its own namespace: `kubectl` output plus a live request/response, with at least one setting supplied by a ConfigMap or Secret.
- [ ] Service reached through a tunnel from a device that is not the cluster machine.
- [ ] Prometheus and Grafana run in their own namespace, and the dashboard shows real metrics emitted by the team's own app plus pod CPU and memory from the cluster — not a stock demo dashboard.
- [ ] The 16 Oct demo runs a live k6 load test with metrics updating in real time.
- [ ] Write-up records how the team split the topics and the steps, and every member presented part of the demo.
- [ ] Every capture is committed in the repo next to the write-up.
- [ ] Git history shows incremental work throughout, not one final paste.
- [ ] Slide deck used at the half-way presentation (16 Oct) and extended into the final (30 Oct).
- [ ] Both presentations delivered live: the 16 Oct half-way presentation and the 30 Oct final.

## 7. Out of Scope

- Any specific company's production infrastructure, deployment configs, or CI setup — this is general research plus your own local proof of work.
- Managed Kubernetes (EKS/GKE/AKS) or any paid cloud — your own machines and free tiers only.
- Production-grade ingress, TLS, and DNS — a free tunnel is enough to reach the service from outside.
- Alerting and on-call rules — dashboards are the target, paging is not.
- A vulnerability-scanning gate in CI — know the tools (3.1c); running one is optional.
- An automated GitOps deploy — `kubectl apply` by hand is the minimum, Argo CD/Flux is a bonus.
- A multi-machine cluster — one machine is the minimum, more nodes is a bonus.
- Full OpenTelemetry instrumentation of a real app — the request rate, latency, and error metrics in 3.2.4 are enough. Traces are a bonus.
- A production-grade supply-chain verification pipeline (e.g. policy-enforced attestation checks) — know what SBOMs and attestations are, generate one as a bonus; enforcement is not required.
- A production RUM deployment or a real frontend for the toy app — one Web Vitals capture (bonus 8) is enough, full browser-to-backend trace propagation is a bonus.

## 8. Review Points

- **Fri 16 Oct:** Half-way presentation to Track Leader and stakeholder (Thang Le).
- **Fri 30 Oct:** Final presentation to Track Leader and stakeholder (Thang Le).

## 9. References

- GitHub Actions: `https://docs.github.com/en/actions` — changelog (Immutable Actions rollout status): `https://github.blog/changelog/` — Artifact Attestations: `https://docs.github.com/en/actions/concepts/security/artifact-attestations`
- Docker: `https://docs.docker.com/` — Dockerfile best practices, Docker Hardened Images (verify claims here, not in third-party summaries).
- Kubernetes: `https://kubernetes.io/docs/` — core concepts: `https://kubernetes.io/docs/concepts/` — local clusters: `kind` (`https://kind.sigs.k8s.io/`), `minikube` (`https://minikube.sigs.k8s.io/`), `k3s` (`https://docs.k3s.io/`, multi-machine path), `k3d` (`https://k3d.io/`).
- Kubernetes storage: `https://kubernetes.io/docs/concepts/storage/` — Secrets: `https://kubernetes.io/docs/concepts/configuration/secret/` — minikube volumes: `https://minikube.sigs.k8s.io/docs/handbook/persistent_volumes/` — k3s storage: `https://docs.k3s.io/storage`
- GitOps: Argo CD (`https://argo-cd.readthedocs.io/`) — Flux (`https://fluxcd.io/flux/`) — OpenGitOps principles (`https://opengitops.dev/`).
- Tunnels / remote access: cloudflared (`https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/`) — ngrok (`https://ngrok.com/docs`) — Tailscale (`https://tailscale.com/kb`).
- Prometheus: `https://prometheus.io/docs/` — Grafana: `https://grafana.com/docs/` — kube-prometheus-stack: `https://github.com/prometheus-community/helm-charts`.
- OpenTelemetry: `https://opentelemetry.io/docs/` — signal status (source for maturity claims): `https://opentelemetry.io/status/` — Collector (source for version/release notes): `https://opentelemetry.io/docs/collector/`
- OpenTelemetry Semantic Conventions: `https://opentelemetry.io/docs/specs/semconv/`
- Jaeger (trace viewer): `https://www.jaegertracing.io/` — k6 (load testing): `https://k6.io/docs/`
- Core Web Vitals: `https://web.dev/vitals/` — `web-vitals` JS library: `https://github.com/GoogleChrome/web-vitals` — Chrome UX Report: `https://developer.chrome.com/docs/crux`
- Chrome DevTools Performance panel / Trace Viewer: `https://developer.chrome.com/docs/devtools/performance`
- OpenTelemetry JS (Web SDK, browser auto-instrumentations): `https://opentelemetry.io/docs/languages/js/`
- W3C Trace Context (the `traceparent` propagation header linking browser and backend traces): `https://www.w3.org/TR/trace-context/`
- Server-Timing API / custom metrics: `https://web.dev/custom-metrics/#server-timing-api`
