# Admission Webhooks (Experimental)

**Status**: 🧪 Experimental - Disabled by default in v0.2.0

---

## Overview

kspec includes admission webhook support for real-time Pod validation. Webhooks provide immediate feedback when non-compliant workloads are created or updated, preventing violations before they enter the cluster.

**Current Status in v0.2.0**:
- ✅ **Webhook code implemented** - `pkg/webhooks/pod_webhook.go`
- ⚠️ **Disabled by default** - Requires cert-manager setup
- 🚧 **TLS provisioning not automated** - Manual setup required
- 📅 **Planned for v0.3.0** - Automated cert-manager integration

---

## Why Webhooks Are Disabled

Admission webhooks require TLS certificates to function. Without proper certificate provisioning:

1. **Operator crashes on startup** - Cannot find TLS certificates
2. **Cluster-wide impact** - `failurePolicy=fail` blocks ALL pod creations if webhook is misconfigured
3. **Complex setup** - Requires cert-manager or manual certificate management

**v0.2.0 Decision**: Disable webhooks by default to ensure kspec is **safe to install on any cluster** without prerequisites.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    kspec Operator                           │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Admission Webhook (Experimental)                    │  │
│  │                                                      │  │
│  │  • Validates Pods against ClusterSpecification      │  │
│  │  • Checks security context requirements             │  │
│  │  • Enforces image policies                          │  │
│  │  • Real-time validation (fail-fast)                 │  │
│  └──────────────────────────────────────────────────────┘  │
│                         ↓ TLS                               │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│          Kubernetes API Server                              │
│                                                             │
│  • Intercepts Pod create/update requests                   │
│  • Calls webhook for validation                            │
│  • Blocks non-compliant Pods (failurePolicy=fail)          │
└─────────────────────────────────────────────────────────────┘
```

---

## Current Implementation

### Pod Validator

**File**: `pkg/webhooks/pod_webhook.go`

**Validates**:
- ✅ Security context requirements (runAsNonRoot, allowPrivilegeEscalation)
- ✅ Resource limits and requests
- ✅ Privileged container prevention
- ✅ Host namespace restrictions
- ✅ Image registry policies

**Kubebuilder Marker**:
```go
// +kubebuilder:webhook:path=/validate-v1-pod,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=vpod.kspec.io,admissionReviewVersions=v1
```

### Failure Policy

**Current**: `failurePolicy=fail` (fail-closed)

- **Safe**: Prevents violations from entering the cluster
- **Risk**: If webhook is unavailable, ALL pod creations fail cluster-wide
- **Mitigation**: Webhooks disabled by default in v0.2.0

---

## Enabling Webhooks (Manual Setup)

⚠️ **WARNING**: Only enable webhooks if you understand the risks and have cert-manager installed.

### Prerequisites

1. **cert-manager installed** in your cluster:
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
   ```

2. **kspec operator running**:
   ```bash
   kubectl get pods -n kspec-system
   ```

### Step 1: Create Certificate Issuer

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: kspec-selfsigned-issuer
  namespace: kspec-system
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: kspec-webhook-cert
  namespace: kspec-system
spec:
  secretName: kspec-webhook-tls
  duration: 2160h # 90 days
  renewBefore: 360h # 15 days
  subject:
    organizations:
      - kspec
  isCA: false
  privateKey:
    algorithm: RSA
    encoding: PKCS1
    size: 2048
  usages:
    - server auth
    - client auth
  dnsNames:
    - kspec-webhook-service.kspec-system.svc
    - kspec-webhook-service.kspec-system.svc.cluster.local
  issuerRef:
    name: kspec-selfsigned-issuer
    kind: Issuer
```

### Step 2: Create Webhook Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kspec-webhook-service
  namespace: kspec-system
spec:
  ports:
    - port: 443
      targetPort: 9443
      protocol: TCP
      name: webhook
  selector:
    app.kubernetes.io/name: kspec-operator
```

### Step 3: Create ValidatingWebhookConfiguration

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: kspec-validating-webhook-configuration
  annotations:
    cert-manager.io/inject-ca-from: kspec-system/kspec-webhook-cert
webhooks:
  - name: vpod.kspec.io
    admissionReviewVersions:
      - v1
    clientConfig:
      service:
        name: kspec-webhook-service
        namespace: kspec-system
        path: /validate-v1-pod
    failurePolicy: Fail
    matchPolicy: Equivalent
    rules:
      - apiGroups: [""]
        apiVersions: ["v1"]
        operations: ["CREATE", "UPDATE"]
        resources: ["pods"]
        scope: "*"
    sideEffects: None
    timeoutSeconds: 10
