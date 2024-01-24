FROM --platform=${BUILDPLATFORM} golang:1.21.4

ARG VERSION
ARG BUILDPLATFORM
ARG BUILDARCH
ARG TARGETARCH

COPY . /go/src/github.com/HimbeerserverDE/mt-multiserver-proxy

RUN mkdir /usr/local/mt-multiserver-proxy
RUN GOARCH=${TARGETARCH} go install github.com/HimbeerserverDE/mt-multiserver-proxy/cmd/...@${VERSION:-`(cd /go/src/github.com/HimbeerserverDE/mt-multiserver-proxy && TZ=UTC git --no-pager show --quiet --abbrev=12 --date='format-local:%Y%m%d%H%M%S' --format='v0.0.0-%cd-%h')`}
RUN if [ "${TARGETARCH}" = "${BUILDARCH}" ]; then mv /go/bin/mt-* /usr/local/mt-multiserver-proxy/; else mv /go/bin/linux_${TARGETARCH}/mt-* /usr/local/mt-multiserver-proxy/; fi

VOLUME ["/usr/local/mt-multiserver-proxy"]

EXPOSE 40000/udp

CMD ["/usr/local/mt-multiserver-proxy/mt-multiserver-proxy"]
