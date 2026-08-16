# localsend-nas

<img src="web/logo.png" alt="localsend-nas logo" width="128" align="right">

Send-only [LocalSend](https://localsend.org) node with a web UI, built for
NAS-style deployments: browse host-mounted share directories in a browser,
assemble a basket of files, pick a discovered device, and the server sends the
files directly to the recipient's LocalSend app — the files never
round-trip through your laptop.

Speaks LocalSend Protocol v2.2 over HTTPS (port 53317 by default) and
announces itself via multicast, so it shows up in other devices' lists.
Incoming transfers are politely declined (`403`): this node is send-only.

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
