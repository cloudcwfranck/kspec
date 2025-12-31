'use client';

import Link from 'next/link';
import { DocSection } from '@/lib/docs';

interface DocSidebarProps {
  sections: DocSection[];
  currentSlug: string;
}

export default function DocSidebar({ sections, currentSlug }: DocSidebarProps) {
  return (
    <nav className="sticky top-24 space-y-8">
      {sections.map((section) => (
        <div key={section.title}>
          <h3 className="font-semibold text-sm uppercase text-gray-500 mb-3">
            {section.title}
          </h3>
          <ul className="space-y-2">
            {section.pages.map((page) => {
              const isActive = currentSlug === page.slug;
              return (
                <li key={page.slug}>
                  <Link
                    href={`/docs/${page.slug}`}
                    className={`block text-sm py-1 px-3 rounded-md transition-colors ${
                      isActive
                        ? 'bg-primary-50 text-primary-700 font-medium'
                        : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
                    }`}
                  >
                    {page.title}
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}
