FROM quay.io/rebuy/rebuy-go-sdk:{{.Version}} as builder

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/dist/terraform-provider-graylog /usr/local/bin/
