# kspec GitOps Deployment

Deploy kspec with **zero local commands** using GitOps tools like ArgoCD, Flux, or plain `kubectl apply`.

## 🚀 Quick Deploy (No Local Commands)

### Option 1: ArgoCD

```yaml
# Create this file in your GitOps repo: apps/kspec.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kspec
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/cloudcwfranck/kspec
    targetRevision: main
    path: config/dashboard
  destination:
    server: https://kubernetes.default.svc
    namespace: kspec-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

**That's it!** ArgoCD will:
- Deploy the operator
- Deploy the web dashboard
- Set up all CRDs and RBAC
- Auto-sync on git changes

### Option 2: Flux

```yaml
# Create this file in your GitOps repo: clusters/production/kspec.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: kspec
  namespace: flux-system
spec:
  interval: 10m
  path: ./config/dashboard
  prune: true
  sourceRef:
    kind: GitRepository
    name: kspec
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: kspec
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/cloudcwfranck/kspec
  ref:
    branch: main
```

**Flux will automatically**:
- Sync manifests from the repo
- Deploy to your cluster
- Update on git changes

### Option 3: Raw Manifests (Manual GitOps)

```bash
# Just commit this file to your repo: k8s/kspec/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- https://github.com/cloudcwfranck/kspec/config/dashboard?ref=main

# Optional: customize settings
namespace: kspec-system

images:
- name: kspec-operator:latest
  newName: ghcr.io/cloudcwfranck/kspec-operator
  newTag: v0.2.0
- name: kspec-dashboard:latest
  newName: ghcr.io/cloudcwfranck/kspec-dashboard
  newTag: v0.2.0
```

## 📊 Access the Dashboard

### Option 1: Port Forward (No Ingress Needed)

```bash
# Just add this to your README - users run ONLY this one command:
kubectl port-forward -n kspec-system svc/kspec-dashboard 8000:80
```

Then open: http://localhost:8000

### Option 2: Ingress (Public Access)

Edit `config/dashboard/deployment.yaml` and update:

```yaml
spec:
  rules:
  - host: kspec.yourcompany.com  # Your domain
```

Commit to git → ArgoCD/Flux deploys → Dashboard available at https://kspec.yourcompany.com

### Option 3: LoadBalancer

Change Service type in your GitOps overlay:

```yaml
# overlays/production/service-patch.yaml
apiVersion: v1
kind: Service
metadata:
  name: kspec-dashboard
  namespace: kspec-system
spec:
  type: LoadBalancer  # Change from ClusterIP
```

## 🔧 GitOps Workflow

### 1. Initial Setup (One-Time)

```
your-gitops-repo/
├── apps/
│   └── kspec.yaml              # ArgoCD Application
├── clusters/
│   ├── production/
│   │   ├── kspec/
│   │   │   ├── kustomization.yaml
│   │   │   ├── clusterspec.yaml
│   │   │   └── clusters.yaml
│   └── staging/
│       └── kspec/
│           └── ...
```

### 2. Add Clusters (GitOps)

Create `clusters/production/kspec/clusters.yaml`:

```yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterTarget
metadata:
  name: prod-us-east
  namespace: kspec-system
spec:
  apiServerURL: https://prod-us-east.k8s.local:6443
  authMode: serviceAccount
  serviceAccountSecretRef:
    name: prod-us-east-sa
    key: token
  allowEnforcement: false  # Read-only
---
apiVersion: v1
kind: Secret
metadata:
  name: prod-us-east-sa
  namespace: kspec-system
type: Opaque
stringData:
  token: "YOUR_SERVICE_ACCOUNT_TOKEN"
```

Commit → Push → Auto-deployed!

### 3. Define Compliance (GitOps)

Create `clusters/production/kspec/clusterspec.yaml`:

```yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: prod-baseline
  namespace: kspec-system
spec:
  # Scan all production clusters
  clusterRef:
    name: prod-us-east

  # Your compliance requirements
  kubernetesVersion: ">=1.28.0"
  podSecurityStandards:
    enforce:
      - restricted
  networkPolicies:
    required: true
  rbac:
    enforceRBAC: true
```

Commit → Push → Scans start automatically!

### 4. View Compliance (Browser)

Just open your dashboard URL - no commands needed!

## 🎯 Benefits of GitOps Approach

✅ **No Local Tools** - Just browser and git
✅ **Audit Trail** - All changes in git history
✅ **PR Reviews** - Approve compliance changes via PR
✅ **Automatic Sync** - ArgoCD/Flux keeps cluster in sync
✅ **Rollback** - `git revert` to undo changes
✅ **Multi-Env** - Same manifests, different overlays

## 📁 Recommended Repo Structure

```
infrastructure/
├── base/
│   └── kspec/
│       ├── kustomization.yaml
│       ├── namespace.yaml
│       ├── operator.yaml
│       └── dashboard.yaml
├── overlays/
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   ├── clusters.yaml
│   │   └── clusterspec.yaml
│   └── production/
│       ├── kustomization.yaml
│       ├── clusters.yaml
│       ├── clusterspec.yaml
│       └── ingress.yaml
└── apps/
    ├── kspec-staging.yaml    # ArgoCD App
    └── kspec-production.yaml # ArgoCD App
```

## 🔒 Secret Management

### Option 1: Sealed Secrets

```yaml
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: prod-cluster-credentials
  namespace: kspec-system
spec:
  encryptedData:
    token: AgBHx7Qw...  # Encrypted, safe to commit
```

### Option 2: External Secrets Operator

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: cluster-credentials
  namespace: kspec-system
spec:
  secretStoreRef:
    name: vault
    kind: ClusterSecretStore
  target:
    name: prod-cluster-sa
  data:
  - secretKey: token
    remoteRef:
      key: kspec/prod-cluster
      property: sa-token
```

### Option 3: SOPS

```yaml
# Encrypt with SOPS
sops --encrypt clusters.yaml > clusters.enc.yaml

# Flux auto-decrypts
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: kspec
spec:
  decryption:
    provider: sops
```

## 🚦 CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy kspec
on:
  push:
    branches: [main]
    paths:
      - 'infrastructure/kspec/**'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Validate manifests
      run: |
        kubectl kustomize infrastructure/overlays/production | \
        kubectl apply --dry-run=server -f -
```

## 📈 Monitoring Setup (GitOps)

### Prometheus ServiceMonitor

Already included in `config/dashboard/deployment.yaml`!

Just ensure Prometheus Operator is installed, and metrics are auto-scraped.

### Grafana Dashboard ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kspec-grafana-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  kspec-dashboard.json: |
    {
      "dashboard": {
        "title": "kspec Compliance",
        "panels": [...]
      }
    }
```

Commit → Grafana auto-imports!

## 🔄 Updates (GitOps)

```yaml
# Update image version in kustomization.yaml
images:
- name: kspec-dashboard:latest
  newTag: v0.3.0  # Update here

# Commit → Push → Auto-deployed
```

## 🎓 Example: Complete GitOps Setup

```bash
# 1. Fork the kspec repo
# 2. Add to ArgoCD:

kubectl apply -f - <<EOF
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kspec
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/YOUR-ORG/kspec
    targetRevision: main
    path: config/dashboard
  destination:
    server: https://kubernetes.default.svc
    namespace: kspec-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
EOF

# 3. Open browser to ArgoCD UI - done!
```

## 📞 Support

- Issues: https://github.com/cloudcwfranck/kspec/issues
- Docs: https://github.com/cloudcwfranck/kspec/tree/main/docs

---

**Bottom line:** Store manifests in git → ArgoCD/Flux deploys → Open browser → See compliance. No CLI needed! 🎉
