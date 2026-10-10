# User Stories: Checkpoint 1 (Pillar 1: CI/CD & Container Security)

**Source:** [PROJECT_PLAN.md](PROJECT_PLAN.md), Sections 2, 3, 3.1, 4.1–4.4, 6, 7  
**Pillar owner / decision maker:** Engineer A  
**Window:** Fri 9 Oct (kickoff) → **★ CP1: Mon 12 Oct, 09:00** → merge and official captures by Mon 12 Oct, 11:30  
**Rubric step covered:** Step 1 (public repo, distroless `amd64` Dockerfile, green GitHub Actions CI pushing to `ghcr.io`, architecture audit)

---

## 1. Checkpoint Goal

> By 11:30 on Mon 12 Oct, `main` holds one tested Go service that the team has agreed on. CI builds it into a distroless, non-root `amd64` image and publishes it to GHCR. That image is the foundation Pillar 2 deploys, and its `/metrics` endpoint is what Pillar 3 scrapes.

**Success measures**
- All 3 engineers have a personal public repo with at least one CI run (green, or a failure documented in `NOTES.md`).
- CP1 starts at 09:00 and finishes within its 75-minute timebox, with a recorded decision.
- The first CI run on `main` after the merge is green and pushes `:<sha>` and `:latest`.
- `step-1-ci-run.png` and `step-1-ghcr-image.png` are committed from that `main` run.
- Pillar 2 can pull the official image without extra credentials.

---

## 2. Personas

| Persona | Who | What they need from CP1 |
| :--- | :--- | :--- |
| **Solo Builder** | Each of Eng A, B, C | A clear contract and a repo layout shared with the others, so each person can build alone and the three builds are easy to compare at CP1. |
| **Pillar Owner** | Eng A | Comparable builds and clear criteria, so the decision takes 5 minutes, not 50. |
| **Downstream Builder** | Eng B (Pillar 2), Eng C (Pillar 3) | A stable image, port, env vars, and metric names they can build manifests and dashboards against. |
| **Grader / Stakeholder** | Internship Supervisor (grader); Thang Le (Track Leader) | Evidence that Step 1 is real, individual, progressive, and documented. |

---

## 3. Story Map

| ID | Story | Persona | Priority | Pts | Owner | Due |
| :--- | :--- | :--- | :---: | :---: | :--- | :--- |
| **Epic A: Kickoff Foundation** | | | | | | |
| CP1-01 | Personal repo setup (and empty team repo) | Solo Builder | Must | 1 | Each (Eng A: team repo) | Fri 9, AM |
| CP1-02 | Shared app contract | Downstream Builder | Must | 1 | All (Eng A records) | Fri 9, kickoff |
| CP1-03 | Local toolchain ready | Solo Builder | Must | 1 | Each | Fri 9, AM |
| CP1-04 | Hardware & architecture audit | Grader | Must | 1 | Eng A | Fri 9 |
| **Epic B: Solo Build** (each engineer, in their personal repo) | | | | | | |
| CP1-05 | Go service implements the contract | Downstream Builder | Must | 3 | Each | Sat 10, AM |
| CP1-06 | RED metrics on `/metrics` | Downstream Builder (Eng C) | Must | 2 | Each | Sat 10, AM |
| CP1-07 | Local test, lint, build via `Makefile` | Solo Builder | Must | 2 | Each | Sat 10, AM |
| CP1-08 | Distroless, non-root `amd64` image | Grader | Must | 3 | Each | Sat 10, AM |
| CP1-09 | Personal CI pipeline to GHCR | Solo Builder | Must | 4 | Each | Sat 10, AM |
| CP1-10 | Pillar 1 learning notes | Grader | Must | 1 | Each | Sat 10, AM |
| *CP1 session (Mon 12, 09:00–10:15): demos and decision, see Section 5* | | | | | | |
| **Epic C: Merge & Official Artifacts** (Mon 12, 10:15–13:00) | | | | | | |
| CP1-11 | Checkpoint PR into `main` (with decision record) | Downstream Builder | Must | 3 | Eng A (B, C review) | Mon 12, 11:30 |
| CP1-12 | Official image published and pullable | Downstream Builder (Eng B) | Must | 1 | Eng A | Mon 12, 11:30 |
| CP1-13 | Graded Step 1 captures | Grader | Must | 1 | Eng A | Mon 12, 11:30 |
| CP1-14 | Switch Pillar 2 to the official image (CP2 prerequisite) | Downstream Builder | Must | 1 | Each | Mon 12, 13:00 |
| **Epic D: Research** | | | | | | |
| CP1-15 | Research seeds for 3.1a / 3.1c | Grader | Should | 2 | Eng A | Mon 12 |

