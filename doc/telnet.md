# Telnet interface
mt-multiserver-proxy provides a telnet interface that can be used to
execute proxy chat commands.
## Differences to chat interface
Telnet clients can execute any chat command. They do not have any
privileges, only the chat command permission check will succeed.
## Security
There is no authentication at all. For this reason the telnet server
only listens on the loopback interface by default.
## Connecting
The telnet server listens on the IPv6 loopback address "::1"
and TCP port 40010 by default. Use the telnet command to connect.
## Disconnecting
Type \quit or \q to close the connection. All telnet clients will also
be disconnected when the proxy shuts down.
