# alertmanager-sentry-gateway

alertmanager-sentry-gateway is a webhook gateway for [Alertmanager](https://github.com/prometheus/alertmanager). This gateway receives webhooks from Alertmanager and sends alert information as an event to [Sentry](https://sentry.io).

## Install

### Just want the binary?

Go to the [releases page](https://github.com/summerwind/alertmanager-sentry-gateway/releases), find the version you want, and download the tarball file.

### Run as container?

```
$ docker pull summerwind/alertmanager-sentry-gateway:latest
```

### Building binary yourself

To build the binary you need to install [Go](https://golang.org/) and [Glide](https://github.com/Masterminds/glide).

```
$ make
```

## Usage

Sentry's DSN is required to run this gateway.

```
$ sentry-gateway --dsn ${SENTRY_DSN}
```

Event body of Sentry can be customized with a template file as follows. The data passed to the template file is an [Alert](https://godoc.org/github.com/prometheus/alertmanager/template#Alert) of Alertmanager.

```
$ vim template.tmpl
```
```
{{ .Labels.alertname }} - {{ .Labels.instance }}
{{ .Annotations.description }}

Labels:
{{ range .Labels.SortedPairs }} - {{ .Name }} = {{ .Value }}
{{ end }}
```
```
$ sentry-gateway --dsn ${SENTRY_DSN} --template template.tmpl
```

## Alertmanager Configuration

To enable Alertmanager to send alerts to this gateway you need to configure a webhook in Alertmanager.

```
receivers:
- name: team
  webhook_configs:
  - url: 'http://127.0.0.1:9096/sentry'
    send_resolved: false
```
