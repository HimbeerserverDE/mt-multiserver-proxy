FROM golang:1.17.9-stretch as builder

COPY proxy /go/proxy
WORKDIR /go/proxy

RUN go build .

WORKDIR /go/proxy/cmd/mt-multiserver-proxy
RUN go build .

# plugins:
COPY plugins /go/plugin_src
COPY plugin_installer.sh /go/plugin_installer.sh
RUN sh /go/plugin_installer.sh

COPY ./config.json /go/proxy/cmd/mt-multiserver-proxy/config.json

EXPOSE 40000/udp
EXPOSE 40010/tcp

CMD [ "/go/proxy/cmd/mt-multiserver-proxy/mt-multiserver-proxy"]
