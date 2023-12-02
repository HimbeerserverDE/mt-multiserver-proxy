# Plugins
mt-multiserver-proxy loads all plugin files in the `plugins` directory
on startup. Any errors will be logged and do not prevent other plugins
from being loaded. Plugins **cannot** be (re)loaded at runtime, you
need to restart the proxy.

## Installing plugins
The recommended way to install plugins is cd'ing into the `plugins` directory,
downloading the source code (e.g. using `git clone`) into it
and starting the proxy without setting the `NoAutoPlugins` config option
to `true`.

The proxy will detect its own version automatically
and attempt to build the plugin against it, making this the easiest way
for end users to deal with versioning.

This won't work in development builds.

### Manual installation
To install a plugin manually, clone the repository, cd into it and run:

```
go build -buildmode=plugin
```

A .so file will be created. Copy or move this file into the `plugins`
directory. Restart the proxy to load the plugin.

### Automatic version management
To make dealing with version issues easier for end users
the `mt-build-plugin` tool is provided. It automatically detects
the correct proxy version and builds the plugin in the working directory
against it. The resulting .so file can then be used as explained above.

To use this, clone the plugin repository, cd into it and run:

```
mt-build-plugin
```

This tool won't work in development builds.

## Developing plugins
A plugin is simply a main package without a main function. Use the init
functions instead. Plugins can import
`github.com/HimbeerserverDE/mt-multiserver-proxy` and use the exported
symbols to control the behavior of the proxy. The API is documented
[here](https://pkg.go.dev/github.com/HimbeerserverDE/mt-multiserver-proxy).
**The plugin API may change at any time without warning.
Crucially, symbols may be renamed or deleted and fields may be deleted
from type definitions.**

## Common issues
If mt-multiserver-proxy prints an error similar to this:

```
plugin.Open: plugin was built with a different version of package github.com/HimbeerserverDE/mt-multiserver-proxy
```

it usually means that either the plugin or the proxy is out of date.
Upgrade the proxy and associated helper programs, then run

```
mt-build-plugin
```

in the plugin repository. It is also possible to manually run

```
go get github.com/HimbeerserverDE/mt-multiserver-proxy
```

and rebuild the plugin. You should compile the plugin and the proxy
on the same machine since the build environment needs to be identical.
My build environment can be found in
[build_env.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/build_env.md).

## Using plugins with development builds
If you want to use plugins with a proxy binary produced by `go build`
or `go run` (usually for contributing to the proxy), you have to temporarily
edit the go.mod file of your plugin. Find the line that says
`require github.com/HimbeerserverDE/mt-multiserver-proxy SOMEVERSION`
and copy everything excluding the `require `. Then append a new line:
`replace github.com/HimbeerserverDE/mt-multiserver-proxy SOMEVERSION => ../path/to/proxy/repo/`.
Now rebuild and install the plugin and it should be loaded.

As of now there is no way to automate this, though the go toolchain
provides everything needed to implement it.
Expect this feature to be added soon.
