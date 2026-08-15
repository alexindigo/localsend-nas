# Changelog

## 2026-08-15 (post-v0.1.0)

### Feature

Operability endpoints for automation: the LXC install script, Docker
operators, and monitoring can now verify the service over plain HTTP
without touching the mTLS LocalSend port. `GET /api/health` reports
liveness plus version/identity; `GET /api/selftest` runs per-subsystem
checks (share roots, LocalSend port self-dial, multicast listener) and
returns 503 with per-check error strings on failure, so
`curl -sf …/api/selftest` works as an install gate.

- feat: /api/health + /api/selftest operability endpoints (`ab3bce6`)

## 2026-08-15

### Feature

Initial build of localsend-nas: a send-only LocalSend Protocol v2.2 node with
an embedded web UI, aimed at NAS-style deployments where files should travel
share → server → recipient device without round-tripping through the
browser's machine. The node announces itself over multicast (224.0.0.167:53317),
keeps a persistent ECDSA identity whose SHA-256 fingerprint is its device ID,
and speaks mutual TLS the way v2.1+ receivers require (client certificate
presented unconditionally, uppercase-hex fingerprint). Incoming transfers are
politely declined with 403 while /info and /register keep the node visible in
other devices' lists. The web UI offers share browsing with a basket flow,
device management (discovered + manual IP targets), and live transfer progress
over SSE; it ships dark mode (system-aware with a persisted toggle) and a
project logo/favicon. File access is read-only and path-confined per share
root, with symlink-escape and traversal guards covered by unit tests.
Verified end-to-end against the official LocalSend CLI 1.18.0 (byte-identical
delivery). Docker image published to GHCR (multi-arch amd64/arm64) alongside
the release tarballs.

- feat: identity + config (persistent cert, share parsing) (`6980f2c`)
- feat: localsend protocol client (info/prepare/upload/cancel) (`66078aa`)
- feat: discovery — multicast announce/listen, registry, manual probe (`a1c1649`)
- feat: send-only reject server on :53317 (`e977ee5`)
- feat: share store with confined read-only access (`de4c8c8`)
- feat: transfer manager (basket sessions, progress, cancel) (`afd2ff8`)
- feat: http API + embedded SPA (`d1d5d61`)
- feat: web UI dark mode (system-aware, persisted toggle) (`eb35469`)
- feat: favicon from project logo (`6cae9cc`)
- feat: Docker image (multi-arch GHCR publish on tag) (`2ab9df6`)

### Docs

Original project logo (teal gradient + send stripes), SVG source plus PNG
renders; referenced from the README and reused as the favicon.

- docs: project logo (`753a255`)

### CI

Tag-driven release pipeline: cross-compiled static binaries packed as
`localsend-nas_<ver>_linux_{amd64,arm64}.tar.gz` with checksums and a
GitHub Release, plus the Docker→GHCR publish workflow.

- ci: tag-triggered release builds (tar.gz, amd64/arm64) (`8f22682`)

### Chore

Repo scaffolding (go.mod pinned Go, LGPL-3.0, README, .gitignore) and module
path settlement to the personal account.

- chore: repo init — go.mod, LICENSE, README skeleton (`8de3754`)
- chore: module path → github.com/alexindigo/localsend-nas (`8bbdf5f`)
