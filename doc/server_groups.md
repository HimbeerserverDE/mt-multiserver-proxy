# Server groups

Servers can be made members of multiple server groups by listing them
in the `Groups` subfield of the server definition in the config.
Configuration options that support server groups will randomly choose
from their member servers every time they are applied to a client.

Fallback servers cannot be server groups.

If there is a server group with the same name as a regular server,
the regular server is preferred, rendering the group inaccessible.

## Use cases

Server groups provide a simple builtin load balancing solution.
The default server may be a server group, distributing the load
over multiple Minetest lobby servers.

Server groups are also available to plugins, making it possible to use them
for custom features such as requiring special permissions to access a certain
server group.
