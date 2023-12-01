# mt-multiserver-proxy

mt-multiserver-proxy is a reverse proxy designed for linking
multiple Minetest servers together. It is the successor to multiserver.

## mt
This project was made possible by [anon55555's mt module](https://github.com/anon55555/mt).

## Installation
Go 1.21 or higher is required. Run

```
go install github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/...@latest
```

to download and compile the project. A mt-multiserver-proxy executable
will be created in your ${GOBIN} directory. The same command is also
used to upgrade to the latest version. You will need to recompile
all plugins after upgrading.

In addition to the main `mt-multiserver-proxy` binary the following
additional utilities are installed:

* [mt-auth-convert](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/auth_backends.md#mt-auth-convert): Helper program to convert between authentication database formats.
* [mt-build-plugin](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/plugins.md#automatic-version-management): Utility for building plugins against the correct proxy version.

You can replace the `...` in the installation command
with any of the binary names to limit installation and updating
to a single executable. **This is not recommended, however,
as it can cause version mismatches between them.**

## Usage

### Starting
Run `${GOBIN}/mt-multiserver-proxy`. The configuration file and other required
files are created automatically in the directory the executable is in,
so make sure to install the executable to the desired location.
Symlinks to the executable will be followed, only the real path matters.

### Stopping
mt-multiserver-proxy reacts to SIGINT, SIGTERM and SIGHUP. It stops listening
for new connections, kicks all clients, disconnects from all servers
and exits. If some clients aren't responding, mt-multiserver-proxy waits until
they have timed out.

## Configuration
The configuration file name and format including a minimal example
are described in [doc/config.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/config.md).
**All internal servers need to allow empty passwords
and must not be reachable from the internet!**

## Authentication database migration
It is possible to import existing Minetest authentication databases.
See [doc/auth_backends.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/auth_backends.md)
for details.

## Chat commands
The default chat commands can be installed as a [plugin](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands).

## Telnet interface
Chat commands can also be executed over a telnet connection.
See [doc/telnet.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/telnet.md)
for details.

## Plugins
This proxy supports loading Go plugins.
Consult [doc/plugins.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/plugins.md)
for details on how to develop or install them.
