FROM golang:1.12-alpine as builder

ARG BUILD_FLAGS

RUN apk add --no-cache curl git && \
    curl -sS https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR $GOPATH/src/alertmanager-sentry-gateway
ADD Gopkg.toml Gopkg.lock ./
RUN dep ensure -vendor-only -v
ADD sentry-gateway.go ./
RUN sh -xc "GOARCH=amd64 GOOS=linux go build ${BUILD_FLAGS}"


FROM alpine:3.7
LABEL maintainer="Moto Ishizawa <summerwind.jp>"

COPY --from=builder /go/src/alertmanager-sentry-gateway/alertmanager-sentry-gateway /bin

ENTRYPOINT ["alertmanager-sentry-gateway"]
