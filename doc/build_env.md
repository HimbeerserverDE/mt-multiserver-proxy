# Build environment
## `go version`
```
go version go1.17 linux/amd64
```
## Build commands
### Compile development version to check for errors
```
go install -race github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/mt-multiserver-proxy
```
### Install and run latest version
```
go install -race github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/mt-multiserver-proxy@latest && ~/go/bin/mt-multiserver-proxy
```
## Formatting
```
go fmt
```
