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
          <h3 className="font-semibold text-sm uppercase text-[#707070] mb-3">
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
                        ? 'bg-primary-600/10 text-primary-500 font-medium border border-primary-600/20'
                        : 'text-[#a0a0a0] hover:text-white hover:bg-[#1a1a1a]'
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
