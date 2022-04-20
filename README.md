# mt-multiserver-proxy

mt-multiserver-proxy is a reverse proxy designed for linking
multiple Minetest servers together. It is the successor to multiserver.

## mt
This project was made possible by [anon55555's mt module](https://github.com/anon55555/mt).

## Installation
Go 1.18 or higher is required. Run

`go install github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/mt-multiserver-proxy@latest`

to download and compile the project. A mt-multiserver-proxy executable
will be created in your $GOBIN directory. The same command is also
used to upgrade to the latest version. You will need to recompile
all plugins after upgrading.

## Usage

### Starting
Run `$GOBIN/mt-multiserver-proxy`. The configuration file and other required
files are created automatically in the directory the executable
(or symlink to said executable) is in, so make sure to move the
executable to the desired location or use a symlink.

### Stopping
mt-multiserver-proxy reacts to SIGINT, SIGTERM and SIGHUP. It stops listening
for new connections, kicks all clients, disconnects from all servers
and exits. If some clients aren't responding, mt-multiserver-proxy waits until
they have timed out.

## Configuration
The configuration file name and format including a minimal example
are described in [doc/config.md](doc/config.md).
__All internal servers need to allow empty passwords
and must not be reachable from the internet!__

## Chat commands
The default chat commands can be installed as a [plugin](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands).

## Telnet interface
Chat commands can also be executed over a telnet connection.
See [telnet.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/telnet.md)
for details.
