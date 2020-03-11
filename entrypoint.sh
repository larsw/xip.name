#!/bin/sh
/xip &
/usr/sbin/nginx -g 'daemon off; pid /tmp/nginx.pid; error_log /dev/stdout info;'

