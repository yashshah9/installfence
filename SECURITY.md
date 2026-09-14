# Security Policy

## Reporting a vulnerability

Email **yash376351@gmail.com** with the repo name, a short description, and steps to reproduce. Please do not open a public issue for exploitable findings until we have had a reasonable chance to respond.

## Threat model (honest)

installfence reduces blast radius for **package install scripts** by hiding common secret paths/env keys and (on Linux) running under bubblewrap.

- It is **not** a full secure enclave or substitute for container/VM isolation.
- macOS `sandbox-exec` support is **best-effort**, not bubblewrap parity.
- Without a working sandbox backend, the default path may fall through to unsandboxed execution unless you pass `--require-sandbox`.
- `allow_write_paths` deliberately weakens the read-only root for listed paths — only add directories you understand.
- Do not treat a green install as proof that a package is safe.
