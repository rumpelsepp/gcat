# gcat

`gcat` is a tool for penetration testers and sysadmins.
Its design is roughly based on `socat` (hence the name).
However, `gcat` provides the following delta to `socat`:

- `serve` command: `gcat` allows starting several different servers for quick usage.
  The authors main purpose might be penetration tests or quick 'n' dirty lab setups.
  Here is an excerpt for supported protocols: `ftp`, `http`, `ssh`, `webdav`.

- `proxy` command: it works similar to `socat`. Data is proxied between two proxy modules, 
  specified as command line arguments.

## Examples

### Netcat

```
$ gcat proxy tcp-listen://localhost:1234 -
```

### HTTP Server

```
$ gcat serve http
```
