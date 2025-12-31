import Link from 'next/link';

export default function HomePage() {
  return (
    <>
      {/* Hero Section */}
      <section className="relative overflow-hidden pt-32 pb-20">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-br from-primary-50 via-white to-gray-50" />
          <div className="absolute top-0 right-0 w-96 h-96 bg-primary-100 rounded-full blur-3xl opacity-30" />
          <div className="absolute bottom-0 left-0 w-96 h-96 bg-primary-200 rounded-full blur-3xl opacity-20" />
        </div>

        <div className="max-w-6xl mx-auto px-6">
          <div className="text-center max-w-4xl mx-auto">
            <h1 className="text-6xl md:text-7xl font-bold tracking-tight mb-6 text-balance">
              Policy enforcement{' '}
              <span className="text-primary-600">for Kubernetes</span>
            </h1>
            <p className="text-xl md:text-2xl text-gray-600 mb-12 text-balance leading-relaxed">
              Define security, compliance, and operational policies as code.
              Automated enforcement across all your clusters.
            </p>
            <div className="flex gap-4 justify-center flex-wrap">
              <Link href="/docs" className="btn-primary">
                Read documentation
              </Link>
              <Link href="/install" className="btn-secondary">
                Install kspec
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Value Props */}
      <section className="py-20 bg-white">
        <div className="max-w-6xl mx-auto px-6">
          <div className="grid md:grid-cols-3 gap-12">
            <div className="space-y-4">
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center">
                <svg className="w-6 h-6 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold">Security by default</h3>
              <p className="text-gray-600 leading-relaxed">
                Enforce pod security standards, image policies, network segmentation, and RBAC rules automatically across your fleet.
              </p>
            </div>

            <div className="space-y-4">
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center">
                <svg className="w-6 h-6 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold">Compliance guardrails</h3>
              <p className="text-gray-600 leading-relaxed">
                Meet regulatory requirements with declarative policies. Track compliance status in real-time with automated reports.
              </p>
            </div>

            <div className="space-y-4">
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center">
                <svg className="w-6 h-6 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
                </svg>
              </div>
              <h3 className="text-xl font-semibold">Multi-cluster</h3>
              <p className="text-gray-600 leading-relaxed">
                Manage policies across development, staging, and production. Single source of truth for your entire fleet.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section className="py-20 bg-gradient-to-b from-gray-50 to-white">
        <div className="max-w-6xl mx-auto px-6">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">How it works</h2>
            <p className="text-xl text-gray-600">Three simple steps to secure your clusters</p>
          </div>

          <div className="grid md:grid-cols-3 gap-8">
            <div className="relative">
              <div className="bg-white rounded-xl p-8 border border-gray-200 shadow-sm h-full">
                <div className="text-primary-600 font-bold text-sm mb-4">STEP 1</div>
                <h3 className="text-xl font-semibold mb-3">Define policies</h3>
                <p className="text-gray-600 mb-4">
                  Create ClusterSpecification resources with your security, compliance, and operational requirements.
                </p>
                <div className="bg-gray-900 rounded-lg p-4 font-mono text-xs text-gray-300 overflow-x-auto">
                  <pre className="whitespace-pre">{`apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
metadata:
  name: production-spec
spec:
  enforcementMode: enforce
  policies:
    - id: pod-security
      severity: high`}</pre>
                </div>
              </div>
              {/* Connector line */}
              <div className="hidden md:block absolute top-1/2 -right-4 w-8 h-0.5 bg-gray-200" />
            </div>

            <div className="relative">
              <div className="bg-white rounded-xl p-8 border border-gray-200 shadow-sm h-full">
                <div className="text-primary-600 font-bold text-sm mb-4">STEP 2</div>
                <h3 className="text-xl font-semibold mb-3">Controller enforces</h3>
                <p className="text-gray-600 mb-4">
                  kspec controller translates your specs into Kyverno policies and admission webhooks.
                </p>
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-sm">
                    <div className="w-2 h-2 bg-emerald-500 rounded-full" />
                    <span className="text-gray-700">Policies created</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <div className="w-2 h-2 bg-emerald-500 rounded-full" />
                    <span className="text-gray-700">Webhooks configured</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <div className="w-2 h-2 bg-emerald-500 rounded-full" />
                    <span className="text-gray-700">Real-time enforcement</span>
                  </div>
                </div>
              </div>
              <div className="hidden md:block absolute top-1/2 -right-4 w-8 h-0.5 bg-gray-200" />
            </div>

            <div className="bg-white rounded-xl p-8 border border-gray-200 shadow-sm">
              <div className="text-primary-600 font-bold text-sm mb-4">STEP 3</div>
              <h3 className="text-xl font-semibold mb-3">Monitor compliance</h3>
              <p className="text-gray-600 mb-4">
                View compliance reports, drift detection, and audit logs in real-time.
              </p>
              <div className="bg-gradient-to-br from-primary-50 to-white rounded-lg p-4 border border-primary-100">
                <div className="text-xs font-semibold text-gray-500 uppercase mb-2">Compliance Score</div>
                <div className="text-3xl font-bold text-primary-600 mb-2">98.5%</div>
                <div className="text-xs text-gray-600">3 clusters monitored</div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 bg-white">
        <div className="max-w-4xl mx-auto px-6 text-center">
          <h2 className="text-4xl font-bold mb-6">Ready to secure your clusters?</h2>
          <p className="text-xl text-gray-600 mb-8">
            Get started with kspec in minutes. Free and open source.
          </p>
          <div className="flex gap-4 justify-center flex-wrap">
            <Link href="/docs" className="btn-primary">
              View documentation
            </Link>
            <Link href="/changelog" className="btn-secondary">
              See what's new
            </Link>
          </div>
        </div>
      </section>
    </>
  );
}
