FROM golang:1.22-bookworm AS builder
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends bubblewrap && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -o /installfence ./cmd/installfence

FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends bubblewrap ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /installfence /usr/local/bin/installfence
ENTRYPOINT ["installfence"]

FROM builder AS dev
WORKDIR /src
CMD ["go", "test", "./...", "-v"]
