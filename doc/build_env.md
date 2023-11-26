# Build environment

## `go version`
```
go version go1.21.4 linux/amd64
```

## Build commands
All commands are run in the project root directory.

### Compile development version to check for errors
```
go install -race github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/...
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
