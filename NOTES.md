# Notes

## Pillar 1

- Image `devops-app` (distroless `static:nonroot`, `linux/amd64`): **19.9 MB** uncompressed in `docker image ls`, 4.74 MB compressed content.
- Pulling `golang:1.27` stalled on the same layers from both Docker Hub and `mirror.gcr.io`, while `curl` fetched those layers at about 10 MB/s. Docker Desktop's pull engine was stuck. Restarting Docker Desktop fixed it, and the pull then took 21 s.
