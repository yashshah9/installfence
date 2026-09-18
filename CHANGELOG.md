# Changelog

## [0.7.1] - 2026-09-19

- Darwin hide-paths: QuoteMeta + deny writes


## [0.7.0] - 2026-09-18

### Added
- `installfence compat list|run` — package compatibility matrix (100 packages across pip/npm/uv)
- `config/compat-matrix.yaml` with `smoke` tags for fast CI
- `--mode dry-run|install`, `--tag`, `--ecosystem`, `--fail-under`, `--json`
- Compose services `compat-dry` / `compat-install` and weekly GitHub Actions workflow

## [0.6.0] - 2026-09-14

### Added
- Policy `npm_wrap_scripts: always|allowlisted|never` (default `always`) for npm/npx install-like commands
- Optional `npm_script_allowlist` — empty allowlist with `allowlisted` forces `--ignore-scripts`; non-empty keeps sandboxed installs (list reserved for future script-name filtering)
- `installfence npx` convenience wrapper

## [0.5.0] - 2026-09-14

### Added
- Parse bubblewrap/sandbox stderr for deny patterns (`Permission denied`, `Operation not permitted`, `bwrap:`, `EACCES`) and record structured violations
- Sandbox runners tee stderr into the violation collector (still streams to the terminal)

## [0.4.0] - 2026-09-14

### Added
- `allow_write_paths` policy field — bind-mount writable exceptions for native builds (host path must exist)

## [0.3.0] - 2026-08-19

### Added
- `installfence shim install|uninstall|status|env` PATH wrappers for pip/uv/npm/pnpm/yarn

## [0.2.0] - 2026-08-19

### Added
- Violation collector with `--json-violations` and `--fail-on-violation` (exit 42)
- `--require-sandbox` fail-closed mode
- Runtime `hide_env_keys` via process env scrub + bwrap `--unsetenv`
- macOS `sandbox-exec` backend (best-effort Seatbelt profile)
- `go.sum` so Docker/CI builds no longer fail on missing cobra checksums

### Notes
- Real bubblewrap needs Linux user namespaces (Docker: `privileged: true`)

## [0.1.0] - 2026-08-18

### Added
- Initial CLI with `health`, `run`, `pip`, `uv`, `npm` subcommands
- YAML policy loading with default secret-hiding paths
- Bubblewrap sandbox integration (with passthrough fallback)
- Docker dev/test environment
