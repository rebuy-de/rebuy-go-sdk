# disabled syntax = docker/dockerfile:experimental

FROM golang:1.16-alpine as builder

RUN apk add --no-cache git curl openssl bash
RUN apk add --no-cache -X http://dl-cdn.alpinelinux.org/alpine/edge/community podman

# Install Go Tools
# RUN --mount=type=cache,id=go-build-cache,target=/root/.cache/go-build \
RUN GO111MODULE= go get -u golang.org/x/lint/golint
RUN GO111MODULE= go get -u github.com/gobuffalo/packr/v2/packr2

# Configure Go
ENV GOPATH= CGO_ENABLED=0 GO111MODULE=on

# Note: We need to copy the whole directory, because the .git directory needs
# to be part of the Docker context to determine the version.

COPY . /sdk

#RUN \
#    --mount=type=cache,id=go-build-cache,target=/root/.cache/go-build \
#    --mount=type=cache,id=go-pkg-cache,target=/go/pkg \
RUN \
    set -e \
    && cd /sdk \
    && ./buildutil \
    && cp ./dist/buildutil /usr/local/bin \
    && buildutil version \
    && rm -rf /sdk \
    && mkdir /build

WORKDIR /build

ONBUILD COPY . .
ONBUILD RUN packr2
#ONBUILD RUN \
#    --mount=type=cache,id=go-build-cache,target=/root/.cache/go-build \
#    --mount=type=cache,id=go-pkg-cache,target=/go/pkg \
ONBUILD RUN \
    buildutil
