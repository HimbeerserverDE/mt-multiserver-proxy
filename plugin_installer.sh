#! /bin/sh

for folder in $(ls /go/plugin_src/)
do
	echo building $folder
	cd /go/plugin_src/$folder
	go build -buildmode=plugin
	mkdir -p /go/proxy/cmd/mt-multiserver-proxy/plugins/
	cp *.so /go/proxy/cmd/mt-multiserver-proxy/plugins/
done

echo Installed plugins:
ls /go/proxy/cmd/mt-multiserver-proxy/plugins/

