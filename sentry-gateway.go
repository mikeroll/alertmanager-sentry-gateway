package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/prometheus/alertmanager/notify"
	amt "github.com/prometheus/alertmanager/template"
	"github.com/spf13/cobra"
)

var (
	VERSION string = "latest"
	COMMIT  string = "HEAD"
)

const (
	defaultTemplate   = "{{ .Labels.alertname }} - {{ .Labels.instance }}\n{{ .Annotations.description }}"
	defaultListenAddr = "0.0.0.0:9096"
)

func main() {
	var cmd = &cobra.Command{
		Use:   "sentry-gateway",
		Short: "Sentry gateway for Alertmanager",
		RunE:  run,
	}

	cmd.Flags().StringP("dsn", "d", "", "Sentry DSN")
	cmd.Flags().StringP("sentry-url", "u", "", "Sentry URL")
	cmd.Flags().StringP("environment", "e", "", "Sentry Environment")
	cmd.Flags().StringP("template", "t", "", "Path of the template file of event message")
	cmd.Flags().StringArrayP("fingerprint-templates", "f", []string{}, "List of templates to use as Sentry event fingerprint")
	cmd.Flags().BoolP("dumb-timestamps", "s", false, "Whether to use time.Now instead of alert StartsAt/EndsAt")
	cmd.Flags().StringP("addr", "a", "", "Address to listen on for WebHook")
	cmd.Flags().Bool("version", false, "Display version information and exit")

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

type gatewayRequest struct {
	dsn     string
	message *notify.WebhookMessage
}

func run(cmd *cobra.Command, args []string) error {
	v, err := cmd.Flags().GetBool("version")
	if err != nil {
		return err
	}

	if v {
		version()
		os.Exit(0)
	}

	defaultDSN, err := cmd.Flags().GetString("dsn")
	if err != nil {
		return err
	}
	if defaultDSN == "" {
		defaultDSN = os.Getenv("SENTRY_DSN")
	}

	sentryURL, err := cmd.Flags().GetString("sentry-url")
	if err != nil {
		return err
	}
	if sentryURL == "" {
		sentryURL = os.Getenv("SENTRY_URL")
	}

	if defaultDSN == "" && sentryURL == "" {
		return errors.New("one of `dsn,sentry-url` is required")
	}

	tmplPath, err := cmd.Flags().GetString("template")
	if err != nil {
		return err
	}

	var tmpl string
	if tmplPath != "" {
		file, err := ioutil.ReadFile(tmplPath)
		if err != nil {
			return err
		}

		tmpl = string(file)
	} else if envTmpl := os.Getenv("SENTRY_GATEWAY_TEMPLATE"); envTmpl != "" {
		tmpl = envTmpl
	} else {
		tmpl = defaultTemplate
	}

	addr, err := cmd.Flags().GetString("addr")
	if err != nil {
		return err
	}

	if addr == "" {
		if envAddr := os.Getenv("SENTRY_GATEWAY_ADDR"); envAddr != "" {
			addr = envAddr
		} else {
			addr = defaultListenAddr
		}
	}

	t, err := createTemplate(tmpl)
	if err != nil {
		return err
	}

	fingerprintTemplates, err := cmd.Flags().GetStringArray("fingerprint-templates")
	if err != nil {
		return err
	}
	if len(fingerprintTemplates) == 0 {
		fingerprintTemplates = strings.Split(os.Getenv("SENTRY_GATEWAY_FINGERPRINT_TEMPLATES"), ",")
	}

	var fpTemplates []*template.Template
	for _, templateString := range fingerprintTemplates {
		fpTemplate, err := createTemplate(templateString)
		if err != nil {
			return err
		}
		fpTemplates = append(fpTemplates, fpTemplate)
	}

	dumbTimestamps, err := cmd.Flags().GetBool("dumb-timestamps")
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("dumb-timestamps") {
		if envDT, err := strconv.ParseBool(os.Getenv("SENTRY_GATEWAY_DUMB_TIMESTAMPS")); err == nil {
			dumbTimestamps = envDT
		}
	}

	hookChan := make(chan gatewayRequest)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		dsn := defaultDSN

		if sentry, err := url.Parse(sentryURL); sentryURL != "" && err == nil {
			if token, _, ok := r.BasicAuth(); ok && r.URL.Path != "/" {
				dsn = fmt.Sprintf("%s://%s@%s%s", sentry.Scheme, token, sentry.Host, r.URL.Path)
			}
		}

		var wh notify.WebhookMessage

		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()

		err := decoder.Decode(&wh)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid webhook: %s\n", err)
			return
		}

		hookChan <- gatewayRequest{dsn, &wh}
	})

	s := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Unable to start server: %s\n", err)
			os.Exit(1)
		}
	}()

	go worker(hookChan, t, fpTemplates, dumbTimestamps)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = s.Shutdown(ctx)
	if err != nil {
		return err
	}

	for len(hookChan) > 0 {
		time.Sleep(1)
	}
	close(hookChan)

	return nil
}

func createTemplate(templateString string) (*template.Template, error) {
	t := template.New("").Option("missingkey=zero")
	t.Funcs(template.FuncMap(amt.DefaultFuncs))
	return t.Parse(templateString)
}

func getEventTimestamp(alert amt.Alert, dumb bool) raven.Timestamp {
	if dumb {
		return raven.Timestamp(time.Now())
	}

	return raven.Timestamp(map[string]time.Time{
		"firing":   alert.StartsAt,
		"resolved": alert.EndsAt,
	}[alert.Status])
}

func getEventTags(alert amt.Alert) []raven.Tag {
	var tags []raven.Tag
	for _, label := range alert.Labels.SortedPairs() {
		tags = append(tags, raven.Tag{Key: label.Name, Value: label.Value})
	}
	return tags
}

func getEventFingerprint(alert amt.Alert, fingerprintTemplates []*template.Template) []string {
	var fingerprint []string
	for _, fpTemplate := range fingerprintTemplates {
		var fp bytes.Buffer

		err := fpTemplate.Execute(&fp, alert)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid fingerprint template: %s\n", err)
			continue
		}

		fingerprint = append(fingerprint, fp.String())
	}
	return fingerprint
}

func worker(
	hookChan chan gatewayRequest,
	t *template.Template,
	fingerprintTemplates []*template.Template,
	dumbTimestamps bool,
) {
	ravenClients := map[string]*raven.Client{}

	for req := range hookChan {
		dsn, wh := req.dsn, req.message

		client := ravenClients[dsn]
		if client == nil {
			if newClient, err := raven.NewClient(dsn, map[string]string{}); err == nil {
				ravenClients[dsn] = newClient
				client = newClient
			} else {
				fmt.Fprintf(os.Stderr, "Could not init Sentry client: %s\n", err)
				continue
			}
		}

		for _, alert := range wh.Alerts {
			var buf bytes.Buffer

			err := t.Execute(&buf, alert)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid template: %s\n", err)
				continue
			}

			packet := &raven.Packet{
				Timestamp: getEventTimestamp(alert, dumbTimestamps),
				Message:   buf.String(),
				Extra: map[string]interface{}{
					"starts_at": alert.StartsAt,
					"ends_at":   alert.EndsAt,
				},
				Logger:      "alertmanager",
				Tags:        getEventTags(alert),
				Fingerprint: getEventFingerprint(alert, fingerprintTemplates),
			}

			eventID, ch := client.Capture(packet, alert.Labels)
			<-ch

			log.Printf("event_id:%s alert_name:%s\n", eventID, alert.Labels["alertname"])
		}
	}
}

func version() {
	fmt.Printf("Version: %s (%s)\n", VERSION, COMMIT)
}
