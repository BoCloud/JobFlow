FROM alpine:latest

COPY bin/manager /manager

ENTRYPOINT ["/manager"]