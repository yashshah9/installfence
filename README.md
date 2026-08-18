# installfence

Sandboxed package installation for **pip**, **uv**, **npm**, and other package managers. Install scripts run inside a kernel-level sandbox where SSH keys, cloud credentials, and `.env` files are hidden.

> **Status:** v0.1 foundation — bubblewrap sandbox wrapper with policy YAML; violation reporting and macOS backend are next.

## Problem

Package install hooks execute arbitrary code with your full user privileges. npm 12 now blocks lifecycle scripts by default, but approved scripts still run unsandboxed. **Python has no equivalent** — every `pip install` can read `~/.ssh` and environment secrets.

## Key features (v0.1)

- Wrap `pip`, `uv`, `npm`, or any command in a bubblewrap sandbox
- Default policy hides `~/.ssh`, `~/.aws`, `~/.gnupg`, and secret env vars
- YAML policy files for per-project customization
- `--dry-run` mode for CI validation without bubblewrap
- Graceful fallback when bubblewrap is not installed (with warning)

## Architecture

```
installfence CLI (Go/cobra)
    │
    ├── policy loader (YAML)
    └── sandbox runner (bubblewrap)
            └── passthrough fallback
```

| Component | Technology | Why |
|-----------|------------|-----|
| Language | Go 1.22 | Single static binary, easy distribution |
| CLI | cobra | Standard Go CLI framework |
| Sandbox | bubblewrap | Lightweight, kernel namespaces, no daemon |
| Policy | YAML | Human-readable, version-controllable |

## Installation

```bash
go install github.com/installfence/installfence/cmd/installfence@latest
# or
go build -o bin/installfence ./cmd/installfence
```

## Local development

```bash
go mod tidy
go test ./... -v
go build -o bin/installfence ./cmd/installfence
./bin/installfence health
./bin/installfence run --dry-run pip install requests
```

## Docker

Requires Linux with bubblewrap for real sandboxing:

```bash
docker compose run --rm dev      # run tests
docker compose run --rm health   # health check
docker compose run --rm sandbox-pip
```

## Configuration

Policy file (`config/default-policy.yaml`):

```yaml
allow_network: true
hide_paths:
  - ~/.ssh
  - ~/.aws
hide_env_keys:
  - GITHUB_TOKEN
  - AWS_SECRET_ACCESS_KEY
```

| Flag | Description |
|------|-------------|
| `--policy PATH` | Custom policy YAML |
| `--dry-run` | Show plan, don't enforce sandbox |

## Usage

```bash
installfence health
installfence pip install requests
installfence uv pip install -r requirements.txt
installfence npm install
installfence run -- pip install -e .
installfence run --dry-run npm ci
```

## Running tests

```bash
go test ./... -v
```

## Roadmap

- [ ] Violation reporting (log blocked file access attempts)
- [ ] macOS sandbox-exec backend
- [ ] Shell shims for transparent PATH interception
- [ ] Top-100 package compatibility test matrix

## License

MIT

## Known limitations (v0.1)

- Linux bubblewrap only (macOS/Windows backends not implemented)
- No install-script allowlist integration with npm 12
- Native builds may need policy exceptions for compiler access
- Violation reporting is not yet implemented
