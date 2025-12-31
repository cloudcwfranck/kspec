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
          <h3 className="font-semibold text-sm uppercase text-vercel-text-muted mb-3">
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
                        ? 'bg-accent/10 text-accent font-medium border border-accent/20'
                        : 'text-vercel-text-secondary hover:text-vercel-text hover:bg-vercel-bg-subtle'
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
