# Docker

The proxy can be run in Docker if you with to do so.
You may build it yourself or pull the latest image from
[GitHub Packages](https://github.com/HimbeerserverDE/mt-multiserver-proxy/pkgs/container/mt-multiserver-proxy).

This repository contains a `Dockerfile` at its root.
It can be used to build a release version into an image.
Development versions can be built using the `devel.Dockerfile`.

## Pre-built image

You can pull the latest version from [GitHub Packages](https://github.com/HimbeerserverDE/mt-multiserver-proxy/pkgs/container/mt-multiserver-proxy).
Simply follow the instructions at [Run](#run), replacing the image `mt-multiserver-proxy`
with `ghcr.io/himbeerserverde/mt-multiserver-proxy:main`.

Example (without data volume):

```
docker run \
	-it \
	-p 40000:40000/udp \
	--name mt-multiserver-proxy \
	ghcr.io/himbeerserverde/mt-multiserver-proxy:main
```

## Build

You can replace the `-t` option with anything you need.
For example, you can add a `:devel` suffix for development builds,
though remember to refer the image using that name in compose files
or the `docker run` command.

The images are intended to be built by the default buildx builder.

### Regular

To build an image of the current commit, run the following command
from the repository root:

```
docker buildx build -t mt-multiserver-proxy --load .
```

This works well with CI because it doesn't rely on the Go proxy being up-to-date.

It is also possible to build a specific version into an image:

```
docker buildx build -t mt-multiserver-proxy --load --build-arg version=VERSION .
```

where `VERSION` is a Go pseudo-version or `latest` for the latest version
known by the Go proxy.

### Development

To build an image of the checked-out commit, run the following command
from the repository root:

```
docker buildx build -t mt-multiserver-proxy --load -f devel.Dockerfile .
```

### Cross-compilation

You can add the `--platform linux/ARCH` argument to any of the build commands.

Example (ARMv8 64-bit):

```
docker buildx build --platform linux/arm64 -t mt-multiserver-proxy --load .
```

## Run

You can change the external port or the container name to suit your needs.

To run the proxy in a container, run the following command:

```
docker run \
	-it \
	-p 40000:40000/udp \
	--name mt-multiserver-proxy \
	mt-multiserver-proxy
```

In most cases you'll want to use a volume for configuration,
authentication databases, logs, caching and plugins:

```
docker run \
	-it \
	-v mtproxy_data:/usr/local/mt-multiserver-proxy
	-p 40000:40000/udp \
	--name mt-multiserver-proxy \
	mt-multiserver-proxy
```

which assumes that you've already set up a `mtproxy_data` volume
using the `docker volume` command.

Or use compose:

```
services:
  proxy:
    container_name: mt-multiserver-proxy
	image: mt-multiserver-proxy
	ports:
	  - "40000:40000/udp"
	restart: unless-stopped
	volumes:
	  - mtproxy_data:/usr/local/mt-multiserver-proxy
volumes:
  mtproxy_data:
    external: true
```

which assumes that you've already set up a `mtproxy_data` volume
using the `docker volume` command.

Then use the volume to configure the proxy, add plugins, etc.

## mt-auth-convert

You can run mt-auth-convert inside the container:

```
docker run \
	-it \
	-p 40000:40000/udp \
	--name mt-multiserver-proxy \
	mt-multiserver-proxy \
	mt-auth-convert PARAMS
```

If using a volume:

```
docker run \
	-it \
	-v mtproxy_data:/usr/local/mt-multiserver-proxy
	-p 40000:40000/udp \
	--name mt-multiserver-proxy \
	mt-multiserver-proxy \
	mt-auth-convert PARAMS
```

Or use compose:

```
docker compose run proxy /usr/local/mt-multiserver-proxy/mt-auth-convert PARAMS
```

Consult the [mt-auth-convert documentation](https://github.com/HimbeerserverDE/mt-multiserver-proxy/blob/main/doc/auth_backends.md#mt-auth-convert)
for what `PARAMS` to use.
