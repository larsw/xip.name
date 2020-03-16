FROM golang:alpine AS build
RUN apk add --no-cache curl libcap && \
    curl -L -o upx-3.96-amd64_linux.tar.xz https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz && \
    tar xf upx-3.96-amd64_linux.tar.xz && \
    adduser -D -g "" -h "/nonexistent" -s "/sbin/nologin" -H -u 1001 xip

COPY go.mod *.go /go/src/github.com/larsw/xip.name/
WORKDIR /go/src/github.com/larsw/xip.name/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o xip xip.go && \
    /go/upx-3.96-amd64_linux/upx ./xip

RUN setcap 'cap_net_bind_service=+ep' ./xip
    
FROM scratch as minimal
ARG CREATED 
ARG COMMIT
ARG VERSION

ENV XIP_VERBOSE=false
ENV XIP_FQDN=xip.name.
ENV XIP_ADDR=:53
ENV XIP_IP=127.0.0.1

LABEL org.opencontainers.org.authors="Lars Wilhelmsen <lars@sral.org>" \
      org.opencontainers.org.source="https://github.com/larsw/xip.name/" \
      org.opencontainers.org.revision=$COMMIT \
      org.opencontainers.org.created=$CREATED \
      org.opencontainers.org.version=$VERSION
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /go/src/github.com/larsw/xip.name/xip /
USER xip:xip
EXPOSE 53/tcp 53/udp
ENTRYPOINT ["/xip"]

FROM alpine@sha256:ab00606a42621fb68f2ed6ad3c88be54397f981a7b70a79db3d1172b11c4367d AS alpine 
ARG CREATED 
ARG COMMIT
ARG VERSION

ENV XIP_VERBOSE=false
ENV XIP_FQDN=xip.name.
ENV XIP_ADDR=:53
ENV XIP_IP=127.0.0.1

LABEL org.opencontainers.org.authors="Lars Wilhelmsen <lars@sral.org>" \
      org.opencontainers.org.source="https://github.com/larsw/xip.name/" \
      org.opencontainers.org.revision=$COMMIT \
      org.opencontainers.org.created=$CREATED \
      org.opencontainers.org.version=$VERSION

EXPOSE 53/tcp 53/udp

COPY --from=build /go/src/github.com/larsw/xip.name/xip /

ENTRYPOINT ["./xip"]

FROM alpine AS alpine-web

COPY entrypoint.sh /
RUN chmod +x /entrypoint.sh

# Install and configure nginx
RUN apk add --no-cache nginx && \
    sed -i '/access_log/s|/[^;]\+|/dev/stdout|' /etc/nginx/nginx.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY usr/share/nginx/html/ /usr/share/nginx/html/

# Expose volume so that the website can be customized
VOLUME /usr/share/nginx/html/

# Enable both DNS lookup via UDP and TCP + web
EXPOSE 80/tcp

ENTRYPOINT ["/entrypoint.sh"]
