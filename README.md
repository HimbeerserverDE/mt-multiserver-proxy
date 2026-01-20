# mt-multiserver-proxy

mt-multiserver-proxy is a reverse proxy designed for linking
multiple Minetest servers together. It is the successor to multiserver.

## mt

This project was made possible by
[anon55555's mt module](https://github.com/anon55555/mt).

## Supported Minetest versions

**Each commit only supports a single Minetest minor version.**

This is because each minor version breaks compatibility with its predecessors.
Patch versions should be safe. Example:
A proxy commit for 5.5.0 should also work with 5.5.1
but is highly likely to break when used with 5.6.0.

**All internal servers should have strict version checking enabled.
Otherwise version mismatches may remain undetected.**

Compatibility breaks because upstream Minetest
doesn't take the protocol version into account when sending packets
and instead expects the receiver to ignore any new fields it doesn't recognize.
This causes a *trailing data* error on the proxy
that prevents the packet from being parsed and processed.

### Proxy updates

Only the currently supported Minetest version will get proxy updates,
i.e. features and bug fixes won't be backported to earlier Minetest versions.
If you need this you can manually merge the commits yourself.

### Commit hashes for Minetest releases

The latest `main` usually supports the latest Minetest release
unless it isn't supported yet. The following list contains the commit hashes
for all versions that were ever supported:

* Minetest 5.3: [18e7ba7977d1a94880ca6f0bc6d70dab0dc696e2](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/18e7ba7977d1a94880ca6f0bc6d70dab0dc696e2), chat command plugin: unknown
* Minetest 5.4: [4c90fdd8d212a1c94a7e9ffad587ef610dcc243b](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/4c90fdd8d212a1c94a7e9ffad587ef610dcc243b), chat command plugin: [bae3cf2cf232a90203677464d83bbffc50be77b1](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/bae3cf2cf232a90203677464d83bbffc50be77b1)
* Minetest 5.5: [dd9e80d6a9a7031c97c64a1979e1e514c092a4cd](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/dd9e80d6a9a7031c97c64a1979e1e514c092a4cd), chat command plugin: [fc27ae7c87be94a39bb9ccb15f2ad0b27fcac76c](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/fc27ae7c87be94a39bb9ccb15f2ad0b27fcac76c)
* Minetest 5.6: [04705a9129afe3e3f5414af1799667efcc57d3eb](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/04705a9129afe3e3f5414af1799667efcc57d3eb), chat command plugin: [4020944da5bce99b878fae4c2d9709f610f4cf6a](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/4020944da5bce99b878fae4c2d9709f610f4cf6a)
* Minetest 5.7: [629d57a651b46539af3ffed36fb0649b3ea6d346](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/629d57a651b46539af3ffed36fb0649b3ea6d346), chat command plugin: [718f8defad54adc04ac81f535b6d59c82a13298e](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/718f8defad54adc04ac81f535b6d59c82a13298e)
* Minetest 5.8: [efeceb162b2fd45994bf09023eea065519b6b89b](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/15c56b8806030984c2cfdc04a5455a366eca44d4), chat command plugin: [15c56b8806030984c2cfdc04a5455a366eca44d4](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/15c56b8806030984c2cfdc04a5455a366eca44d4)
* Minetest 5.9: [143f14722b6c23cebd9a625e517d5988e8330baa](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/143f14722b6c23cebd9a625e517d5988e8330baa), chat command plugin: [86bd26badf51258be23a73bb48e5b55b28aa2c07](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/86bd26badf51258be23a73bb48e5b55b28aa2c07)
* Luanti 5.10: [7800bf490fa92879dfc46a54836624a8d1c6c6f6](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/7800bf490fa92879dfc46a54836624a8d1c6c6f6), chat command plugin: [8ea5400bdd4f68bbccb8f25e8f60c1346d218ff8](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/8ea5400bdd4f68bbccb8f25e8f60c1346d218ff8)
* Luanti 5.11: [278d619d28f7d17e44c55311e2221dda3c86ca4e](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/278d619d28f7d17e44c55311e2221dda3c86ca4e), chat command plugin: [f69d016fd7b84b594250913a8a52eacc73788265](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/f69d016fd7b84b594250913a8a52eacc73788265)
* Luanti 5.12: [3b16493fbc7d81653a8f445dd875349679cf7d2a](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/3b16493fbc7d81653a8f445dd875349679cf7d2a), chat command plugin: [2613b70407e0b69f34fd31d11cee993ea330ccf7](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/2613b70407e0b69f34fd31d11cee993ea330ccf7)
* Luanti 5.13: [812055bbce5e84c383dfab29440e48cc6bebd349](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/812055bbce5e84c383dfab29440e48cc6bebd349), chat command plugin: [68a6384793beb321b695f71b5078d3b23a338aea](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/68a6384793beb321b695f71b5078d3b23a338aea)
* Luanti 5.14: [a80b19b772df2c47a14de3d6eb02c13389a5707c](https://github.com/HimbeerserverDE/mt-multiserver-proxy/commit/a80b19b772df2c47a14de3d6eb02c13389a5707c), chat command plugin: [f8f0e059b06d84c86f644dfc94c16c1b31ccc731](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands/commit/f8f0e059b06d84c86f644dfc94c16c1b31ccc731)
* Luanti 5.15: latest

The chat command plugin commit hashes are mainly specified for old proxy
versions that didn't support automatic plugin building and version management
yet. Using the `mt-build-plugin` tool should be sufficient, though there may
be API changes preventing the plugin from compiling against an old proxy
version in which case the commit hashes are needed too. Conclusively it's
important to downgrade the plugin to that version if you want it to work with
an old proxy version without automatic plugin building and version management
or if it doesn't compile against old proxy versions anymore.

### Minetest development builds

Development builds aren't supported at all
because it would be a monumental maintenance effort.
If you have to use one, try the proxy version for its release first
and continue with the proxy version for the last release
preceeding the development build.
If this doesn't work you'll have to edit the code of the proxy yourself.

**Development builds may pass the version check performed by the proxy
and experience major breakage.** This is because the protocol version
isn't bumped when a new development phase is started after a release.

## Installation

It is recommended to explicitly set the `GOBIN` environment variable
to a directory that is only used for the proxy binaries, databases
and configuration files.

Go 1.21 or higher is required. Run

```
export GOBIN=~/.local/share/mt-multiserver-proxy
mkdir -p ${GOBIN}
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

### Development builds

Build the following binaries from the proxy repository directory:

```
go build -race ./cmd/mt-auth-convert
go build -race ./cmd/mt-build-plugin
go build -race ./cmd/mt-multiserver-proxy
```

*Do not move the binaries! Doing so breaks automatic plugin builds.*

### Docker

The proxy can be run in Docker. See [doc/docker.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/docker.md)
for instructions and details.

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

## Plugins

This proxy supports loading Go plugins.
Consult [doc/plugins.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/plugins.md)
for details on how to develop or install them.

## Docker

The proxy can be run in Docker.
See [doc/docker.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/docker.md)
for details.
