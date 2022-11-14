# gcat

`gcat` is a tool for penetration testers and sysadmins.
Its design is roughly based on `socat` (hence the name).
However, `gcat` provides the following delta to `socat`:

- `serve` command: `gcat` allows starting several different servers for quick usage.
  The `serve` command might be used in penetration tests or quick 'n' dirty lab setups.
  Here is an excerpt for supported protocols: `doh`, `ftp`, `http`, `ssh`, `webdav`.

- `proxy` command: it works similar to `socat`. Data is copied between two proxy modules (such as `quic`, `tls`, or `stdin`) specified as command line arguments.

- Written in Go: it is easy to compile `gcat` to a static binary with **no** runtime dependencies.
