package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	_ = admissionv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
}

// Server implements the admission webhook server
type Server struct {
	Client         client.Client
	Port           int
	CircuitBreaker *CircuitBreaker
}

// NewServer creates a new webhook server
func NewServer(client client.Client, port int) *Server {
	return &Server{
		Client:         client,
		Port:           port,
		CircuitBreaker: NewCircuitBreaker(),
	}
}

// Start starts the webhook server
func (s *Server) Start(ctx context.Context) error {
	log := log.FromContext(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/validate", s.handleValidate)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/metrics", s.handleMetrics)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Port),
		Handler: mux,
	}

	log.Info("Starting webhook server", "port", s.Port)

	// TLS certificate paths (mounted from cert-manager Secret)
	certPath := "/tmp/k8s-webhook-server/serving-certs/tls.crt"
	keyPath := "/tmp/k8s-webhook-server/serving-certs/tls.key"

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
			log.Error(err, "Webhook server failed")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Info("Shutting down webhook server")
	return server.Shutdown(context.Background())
}

// handleValidate handles admission review requests for pod validation
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := log.FromContext(ctx)

	// Check circuit breaker
	if s.CircuitBreaker.IsTripped() {
		log.Info("Circuit breaker tripped, allowing request with warning")
		// Fail-open: allow request but warn
		response := &admissionv1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "admission.k8s.io/v1",
				Kind:       "AdmissionReview",
			},
			Response: &admissionv1.AdmissionResponse{
				Allowed:  true,
				Warnings: []string{"Webhook validation temporarily disabled due to high error rate"},
			},
		}
		responseBytes, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
		return
	}

	// Track validation success/failure
	defer func() {
		if r := recover(); r != nil {
			s.CircuitBreaker.RecordError()
			log.Error(fmt.Errorf("panic in validation: %v", r), "Panic recovered")
		}
	}()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.CircuitBreaker.RecordError()
		log.Error(err, "Failed to read request body")
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Decode admission review
	admissionReview := &admissionv1.AdmissionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, admissionReview); err != nil {
		s.CircuitBreaker.RecordError()
		log.Error(err, "Failed to decode admission review")
		http.Error(w, "Failed to decode admission review", http.StatusBadRequest)
		return
	}

	// Validate the request
	response := s.validate(ctx, admissionReview.Request)

	// Create response admission review
	responseReview := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: response,
	}
	responseReview.Response.UID = admissionReview.Request.UID

	// Encode and send response
	responseBytes, err := json.Marshal(responseReview)
	if err != nil {
		s.CircuitBreaker.RecordError()
		log.Error(err, "Failed to marshal response")
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	// Record success
	s.CircuitBreaker.RecordSuccess()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

// validate validates a pod against all active ClusterSpecs
func (s *Server) validate(ctx context.Context, request *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	log := log.FromContext(ctx)

	// Only validate Pods
	if request.Kind.Kind != "Pod" {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Decode pod
	pod := &corev1.Pod{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(request.Object.Raw, nil, pod); err != nil {
		log.Error(err, "Failed to decode pod")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to decode pod: %v", err),
			},
		}
	}

	// Get all ClusterSpecs with enforcement enabled
	var clusterSpecs kspecv1alpha1.ClusterSpecificationList
	if err := s.Client.List(ctx, &clusterSpecs); err != nil {
		log.Error(err, "Failed to list ClusterSpecs")
		// Fail open - allow pod if we can't check ClusterSpecs
		return &admissionv1.AdmissionResponse{
			Allowed:  true,
			Warnings: []string{"Failed to check cluster specifications, allowing by default"},
		}
	}

	// Validate pod against each active ClusterSpec
	for _, clusterSpec := range clusterSpecs.Items {
		// Skip if enforcement not enabled
		if clusterSpec.Spec.Enforcement == nil || !clusterSpec.Spec.Enforcement.Enabled {
			continue
		}

		// Skip if webhooks not enabled
		if clusterSpec.Spec.Webhooks == nil || !clusterSpec.Spec.Webhooks.Enabled {
			continue
		}

		// Skip if mode is monitor (no enforcement)
		if clusterSpec.Spec.Enforcement.Mode == "monitor" {
			continue
		}

		// Validate pod against this ClusterSpec
		if allowed, reason := s.validatePodAgainstSpec(ctx, pod, &clusterSpec); !allowed {
			// In audit mode, allow but warn
			if clusterSpec.Spec.Enforcement.Mode == "audit" {
				log.Info("Pod violates ClusterSpec (audit mode)",
					"pod", pod.Name,
					"namespace", pod.Namespace,
					"clusterSpec", clusterSpec.Name,
					"reason", reason)
				return &admissionv1.AdmissionResponse{
					Allowed:  true,
					Warnings: []string{fmt.Sprintf("Policy violation (audit): %s", reason)},
				}
			}

			// In enforce mode, deny
			log.Info("Pod violates ClusterSpec (enforce mode)",
				"pod", pod.Name,
				"namespace", pod.Namespace,
				"clusterSpec", clusterSpec.Name,
				"reason", reason)
			return &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: fmt.Sprintf("Pod violates cluster specification %s: %s", clusterSpec.Name, reason),
				},
			}
		}
	}

	// Pod is valid
	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

