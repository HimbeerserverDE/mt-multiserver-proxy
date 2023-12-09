FROM golang:1.21.4

COPY . /go/src/github.com/HimbeerserverDE/mt-multiserver-proxy

WORKDIR /go/src/github.com/HimbeerserverDE/mt-multiserver-proxy

RUN mkdir /usr/local/mt-multiserver-proxy
RUN GOBIN=/usr/local/mt-multiserver-proxy go install github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/...

VOLUME ["/usr/local/mt-multiserver-proxy"]

EXPOSE 40000/udp

CMD ["/usr/local/mt-multiserver-proxy/mt-multiserver-proxy"]
