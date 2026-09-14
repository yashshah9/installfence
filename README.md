# installfence

Sandboxed package installation for **pip**, **uv**, **npm**, and other package managers. Install scripts run inside a kernel-level sandbox where SSH keys, cloud credentials, and `.env` files are hidden.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go 1.22+](https://img.shields.io/badge/go-1.22+-00ADD8.svg)](https://go.dev/)
[![CI](https://github.com/yashshah9/installfence/actions/workflows/ci.yml/badge.svg)](https://github.com/yashshah9/installfence/actions/workflows/ci.yml)

> **Status:** v0.6 — npm wrap script modes (`always`/`allowlisted`/`never`), violation reporting, PATH shims, and `allow_write_paths`.

## 60-second try

```bash
go install github.com/yashshah9/installfence/cmd/installfence@v0.6.0
installfence health
# or with Docker:
docker compose run --rm health
docker compose run --rm env-probe   # privileged; proves env secrets stay hidden
```

## Why this vs alternatives

| Approach | Strength | Gap |
|----------|----------|-----|
| **installfence** | Policy YAML + bubblewrap/shim for pip/uv/npm | Linux-first; macOS is best-effort |
| `npm install --ignore-scripts` | Blocks lifecycle scripts | No Python/uv coverage; approved scripts still unsandboxed |
| Manual bubblewrap | Full control | No package-manager UX or default secret policy |
| Full Docker per install | Strong isolation | Heavyweight for every `pip install` |

## Problem

Package install hooks execute arbitrary code with your full user privileges. npm 12 now blocks lifecycle scripts by default, but approved scripts still run unsandboxed. **Python has no equivalent** — every `pip install` can read `~/.ssh` and environment secrets.

## Key features (v0.6)

- Wrap `pip`, `uv`, `npm`, `npx`, or any command in a bubblewrap sandbox (Linux)
- PATH shims: `installfence shim install` then `eval "$(installfence shim env)"`
- Default policy hides `~/.ssh`, `~/.aws`, `~/.gnupg`, and secret env vars at runtime
- `npm_wrap_scripts` — `always` (sandbox lifecycle scripts), `never` (force `--ignore-scripts`), or `allowlisted` (empty list → ignore-scripts; non-empty → sandboxed, list reserved for future filtering)
- `allow_write_paths` — bind-mount writable exceptions for native builds (paths must exist on the host)
- Stderr deny parsing → violation records (`Permission denied`, `EACCES`, `bwrap:`)
- `--json-violations` / `--fail-on-violation` (exit 42) for CI
- `--require-sandbox` instead of unsandboxed passthrough
- macOS `sandbox-exec` backend when available
- `--dry-run` mode without enforcing

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
go install github.com/yashshah9/installfence/cmd/installfence@v0.6.0
# or latest tip of main:
go install github.com/yashshah9/installfence/cmd/installfence@latest
# or from a clone:
go build -o bin/installfence ./cmd/installfence
```

## Local development

```bash
go mod tidy
go test ./... -v
go build -o bin/installfence ./cmd/installfence
./bin/installfence health
./bin/installfence --dry-run pip install requests
./bin/installfence run --dry-run -- pip install requests
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
# Writable overrides for native builds (host path must exist):
# allow_write_paths:
#   - /tmp/build-cache
# npm install/ci/add lifecycle scripts:
# npm_wrap_scripts: always   # always|allowlisted|never
# npm_script_allowlist: []   # empty + allowlisted → --ignore-scripts
```

`allow_write_paths` adds `--bind` mounts on top of the read-only root. Use it when compilers or package build scripts need a writable cache or scratch directory. Paths are expanded (`~` supported) and skipped if they do not exist on the host.

`npm_wrap_scripts` controls lifecycle scripts on `npm`/`npx` `install`/`ci`/`add` (and `i`). Default `always` sandboxes as today. `never` appends `--ignore-scripts`. `allowlisted` with an empty `npm_script_allowlist` also forces `--ignore-scripts`; a non-empty list keeps the current sandboxed behavior (the names themselves are reserved for future script-name filtering — complementary to npm 12, not a replacement).

| Flag | Description |
|------|-------------|
| `--policy PATH` | Custom policy YAML |
| `--dry-run` | Show plan, don't enforce sandbox |
| `--require-sandbox` | Fail if no sandbox backend is available |
| `--fail-on-violation` | Exit 42 when a violation is recorded |
| `--json-violations` | NDJSON violation stream on stderr |

## Usage

```bash
installfence health
installfence --dry-run pip install requests
installfence pip install requests
installfence uv pip install -r requirements.txt
installfence npm install
installfence npx --yes cowsay hi
installfence run -- pip install -e .
installfence shim install
eval "$(installfence shim env)"
```

## Running tests

```bash
go test ./... -v
```

## Roadmap

- [x] Violation reporting (structured logs + CI exit code)
- [x] macOS sandbox-exec backend (best-effort)
- [x] Shell shims for transparent PATH interception
- [x] `allow_write_paths` policy for native builds
- [x] npm wrap script modes (`npm_wrap_scripts` / `npm_script_allowlist`)
- [ ] Top-100 package compatibility test matrix

## Known limitations (v0.6)

- Flags such as `--dry-run` and `--require-sandbox` must go **before** `pip`/`npm`/`uv` (`installfence pip` disables cobra flag parsing so pip flags pass through)
- Linux bubblewrap needs user namespaces (Docker Compose `privileged: true` for real sandbox)
- macOS Seatbelt profile is best-effort, not a full bwrap parity matrix
- `npm_script_allowlist` does not yet filter by script name (non-empty list only opts out of forced `--ignore-scripts`)
- Native builds may need `allow_write_paths` exceptions for compiler/cache access
- `allow_write_paths` entries that do not exist on the host are skipped

## License

MIT
