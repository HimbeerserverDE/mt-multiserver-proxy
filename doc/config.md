# Configuration file

## Location
The configuration file is automatically created in the working directory.
The file name is `config.json`.

## Example
This is an example configuration file with two servers. Remember to install
[the chat command plugin](https://github.com/HimbeerserverDE/mt-multiserver-chatcommands) and to configure the permissions.

```json
{
	"Servers": {
		"ServerName1": {
			"Addr": "minetest.local:30000"
		},
		"ServerName2": {
			"Addr": "minetest.local:30001"
		}
	}
}
```

## Format
The configuration file contains JSON data. The fields are as follows.

> `NoPlugins`
```
Type: bool
Default: false
Description: Plugins are not loaded if this is true.
```

> `NoAutoPlugins`
```
Type: bool
Default: false
Description: Plugin subdirectories are not built automatically if this is true.
```

> `CmdPrefix`
```
Type: string
Default: ">"
Description: A chat message is handled as a chat command
if it is prefixed by this.
```

> `RequirePasswd`
```
Type: bool
Default: false
Description: Empty passwords are rejected if this is true.
```

> `SendInterval`
```
Type: float32
Default: 0.09
Description: The recommended send interval for clients.
The proxy itself doesn't have a fixed send interval.
```

> `UserLimit`
```
Type: int
Default: 10
Description: The maximum number of players that can be connected to the proxy
at the same time.
```

> `AuthBackend`
```
Type: string
Default: "files"
Values: "files", "mtsqlite3", "mtpostgresql"
Description: The authentication backend to use.
Consider converting your existing database instead of loading it directly.
```

> `AuthPostgresConn`
```
Type: string
Default: ""
Description: The postgres connection string for the authentication database.
Used in conjunction with the mtpostgresql authentication backend.
```

> `NoTelnet`
```
Type: bool
Default: false
Description: The telnet server is not started if this is true.
```

> `TelnetAddr`
```
Type: string
Default: "[::1]:40010"
Description: The telnet server will listen for new clients on this
address.
```

> `BindAddr`
```
Type: string
Default: ":40000"
Description: The proxy will listen for new clients on this address.
```

> `Servers`
```
Type: map[string]Server
Default: map[string]Server{}
Description: The list of internal servers served by this proxy.
The first server is the default server new clients are connected to.
It also acts as a fallback server if a connection
to another server fails or closes.
```

> `Server.Addr`
```
Type: string
Default: ""
Description: The network address and port of an internal server.
```

> `Server.MediaPool`
```
Type: string
Default: Server name (map key)
Description: The media pool this server is part of.
See [media_pools.md](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/media_pools.md)
for more information.
```

> `Server.Fallbacks`
```
Type: []string
Default: []string{}
Description: The names of the servers a client should fall back to
if this server shuts down or crashes gracefully. Connection attempts
are made in the order in which the servers are given. As soon as
a connection is successful the other fallback servers in this list
will be ignored.
```

> `ForceDefaultSrv`
```
Type: bool
Default: false
Description: Players are connected to the default server instead of
the server they were playing on if this is true.
```

> `KickOnNewPool`
```
Type: bool
Default: false
Description: Players are kicked if a new media pool is added
by reloading the config if this is true.
```

> `CSMRF`
```
Type: CSMRF
Default: CSMRF{}
Description: The CSM Restriction Flags to send to clients.
Don't rely on this since it can trivially be bypassed.
```

> `CSMRF.NoCSMs`
```
Type: bool
Default: false
Description: Loading CSMs is disabled if this is true.
```

> `CSMRF.ChatMsgs`
```
Type: bool
Default: false
Description: CSMs can send chat messages if this is true.
```

> `CSMRF.ItemDefs`
```
Type: bool
Default: false
Description: CSMs can read item definitions.
```

> `CSMRF.NodeDefs`
```
Type: bool
Default: false
Description: CSMs can read node definitions.
```

> `CSMRF.NoLimitMapRange`
```
Type: bool
Default: false
Description: CSMs can look nodes up no matter how far away they are.
```

> `CSMRF.PlayerList`
```
Type: bool
Default: false
Description: CSMs can access the player list.
```

> `MapRange`
```
Type: uint32
Default: 0
Description: The maximum distance from which CSMs can read the map.
```

> `FallbackServers`
```
Type: []string
Default: []string{}
Description: Names of general fallback servers to connect to
if a connection attempt fails or an existing connection
to a game server is lost.
```

> `DropCSMRF`
```
Type: bool
Default: false
Description: Servers cannot override CSM Restriction Flags if this is true.
```

> `Groups`
```
Type: map[string][]string
Default: map[string][]string{}
Description: The list of permission groups.
```

> `Groups[k]`
```
Type: []string
Default: []string{}
Description: The list of permissions the group has.
```

> `UserGroups`
```
Type: map[string]string
Default: map[string]string{}
Description: This sets the group of a user.
```

> `UserGroups[k]`
```
Type: string
Default: "default"
Description: The group of the user.
```

> `List`
```
Type: List
Default: List{}
Description: This contains information on how to announce to the server list.
```

> `List.Enable`
```
Type: bool
Default: false
Description: If this is set to true server list announcements are sent.
```

> `List.Addr`
```
Type: string
Default: ""
Description: The base URL of the list server.
```

> `List.Interval`
```
Type: int
Default: 300
Description: The interval between server list announcements.
```

> `List.Name`
```
Type: string
Default: ""
Values: Any non-zero string
Description: The name to be displayed on the server list.
```

> `List.Desc`
```
Type: string
Default: ""
Description: The description for the server list.
```

> `List.URL`
```
Type: string
Default: ""
Description: The website for this server.
```

> `List.Creative`
```
Type: bool
Default: false
Description: The creative server list flag.
```

> `List.Dmg`
```
Type: bool
Default: false
Description: The damage server list flag.
```

> `List.PvP`
```
Type: bool
Default: false
Description: The PvP server list flag.
```

> `List.Game`
```
Type: string
Default: ""
Description: The subgame displayed on the server list.
```

> `List.FarNames`
```
Type: bool
Default: false
Description: The server list flag that shows whether far players are visible.
```

> `List.Mods`
```
Type: []string
Default: []string{}
Description: The list of mods to be displayed on the server list.
```
