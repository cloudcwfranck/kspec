import Link from 'next/link';

export default function Footer() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="bg-[#0a0a0a] border-t border-[#2a2a2a]">
      <div className="max-w-7xl mx-auto px-6 py-12">
        <div className="grid md:grid-cols-4 gap-8 mb-8">
          {/* Brand */}
          <div>
            <div className="flex items-center gap-2 mb-4">
              <div className="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-lg">k</span>
              </div>
              <span className="font-bold text-xl text-white">kspec</span>
            </div>
            <p className="text-sm text-[#a0a0a0]">
              Policy enforcement for Kubernetes clusters
            </p>
          </div>

          {/* Product */}
          <div>
            <h3 className="font-semibold mb-4 text-white">Product</h3>
            <ul className="space-y-2 text-sm">
              <li>
                <Link href="/docs" className="text-[#a0a0a0] hover:text-white transition-colors">
                  Documentation
                </Link>
              </li>
              <li>
                <Link href="/install" className="text-[#a0a0a0] hover:text-white transition-colors">
                  Install
                </Link>
              </li>
              <li>
                <Link href="/changelog" className="text-[#a0a0a0] hover:text-white transition-colors">
                  Changelog
                </Link>
              </li>
              <li>
                <Link href="/status" className="text-[#a0a0a0] hover:text-white transition-colors">
                  Status
                </Link>
              </li>
            </ul>
          </div>

          {/* Resources */}
          <div>
            <h3 className="font-semibold mb-4 text-white">Resources</h3>
            <ul className="space-y-2 text-sm">
              <li>
                <Link href="/docs/getting-started" className="text-[#a0a0a0] hover:text-white transition-colors">
                  Getting Started
                </Link>
              </li>
              <li>
                <Link href="/docs/guides" className="text-[#a0a0a0] hover:text-white transition-colors">
                  Guides
                </Link>
              </li>
              <li>
                <Link href="/docs/api-reference" className="text-[#a0a0a0] hover:text-white transition-colors">
                  API Reference
                </Link>
              </li>
              <li>
                <a
                  href="https://github.com/cloudcwfranck/kspec/tree/main/examples"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[#a0a0a0] hover:text-white transition-colors"
                >
                  Examples
                </a>
              </li>
            </ul>
          </div>

          {/* Community */}
          <div>
            <h3 className="font-semibold mb-4 text-white">Community</h3>
            <ul className="space-y-2 text-sm">
              <li>
                <a
                  href="https://github.com/cloudcwfranck/kspec"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[#a0a0a0] hover:text-white transition-colors"
                >
                  GitHub
                </a>
              </li>
              <li>
                <a
                  href="https://github.com/cloudcwfranck/kspec/issues"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[#a0a0a0] hover:text-white transition-colors"
                >
                  Issues
                </a>
              </li>
              <li>
                <a
                  href="https://github.com/cloudcwfranck/kspec/discussions"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[#a0a0a0] hover:text-white transition-colors"
                >
                  Discussions
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom */}
        <div className="pt-8 border-t border-[#2a2a2a] flex flex-col md:flex-row justify-between items-center gap-4">
          <p className="text-sm text-[#a0a0a0]">
            © {currentYear} kspec. Open source under Apache 2.0 License.
          </p>
          <div className="flex gap-6 text-sm text-[#a0a0a0]">
            <a
              href="https://github.com/cloudcwfranck/kspec/blob/main/LICENSE"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-white transition-colors"
            >
              License
            </a>
            <a
              href="https://github.com/cloudcwfranck/kspec"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-white transition-colors"
            >
              GitHub
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
