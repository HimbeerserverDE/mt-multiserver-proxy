# Media Pools

All servers must be part of a media pool. By default the name of the server
is used.

## Background

When the proxy sends any content-related packets to the client,
it prefixes any content names such as node names or media file names
with the media pool of the current server and an underscore.
The purpose of this is to allow servers to have different media
with the same name and to avoid some other multiplexing issues.

## When to use media pools?

In general, custom media pools are not required.
There are reasons to use them:

* reducing memory and storage usage on the client
* dynamically adding servers

### Reducing RAM and disk usage

The client has to store all media it receives in memory and in its cache.
Minetest doesn't do this very efficiently: Identical files with different
names will not share memory, a copy will be made. Even if they did share
memory the references would still consume memory themselves but that would
probably be negligable.

This may not look like a big issue but it is. Many machines, especially
phones, still only have 4 GB of RAM or even less. It's quite easy to
exceed this limit even with lightweight or basic subgames. This will make
devices that don't have enough memory unable to connect. The game will crash
while downloading media.

The unnecessarily redundant caching will fill the permanent storage with
unneeded files too. This isn't as big of a problem as the cache isn't
(or at least shouldn't) be required for the engine to work. However
inexperienced players are going to wonder where their disk space is going.

### Dynamic servers

Dynamic servers always require the use of media pools.
The reason is that connected clients can't get the new
content without reconnecting due to engine limitations. Media pools are
pushed to the client when it connects. This requires the first server of the
media pool to be reachable. This means you can make a dummy server for the
media and prevent players from connecting to it, or just use a hub server
as the media master.

## How to use media pools?

Simply specify the name of the media pool you'd like the server to be part of
in the MediaPool field of the server definition. All server you do this for
will be part of the pool. Alternatively you can specify the name of another
server if that server doesn't have a custom media pool set or if it's the same
as its name. This will result in the servers being in a media pool that has
the same name as that server. You can use it to your advantage when creating
and naming dummy servers.
