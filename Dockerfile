FROM alpine:3.4
MAINTAINER Moto Ishizawa "summerwind.jp"

COPY ./alertmanager-sentry-gateway /bin/alertmanager-sentry-gateway

ENTRYPOINT ["alertmanager-sentry-gateway"]
