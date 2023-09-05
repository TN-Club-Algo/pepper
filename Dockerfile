FROM ubuntu:latest

LABEL author="Alexandre Duchesne"

# Add main jar
COPY pepper/pepper /pepper

VOLUME /tmp

ENTRYPOINT "/pepper"