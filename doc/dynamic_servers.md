# Dynamic servers

## About

While servers are traditionally defined in the config
plugins have the ability to add new servers at runtime.
Dynamic servers can be deleted when they are no longer needed
and no players are connected to them. Statically defined servers
cannot be removed at runtime. Dynamic servers must be
part of a media pool that has at least one static member.
They are lost when the proxy restarts.

This feature can be useful to implement things like starting
minigame servers as needed.

## Adding servers at runtime

A plugin may call [AddServer](https://pkg.go.dev/github.com/HimbeerserverDE/mt-multiserver-proxy#AddServer)
to dynamically create a server at any time.
It returns a boolean indicating success.

### Conditions

* Server name is not taken
* Media pool contains at least one static member

## Removing servers at runtime

A plugin may call [RmServer](https://pkg.go.dev/github.com/HimbeerserverDE/mt-multiserver-proxy#RmServer)
to dynamically delete a dynamic server at any time.
It returns a boolean indicating success or an already inexistent server.

### Conditions

* Server was dynamically added
* No player connections
