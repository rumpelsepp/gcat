# gcat

`gcat` is a tool for penetration testers and sysadmins.
Its design is roughly based on `socat` (hence the name).
However, `gcat` provides the following delta to `socat`:

- `serve` command: `gcat` allows starting several different servers for quick usage.
  The `serve` command might be used in penetration tests or quick 'n' dirty lab setups.
  Here is an excerpt for supported protocols: `doh`, `ftp`, `http`, `ssh`, `webdav`.

- `proxy` command: it works similar to `socat`. Data is proxied between two proxy modules, 
  specified as command line arguments. The `proxy` command uses URLs for its arguments.

- Written in Go: it is trivial to compile `gcat` to a static binary with no runtime dependencies.

## Proxy

The `proxy` command needs two arguments which specify the data pipeline.
The arguments are URLs; in some rare cases it might be required to escape certain parts of the url.

If the second argument is not present, `gcat` defaults to `-` which is `stdio`.
Thus, the second argument in all the following examples is `-`.

### exec

Read and write from/to a command.

Arguments:

* `cmd`: The relevant command.

Example:

```
$ gcat proxy 'exec:?cmd=cat -'
```

Short form:

```
$ gcat proxy 'exec:cat -'
```

### stdio

No arguments.

Example:

```
$ gcat proxy stdio:
```

Short form:

```
$ gcat proxy -
```

### tcp

Act as a TCP client.

Arguments:

* Host: The target to connect to.

Example:

```
$ gcat proxy tcp://localhost:1234
```

### tcp-listen 

Act as a TCP server.

Example:

```
$ gcat proxy tcp-listen://localhost:1234
```

### tun 

Allocate a `tun` device and proxy ip traffic.

Arguments:

- Host: IP address to assign to the device
- Path: Subnet Mask 
- mtu: The MTU of the `tun` device (default 1500) 
- dev: Device name; can include `%d` for letting the kernel chose an index.

Example:

```
# gcat proxy 'tun://10.0.0.1/24?dev=tun%d'
```

Note: Root permissions or `CAP_NET_ADMIN` required.

## Serve 

The `serve` command is implemented with a traditional command line interface.
Just try out the `--help` pages.

## Examples

Listen on localhost tcp port 1234.

```
$ gcat proxy tcp-listen://localhost:1234 -
```

Forward TCP traffic from `localhost:8080` to `1.1.1.1:80`:

```
$ gcat proxy tcp-listen://localhost:1234 tcp://1.1.1.1:80
```

[Tunnel IP traffic through SSH](https://rumpelsepp.org/blog/vpn-over-ssh/) (an SSH proxy might eventually be included as well):

```
# gcat proxy "tun://192.168.255.1/24" exec:'ssh -l root HOST "gcat tun://192.168.255.2/24"'
```

### HTTP Server

```
$ gcat serve http
```

### SSH Server with Host Key and authorized\_keys

```
$ gcat serve ssh -k /etc/ssh/ssh_host_ed25519_key -a ~/.ssh/authorized_keys
```
