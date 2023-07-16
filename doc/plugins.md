# Plugins
mt-multiserver-proxy loads all plugin files in the `plugins` directory
on startup. Any errors will be logged and do not prevent other plugins
from being loaded. Plugins **cannot** be (re)loaded at runtime, you
need to restart the proxy.

## Installing plugins
To install a plugin, clone the repository, cd into it and run:

```
go build -buildmode=plugin
```

A .so file will be created. Copy or move this file into the `plugins`
directory. Restart the proxy to load the plugin.

## Developing plugins
A plugin is simply a main package without a main function. Use the init
functions instead. Plugins can import
`github.com/HimbeerserverDE/mt-multiserver-proxy` and use the exported
symbols to control the behavior of the proxy. The API is documented
[here](https://pkg.go.dev/github.com/HimbeerserverDE/mt-multiserver-proxy).
__The plugin API may change at any time without warning.__

## Common issues
If mt-multiserver-proxy prints an error like this:
```
plugin.Open: plugin was built with a different version of package github.com/HimbeerserverDE/mt-multiserver-proxy
```
it usually means that either the plugin or the proxy is out of date.
Upgrade the proxy, then run
`go get -u github.com/HimbeerserverDE/mt-multiserver-proxy` in the plugin
repository and rebuild it. You should compile the plugin and the proxy
on the same machine since the build environment needs to be identical.
My build environment can be found in
[build_env.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/build_env.md).

If the above steps didn't fix the error, or if you are making changes
to the proxy itself and want to use plugins, you have to temporarily
edit the go.mod file of your plugin. Find the line that says
`require github.com/HimbeerserverDE/mt-multiserver-proxy SOMEVERSION`
and copy everything excluding the `require `. Then append a new line:
`replace github.com/HimbeerserverDE/mt-multiserver-proxy SOMEVERSION => ../path/to/proxy/repo/`.
Now rebuild and install the plugin and it should be loaded.
