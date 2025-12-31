package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackNotifier sends alerts to Slack via incoming webhooks
type SlackNotifier struct {
	WebhookURL  string
	Channel     string
	Username    string
	IconEmoji   string
	Enabled_    bool
	EventFilter []string // List of event types to send (empty = all)
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL, channel, username, iconEmoji string) *SlackNotifier {
	if username == "" {
		username = "kspec-bot"
	}
	if iconEmoji == "" {
		iconEmoji = ":shield:"
	}

	return &SlackNotifier{
		WebhookURL: webhookURL,
		Channel:    channel,
		Username:   username,
		IconEmoji:  iconEmoji,
		Enabled_:   true,
	}
}

// Send sends an alert to Slack
func (s *SlackNotifier) Send(ctx context.Context, alert Alert) error {
	if s.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL is not configured")
	}

	payload := s.buildPayload(alert)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.WebhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

// Name returns the name of this notifier
func (s *SlackNotifier) Name() string {
	return "slack"
}

// Enabled returns whether this notifier is enabled
func (s *SlackNotifier) Enabled() bool {
	return s.Enabled_
}

// ShouldSend determines if this alert should be sent based on event filters
func (s *SlackNotifier) ShouldSend(alert Alert) bool {
	// If no filters configured, send all
	if len(s.EventFilter) == 0 {
		return true
	}

	// Check if alert's event type is in the filter list
	for _, eventType := range s.EventFilter {
		if eventType == alert.EventType {
			return true
		}
	}

	return false
}

// buildPayload constructs the Slack message payload
func (s *SlackNotifier) buildPayload(alert Alert) map[string]interface{} {
	attachment := map[string]interface{}{
		"color":     s.alertColor(alert.Level),
		"title":     alert.Title,
		"text":      alert.Description,
		"footer":    fmt.Sprintf("Source: %s", alert.Source),
		"ts":        alert.Timestamp.Unix(),
		"fields":    s.buildFields(alert),
		"mrkdwn_in": []string{"text", "fields"},
	}

	payload := map[string]interface{}{
		"username":    s.Username,
		"icon_emoji":  s.IconEmoji,
		"attachments": []interface{}{attachment},
	}

	if s.Channel != "" {
		payload["channel"] = s.Channel
	}

	return payload
}

// alertColor returns the Slack color code for an alert level
func (s *SlackNotifier) alertColor(level AlertLevel) string {
	switch level {
	case AlertLevelCritical:
		return "danger" // Red
	case AlertLevelWarning:
		return "warning" // Yellow
	case AlertLevelInfo:
		return "good" // Green
	default:
		return "#808080" // Gray
	}
}

// buildFields creates Slack attachment fields from alert metadata
func (s *SlackNotifier) buildFields(alert Alert) []map[string]interface{} {
	fields := []map[string]interface{}{
		{
			"title": "Severity",
			"value": string(alert.Level),
			"short": true,
		},
	}

	if alert.EventType != "" {
		fields = append(fields, map[string]interface{}{
			"title": "Event Type",
			"value": alert.EventType,
			"short": true,
		})
	}

	// Add labels as fields
	for key, value := range alert.Labels {
		fields = append(fields, map[string]interface{}{
			"title": key,
			"value": value,
			"short": true,
		})
	}

	// Add selected metadata fields
	if alert.Metadata != nil {
		if cluster, ok := alert.Metadata["cluster"].(string); ok {
			fields = append(fields, map[string]interface{}{
				"title": "Cluster",
				"value": cluster,
				"short": true,
			})
		}

		if count, ok := alert.Metadata["count"].(int); ok {
			fields = append(fields, map[string]interface{}{
				"title": "Count",
				"value": fmt.Sprintf("%d", count),
				"short": true,
			})
		}

		if score, ok := alert.Metadata["compliance_score"].(float64); ok {
			fields = append(fields, map[string]interface{}{
				"title": "Compliance Score",
				"value": fmt.Sprintf("%.1f%%", score),
				"short": true,
			})
		}
	}

	return fields
}