```

### Step 4: Update Operator Deployment

Edit the operator deployment to enable webhooks:

```bash
kubectl edit deployment kspec-operator -n kspec-system
```

Change:
```yaml
args:
  - --enable-webhooks=false
```

To:
```yaml
args:
  - --enable-webhooks=true
```

Add volume mount for TLS certificates:
```yaml
volumeMounts:
  - name: webhook-certs
    mountPath: /tmp/k8s-webhook-server/serving-certs
    readOnly: true

volumes:
  - name: webhook-certs
    secret:
      secretName: kspec-webhook-tls
```

### Step 5: Verify Webhook

```bash
# Check certificate is ready
kubectl get certificate -n kspec-system

# Check webhook configuration
kubectl get validatingwebhookconfiguration kspec-validating-webhook-configuration

# Test webhook (should be denied)
kubectl run test-privileged --image=nginx --restart=Never -- --privileged
# Expected: Error from server (Forbidden): admission webhook "vpod.kspec.io" denied the request
```

---

## Why Not Enabled by Default?

### Complexity vs Value Trade-off

**Complexity Added**:
- cert-manager dependency (extra component to install)
- TLS certificate lifecycle management
- Webhook availability monitoring
- Failure domain expansion (webhook failure = cluster-wide pod creation failure)

**Value in v0.2.0**:
- **Kyverno provides same enforcement** with simpler setup
- **CLI `kspec enforce`** generates Kyverno policies
- **Monitoring is primary use case** for v0.2.0 operator

**Decision**: Defer webhook TLS automation to v0.3.0 when enforcement is operator-native.

---

## Recommended Enforcement Path (v0.2.0)

Instead of webhooks, use **Kyverno** for policy enforcement:

### 1. Install Kyverno

```bash
kubectl create -f https://github.com/kyverno/kyverno/releases/download/v1.10.0/install.yaml
```

### 2. Generate Policies from ClusterSpec

```bash
kspec enforce --spec cluster-spec.yaml
```

This automatically:
- ✅ Generates Kyverno ClusterPolicies from your ClusterSpecification
- ✅ Applies policies to cluster
- ✅ Validates policies are active
- ✅ No TLS certificates required
- ✅ Production-tested and stable

### 3. Monitor Enforcement

```bash
# View applied policies
kubectl get clusterpolicy

# Check policy reports
kubectl get policyreport -A

# Verify enforcement
kubectl run test-privileged --image=nginx --privileged
# Expected: Denied by Kyverno
```

---

## Roadmap

### v0.3.0: Automated Webhook TLS
- ✅ Automated cert-manager integration
- ✅ ValidatingWebhookConfiguration manifests included
- ✅ Webhook Service included
- ✅ Certificate lifecycle automation
- ✅ Enabled by default (with opt-out)

### v0.4.0: Advanced Webhook Features
- Multi-resource validation (Deployments, StatefulSets, DaemonSets)
- Mutating webhooks for auto-remediation
- Custom validation rules from ClusterSpecification
- Webhook metrics and alerting

---

## FAQ

### Q: Why are webhooks experimental?
**A**: TLS certificate provisioning is complex and creates a dependency on cert-manager. We want v0.2.0 to be installable on any cluster without prerequisites.

### Q: Can I use webhooks in production?
**A**: Yes, if you follow the manual setup steps above and have cert-manager installed. However, we recommend using Kyverno for enforcement in v0.2.0.

### Q: What happens if I enable webhooks without TLS?
**A**: The operator will crash on startup with: `open /tmp/k8s-webhook-server/serving-certs/tls.crt: no such file or directory`

### Q: Will Kyverno be required in the future?
**A**: No. In v0.3.0+, the operator will have native policy enforcement via webhooks. Kyverno will remain an optional alternative.

### Q: How do I test webhook validation logic without enabling them?
**A**: Use the CLI `kspec validate` command to test validation rules against YAML files.

---

## Get Help

- 📖 **Documentation**: [Operator Quickstart](./OPERATOR_QUICKSTART.md)
- 🐛 **Issues**: https://github.com/cloudcwfranck/kspec/issues
- 💬 **Discussions**: https://github.com/cloudcwfranck/kspec/discussions

---

**Last Updated**: 2025-12-27 (v0.2.0)
