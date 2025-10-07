# Build environment

## `go version`

```
go version go1.25.1 X:nodwarf5 linux/amd64
```

## Build commands

All commands are run in the project root directory.

### Compile development version to check for errors

```
go build -race ./cmd/mt-auth-convert
go build -race ./cmd/mt-build-plugin
go build -race ./cmd/mt-multiserver-proxy
```

### Install and run latest version

```
go install github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/...@latest && mt-multiserver-proxy
```

## Formatting

```
goimports -l -w .
# go fmt
```
