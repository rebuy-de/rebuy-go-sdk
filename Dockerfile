# disabled syntax = docker/dockerfile:experimental

FROM golang:1.17-alpine as builder

RUN apk add --no-cache git curl openssl bash

# Install Go Tools
# RUN --mount=type=cache,id=go-build-cache,target=/root/.cache/go-build \
RUN GO111MODULE= go get -u golang.org/x/lint/golint

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
#ONBUILD RUN \
#    --mount=type=cache,id=go-build-cache,target=/root/.cache/go-build \
#    --mount=type=cache,id=go-pkg-cache,target=/go/pkg \
ONBUILD RUN \
    buildutil
