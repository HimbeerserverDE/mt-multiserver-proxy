# Authentication backends

## Supported backends
All backends prefixed with `mt` are implementations of the upstream backends.
They store ban information in `ipban.txt` in the Minetest format.

### files
This is the default authentication backend unless specified otherwise
in the [config](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/config.md).
It creates a directory named `auth` in the proxy directory. It contains subdirectories
for each user. These are home to several files (created on demand):

* `salt`: The binary SRP salt of the user.
* `verifier`: The binary SRP verifier of the user.
* `timestamp`: An empty file whose access timestamps are used to keep track of reads or writes to the user's authentication entry.
* `last_server`: The name of the last server the user was connected to.

There's also a `ban` directory that holds files named after banned IP addresses
containing the username that was banned.

One of the main advantages of this format is that it is custom,
allowing the proxy to store anything it needs
and providing future expandibility. It's also very simple and easily readable
for humans or shell scripts.

### mtsqlite3
This backend is partially compatible with regular Minetest `auth.sqlite` databases.
The proxy is able to run using this backend and the authentication information
can be converted by [mt-auth-convert](#mt-auth-convert).
However storing a player's last server is not supported with this backend
and no conversions involving it will ever output server information.

### mtpostgresql
This backend provides partial compatibility with regular Minetest PostgreSQL
databases. The proxy is able to run using this backend and the authentication
information can be converted by [mt-auth-convert](#mt-auth-convert).
However storing a player's last server is not supported with this backend
and no conversions involving it will ever output server information.

Postgres connection strings are required to use this backend.
The proxy uses a configuration value for this
while the converter gets them from command-line arguments.

## Dealing with existing Minetest databases
If possible you should always convert your existing database
to the `files` format. An alternative is to reconfigure the proxy
to use the existing format directly at the cost of reduced functionality.
This method currently does not support storing the last server
a user was connected to, for example.

## mt-auth-convert
There's a tool that is able to convert between the supported backends.

### Installation
```
go install github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/mt-auth-convert@latest
```

Please specify the version explicitly if @latest differs from your proxy version.

### Usage
1. Move the binary to the directory the proxy binary is located in. The same rules apply regarding symlinks.
2. Move or copy the source database to the directory.
3. Stop the proxy.
4. Run the conversion tool.
5. (optional) Reconfigure the proxy to use the new backend.
6. Start the proxy.
7. (optional) Check if everything is working.

Unused Postgres connection strings should be set to nil,
though any other value should work as well.

Example (converting Minetest's auth.sqlite to the `files` backend):

```
mt-auth-convert mtsqlite3 files nil nil
```

Example (converting Minetest's PostgreSQL database to the `files` backend):

```
mt-auth-convert mtpostgresql files 'host=localhost user=mt dbname=mtauth sslmode=disable' nil
```
