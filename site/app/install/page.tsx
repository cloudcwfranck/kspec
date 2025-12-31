import { Metadata } from 'next';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Install kspec',
  description: 'Installation instructions for kspec operator and CLI',
};

export default function InstallPage() {
  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <div className="border-b border-gray-200 py-16">
        <div className="max-w-4xl mx-auto px-6">
          <h1 className="text-5xl font-bold mb-4 text-gray-900">Install kspec</h1>
          <p className="text-xl text-gray-600">
            Get started with kspec in your Kubernetes cluster
          </p>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-4xl mx-auto px-6 py-12">
        {/* Prerequisites */}
        <section className="mb-16">
          <h2 className="text-3xl font-bold mb-6 text-gray-900">Prerequisites</h2>
          <div className="bg-gray-50 rounded-xl p-6 space-y-3 border border-gray-200">
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-primary-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <span className="text-gray-700">Kubernetes cluster (v1.24+)</span>
            </div>
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-primary-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <span className="text-gray-700">kubectl configured with cluster access</span>
            </div>
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-primary-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <span className="text-gray-700">cert-manager installed (for webhook certificates)</span>
            </div>
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-primary-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <span className="text-gray-700">Kyverno installed (policy engine)</span>
            </div>
          </div>
        </section>

        {/* Step 1: Install Dependencies */}
        <section className="mb-16">
          <h2 className="text-3xl font-bold mb-6 text-gray-900">Step 1: Install Dependencies</h2>

          <div className="space-y-8">
            {/* cert-manager */}
            <div>
              <h3 className="text-xl font-semibold mb-3 text-gray-900">Install cert-manager</h3>
              <div className="bg-gray-900 border border-gray-200 rounded-xl p-4 overflow-x-auto">
                <pre className="text-gray-900-secondary text-sm font-mono">
{`kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=120s`}
                </pre>
              </div>
            </div>

            {/* Kyverno */}
            <div>
              <h3 className="text-xl font-semibold mb-3 text-gray-900">Install Kyverno (via Helm)</h3>
              <div className="bg-amber-500/10 border border-amber-500/30 rounded-xl p-4 mb-3">
                <p className="text-sm text-amber-400">
                  <strong>Important:</strong> You must install Kyverno using Helm, not raw manifests.
                </p>
              </div>
              <div className="bg-gray-900 border border-gray-200 rounded-xl p-4 overflow-x-auto">
                <pre className="text-gray-900-secondary text-sm font-mono">
{`helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update

helm install kyverno kyverno/kyverno \\
  --namespace kyverno \\
  --create-namespace \\
  --wait \\
  --timeout=5m`}
                </pre>
              </div>
            </div>
          </div>
        </section>

        {/* Step 2: Install kspec */}
        <section className="mb-16">
          <h2 className="text-3xl font-bold mb-6 text-gray-900">Step 2: Install kspec Operator</h2>

          <div className="space-y-6">
            {/* Using manifests */}
            <div>
              <h3 className="text-xl font-semibold mb-3 text-gray-900">Option A: Using kubectl</h3>
              <div className="bg-gray-900 border border-gray-200 rounded-xl p-4 overflow-x-auto">
                <pre className="text-gray-900-secondary text-sm font-mono">
{`# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/cloudcwfranck/kspec/main/config/crd/bases/kspec.io_clusterspecifications.yaml
kubectl apply -f https://raw.githubusercontent.com/cloudcwfranck/kspec/main/config/crd/bases/kspec.io_clustertargets.yaml
kubectl apply -f https://raw.githubusercontent.com/cloudcwfranck/kspec/main/config/crd/bases/kspec.io_compliancereports.yaml
kubectl apply -f https://raw.githubusercontent.com/cloudcwfranck/kspec/main/config/crd/bases/kspec.io_driftreports.yaml

# Install operator
kubectl apply -k https://github.com/cloudcwfranck/kspec/config/default

# Verify installation
kubectl get pods -n kspec-system -l control-plane=controller-manager`}
                </pre>
              </div>
            </div>

            {/* Using kustomize */}
            <div>
              <h3 className="text-xl font-semibold mb-3 text-gray-900">Option B: Using kustomize</h3>
              <div className="bg-gray-900 border border-gray-200 rounded-xl p-4 overflow-x-auto">
                <pre className="text-gray-900-secondary text-sm font-mono">
{`# Clone the repository
git clone https://github.com/cloudcwfranck/kspec.git
cd kspec

# Build and apply
kubectl apply -k config/default

# Verify installation
kubectl get deployment -n kspec-system kspec-operator-controller-manager`}
                </pre>
              </div>
            </div>
          </div>
        </section>

        {/* Step 3: Create first spec */}
        <section className="mb-16">
          <h2 className="text-3xl font-bold mb-6 text-gray-900">Step 3: Create Your First ClusterSpecification</h2>
          <p className="text-gray-900-secondary mb-6">
            Create a ClusterTarget to reference your cluster, then define a ClusterSpecification with policies.
          </p>

          <div className="space-y-6">
            {/* ClusterTarget */}
            <div>
              <h3 className="text-lg font-semibold mb-3 text-gray-900">Create ClusterTarget</h3>
              <div className="bg-gray-900 border border-gray-200 rounded-xl p-4 overflow-x-auto">
                <pre className="text-gray-900-secondary text-sm font-mono">
{`cat <<EOF | kubectl apply -f -
apiVersion: kspec.io/v1alpha1
kind: ClusterTarget
metadata:
  name: production-cluster
  namespace: kspec-system
spec:
  inCluster: true
  platform: eks
  version: "1.28.0"
EOF`}
                </pre>
              </div>
            </div>

            {/* ClusterSpecification */}
            <div>
              <h3 className="text-lg font-semibold mb-3 text-gray-900">Create ClusterSpecification</h3>
              <div className="bg-gray-900 border border-gray-200 rounded-xl p-4 overflow-x-auto">
                <pre className="text-gray-900-secondary text-sm font-mono">
{`cat <<EOF | kubectl apply -f -
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: production-spec
  namespace: kspec-system
spec:
  targetClusterRef:
    name: production-cluster
  enforcementMode: monitor  # Start in monitor mode
  policies:
    - id: "pod-security-baseline"
      title: "Pod Security Standards - Baseline"
      description: "Enforce baseline pod security requirements"
      severity: high
      checks:
        - id: "require-run-as-non-root"
          title: "Require runAsNonRoot"
          kyvernoPolicy: |
            apiVersion: kyverno.io/v1
            kind: ClusterPolicy
            metadata:
              name: require-run-as-non-root
            spec:
              validationFailureAction: audit
              background: true
              rules:
              - name: check-runAsNonRoot
                match:
                  any:
                  - resources:
                      kinds:
                      - Pod
                validate:
                  message: "Containers must run as non-root user"
                  pattern:
                    spec:
                      securityContext:
                        runAsNonRoot: true
EOF`}
                </pre>
              </div>
            </div>
          </div>
        </section>

        {/* Step 4: Verify */}
        <section className="mb-16">
          <h2 className="text-3xl font-bold mb-6 text-gray-900">Step 4: Verify Installation</h2>
          <div className="bg-gray-900 border border-gray-200 rounded-xl p-4 overflow-x-auto">
            <pre className="text-gray-900-secondary text-sm font-mono">
{`# Check operator logs
kubectl logs -n kspec-system -l control-plane=controller-manager --tail=20

# Check that policies were created
kubectl get clusterpolicy

# Check compliance reports
kubectl get compliancereport -n kspec-system

# Switch to enforce mode when ready
kubectl patch clusterspecification production-spec -n kspec-system \\
  --type='json' \\
  -p='[{"op": "replace", "path": "/spec/enforcementMode", "value": "enforce"}]'`}
            </pre>
          </div>
        </section>

        {/* Next Steps */}
        <section className="bg-accent/10 rounded-xl p-8 border border-accent/20">
          <h2 className="text-2xl font-bold mb-4 text-gray-900">Next Steps</h2>
          <ul className="space-y-3">
            <li className="flex items-start gap-3">
              <svg className="w-5 h-5 text-primary-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <Link href="/docs/getting-started" className="font-medium text-primary-600 hover:text-primary-600-hover">
                  Getting Started Guide →
                </Link>
                <p className="text-sm text-gray-900-secondary mt-1">Learn the basics of kspec</p>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <svg className="w-5 h-5 text-primary-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <Link href="/docs/guides" className="font-medium text-primary-600 hover:text-primary-600-hover">
                  Policy Guide →
                </Link>
                <p className="text-sm text-gray-900-secondary mt-1">Write effective security policies</p>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <svg className="w-5 h-5 text-primary-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <a
                  href="https://github.com/cloudcwfranck/kspec/tree/main/examples"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-medium text-primary-600 hover:text-primary-600-hover"
                >
                  Example Policies →
                </a>
                <p className="text-sm text-gray-900-secondary mt-1">Browse pre-built policy examples</p>
              </div>
            </li>
          </ul>
        </section>

        {/* Support */}
        <div className="mt-12 text-center text-sm text-gray-900-muted">
          <p>
            Need help?{' '}
            <a
              href="https://github.com/cloudcwfranck/kspec/issues"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary-600 hover:text-primary-600-hover"
            >
              Open an issue on GitHub
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}
