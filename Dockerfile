FROM alpine:3.14.3

COPY bin/manager /manager

ENTRYPOINT ["/manager"]