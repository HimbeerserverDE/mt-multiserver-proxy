# Permissions

The proxy comes with a permission system that can be used by plugins.
Some builtin features use it as well, namely chat commands.

## Design

Users cannot be assigned any permissions, but they can be part of a group.
Unless specified explicitly in the config users are assigned the `default`
group.

Groups can be assigned multiple permissions. These permissions then apply
to all players who are members of that group. Inexistent groups do not have
any permissions, so with no explicit configuration nobody has any permissions.

When granting permissions to a group, trailing wildcards are supported.
Any permission ending with a `*` will grant all permissions that start with
the string preceeding it. For example `cmd_*` grants access to all
chat commands provided by the official plugin.

## Configuration

Permissions are set in the config and cannot be modified by the proxy directly.
If necessary a plugin can directly access the configuration file, modify it
and perform a reload. This is not recommended, but the most likely use case
is a rank system which would reasonably have to be synchronized with the actual
Minetest servers as well, requiring a custom architecture anyway.
That architecture can then replace proxy permissions in the places
the rank system is needed in. One example would be exclusive servers:
There could be a chat command to switch to them. This command would need to be
available to all players, but it would perform an internal check with the rank
system to limit access to a subset of all players.
