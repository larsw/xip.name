FROM golang:alpine AS build

COPY go.mod *.go /go/src/github.com/larsw/xip.name/
WORKDIR /go/src/github.com/larsw/xip.name/
RUN go build -o xip xip.go

FROM alpine:3.11.3 as runtime
COPY entrypoint.sh /
RUN chmod +x /entrypoint.sh
COPY --from=build /go/src/github.com/larsw/xip.name/xip /

# Install and configure nginx
RUN apk add --no-cache nginx && \
    sed -i '/access_log/s|/[^;]\+|/dev/stdout|' /etc/nginx/nginx.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY usr/share/nginx/html/ /usr/share/nginx/html/

# Expose volume so that the website can be customized
VOLUME /usr/share/nginx/html/

# Enable both DNS lookup via UDP and TCP + web
EXPOSE 53/tcp 53/udp 80/tcp

# Document the environment variables that xip uses.
ENV XIP_VERBOSE=false
ENV XIP_FQDN=xip.name.
ENV XIP_ADDR=:53
ENV XIP_IP=127.0.0.1

ENTRYPOINT ["/entrypoint.sh"]
