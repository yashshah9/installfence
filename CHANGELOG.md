# Changelog

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
- Full bwrap audit/seccomp violation parsing is still open

## [0.1.0] - 2026-08-18

### Added
- Initial CLI with `health`, `run`, `pip`, `uv`, `npm` subcommands
- YAML policy loading with default secret-hiding paths
- Bubblewrap sandbox integration (with passthrough fallback)
- Docker dev/test environment
