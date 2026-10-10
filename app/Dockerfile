# Builder: golang:1.27 matches the go directive in go.mod.
FROM golang:1.27 AS build

WORKDIR /src

# Download modules first so this layer is cached until go.mod or go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

# Static amd64 binary: distroless/static has no libc.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/devops-app .

# Runtime: no shell, no package manager, runs as nonroot.
FROM gcr.io/distroless/static:nonroot

COPY --from=build /out/devops-app /devops-app

# Numeric UID:GID so Kubernetes runAsNonRoot can verify the user.
USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/devops-app"]
