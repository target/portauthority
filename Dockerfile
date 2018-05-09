#Builder stage to create the binary
FROM golang:1.9.2-alpine3.7 as builder

RUN apk add --update \
    curl git;

RUN curl https://glide.sh/get | sh

ADD . /go/src/github.com/target/portauthority
WORKDIR /go/src/github.com/target/portauthority

RUN glide install -v

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o portauthority

#Final image stage
FROM alpine:3.7

#Make sure we are patching all packages
RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
        ca-certificates \
    && update-ca-certificates

COPY --from=builder /go/src/github.com/target/portauthority/portauthority /usr/bin/
ENTRYPOINT ["portauthority"]
