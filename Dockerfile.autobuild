#Builder stage to create the binary
FROM golang:1.9.2-alpine3.7 as builder

RUN apk add --update \
    curl git;

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN curl https://glide.sh/get | sh

ADD . /go/src/github.com/target/portauthority
WORKDIR /go/src/github.com/target/portauthority

RUN glide install -v

RUN VERSION=$(git for-each-ref refs/tags --sort=-taggerdate --format='%(refname:short)' --count=1) && echo version=$VERSION; \
  if [ "$VERSION" == "" ] || [ -z "$VERSION" ]; \
  then echo "DEV BUILD" && go build -ldflags "-X main.appVersion=dev" -o portauthority; \
  else echo "TAG BUILD" && go build -ldflags "-X main.appVersion=$VERSION" -o portauthority; \
  fi


#Final image stage
FROM alpine:3.8

#Make sure we are patching all packages
RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
        ca-certificates \
    && update-ca-certificates

COPY --from=builder /go/src/github.com/target/portauthority/portauthority /usr/bin/
ENTRYPOINT ["portauthority"]
