# Contributing

## Running tests

Prefer Docker Compose:

```bash
docker compose run --rm dev      # go test ./...
docker compose run --rm health
```

Locally:

```bash
go test ./... -v
go build -o bin/installfence ./cmd/installfence
./bin/installfence health
```

## Pull requests

- Keep changes focused; include `go test ./...` results
- Update README/CHANGELOG for user-facing policy or CLI changes
- Real bubblewrap tests need Linux user namespaces (`privileged: true` in Compose)

## Commit style

- Imperative subject line; mention the user-facing why when relevant
- Do not add AI co-author trailers (e.g. Co-authored-by: Cursor) to commits.
