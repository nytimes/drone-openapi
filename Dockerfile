FROM alpine:3.6

ADD drone-openapi /bin/
ENTRYPOINT ["/bin/drone-openapi"]
