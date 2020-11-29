# alertmanager-sentry-gateway

alertmanager-sentry-gateway is a webhook gateway for [Alertmanager](https://github.com/prometheus/alertmanager). This gateway receives webhooks from Alertmanager and sends alert information as an event to [Sentry](https://sentry.io).

## Install

### Just want the binary?

Go to the [releases page](https://github.com/mikeroll/alertmanager-sentry-gateway/releases), find the version you want, and download the tarball file.

### Run as container?
```
# <= v0.3.0
$ docker pull summerwind/alertmanager-sentry-gateway:<tag>

# >= v0.4.0
$ docker pull mikeroll/alertmanager-sentry-gateway:<tag>
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

### DSN proxying
It is possible to support forwarding events to any Sentry DSN (as opposed to locking the gateway into a single one via `SENTRY_DSN`). To enable such DSN proxying, provide the base url of the target Sentry instance via `--sentry-url`/`SENTRY_URL`. Gateway url can the be specified as follows, similar to a "real" DSN:  
```
<gateway_scheme>//<project_secret>@<gateway_host>/<project_id>
```
E.g. given `SENTRY_URL=https://my.hosted.sentry:8000/`, with the gateway running at `http://sentry.gateway:9096/`, the gateway url should be given as `http://a1b2c3d4e5f6@sentry.gateway:9096/42` and the corresponding DSN will be reconstructed as `https://a1b2c3d4e5f6@my.hosted.sentry:8000/42`.

### Sentry environment
Default environment is taken from argument `environment` or from env variable `SENTRY_ENVIRONMENT`.  
If you are using DSN proxying, then there is also a way to specify environment via url path:
`http://a1b2c3d4e5f6@sentry.gateway:9096/42/my_environment`
Replace `my_environment` with any other valid string, and gateway will pass that environment with event.

### Sentry environment from alert label
There is also a third way to specify sentry environment, which would come from alert label itself. You can specify alert label via command line argument: `environment-label` or via environment variable: `SENTRY_ENVIRONMENT_LABEL`.  
For example:
You have alert with label `my-label` which has value `my-sentry-environment`. 
Then you specify it like so: `--environment-label=my-label` and if alert coming in has this label, it will set sentry environment for that alert equal to the value of that label.  
This overwrites any existing sentry environment which was set via URL or `--environment` argument. So that allows to ingest alerts that might not have that label, in that case they will use sentry environment from previous methods.


### Event body
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

Alternatively, provide a `SENTRY_GATEWAY_TEMPLATE` variable with the template string:
```
$ export SENTRY_GATEWAY_TEMPLATE="{{ .Labels.alertname }} - {{ .Labels.instance }}"
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
