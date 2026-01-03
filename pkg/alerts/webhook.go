package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"
)

// WebhookNotifier sends alerts to generic HTTP webhooks
type WebhookNotifier struct {
	Name_         string
	URL           string
	Method        string
	Headers       map[string]string
	Template      string
	EventFilter   []string // List of event types to send (empty = all)
	Enabled_      bool
	RetryAttempts int
	Timeout       time.Duration
}

// NewWebhookNotifier creates a new generic webhook notifier
func NewWebhookNotifier(name, url, method string, headers map[string]string, tmpl string) *WebhookNotifier {
	if method == "" {
		method = "POST"
	}

	return &WebhookNotifier{
		Name_:         name,
		URL:           url,
		Method:        method,
		Headers:       headers,
		Template:      tmpl,
		Enabled_:      true,
		RetryAttempts: 3,
		Timeout:       10 * time.Second,
	}
}

// Send sends an alert to the webhook
func (w *WebhookNotifier) Send(ctx context.Context, alert Alert) error {
	if w.URL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	payload, err := w.renderPayload(alert)
	if err != nil {
		return fmt.Errorf("failed to render payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= w.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := w.sendRequest(ctx, payload)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", w.RetryAttempts+1, lastErr)
}

// Name returns the name of this notifier
func (w *WebhookNotifier) Name() string {
	return w.Name_
}

// Enabled returns whether this notifier is enabled
func (w *WebhookNotifier) Enabled() bool {
	return w.Enabled_
}

// ShouldSend determines if this alert should be sent based on event filters
func (w *WebhookNotifier) ShouldSend(alert Alert) bool {
	// If no filters configured, send all
	if len(w.EventFilter) == 0 {
		return true
	}

	// Check if alert's event type is in the filter list
	for _, eventType := range w.EventFilter {
		if eventType == alert.EventType {
			return true
		}
	}

	return false
}

// sendRequest sends the HTTP request
func (w *WebhookNotifier) sendRequest(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, w.Method, w.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range w.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: w.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

// renderPayload renders the payload using the template or default JSON
func (w *WebhookNotifier) renderPayload(alert Alert) ([]byte, error) {
	if w.Template != "" {
		// Use custom template
		return w.renderTemplate(alert)
	}

	// Use default JSON payload
	return w.defaultPayload(alert)
}

// renderTemplate renders the payload using a Go template
func (w *WebhookNotifier) renderTemplate(alert Alert) ([]byte, error) {
	tmpl, err := template.New("webhook").Parse(w.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, alert); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// defaultPayload creates the default JSON payload
func (w *WebhookNotifier) defaultPayload(alert Alert) ([]byte, error) {
	payload := map[string]interface{}{
		"level":       string(alert.Level),
		"title":       alert.Title,
		"description": alert.Description,
		"source":      alert.Source,
		"timestamp":   alert.Timestamp.Format(time.RFC3339),
		"event_type":  alert.EventType,
	}

	if len(alert.Labels) > 0 {
		payload["labels"] = alert.Labels
	}

	if len(alert.Metadata) > 0 {
		payload["metadata"] = alert.Metadata
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return data, nil
}

// PagerDutyNotifier is a specialized webhook notifier for PagerDuty
type PagerDutyNotifier struct {
	*WebhookNotifier
	IntegrationKey string
}

// NewPagerDutyNotifier creates a PagerDuty notifier
func NewPagerDutyNotifier(integrationKey string) *PagerDutyNotifier {
	// PagerDuty Events API v2 endpoint
	template := `{
  "routing_key": "{{ .IntegrationKey }}",
  "event_action": "{{ if eq .Level "critical" }}trigger{{ else }}acknowledge{{ end }}",
  "payload": {
    "summary": "{{ .Title }}",
    "severity": "{{ if eq .Level "critical" }}critical{{ else if eq .Level "warning" }}warning{{ else }}info{{ end }}",
    "source": "{{ .Source }}",
    "timestamp": "{{ .Timestamp.Format "2006-01-02T15:04:05Z07:00" }}",
    "custom_details": {{ .Metadata | toJson }}
  }
}`

	webhook := NewWebhookNotifier(
		"pagerduty",
		"https://events.pagerduty.com/v2/enqueue",
		"POST",
		map[string]string{
			"Content-Type": "application/json",
		},
		template,
	)

	return &PagerDutyNotifier{
		WebhookNotifier: webhook,
		IntegrationKey:  integrationKey,
	}
}
