# disabled syntax = docker/dockerfile:experimental

FROM golang:1.18-rc-alpine as builder

RUN apk add --no-cache git curl openssl bash

# Configure Go
ENV GOPATH= CGO_ENABLED=0 GO111MODULE=on

# Install Go Tools
RUN go install golang.org/x/lint/golint@latest

# Note: We need to copy the whole directory, because the .git directory needs
# to be part of the Docker context to determine the version.

COPY . /sdk

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
ONBUILD RUN \
    buildutil
