FROM golang:1.13-alpine as builder

ARG BUILD_FLAGS

RUN apk add --no-cache curl git && \
    curl -sS https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR $GOPATH/src/alertmanager-sentry-gateway
ADD go.mod go.sum ./
RUN go mod vendor
ADD sentry-gateway.go ./
RUN sh -xc "GOARCH=amd64 GOOS=linux go build ${BUILD_FLAGS}"


FROM alpine:3.11
LABEL maintainer="Pavel Tumik <pavel.tumik@gmail.com>"

COPY --from=builder /go/src/alertmanager-sentry-gateway/alertmanager-sentry-gateway /bin

ENTRYPOINT ["alertmanager-sentry-gateway"]
