# Server selectors

If needed, plugins can provide a custom function to choose the server to
connect a new client to. To do this, call the [RegisterSrvSelector](https://pkg.go.dev/github.com/HimbeerserverDE/mt-multiserver-proxy#RegisterSrvSelector)
function.

Use the `SrvSelector` configuration option to choose one of the registered
server selectors. This option is reloadable.

If the configured server selector doesn't exist or doesn't return a server, the
proxy uses the builtin server selection strategy. In the case where the
configured server selector doesn't exist, a log message is generated.
