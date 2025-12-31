package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlertConfigSpec defines the desired state of AlertConfig
type AlertConfigSpec struct {
	// Slack configuration for alert notifications
	// +optional
	Slack *SlackConfig `json:"slack,omitempty"`

	// Webhooks is a list of generic webhook configurations
	// +optional
	Webhooks []WebhookConfig `json:"webhooks,omitempty"`

	// Routes defines how alerts are routed to different notifiers
	// +optional
	Routes []AlertRoute `json:"routes,omitempty"`

	// DefaultSeverity is the minimum severity level for alerts (default: warning)
	// +kubebuilder:validation:Enum=info;warning;critical
	// +kubebuilder:default:="warning"
	// +optional
	DefaultSeverity string `json:"defaultSeverity,omitempty"`

	// Enabled globally enables or disables all alerting
	// +kubebuilder:default:=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// SlackConfig defines Slack notification settings
type SlackConfig struct {
	// Enabled enables or disables Slack notifications
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled"`

	// WebhookURL is the Slack incoming webhook URL
	// This should be stored in a Secret and referenced here
	// Format: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX
	// +optional
	WebhookURL string `json:"webhookURL,omitempty"`

	// WebhookURLSecretRef references a Secret containing the webhook URL
	// The secret should have a key 'url' with the webhook URL
	// +optional
	WebhookURLSecretRef *SecretReference `json:"webhookURLSecretRef,omitempty"`

	// Channel is the Slack channel to post to (e.g., "#kspec-alerts")
	// +optional
	Channel string `json:"channel,omitempty"`

	// Username is the bot username to display
	// +kubebuilder:default:="kspec-bot"
	// +optional
	Username string `json:"username,omitempty"`

	// IconEmoji is the emoji to use as the bot icon (e.g., ":shield:")
	// +kubebuilder:default:=":shield:"
	// +optional
	IconEmoji string `json:"iconEmoji,omitempty"`

	// Events is a list of event types to send to Slack
	// If empty, all events are sent
	// Possible values: DriftDetected, ComplianceFailure, PolicyViolation, CircuitBreakerTripped, RemediationPerformed
	// +optional
	Events []string `json:"events,omitempty"`
}

// WebhookConfig defines a generic webhook notification
type WebhookConfig struct {
	// Name is a unique identifier for this webhook
	Name string `json:"name"`

	// URL is the webhook endpoint
	// +optional
	URL string `json:"url,omitempty"`

	// URLSecretRef references a Secret containing the webhook URL
	// +optional
	URLSecretRef *SecretReference `json:"urlSecretRef,omitempty"`

	// Method is the HTTP method to use (default: POST)
	// +kubebuilder:validation:Enum=GET;POST;PUT;PATCH
	// +kubebuilder:default:="POST"
	// +optional
	Method string `json:"method,omitempty"`

	// Headers are custom HTTP headers to send
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// HeadersSecretRef references a Secret containing headers
	// Useful for Authorization headers
	// +optional
	HeadersSecretRef *SecretReference `json:"headersSecretRef,omitempty"`

	// Template is a Go template for the request body
	// If not specified, a default JSON payload is used
	// Template data includes: .Level, .Title, .Description, .Source, .Timestamp, .Labels, .Metadata
	// +optional
	Template string `json:"template,omitempty"`

	// Events is a list of event types to send to this webhook
	// If empty, all events are sent
	// +optional
	Events []string `json:"events,omitempty"`

	// RetryAttempts is the number of retry attempts on failure (default: 3)
	// +kubebuilder:default:=3
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +optional
	RetryAttempts int `json:"retryAttempts,omitempty"`

	// TimeoutSeconds is the request timeout in seconds (default: 10)
	// +kubebuilder:default:=10
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=60
	// +optional
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

// AlertRoute defines how alerts are routed based on labels
type AlertRoute struct {
	// Match is a map of label key-value pairs to match
	// All labels must match for this route to apply
	Match map[string]string `json:"match"`

	// Notifiers is a list of notifier names to send matching alerts to
	// Names can be "slack" or webhook names
	Notifiers []string `json:"notifiers"`

	// Continue indicates whether to continue matching other routes after this one
	// +kubebuilder:default:=false
	// +optional
	Continue bool `json:"continue,omitempty"`
}

// SecretReference is a reference to a secret key
type SecretReference struct {
	// Name is the name of the secret
	Name string `json:"name"`

	// Namespace is the namespace of the secret (defaults to AlertConfig namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Key is the key in the secret data
	Key string `json:"key"`
}

// AlertConfigStatus defines the observed state of AlertConfig
type AlertConfigStatus struct {
	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastAlertTime is the timestamp of the last sent alert
	// +optional
	LastAlertTime *metav1.Time `json:"lastAlertTime,omitempty"`

	// AlertsSent is the total number of alerts sent
	// +optional
	AlertsSent int64 `json:"alertsSent,omitempty"`

	// AlertsFailed is the number of alerts that failed to send
	// +optional
	AlertsFailed int64 `json:"alertsFailed,omitempty"`

	// NotifierStatus contains status for each configured notifier
	// +optional
	NotifierStatus map[string]NotifierStatus `json:"notifierStatus,omitempty"`
}

// NotifierStatus represents the status of a specific notifier
type NotifierStatus struct {
	// LastAlertTime is when this notifier last sent an alert
	// +optional
	LastAlertTime *metav1.Time `json:"lastAlertTime,omitempty"`

	// AlertsSent is alerts sent by this notifier
	AlertsSent int64 `json:"alertsSent,omitempty"`

	// AlertsFailed is failed alerts for this notifier
	AlertsFailed int64 `json:"alertsFailed,omitempty"`

	// LastError is the last error message (if any)
	// +optional
	LastError string `json:"lastError,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ac
// +kubebuilder:printcolumn:name="Slack",type="boolean",JSONPath=".spec.slack.enabled"
// +kubebuilder:printcolumn:name="Webhooks",type="integer",JSONPath=".spec.webhooks[*].name"
// +kubebuilder:printcolumn:name="Alerts Sent",type="integer",JSONPath=".status.alertsSent"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AlertConfig is the Schema for the alertconfigs API
type AlertConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertConfigSpec   `json:"spec,omitempty"`
	Status AlertConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertConfigList contains a list of AlertConfig
type AlertConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertConfig{}, &AlertConfigList{})
}
