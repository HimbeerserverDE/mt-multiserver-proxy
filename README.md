# MTConnector
MTConnector is a reverse proxy designed for linking multiple Minetest servers together
## Installation
Go 1.16 or higher is required. Run

`go install github.com/HimbeerserverDE/MTConnector`

to download and compile the project. A MTConnector executable
will be created in your $GOBIN directory.
## Usage
### Starting
Run `$GOBIN/MTConnector`. The configuration file and other required
files are created automatically in the directory the executable
(or symlink to said executable) is in, so make sure to move the
executable to the desired location or use a symlink.
### Stopping
MTConnector reacts to SIGINT, SIGTERM and SIGHUP. It stops listening
for new connections, kicks all clients, disconnects from all servers
and exits. If some clients aren't responding, MTConnector waits until
they have timed out.
## Configuration
The configuration file name and format are described in [doc/config.md](doc/config.md)
**All internal servers need to allow empty passwords and must not be reachable from the internet!**
