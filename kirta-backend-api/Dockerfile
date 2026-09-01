# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/
COPY docs/ ./docs/

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /out/kirta-backend-api ./cmd/main.go

# Pin scanner versions so the image is reproducible.
FROM anchore/syft:v1.51.1 AS syft
FROM anchore/grype:v0.118.0 AS grype

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        python3 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system kirta \
    && useradd --system --gid kirta --no-create-home kirta

WORKDIR /app

COPY --from=builder /out/kirta-backend-api ./kirta-backend-api
COPY --from=syft /syft /usr/local/bin/syft
COPY --from=grype /grype /usr/local/bin/grype

# graph.py imports helper functions from tracer.py.
COPY tracer.py ./tracer.py
COPY sca.py ./sca.py
COPY graph.py ./graph.py
COPY config.example.yaml ./config.example.yaml

RUN chown -R kirta:kirta /app

USER kirta

EXPOSE 8080

# Create config.yaml from config.example.yaml (or mount it) before running.
ENTRYPOINT ["./kirta-backend-api"]
