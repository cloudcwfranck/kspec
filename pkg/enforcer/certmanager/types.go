package certmanager

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Certificate defines a cert-manager Certificate resource.
// This is a vendored subset of github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1
// to avoid heavyweight dependencies while maintaining API compatibility.
type Certificate struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              CertificateSpec   `json:"spec" yaml:"spec"`
	Status            CertificateStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// CertificateSpec defines the desired state of a Certificate.
type CertificateSpec struct {
	// SecretName is the name of the secret resource that will hold the certificate
	SecretName string `json:"secretName"`

	// DNSNames is a list of DNS subjectAltNames to be set on the Certificate
	// +optional
	DNSNames []string `json:"dnsNames,omitempty"`

	// IssuerRef is a reference to the issuer for this certificate
	IssuerRef IssuerRef `json:"issuerRef"`

	// Duration is the requested 'duration' (i.e. lifetime) of the Certificate
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`

	// RenewBefore is the time before expiry when the certificate should be renewed
	// +optional
	RenewBefore *metav1.Duration `json:"renewBefore,omitempty"`

	// Usages is the set of x509 usages that are requested for the certificate
	// +optional
	Usages []KeyUsage `json:"usages,omitempty"`
}

// IssuerRef references a cert-manager Issuer or ClusterIssuer.
type IssuerRef struct {
	// Name of the Issuer/ClusterIssuer resource
	Name string `json:"name"`

	// Kind of the Issuer resource (Issuer or ClusterIssuer)
	// +optional
	Kind string `json:"kind,omitempty"`

	// Group of the Issuer resource
	// +optional
	Group string `json:"group,omitempty"`
}

// KeyUsage specifies valid usage contexts for keys.
type KeyUsage string

const (
	UsageSigning           KeyUsage = "signing"
	UsageDigitalSignature  KeyUsage = "digital signature"
	UsageContentCommitment KeyUsage = "content commitment"
	UsageKeyEncipherment   KeyUsage = "key encipherment"
	UsageKeyAgreement      KeyUsage = "key agreement"
	UsageDataEncipherment  KeyUsage = "data encipherment"
	UsageCertSign          KeyUsage = "cert sign"
	UsageCRLSign           KeyUsage = "crl sign"
	UsageEncipherOnly      KeyUsage = "encipher only"
	UsageDecipherOnly      KeyUsage = "decipher only"
	UsageAny               KeyUsage = "any"
	UsageServerAuth        KeyUsage = "server auth"
	UsageClientAuth        KeyUsage = "client auth"
	UsageCodeSigning       KeyUsage = "code signing"
	UsageEmailProtection   KeyUsage = "email protection"
	UsageSMIME             KeyUsage = "s/mime"
	UsageIPsecEndSystem    KeyUsage = "ipsec end system"
	UsageIPsecTunnel       KeyUsage = "ipsec tunnel"
	UsageIPsecUser         KeyUsage = "ipsec user"
	UsageTimestamping      KeyUsage = "timestamping"
	UsageOCSPSigning       KeyUsage = "ocsp signing"
	UsageMicrosoftSGC      KeyUsage = "microsoft sgc"
	UsageNetscapeSGC       KeyUsage = "netscape sgc"
)

// CertificateStatus defines the observed state of a Certificate.
type CertificateStatus struct {
	// Conditions is a list of certificate conditions
	// +optional
	Conditions []CertificateCondition `json:"conditions,omitempty"`

	// NotAfter is the expiry time of the certificate
	// +optional
	NotAfter *metav1.Time `json:"notAfter,omitempty"`

	// NotBefore is the start time of the certificate validity
	// +optional
	NotBefore *metav1.Time `json:"notBefore,omitempty"`

	// RenewalTime is when the certificate will be renewed
	// +optional
	RenewalTime *metav1.Time `json:"renewalTime,omitempty"`
}

// CertificateCondition contains condition information for a Certificate.
type CertificateCondition struct {
	// Type of the condition
	Type CertificateConditionType `json:"type"`

	// Status of the condition (True, False, Unknown)
	Status ConditionStatus `json:"status"`

	// LastTransitionTime is the timestamp of the last update
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine-readable explanation
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable description
	// +optional
	Message string `json:"message,omitempty"`
}

// CertificateConditionType represents a Certificate condition type.
type CertificateConditionType string

const (
	// CertificateConditionReady indicates the certificate is ready for use
	CertificateConditionReady CertificateConditionType = "Ready"

	// CertificateConditionIssuing indicates the certificate is being issued
	CertificateConditionIssuing CertificateConditionType = "Issuing"
)

// ConditionStatus represents a condition's status.
type ConditionStatus string

const (
	// ConditionTrue means the condition is true
	ConditionTrue ConditionStatus = "True"

	// ConditionFalse means the condition is false
	ConditionFalse ConditionStatus = "False"

	// ConditionUnknown means the condition status is unknown
	ConditionUnknown ConditionStatus = "Unknown"
)

// CertificateGVR returns the GroupVersionResource for Certificate.
func CertificateGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "certificates",
	}
}

// GroupVersionKind returns the GroupVersionKind for Certificate.
func (c *Certificate) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "Certificate",
	}
}

// DeepCopyObject implements runtime.Object interface for Certificate.
func (c *Certificate) DeepCopyObject() runtime.Object {
	if c == nil {
		return nil
	}
	out := new(Certificate)
	c.DeepCopyInto(out)
	return out
}

// DeepCopyInto performs a deep copy of Certificate into out.
func (c *Certificate) DeepCopyInto(out *Certificate) {
	*out = *c
	out.TypeMeta = c.TypeMeta
	c.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	c.Spec.DeepCopyInto(&out.Spec)
	c.Status.DeepCopyInto(&out.Status)
}

// DeepCopyInto performs a deep copy of CertificateSpec into out.
func (s *CertificateSpec) DeepCopyInto(out *CertificateSpec) {
	*out = *s
	if s.DNSNames != nil {
		out.DNSNames = make([]string, len(s.DNSNames))
		copy(out.DNSNames, s.DNSNames)
	}
	if s.Duration != nil {
		out.Duration = s.Duration.DeepCopy()
	}
	if s.RenewBefore != nil {
		out.RenewBefore = s.RenewBefore.DeepCopy()
	}
	if s.Usages != nil {
		out.Usages = make([]KeyUsage, len(s.Usages))
		copy(out.Usages, s.Usages)
	}
}

// DeepCopyInto performs a deep copy of CertificateStatus into out.
func (s *CertificateStatus) DeepCopyInto(out *CertificateStatus) {
	*out = *s
	if s.Conditions != nil {
		out.Conditions = make([]CertificateCondition, len(s.Conditions))
		for i := range s.Conditions {
			s.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
	if s.NotAfter != nil {
		out.NotAfter = s.NotAfter.DeepCopy()
	}
	if s.NotBefore != nil {
		out.NotBefore = s.NotBefore.DeepCopy()
	}
	if s.RenewalTime != nil {
		out.RenewalTime = s.RenewalTime.DeepCopy()
	}
}

// DeepCopyInto performs a deep copy of CertificateCondition into out.
func (c *CertificateCondition) DeepCopyInto(out *CertificateCondition) {
	*out = *c
	if c.LastTransitionTime != nil {
		out.LastTransitionTime = c.LastTransitionTime.DeepCopy()
	}
}

// NewCertificate creates a new Certificate with standard defaults.
func NewCertificate(namespace, name, secretName string, dnsNames []string, issuerRef IssuerRef) *Certificate {
	return &Certificate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cert-manager.io/v1",
			Kind:       "Certificate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: CertificateSpec{
			SecretName: secretName,
			DNSNames:   dnsNames,
			IssuerRef:  issuerRef,
			Usages: []KeyUsage{
				UsageDigitalSignature,
				UsageKeyEncipherment,
				UsageServerAuth,
			},
		},
	}
}

// IsCertificateReady checks if a certificate is ready.
func IsCertificateReady(cert *Certificate) bool {
	for _, condition := range cert.Status.Conditions {
		if condition.Type == CertificateConditionReady && condition.Status == ConditionTrue {
			return true
		}
	}
	return false
}
