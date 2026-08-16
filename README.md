<p align="center"><img src="web/logo.png" width="200" alt="localsend-nas logo"></p>

# localsend-nas

A [LocalSend](https://localsend.org) node with a web UI, built for
NAS-style deployments: browse host-mounted share directories in a browser,
assemble a basket of files, pick a discovered device, and the server sends the
files directly to the recipient's LocalSend app — the files never
round-trip through your laptop. Receiving works too: a desktop-app-style
dialog lets you accept inbound files straight into a share.

Speaks LocalSend Protocol v2.2 over HTTPS (port 53317 by default) and
announces itself via multicast, so it shows up in other devices' lists.

## Receiving

On by default. When a device sends files to the NAS, every open UI tab pops a
dialog with the sender, file list, a destination picker (any configured
share), and a countdown (default 30 s):

- **Accept / Decline** — your call, per transfer.
- **Countdown expires** — with a *dropbox* configured, files are auto-accepted
  into that share; otherwise the request is politely rejected.
- **`--read-only`** (env `LOCALSEND_NAS_READ_ONLY=1`) — restores the strict
  v1 send-only posture: no receiving at all (senders get a clean 403).

Countdown length and the dropbox share live in the settings menu (gear icon,
top right) and persist server-side in `<data-dir>/settings.json`.

Received files keep their folder structure, never overwrite existing files
(`name (1).ext` dedupe), and land in shares as normal files — browsable and
re-sendable immediately.

Single static Go binary, no runtime dependencies, embedded web UI.

## Running

Binary (see [releases](https://github.com/alexindigo/localsend-nas/releases)):

```bash
localsend-nas --listen :80 --share movies=/srv/movies --share books=/srv/books
```

Docker (`ghcr.io/alexindigo/localsend-nas`):

```bash
docker run -d --name localsend-nas --network host \
  -v /srv/movies:/srv/movies:ro -v /srv/books:/srv/books:ro \
  -v localsend-nas-data:/data \
  -e LOCALSEND_NAS_SHARES="movies=/srv/movies,books=/srv/books" \
  ghcr.io/alexindigo/localsend-nas:latest
```

`--network host` is recommended: LocalSend discovery rides multicast UDP
53317, which NAT/bridge networking breaks. All flags have
`LOCALSEND_NAS_*` env equivalents (`--listen` → `LOCALSEND_NAS_LISTEN`,
repeatable `--share` → comma-separated `LOCALSEND_NAS_SHARES`, …).

docker-compose equivalent:

```yaml
services:
  localsend-nas:
    image: ghcr.io/alexindigo/localsend-nas:latest
    container_name: localsend-nas
    network_mode: host          # required for multicast discovery
    restart: unless-stopped
    environment:
      LOCALSEND_NAS_SHARES: movies=/srv/movies,books=/srv/books
      # LOCALSEND_NAS_LISTEN: ":8080"   # image default; :80 needs root
      # LOCALSEND_NAS_ALIAS: "Nas living-room"
    volumes:
      - /srv/movies:/srv/movies:ro    # shares are read-only by design
      - /srv/books:/srv/books:ro
      - localsend-nas-data:/data      # persistent device identity

volumes:
  localsend-nas-data:
```

With `network_mode: host` the web UI lands on the host's port 8080
directly; change `LOCALSEND_NAS_LISTEN` if that clashes.

## Operability (for scripts and monitoring)

- `GET /api/health` — liveness + identity: `{"version","protocol","alias","fingerprint","lsPort"}`.
  Always 200 when the process serves HTTP.
- `GET /api/selftest` — subsystem diagnostics: share roots re-stat'd,
  LocalSend port self-dial, multicast listener state, known-device count.
  `200 {"ok":true,…}` when healthy, `503` with per-check error strings
  otherwise — so `curl -sf …/api/selftest` works as an install gate.

## Status

Early development. See the commit history and the plan document for scope.

Logo: `web/logo.svg` (source) / `web/logo.png` (rendered) — the LocalSend
"device + broadcast ring" motif, squared: a NAS box at the center of the
local network. Style derived from the LocalSend logo (© Tien Do Nam,
[Apache-2.0](https://github.com/localsend/localsend)) as an ecosystem
homage — this is not an official LocalSend project.

## License

GNU Lesser General Public License v3.0 (LGPL-3.0). See [LICENSE](LICENSE).
