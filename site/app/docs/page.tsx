import { Metadata } from 'next';
import Link from 'next/link';
import { getDocSidebar } from '@/lib/docs';

export const metadata: Metadata = {
  title: 'Documentation',
  description: 'Complete documentation for kspec policy enforcement platform',
};

export default function DocsIndexPage() {
  const sidebar = getDocSidebar();

  return (
    <div className="min-h-screen bg-white">
      <div className="max-w-7xl mx-auto px-6 py-16">
        <div className="max-w-4xl">
          <h1 className="text-5xl font-bold mb-6">Documentation</h1>
          <p className="text-xl text-gray-600 mb-12">
            Everything you need to know about deploying and using kspec
          </p>

          {/* Quick Links */}
          <div className="grid md:grid-cols-3 gap-6 mb-16">
            <Link
              href="/docs/getting-started"
              className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-shadow"
            >
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h3 className="font-semibold text-lg mb-2">Getting Started</h3>
              <p className="text-sm text-gray-600">
                Quick start guide to install and configure kspec in your cluster
              </p>
            </Link>

            <Link
              href="/docs/guides"
              className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-shadow"
            >
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                </svg>
              </div>
              <h3 className="font-semibold text-lg mb-2">Guides</h3>
              <p className="text-sm text-gray-600">
                In-depth guides for writing policies and managing compliance
              </p>
            </Link>

            <Link
              href="/docs/api-reference"
              className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-shadow"
            >
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                </svg>
              </div>
              <h3 className="font-semibold text-lg mb-2">API Reference</h3>
              <p className="text-sm text-gray-600">
                Complete API documentation for all Custom Resource Definitions
              </p>
            </Link>
          </div>

          {/* All Docs */}
          <h2 className="text-3xl font-bold mb-6">All Documentation</h2>
          <div className="space-y-8">
            {sidebar.map((section) => (
              <div key={section.title}>
                <h3 className="text-xl font-semibold mb-4 text-gray-900">{section.title}</h3>
                <ul className="space-y-2">
                  {section.pages.map((page) => (
                    <li key={page.slug}>
                      <Link
                        href={`/docs/${page.slug}`}
                        className="text-primary-600 hover:text-primary-700 hover:underline"
                      >
                        {page.title}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
