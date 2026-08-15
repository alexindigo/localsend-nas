# localsend-nas

Send-only [LocalSend](https://localsend.org) node with a web UI, built for
NAS-style deployments: browse host-mounted share directories in a browser,
assemble a basket of files, pick a discovered device, and the server sends the
files directly to the recipient's LocalSend app — the files never
round-trip through your laptop.

Speaks LocalSend Protocol v2.2 over HTTPS (port 53317 by default) and
announces itself via multicast, so it shows up in other devices' lists.
Incoming transfers are politely declined (`403`): this node is send-only.

Single static Go binary, no runtime dependencies, embedded web UI.

## Status

Early development. See the commit history and the plan document for scope.

## License

GNU Lesser General Public License v3.0 (LGPL-3.0). See [LICENSE](LICENSE).
