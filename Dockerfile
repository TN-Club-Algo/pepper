FROM ubuntu:latest

LABEL author="Alexandre Duchesne"

RUN apt-get update && apt-get install -y curl iproute2 iputils-ping kmod
RUN modprobe tun

# Add executable
COPY pepper/pepper /pepper

VOLUME /tmp

ENTRYPOINT "/pepper"