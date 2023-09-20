FROM ubuntu:latest

LABEL author="Alexandre Duchesne"

RUN apt-get update && apt-get install -y iproute2 curl

# Add executable
COPY pepper/pepper /pepper

VOLUME /tmp

ENTRYPOINT "/pepper"