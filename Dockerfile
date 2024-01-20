FROM --platform=${BUILDPLATFORM} golang:1.21.4

ARG version=latest
ARG BUILDPLATFORM
ARG BUILDARCH
ARG TARGETARCH

RUN mkdir /usr/local/mt-multiserver-proxy
RUN GOARCH=${TARGETARCH} go install github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/...@${version}
RUN if [ "${TARGETARCH}" = "${BUILDARCH}" ]; then mv /go/bin/mt-* /usr/local/mt-multiserver-proxy/; else mv /go/bin/linux_${TARGETARCH}/mt-* /usr/local/mt-multiserver-proxy/; fi

VOLUME ["/usr/local/mt-multiserver-proxy"]

EXPOSE 40000/udp

CMD ["/usr/local/mt-multiserver-proxy/mt-multiserver-proxy"]
