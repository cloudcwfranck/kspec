'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

export default function Navigation() {
  const pathname = usePathname();

  const isActive = (path: string) => {
    if (path === '/') return pathname === '/';
    return pathname.startsWith(path);
  };

  return (
    <nav className="sticky top-0 z-50 bg-[#0a0a0a]/80 backdrop-blur-xl border-b border-[#2a2a2a]">
      <div className="max-w-7xl mx-auto px-6">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link href="/" className="flex items-center gap-2 group">
            <div className="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center group-hover:bg-primary-500 transition-colors">
              <span className="text-white font-bold text-lg">k</span>
            </div>
            <span className="font-semibold text-lg text-white">kspec</span>
          </Link>

          {/* Navigation Links */}
          <div className="hidden md:flex items-center gap-6">
            <Link
              href="/docs"
              className={`text-sm font-medium transition-colors ${
                isActive('/docs')
                  ? 'text-white'
                  : 'text-[#a0a0a0] hover:text-white'
              }`}
            >
              Docs
            </Link>
            <Link
              href="/install"
              className={`text-sm font-medium transition-colors ${
                isActive('/install')
                  ? 'text-white'
                  : 'text-[#a0a0a0] hover:text-white'
              }`}
            >
              Install
            </Link>
            <Link
              href="/changelog"
              className={`text-sm font-medium transition-colors ${
                isActive('/changelog')
                  ? 'text-white'
                  : 'text-[#a0a0a0] hover:text-white'
              }`}
            >
              Changelog
            </Link>
            <Link
              href="/status"
              className={`text-sm font-medium transition-colors ${
                isActive('/status')
                  ? 'text-white'
                  : 'text-[#a0a0a0] hover:text-white'
              }`}
            >
              Status
            </Link>
            <a
              href="https://github.com/cloudcwfranck/kspec"
              target="_blank"
              rel="noopener noreferrer"
              className="text-[#a0a0a0] hover:text-white transition-colors"
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
              </svg>
            </a>
          </div>
        </div>
      </div>
    </nav>
  );
}
