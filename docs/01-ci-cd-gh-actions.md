# CI/CD with GitHub Actions

## Hardware and architecture audit

The team builds and publishes `linux/amd64` images only. This section records the hardware audit behind that decision, the reasoning, and the known limitation.

### Team hardware

Recorded at kickoff on Fri 9 Oct. The raw `uname -m` output from each member is in [`captures/step-1-arch-audit.txt`](../captures/step-1-arch-audit.txt).

| Member | Laptop | CPU | `uname -m` |
| :--- | :--- | :--- | :--- |
| Eng A | MacBook Pro 13-inch, 2020 (`MacBookPro16,2`) | Intel Core i5-1038NG7 @ 2.00GHz | `x86_64` |
| Eng B | MacBook Pro 13-inch, 2020 (`MacBookPro16,2`) | Intel Core i5-1038NG7 @ 2.00GHz | `x86_64` |
| Eng C | MacBook Pro 13-inch, 2020 (`MacBookPro16,2`) | Intel Core i5-1038NG7 @ 2.00GHz | `x86_64` |

Every member runs `x86_64`, which is the same architecture Docker and Go call `amd64`.

### Why `amd64` only

- GitHub-hosted `ubuntu-latest` runners are `x86_64` machines, so CI builds `linux/amd64` natively. No QEMU emulation is needed, and the build runs at full speed.
- Every laptop in the team is also `x86_64`, so the image CI publishes is the same architecture the k3d cluster runs on. Each node pulls and starts the image natively.
- When the image and the host match, the risk of `exec format error` goes away. That error appears when a container starts a binary built for a different CPU architecture than the node it runs on.
- The Go binary is cross-compiled explicitly with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`, and the CI build step sets `platforms: linux/amd64`, so the target architecture never depends on the machine that runs the build.

### Known limitation

The image does not run natively on Apple Silicon or other ARM (`arm64`) hosts. On an Apple Silicon Mac, Docker Desktop can still run it through emulation, but more slowly. On an ARM-only Kubernetes node, the container fails to start with `exec format error`. No member of the team is affected today, but anyone who later runs the image on ARM hardware is.

Multi-arch builds are deferred to a Sprint 2 stretch goal.

### Risk R1: architecture divergence from the rubric

**Risk.** The assignment expects multi-arch to be considered. A team that builds only `amd64` without explaining why could lose marks for ignoring the Apple Silicon and ARM question.

**Mitigation.**

1. Sprint 1 builds `linux/amd64` only, and does so on purpose. The audit above, the reasoning, and the known limitation are written down in this document.
2. Each member's raw `uname -m` output is captured in `captures/step-1-arch-audit.txt`, so the audit is evidence, not only a claim.
3. The research write-up explains OCI image indexes (manifest lists), which let one tag point to an image for each architecture. This shows the trade-off is deliberate.
4. Multi-arch is a Sprint 2 stretch. The steps are:
   - add `docker/setup-qemu-action` and `docker/setup-buildx-action` to the workflow;
   - set `platforms: linux/amd64,linux/arm64` on `docker/build-push-action`, and stop hard-coding `GOARCH=amd64` in the Dockerfile (use the `TARGETARCH` build argument instead);
   - confirm the published tag is a manifest list with `docker buildx imagetools inspect ghcr.io/<owner>/devops-app:<tag>`.