// validatePodAgainstSpec validates a pod against a ClusterSpec
func (s *Server) validatePodAgainstSpec(ctx context.Context, pod *corev1.Pod, clusterSpec *kspecv1alpha1.ClusterSpecification) (bool, string) {
	// Check workload requirements
	if clusterSpec.Spec.Workloads != nil && clusterSpec.Spec.Workloads.Containers != nil {
		// Check required fields
		for _, req := range clusterSpec.Spec.Workloads.Containers.Required {
			if !s.checkRequiredField(pod, req.Key, req.Value) {
				return false, fmt.Sprintf("Required field %s=%s not satisfied", req.Key, req.Value)
			}
		}

		// Check forbidden fields
		for _, forbidden := range clusterSpec.Spec.Workloads.Containers.Forbidden {
			if s.checkForbiddenField(pod, forbidden.Key, forbidden.Value) {
				return false, fmt.Sprintf("Forbidden field %s=%s found", forbidden.Key, forbidden.Value)
			}
		}
	}

	// Check image requirements
	if clusterSpec.Spec.Workloads != nil && clusterSpec.Spec.Workloads.Images != nil {
		for _, container := range pod.Spec.Containers {
			// Check image digest requirement
			if clusterSpec.Spec.Workloads.Images.RequireDigests {
				if !hasDigest(container.Image) {
					return false, fmt.Sprintf("Container %s must use image digest", container.Name)
				}
			}

			// Check blocked registries
			for _, blockedRegistry := range clusterSpec.Spec.Workloads.Images.BlockedRegistries {
				if matchesRegistry(container.Image, blockedRegistry) {
					return false, fmt.Sprintf("Container %s uses blocked registry %s", container.Name, blockedRegistry)
				}
			}
		}
	}

	return true, ""
}

// checkRequiredField checks if a required field is satisfied
func (s *Server) checkRequiredField(pod *corev1.Pod, key, value string) bool {
	switch key {
	case "securityContext.runAsNonRoot":
		// Check pod-level security context
		if pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.RunAsNonRoot != nil {
			return *pod.Spec.SecurityContext.RunAsNonRoot == (value == "true")
		}
		// Check container-level security contexts
		for _, container := range pod.Spec.Containers {
			if container.SecurityContext != nil && container.SecurityContext.RunAsNonRoot != nil {
				if *container.SecurityContext.RunAsNonRoot != (value == "true") {
					return false
				}
			} else {
				return false // Container doesn't have runAsNonRoot set
			}
		}
		return len(pod.Spec.Containers) > 0

	case "securityContext.allowPrivilegeEscalation":
		for _, container := range pod.Spec.Containers {
			if container.SecurityContext == nil || container.SecurityContext.AllowPrivilegeEscalation == nil {
				return false
			}
			if *container.SecurityContext.AllowPrivilegeEscalation != (value == "true") {
				return false
			}
		}
		return len(pod.Spec.Containers) > 0

	case "resources.limits.memory":
		if value == "true" {
			for _, container := range pod.Spec.Containers {
				if container.Resources.Limits == nil || container.Resources.Limits.Memory().IsZero() {
					return false
				}
			}
			return len(pod.Spec.Containers) > 0
		}
	}

	return true
}

// checkForbiddenField checks if a forbidden field is present
func (s *Server) checkForbiddenField(pod *corev1.Pod, key, value string) bool {
	switch key {
	case "securityContext.privileged":
		for _, container := range pod.Spec.Containers {
			if container.SecurityContext != nil && container.SecurityContext.Privileged != nil {
				if *container.SecurityContext.Privileged == (value == "true") {
					return true
				}
			}
		}

	case "hostNetwork":
		if pod.Spec.HostNetwork == (value == "true") {
			return true
		}

	case "hostPID":
		if pod.Spec.HostPID == (value == "true") {
			return true
		}

	case "hostIPC":
		if pod.Spec.HostIPC == (value == "true") {
			return true
		}
	}

	return false
}

// hasDigest checks if an image uses a digest
func hasDigest(image string) bool {
	// Image digest format: registry/image@sha256:...
	return len(image) > 0 && (image[len(image)-1:] != ":" && contains(image, "@sha256:"))
}

// matchesRegistry checks if an image matches a blocked registry
func matchesRegistry(image, registry string) bool {
	// Simple prefix match
	return len(image) >= len(registry) && image[:len(registry)] == registry
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// handleHealthz handles health check requests
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleReadyz handles readiness check requests
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleMetrics handles metrics requests
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats := s.CircuitBreaker.GetStats()

	response := map[string]interface{}{
		"total_requests":   stats.TotalRequests,
		"error_requests":   stats.ErrorRequests,
		"success_requests": stats.SuccessRequests,
		"error_rate":       stats.ErrorRate,
		"circuit_breaker": map[string]interface{}{
			"tripped":        stats.IsTripped,
			"last_trip_time": stats.LastTripTime,
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}
