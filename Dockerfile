FROM alpine:3.8

#Make sure we are patching all packages
RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
        ca-certificates \
    && update-ca-certificates

COPY portauthority /usr/bin/
ENTRYPOINT ["portauthority"]
