#!/bin/bash
# Quick Start Script for kspec with Dashboard
# This script installs kspec operator with monitoring in one command

set -e

NAMESPACE="${KSPEC_NAMESPACE:-kspec-system}"
ENABLE_GRAFANA="${ENABLE_GRAFANA:-false}"

echo "🚀 kspec Quick Start Installer"
echo "================================"
echo ""
echo "This will install:"
echo "  ✓ kspec Operator with CRDs"
echo "  ✓ Prometheus metrics (via ServiceMonitor)"
if [ "$ENABLE_GRAFANA" = "true" ]; then
echo "  ✓ Grafana with pre-configured dashboards"
fi
echo ""
echo "Namespace: $NAMESPACE"
echo ""

# Step 1: Create namespace
echo "📦 Creating namespace..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Step 2: Install CRDs
echo "📋 Installing CRDs..."
kubectl apply -f config/crd/

# Step 3: Deploy operator
echo "🎯 Deploying kspec operator..."
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kspec-controller-manager
  namespace: $NAMESPACE
  labels:
    control-plane: controller-manager
    app: kspec-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
        app: kspec-operator
    spec:
      serviceAccountName: kspec-controller-manager
      containers:
      - name: manager
        image: kspec-operator:latest  # Replace with your image
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
          name: metrics
          protocol: TCP
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: kspec-operator-metrics
  namespace: $NAMESPACE
  labels:
    app: kspec-operator
spec:
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
  selector:
    control-plane: controller-manager
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kspec-controller-manager
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kspec-operator
rules:
- apiGroups: ["kspec.io"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: [""]
  resources: ["namespaces", "pods", "serviceaccounts", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["rbac.authorization.k8s.io"]
  resources: ["clusterroles", "clusterrolebindings", "roles", "rolebindings"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kspec-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kspec-operator
subjects:
- kind: ServiceAccount
  name: kspec-controller-manager
  namespace: $NAMESPACE
EOF

# Step 4: Install Prometheus ServiceMonitor (if Prometheus Operator is present)
if kubectl get crd servicemonitors.monitoring.coreos.com >/dev/null 2>&1; then
  echo "📊 Creating Prometheus ServiceMonitor..."
  kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kspec-operator
  namespace: $NAMESPACE
  labels:
    app: kspec-operator
spec:
  selector:
    matchLabels:
      app: kspec-operator
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
EOF
else
  echo "⚠️  Prometheus Operator not found - skipping ServiceMonitor"
  echo "   Metrics will still be available at :8080/metrics"
fi

# Step 5: Install Grafana (if requested)
if [ "$ENABLE_GRAFANA" = "true" ]; then
  echo "📈 Installing Grafana..."

  # Check if Helm is available
  if command -v helm &> /dev/null; then
    helm repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true
    helm repo update

    # Install with pre-configured dashboard
    helm upgrade --install grafana grafana/grafana \
      --namespace $NAMESPACE \
      --set persistence.enabled=false \
      --set adminPassword=admin123 \
      --wait

    echo ""
    echo "✅ Grafana installed!"
    echo "   Port forward: kubectl port-forward -n $NAMESPACE svc/grafana 3000:80"
    echo "   Access at: http://localhost:3000 (admin/admin123)"
  else
    echo "⚠️  Helm not found - skipping Grafana installation"
  fi
fi

# Step 6: Wait for operator to be ready
echo ""
echo "⏳ Waiting for operator to be ready..."
kubectl wait --for=condition=available --timeout=60s \
  deployment/kspec-controller-manager -n "$NAMESPACE" || true

# Step 7: Print success message
echo ""
echo "✅ kspec installation complete!"
echo ""
echo "🎯 Quick Start Commands:"
echo ""
echo "  # View live dashboard"
echo "  kspec dashboard --watch"
echo ""
echo "  # Create your first ClusterSpec"
echo "  kubectl apply -f config/samples/"
echo ""
echo "  # View compliance reports"
echo "  kubectl get compliancereports -A"
echo ""
echo "  # Discover and add clusters"
echo "  kspec cluster discover"
echo "  kspec cluster add <context-name> | kubectl apply -f -"
echo ""
echo "  # View Prometheus metrics"
echo "  kubectl port-forward -n $NAMESPACE svc/kspec-operator-metrics 8080:8080"
echo "  curl http://localhost:8080/metrics | grep kspec_"
echo ""

if [ "$ENABLE_GRAFANA" = "true" ]; then
echo "  # Access Grafana Dashboard"
echo "  kubectl port-forward -n $NAMESPACE svc/grafana 3000:80"
echo "  # Then open http://localhost:3000 (admin/admin123)"
echo ""
fi

echo "📚 Documentation: https://github.com/cloudcwfranck/kspec"
echo ""
