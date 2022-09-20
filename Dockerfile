FROM gcr.io/distroless/static

# USER 1001:1001

ADD drone-openapi /bin/

ENTRYPOINT ["/bin/drone-openapi"]
