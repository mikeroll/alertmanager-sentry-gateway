FROM alpine:3.4
MAINTAINER Moto Ishizawa "summerwind.jp"

COPY ./sentry-gateway /bin/sentry-gateway

ENTRYPOINT ["sentry-gateway"]
