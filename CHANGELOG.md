# Changelog

## 2026-08-15 (post-v0.1.2)

### Fix

Theme-toggle icons no longer depend on emoji fonts: bare codepoints without
VS16 fell back to monochrome text glyphs on Linux (U+1F5A5 rendered as an
ambiguous slab that read as a phone) — the toggle now uses inline SVGs that
render identically everywhere and follow `currentColor`. Cancelling a
waiting transfer now flips its state to `cancelled` immediately instead of
appearing stuck until the receiver timed out.

- fix: theme toggle icons as inline SVG (emoji glyphs unreliable on Linux) (`ea29410`)
- fix: cancel transitions to cancelled state immediately (`4024ed4`)

### Feature

UI polish pass, all visible in the header and lists: the v3 logo sits next
to the title (and centered atop the README), the palette is adopted from
localsend.org (deep teal on mint in light, mint on teal-black in dark),
Poppins is self-hosted and embedded, a GitHub repo link lives beside the
theme toggle, device list entries and finished transfers can be cleaned up
with an × button (devices DELETE endpoint now accepts discovered entries;
a live one simply re-announces), and the theme toggle's system icon
reflects the device class (phone on touch, monitor otherwise).

- feat: logo in page header; centered in README (`a8a15e8`)
- feat: self-hosted Poppins webfont; fix header alignment (`5746585`)
- feat: theme toggle system icon reflects device class (`b124c25`)
- feat: adopt localsend.org color palette (`86f5ed8`)
- feat: GitHub repo link in header (`fae0131`)
- feat: × remove for devices and terminal transfers (`35c51f1`)

## 2026-08-15 (post-v0.1.0)

### Fix

The Transfers panel stayed empty while a job was active and only appeared
once it finished: the cancel-button render chained off `Node.append()`,
which returns `undefined`, throwing mid-render for every non-terminal job.
Split into separate statements; verified in headless Chrome with a live
job in both states.

- fix: render cancel button without chaining Node.append (`20e9589`)

### Feature

Operability endpoints for automation: the LXC install script, Docker
operators, and monitoring can now verify the service over plain HTTP
without touching the mTLS LocalSend port. `GET /api/health` reports
liveness plus version/identity; `GET /api/selftest` runs per-subsystem
checks (share roots, LocalSend port self-dial, multicast listener) and
returns 503 with per-check error strings on failure, so
`curl -sf …/api/selftest` works as an install gate.

- feat: /api/health + /api/selftest operability endpoints (`ab3bce6`)

### Docs

New logo: the LocalSend broadcast-ring motif squared — a NAS box at the
center of the local network — white on their teal, with README
attribution to the LocalSend project (© Tien Do Nam, Apache-2.0).
Favicon re-rendered from it, and a new apple-touch-icon plus its
`<link>` makes the UI presentable as a mobile home-screen icon. README
also gained a docker-compose example for Synology/TrueNAS-style hosts.

- docs: LocalSend-style squared logo + apple-touch-icon (`b8f914e`)
- docs: docker-compose example (`ef7f2c4`)

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
