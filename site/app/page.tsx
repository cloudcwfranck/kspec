import Link from 'next/link';

export default function HomePage() {
  return (
    <>
      {/* Hero Section - Vercel style: Clean, massive typography */}
      <section className="relative pt-40 pb-32 bg-white">
        <div className="max-w-6xl mx-auto px-6">
          <div className="text-center max-w-5xl mx-auto">
            <h1 className="text-7xl md:text-8xl lg:text-9xl font-bold tracking-tight mb-8 text-black leading-[1.05]">
              Policy enforcement for Kubernetes
            </h1>
            <p className="text-xl md:text-2xl text-vercel-text-secondary mb-12 max-w-3xl mx-auto leading-relaxed">
              Define security, compliance, and operational policies as code.
              Automated enforcement across all your clusters.
            </p>
            <div className="flex gap-4 justify-center flex-wrap">
              <Link
                href="/docs"
                className="btn-primary"
              >
                Read documentation
              </Link>
              <Link
                href="/install"
                className="px-6 py-3 text-vercel-text-secondary hover:text-vercel-text transition-colors"
              >
                Install kspec →
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Value Props - Vercel minimal style */}
      <section className="py-32 bg-vercel-bg-subtle border-t border-vercel-border">
        <div className="max-w-5xl mx-auto px-6">
          <div className="grid md:grid-cols-3 gap-16">
            <div className="space-y-3">
              <h3 className="text-lg font-semibold text-vercel-text">Security by default</h3>
              <p className="text-vercel-text-secondary leading-relaxed">
                Enforce pod security standards, image policies, network segmentation, and RBAC rules automatically.
              </p>
            </div>

            <div className="space-y-3">
              <h3 className="text-lg font-semibold text-vercel-text">Compliance guardrails</h3>
              <p className="text-vercel-text-secondary leading-relaxed">
                Meet regulatory requirements with declarative policies. Track compliance status in real-time.
              </p>
            </div>

            <div className="space-y-3">
              <h3 className="text-lg font-semibold text-vercel-text">Multi-cluster</h3>
              <p className="text-vercel-text-secondary leading-relaxed">
                Manage policies across development, staging, and production. Single source of truth.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works - Simplified */}
      <section className="py-32 bg-white border-t border-vercel-border">
        <div className="max-w-5xl mx-auto px-6">
          <h2 className="text-3xl font-bold mb-12 text-vercel-text">How it works</h2>

          <div className="space-y-12">
            <div className="space-y-3">
              <div className="text-sm font-medium text-vercel-text-muted">01</div>
              <h3 className="text-xl font-semibold text-vercel-text">Define policies as code</h3>
              <p className="text-vercel-text-secondary leading-relaxed max-w-2xl">
                Create ClusterSpecification resources describing your security, compliance, and operational requirements.
              </p>
            </div>

            <div className="space-y-3">
              <div className="text-sm font-medium text-vercel-text-muted">02</div>
              <h3 className="text-xl font-semibold text-vercel-text">Automatic enforcement</h3>
              <p className="text-vercel-text-secondary leading-relaxed max-w-2xl">
                The kspec controller translates specifications into Kyverno policies and configures admission webhooks.
              </p>
            </div>

            <div className="space-y-3">
              <div className="text-sm font-medium text-vercel-text-muted">03</div>
              <h3 className="text-xl font-semibold text-vercel-text">Monitor compliance</h3>
              <p className="text-vercel-text-secondary leading-relaxed max-w-2xl">
                View real-time compliance reports, drift detection, and audit logs across all clusters.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section - Minimal */}
      <section className="py-32 bg-vercel-bg-subtle border-t border-vercel-border">
        <div className="max-w-5xl mx-auto px-6 text-center">
          <h2 className="text-4xl md:text-5xl font-bold mb-6 text-vercel-text">
            Start enforcing policies today
          </h2>
          <p className="text-xl text-vercel-text-secondary mb-10 max-w-2xl mx-auto">
            Free and open source. Deploy in minutes.
          </p>
          <div className="flex gap-4 justify-center">
            <Link href="/docs" className="btn-primary">
              Get started
            </Link>
            <Link href="/install" className="btn-secondary">
              View install guide →
            </Link>
          </div>
        </div>
      </section>
    </>
  );
}
