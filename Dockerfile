FROM alpine:latest

RUN \
  apk update && \
  apk add dbus iptables curl zip

COPY ".build/releases/linux_amd64/cb" "/usr/local/bin/"

WORKDIR "/root"
ENTRYPOINT [ "/usr/local/bin/cb" ]
