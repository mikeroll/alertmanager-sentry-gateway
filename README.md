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

To build the binary you need to install [Go](https://golang.org/), [dep](https://github.com/golang/dep) and [task](https://github.com/go-task/task).

```
$ task vendor
$ task build
```

### Building a docker image yourself
If you don't want to deal with go-specific stuff and just want a docker image:
```
$ docker build -t <tag> .
```


## Usage

Sentry's DSN is required to run this gateway. Optionally, an environment may be specified.

```
$ alertmanager-sentry-gateway --dsn ${SENTRY_DSN} --environment ${SENTRY_ENVIRONMENT}
```

If you prefer configuration via environment variables, exporting `SENTRY_DSN` and `SENTRY_ENVIRONMENT` to the process will have the same effect - you can then omit the cli arguments entirely.



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
$ alertmanager-sentry-gateway --dsn ${SENTRY_DSN} --template template.tmpl
```

Alternatively, provide a `SENTRY_EVENT_TEMPLATE` variable with the template string:
```
$ export SENTRY_EVENT_TEMPLATE="{{ .Labels.alertname }} - {{ .Labels.instance }}"
$ alertmanager-sentry-gateway
```


### Event fingerprinting
An Sentry event's fingerprint defines the properties of that event that shall be used to tell if multiple events belong to the same group. The fingerprints of outgoing events may be controlled via `--fingerprint-templates`/`SENTRY_GATEWAY_FINGERPRINT_TEMPLATES`, which are used similiarly to the message template. For example:
```
$ alertmanager-sentry-gateway --fingerprint-templates "{{ .Labels.instance }}" "{{ .Labels.alertname }}"
```
will cause events with the same `alertname` and `instance` labels to be grouped into a single issue.
If `--fingerprint-templates` is not supplied, Sentry's default algorithm is used.


## Alertmanager Configuration

To enable Alertmanager to send alerts to this gateway you need to configure a webhook in Alertmanager.

```
receivers:
- name: team
  webhook_configs:
  - url: 'http://127.0.0.1:9096'
    send_resolved: false
```