**Load per person (Must only):** about 19 pts for each Solo Builder. Eng A also carries about 6 pts of owner work plus running the CP1 session, almost all of it on Mon 12, so Eng A starts their solo build on Fri 9 like the others.

**No team repo scaffold, no shared CI template.** Each engineer builds everything in their own personal repo. The team repo is created empty on Fri 9 and gets no code until CP1: it receives only the build chosen in the CP1 session, through the CP1-11 PR. There is no shared reference `ci.yml`: each engineer writes their own under CP1-09, so `ci.yml` becomes one of the artifacts compared at CP1.

---

## 4. Stories & Acceptance Criteria

Criteria that describe behavior are written as **Given / When / Then** scenarios. Criteria that only check that something exists or was done are plain checklist items.

### Epic A: Kickoff Foundation (Fri 9 Oct)

#### CP1-01: Personal repo setup (and empty team repo)
**As a** Solo Builder, **I want** my own public repo with the agreed layout **so that** I can build Pillar 1 on my own, and the three repos are easy to compare side by side at CP1.

**Acceptance criteria**
- [ ] Each engineer creates a personal public repo (`<github-user>/devops-group-b-solo`, used for all three pillars) and shares the link in the team chat.
- [ ] The directory tree from Plan Section 3 exists, using `.gitkeep` where needed:
  - `app/` holds `main.go`, `main_test.go`, `go.mod`, `go.sum`, `Dockerfile`, `.dockerignore`, `.golangci.yml`, and `Makefile`.
  - `.github/workflows/`, `k8s/base/`, `k8s/monitoring/`, `load-tests/`, `docs/`, `captures/` sit at the repo root.
