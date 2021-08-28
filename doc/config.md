# Configuration file
## Location
The configuration file is automatically created in the working directory.
The file name is `config.json`.
## Format
The configuration file contains JSON data. The fields are as follows.

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
Description: The recommended send interval for clients. The proxy itself doesn't have a fixed send interval.
```

> `UserLimit`
```
Type: int
Default: 10
Description: The maximum number of players that can be connected to the proxy at the same time.
```

> `AuthBackend`
```
Type: string
Default: sqlite3
Values: sqlite3
Description: The authentication backend to use. Only SQLite3 is available at the moment.
```

> `BindAddr`
```
Type: string
Default: ":40000"
Description: The proxy will listen for new clients on this address.
```

> `Servers`
```
Type: []Server
Default: []Server{}
Description: The list of internal servers served by this proxy.
```

> `Server.Name`
```
Type: string
Default: ""
Values: Any non-zero string
Description: The unique name an internal server is known as.
```

> `Server.Addr`
```
Type: string
Default: ""
Description: The network address and port of an internal server.
```

> `CSMRF`
```
Type: CSMRF
Default: CSMRF{}
Description: The CSM Restriction Flags to send to clients. Don't rely on this since it can trivially be bypassed.
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
