FROM golang:1.22-bookworm AS builder
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends bubblewrap && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /installfence ./cmd/installfence

FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends bubblewrap ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /installfence /usr/local/bin/installfence
COPY config/compat-matrix.yaml /etc/installfence/compat-matrix.yaml
ENV INSTALLFENCE_COMPAT_MATRIX=/etc/installfence/compat-matrix.yaml
ENTRYPOINT ["installfence"]

# Compat installs need package managers + privileged for real bwrap.
FROM runtime AS compat
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip python3-venv nodejs npm \
    && rm -rf /var/lib/apt/lists/*

FROM builder AS dev
WORKDIR /src
CMD ["go", "test", "./...", "-v"]