- [ ] Paths are fixed for everyone: the Docker build context is `app/`, and CI runs the Go and lint steps with `working-directory: app`. All three personal repos therefore have the same files at the same paths, so each artifact can be compared directly at CP1.
- [ ] Personal `main` holds only Pillar 1. Weekend Pillar 2 work goes on a `p2` branch created from `main`, so `main` still shows the Pillar 1 state at CP1.
- [ ] Eng A creates the empty team repo on Fri 9:
  - It is owned by a GitHub organization (`<org>/devops-observability-research-group-b`), not a personal account. The official image `ghcr.io/<org>/devops-app` then never collides with Eng A's personal `ghcr.io/<eng-a>/devops-app`.
  - It is public, holds only an initial commit (e.g. GitHub's default README), and all 3 engineers have write access.
  - The org settings allow members to create both public and private packages (a package starts private on its first push and is made public afterwards).
- [ ] From Fri 9 to Mon 12, 09:00, nobody scaffolds or pushes code to the team repo. Its layout comes from the chosen build, merged at CP1-11.

#### CP1-02: Shared app contract
**As a** Downstream Builder, **I want** every personal build to expose the same interface **so that** Pillar 2 manifests and Pillar 3 dashboards work on any of the three images.

**Acceptance criteria**
- [ ] Before anyone starts coding, the contract is posted in the team chat and copied into the `README.md` of every personal repo:
  - Port `8080`.
  - `GET /health` → `200` with a small body.
  - `GET /api/work?ms=<int>&fail=<0..1>` → simulated work. `ms` is the delay in milliseconds (default `50`). `fail` is the probability of returning `500` (default `0`). Out-of-range values are clamped (`ms` to `0..2000`, `fail` to `0..1`). An `ms` that is not an integer (e.g. `abc` or `1.5`), or a `fail` that is not a number, returns `400`.
  - `GET /metrics` → Prometheus exposition format.
  - Env vars `APP_ENV` and `MESSAGE` (from the ConfigMap) and `API_KEY` (from the Secret).
- [ ] Metric names are fixed: `http_requests_total` (counter) and `http_request_duration_seconds` (histogram), with labels `method`, `route`, `status`.
- [ ] **Given** `MESSAGE` is set, **when** I `curl` the app, **then** the `MESSAGE` value appears in a response body, so the Step 2 `curl` check can prove the ConfigMap was injected.
- [ ] **Given** `API_KEY` is set, **when** the app starts, logs, or serves any request, **then** the value never appears in logs or responses. At most, the app reports whether it is set.

#### CP1-03: Local toolchain ready
**As a** Solo Builder, **I want** every tool installed on day 1 **so that** missing tools don't block me on the weekend, when no checkpoint is there to catch it.

**Acceptance criteria**
- [ ] `docker`, `k3d`, `kubectl`, `helm`, `k6`, `cloudflared` or `ngrok`, the Go toolchain (the pinned minor version, same as `go.mod` and the Dockerfile builder), and `golangci-lint` (the pinned version agreed at kickoff, same as CI) are installed. The pinned `golangci-lint` release must be built with a Go version at least as new as the `go` directive in `go.mod`; otherwise it refuses to lint the module.
- [ ] `docker run hello-world` succeeds.
- [ ] Each engineer posts "tooling ✅" in the team chat. Blockers go to Eng B, the tooling coach.

#### CP1-04: Hardware & architecture audit
**As a** Grader, **I want** the team's hardware and the `amd64`-only decision documented **so that** I can see the architecture question was addressed on purpose, not ignored.

**Acceptance criteria**
- [ ] A table in `docs/01-ci-cd-gh-actions.md` (written in Eng A's personal repo, carried into the team repo by the CP1-11 PR) lists each member's laptop, CPU, and `uname -m` output (all `x86_64`).
- [ ] Each member's raw `uname -m` output is saved for the graded capture in CP1-13.
- [ ] It gives the reasoning: GitHub Actions runners are `linux/amd64`, so native builds need no QEMU, and the risk of `exec format error` goes away.
- [ ] It states the known limitation (the image will not run natively on Apple Silicon or ARM), and notes that multi-arch is deferred to a Sprint 2 stretch. The risk **R1** mitigation is written out in the doc itself, not linked, because the plan is not part of the team repo.

---

### Epic B: Solo Build (each engineer, Fri 9 – Sat 10 AM, in their personal repo)

#### CP1-05: Go service implements the contract
**As a** Downstream Builder, **I want** a minimal Go HTTP service that meets the contract **so that** Kubernetes probes, the tunnel, and k6 all have something real to hit.

**Acceptance criteria**
- [ ] Uses the standard library `net/http` only, with no web framework. `app/go.mod` and `app/go.sum` are committed.
- [ ] **Given** the app is running, **when** I `curl :8080/health`, **then** I get `200` in under 50 ms.
- [ ] **Given** `MESSAGE="hello"`, **when** I `curl ':8080/api/work?ms=200'`, **then** the response contains `hello` and takes 200–300 ms.
- [ ] **Given** the app is running, **when** I call `/api/work` with no params, **then** the response takes 50–100 ms.
- [ ] **Given** the app is running, **when** I call `/api/work?ms=5000`, **then** the delay is capped and the response takes 2000–2300 ms.
- [ ] **Given** the app is running, **when** I call `/api/work?ms=abc`, `/api/work?ms=1.5`, or `/api/work?fail=abc`, **then** the response is `400`.
- [ ] **Given** the app is running, **when** I call `/api/work?fail=1`, **then** the response is `500`.
- [ ] **Given** the app is running, **when** I call `/api/work?fail=0` or omit `fail`, **then** the response is `200`.
- [ ] **Given** `APP_ENV`, `MESSAGE`, and `API_KEY` are set, **when** the app starts, **then** it reads its config from those env vars.
- [ ] **Given** those env vars are unset (including `API_KEY`), **when** the app starts, **then** it still starts, using safe defaults.
- [ ] **Given** the app is serving requests, **when** it receives `SIGTERM`, **then** it finishes in-flight requests before exiting, so rolling updates in Pillar 2 don't drop requests. *(Should)*

#### CP1-06: RED metrics on `/metrics`
**As** Eng C (Pillar 3), **I want** request Rate, Errors, and Duration exposed by the app **so that** the Grafana dashboard and the k6 demo have meaningful data from the start.

**Acceptance criteria**
- [ ] Uses `prometheus/client_golang`, exposing `http_requests_total{method,route,status}` and `http_request_duration_seconds{method,route,status}`.
- [ ] The `route` label uses the route template (e.g. `/api/work`), not the raw path. Requests to unknown paths are recorded as `route="other"`. Together these keep label cardinality bounded.
- [ ] Histogram buckets cover the full `/api/work` delay range (up to 2 s, e.g. `0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 2.5`), so p95/p99 latency is visible during load.
- [ ] **Given** a freshly started app, **when** I send 5 requests with `?fail=1` and then `curl /metrics`, **then** `http_requests_total{route="/api/work",status="500"}` is 5 (the **Errors** part of RED).
- [ ] **Given** a freshly started app, **when** I send 10 requests to `/api/work` and then `curl /metrics`, **then** `http_requests_total{route="/api/work",status="200"}` is 10.
- [ ] **Given** a freshly started app, **when** I request `/foo/1` and `/foo/2` and then `curl /metrics`, **then** both are counted under `route="other"`, with no series for the raw paths.

#### CP1-07: Local test, lint, build via `Makefile`
**As a** Solo Builder, **I want** the same checks locally as in CI **so that** I catch failures before pushing and wasting CI minutes.

**Acceptance criteria**
- [ ] `app/main_test.go` uses `httptest` to cover at least `/health` (200), `/api/work` (returns `MESSAGE`, `fail=1` returns `500`, non-integer `ms` returns `400`), and `/metrics` (counter increments).
- [ ] The `Makefile` and CI use the same pinned `golangci-lint` version and the same config file (`app/.golangci.yml`).
- [ ] **Given** a clone of my personal repo, **when** I run `make lint`, `make test`, or `make build` in `app/`, **then** each runs with that one command, and `make build` produces the Docker image.
- [ ] **Given** the code on `main`, **when** I run `go test ./...` in `app/`, **then** all tests pass.
- [ ] **Given** the code on `main`, **when** I run `golangci-lint run` in `app/`, **then** it reports zero issues.

#### CP1-08: Distroless, non-root `amd64` image
**As a** Grader, **I want** a minimal, hardened production image **so that** the team shows real container-security practice, which also feeds research section 3.1c.

**Acceptance criteria**
- [ ] Multi-stage `app/Dockerfile`: the builder is a pinned `golang:<x.y>` tag for the minor version in `go.mod`'s `go` directive (e.g. `go 1.x.y` → `golang:1.x`, whose patch release is at least the directive's; the minor is the current stable one at kickoff, recorded in `README.md`), and the runtime is `gcr.io/distroless/static:nonroot`.
- [ ] The binary is built with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`.
- [ ] The Dockerfile keeps the base image's numeric user, or sets `USER 65532:65532`. It never uses `USER nonroot`, because Kubernetes `runAsNonRoot` cannot verify a user given by name.
- [ ] No secrets, `.env` files, or `.git` in the build context. `app/.dockerignore` is present.
- [ ] `EXPOSE 8080` is set.
- [ ] **Given** the built image, **when** I run `docker inspect --format '{{.Config.User}}' <img>`, **then** it prints the numeric `65532` (or `65532:65532`).
- [ ] **Given** the built image, **when** I run `docker image inspect --format '{{.Os}}/{{.Architecture}}' <img>`, **then** it prints `linux/amd64`.
- [ ] **Given** the built image, **when** I run `docker run --entrypoint sh <img>`, **then** it fails because the image has no shell.
- [ ] **Given** the built image, **when** I check its size with `docker image ls`, **then** it is under 30 MB uncompressed (target ~25 MB; GHCR shows the smaller compressed size), and the size is recorded in `NOTES.md`.
- [ ] **Given** the built image, **when** I run `docker run -p 8080:8080 <img>`, **then** the CP1-05 checks pass.

#### CP1-09: Personal CI pipeline to GHCR
**As a** Solo Builder, **I want** every push to my repo to lint, test, build, and publish my image **so that** I have my own image to deploy in Pillar 2 over the weekend.

**Acceptance criteria**
- [ ] The workflow sets `permissions: packages: write, contents: read` and logs in to `ghcr.io` with `GITHUB_TOKEN`. No personal tokens are stored.
- [ ] Triggers: `push` to `main` with `paths-ignore: [captures/**, docs/**]`, and `pull_request` to `main` with no path filter, so required PR checks always report.
- [ ] The image path is `ghcr.io/<owner>/devops-app`, with the owner taken from `github.repository_owner` and lowercased by `docker/metadata-action` or a shell step (GitHub expressions have no `lower()` function). The same file then works unchanged in the team repo if my `ci.yml` is chosen.
- [ ] The `:<sha>` tag is the full 40-character commit SHA (`${{ github.sha }}`), with no `sha-` prefix and no short form, so Pillar 2 manifests can be written from the commit alone.
- [ ] The image carries the `org.opencontainers.image.source` label with the repo URL (`docker/metadata-action` adds it), so the GHCR package links to its repo.
- [ ] The build step sets `platforms: linux/amd64` and `provenance: false`, so GHCR lists only the `linux/amd64` image and no extra `unknown/unknown` entry. Provenance attestations come back with Bonus 4 in Sprint 2.
- [ ] Third-party actions are pinned to a version tag or a commit SHA.
- [ ] After the first successful push, I set the GHCR package visibility to **Public** (new packages start private).
- [ ] The repo has progressive commits (several focused commits, not one dump), per risk **R6**.
- [ ] The first engineer to get a GHCR push working posts the working `permissions` and login snippet in the team chat, so the others don't repeat the same permission debugging.
- [ ] The three jobs are named exactly `Lint`, `Test`, and `Build & Push` (their `name:` fields), with no matrix, which would add suffixes. These are the required check names in CP1-11; any other name never reports and blocks every PR once branch protection is on.
- [ ] **Given** a workflow run, **when** I view its jobs, **then** the stages run in this order: **Lint** (`golangci-lint`) → **Test** (`go test ./...`) → **Build & Push**, and Build & Push runs only if both earlier stages pass.
- [ ] **Given** a pull request to `main`, **when** CI runs, **then** lint, test, and the image build run, but nothing is pushed to GHCR (`push: false`).
- [ ] **Given** a push to `main` in my personal repo, **when** CI runs, **then** all stages pass and `ghcr.io/<github-user>/devops-app` gets the tags `:<sha>` and `:latest`.
- [ ] **Given** a push to `main` that only changes `captures/` or `docs/`, **when** it lands, **then** CI does not run and no new image is published.
- [ ] **Given** a failing test, **when** I push, **then** the pipeline stops before Build & Push and nothing is published.
- [ ] **Given** my image in GHCR, **when** my Pillar 2 cluster pulls it without credentials, **then** the pull succeeds because the package is public.

#### CP1-10: Pillar 1 learning notes
**As a** Grader, **I want** each engineer's individual learning recorded **so that** individual work is visible and the "Challenges & Lessons Learned" slide has real material.

**Acceptance criteria**
- [ ] `NOTES.md` in my personal repo has a "Pillar 1" section with 3–5 bullets: what broke, how it was fixed, and what you would keep.
- [ ] Personal screenshots (green CI run, GHCR tag) are in `captures/` in my personal repo.
- [ ] Anything unfinished is written down as "tried X, failed at Y" (rule 8).
- [ ] *(Due Mon 12, 09:00, not Sat 10.)* At 09:00 on Mon 12 Oct, my personal `main` is tagged `cp1` and the tag is pushed. It marks the Pillar 1 state that gets demoed and is kept as evidence of individual work (risks **R6**, **R8**).

---

### Epic C: Merge & Official Artifacts (Mon 12 Oct, 10:15–13:00)

#### CP1-11: Checkpoint PR into `main` (with decision record)
**As a** Downstream Builder, **I want** the chosen implementation merged into the team repo as one focused PR, with the reasons written down **so that** Pillar 2 builds on a single shared baseline and the team can defend its choices.

**Acceptance criteria**
- [ ] Eng A copies the chosen files from the personal repos (at their `cp1` tags) onto a branch in the team repo created in CP1-01, and opens one PR containing only:
  - the Plan Section 3 directory tree (`.gitkeep` where needed),
  - `app/` (`main.go`, `main_test.go`, `go.mod`, `go.sum`, `Dockerfile`, `.dockerignore`, `Makefile`, `.golangci.yml`) and `.github/workflows/ci.yml`,
  - `README.md` with the app contract, the Go version, the topic split (3.1a–f) and step split (Steps 1–5) with an owner for each, and links to all three personal repos,
  - `docs/01-ci-cd-gh-actions.md` (with the CP1-04 audit),
  - `docs/pow-merge-decisions.md` (the decision record), with the headings `## CP1`, `## CP2`, `## CP3`.
- [ ] The `## CP1` section of the decision record lists: whose version was chosen for each artifact, what was taken from the others, why, which alternatives were rejected and why, and any objections noted in the CP1 session.
- [ ] No personal repo is merged wholesale (`NOTES.md` and personal `captures/` stay in the personal repos).
- [ ] The PR description links the source personal repo and `cp1` commit for each artifact, so the history stays traceable.
- [ ] Eng B and Eng C each approve the PR.
- [ ] The PR is merged with a merge commit (not squash or rebase), so the PR's individual commits stay in `main`'s history (risk **R6**). The team repo's settings allow only merge commits, so later PRs follow the same rule. Everywhere in CP1, "the CP1-11 merge commit" means this commit.
- [ ] CI is green on the PR (build only, nothing pushed) and on the merge commit to `main` (build and push).
- [ ] Once the CP1-13 captures are committed, Eng A turns on branch protection for `main`, ready for CP2 and CP3:
  - require a pull request with 2 approvals;
  - require the status checks `Lint`, `Test`, and `Build & Push` (the CI job names, selectable because they ran on this PR);
  - turn on "Do not allow bypassing the above settings", so admins (including Eng A) are bound too.
- [ ] **Given** branch protection is on, **when** Eng A (an admin) tries a direct push to `main`, **then** the push is rejected.
- [ ] **Given** branch protection is on, **when** a PR's `Test` check fails, **then** the merge button stays blocked.
- [ ] The personal repos stay public and are not deleted (risk **R8**).

#### CP1-12: Official image published and pullable
**As** Eng B, **I want** the official image tags to be predictable and public **so that** my k3d cluster can pull them without an image-pull secret.

**Acceptance criteria**
- [ ] After the first push from `main`, Eng A sets the GHCR package visibility to **Public**, so no image-pull secret is needed. The package is also linked to the team repo through the `org.opencontainers.image.source` label.
- [ ] The README states that manifests should use the `:<sha>` tag (the full commit SHA), and that `:latest` is for convenience only. CI never reuses a SHA tag, but GHCR tags can be overwritten, so pin by digest (`@sha256:…`) where exact immutability matters.
- [ ] **Given** the CP1-11 merge commit on `main`, **when** its CI run finishes, **then** `ghcr.io/<org>/devops-app:<sha>` (the merge commit's full SHA) and `:latest` exist.
- [ ] **Given** the official image, **when** I run `docker image inspect --format '{{.Os}}/{{.Architecture}}'` on it, **then** it prints `linux/amd64`.
- [ ] **Given** a machine that is logged out of GHCR, **when** I run `docker pull ghcr.io/<org>/devops-app:latest`, **then** the pull succeeds.

#### CP1-13: Graded Step 1 captures
**As a** Grader, **I want** Step 1 evidence taken from the shared `main` run **so that** the submission reflects the team system, not one person's sandbox.

**Acceptance criteria**
- [ ] `captures/step-1-ci-run.png` shows the green CI run of the CP1-11 merge commit, with every stage visible.
- [ ] `captures/step-1-ghcr-image.png` shows the GHCR package with the `:<sha>` and `:latest` tags.
- [ ] `captures/step-1-arch-audit.txt` holds each member's `uname -m` output from CP1-04 (all `x86_64`), so the architecture audit is captured, not only documented.
- [ ] Eng A commits the captures directly to `main` before branch protection is turned on (CP1-11). CI ignores `captures/**`, so no new image is built and `:latest` still points to the CP1-11 merge commit.
- [ ] All captures are committed to `main` by 11:30. The Step 1 slide in the Midway Deck uses them.

#### CP1-14: Switch Pillar 2 to the official image (CP2 prerequisite)
**As a** Downstream Builder, **I want** my Pillar 2 Deployment running the official image before CP2 **so that** CP2 compares like for like.

**Acceptance criteria**
- [ ] `<main-sha>` is the full SHA of the latest image-building commit on `main`: the CP1-11 merge commit (the image captured in CP1-13), or a fix PR's merge commit if one has landed since.
- [ ] Each engineer updates their Deployment to `ghcr.io/<org>/devops-app:<main-sha>` between 11:30 and 13:00.
- [ ] The `curl` check still returns the ConfigMap `MESSAGE`. If it breaks, the cause is posted to the team chat and Eng A ships a fix PR (rule 8 exception). Eng B and Eng C review it as soon as it opens, and once it merges everyone redeploys with the fix's SHA.
- [ ] If an engineer's Pillar 2 Deployment does not exist yet, the story is done once that is posted in the team chat (rule 9: no catch-up debt).

---

### Epic D: Research

#### CP1-15: Research seeds for 3.1a / 3.1c (Should)
**As a** Grader, **I want** CP1 experience turned into research notes right away **so that** the 40% research score draws on hands-on evidence, not only theory.

**Acceptance criteria**
- [ ] `docs/01-ci-cd-gh-actions.md` and `docs/03-containers-distroless-sbom.md` each have an outline and at least 3 official-doc citations (GitHub Docs, Docker Docs, distroless repo).
- [ ] Each document includes at least one concrete CP1 finding (e.g. the image size of distroless vs. a `golang` base, or a GHCR permission error and its fix).
- [ ] Both land in the team repo through a docs PR after CP1-11 (`docs/01` first arrives with the CP1-11 PR; this PR extends it and adds `docs/03`).

---

## 5. CP1 Session Agenda (Mon 12 Oct, 09:00–10:15)

The session follows Plan Section 4.1 and is run by Eng A as Pillar Owner. A timekeeper is named at kickoff. Its output is the decision that CP1-11 merges and records.

**Demos (3 × 12 min)**
- Each engineer shows the latest CI run, the GHCR tag, `docker run`, `curl` on all 3 endpoints, and one key choice or one failure.
- What gets demoed is the personal repo's `main` at its `cp1` tag (CP1-10). No live fixes.
- Unfinished work is presented as-is (rule 9: no catch-up debt).

**Compare (15 min)**
- Builds are compared against the 5 criteria in Plan Section 4.1: works end to end, simplicity, fit with Section 3.1, security (non-root, pinned versions, no secrets), and readiness for Pillars 2 and 3.
- Objections from Eng B or Eng C are noted for the decision record, not debated beyond this slot.

**Decide (5 min)**
- Eng A decides for each artifact (`app/`, `Dockerfile`, `ci.yml`) and may combine parts from different builds. It is not a vote.

**Record (5 min)**
- Eng A drafts the `## CP1` section of `docs/pow-merge-decisions.md`, which goes into the CP1-11 PR.

---

## 6. CP1 Definition of Done

- [ ] All **Must** stories accepted by Eng A. Eng B and Eng C approve the CP1-11 PR, then confirm CP1-12 (by pulling the image while logged out) and CP1-13 (by checking the committed captures).
- [ ] `main` is green and the official image is public in GHCR, `linux/amd64`, and runs as numeric UID `65532`.
- [ ] Graded Step 1 captures (CI run, GHCR image, architecture audit) and the CP1 decision record are committed to `main`.
- [ ] `main` in the team repo is protected (2 approvals, required CI checks, no admin bypass).
- [ ] Three personal public repos exist with progressive commits, a pushed `cp1` tag, and a "Pillar 1" section in each `NOTES.md`.
- [ ] No secrets in git: no tokens in the workflow, and `API_KEY` comes only from the environment.

## 7. Out of Scope for CP1

- Kubernetes manifests, the tunnel, and the cluster (CP2). The one exception is CP1-14, which only switches an existing Pillar 2 Deployment to the official image.
- Prometheus, Grafana, and k6 (CP3). CP1 only guarantees the `/metrics` contract.
- SBOM generation, SLSA attestations, and image signing (Bonus 4, Sprint 2).
- Non-blocking image vulnerability scan in CI (Trivy or Grype, report only, `exit-code: 0`). Bonus 4 and 3.1c head start: picked up only if CP1-13 is done with time to spare, otherwise Sprint 2.
- Multi-arch builds (`linux/arm64`). Deferred to a Sprint 2 stretch.
