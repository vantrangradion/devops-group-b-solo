# Notes

## Pillar 1

- **Keep:** distroless `static:nonroot` with a `CGO_ENABLED=0` static binary. Image `devops-app` (`linux/amd64`) is **19.9 MB** uncompressed in `docker image ls` and 4.74 MB compressed, runs as `65532`, and has no shell.
- **Broke:** pulling `golang:1.27` stalled on the same layers from both Docker Hub and `mirror.gcr.io`, while `curl` fetched those layers at about 10 MB/s, so the network was fine and Docker Desktop's pull engine was stuck. **Fixed:** restarting Docker Desktop; the pull then took 21 s.
- **Checked the failure path on purpose:** throwaway PR #1 (closed, not merged) first broke `TestHealth`. `Test` failed and `Build & Push` was skipped. After the revert, the image built with `push: false` and nothing reached GHCR (`captures/step-1-pr-test-fail.png`, `captures/step-1-pr-build-no-push.png`). **Keep:** prove the gates with a deliberate failure instead of trusting the YAML.
- **GHCR visibility:** the first push from `main` created the package as private. I set it to Public, then checked an anonymous pull: a manifest request for `ghcr.io/vantrangradion/devops-app:latest` with an anonymous token returns `200`, so Pillar 2 needs no image-pull secret.
- **Keep:** each version pin lives in one place per tool (`go.mod` → `setup-go` via `go-version-file`; golangci-lint `v2.14.0` in the `Makefile` and `ci.yml`). `make lint` refuses to run on a different golangci-lint version, so local and CI lint results can't drift.
