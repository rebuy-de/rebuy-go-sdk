FROM golang:1.18beta2-alpine as builder

RUN apk add --no-cache git openssl

ENV CGO_ENABLED=0
RUN go install golang.org/x/lint/golint@latest
