import { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { MDXRemote } from 'next-mdx-remote/rsc';
import { getAllDocs, getDocBySlug, getDocSidebar } from '@/lib/docs';
import DocSidebar from '@/components/DocSidebar';

interface PageProps {
  params: {
    slug: string[];
  };
}

export async function generateStaticParams() {
  const docs = getAllDocs();
  return docs.map((doc) => ({
    slug: doc.slug.split('/'),
  }));
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const slug = params.slug.join('/');
  const doc = getDocBySlug(slug);

  if (!doc) {
    return {
      title: 'Not Found',
    };
  }

  return {
    title: doc.frontmatter.title,
    description: doc.frontmatter.description || `Documentation for ${doc.frontmatter.title}`,
  };
}

export default function DocPage({ params }: PageProps) {
  const slug = params.slug.join('/');
  const doc = getDocBySlug(slug);

  if (!doc) {
    notFound();
  }

  const sidebar = getDocSidebar();

  return (
    <div className="min-h-screen bg-linear-bg">
      <div className="max-w-7xl mx-auto px-6 py-8">
        <div className="flex gap-12">
          {/* Sidebar */}
          <aside className="hidden lg:block w-64 flex-shrink-0">
            <DocSidebar sections={sidebar} currentSlug={slug} />
          </aside>

          {/* Main Content */}
          <main className="flex-1 max-w-4xl">
            <article className="prose prose-lg max-w-none prose-invert">
              <h1 className="text-5xl font-bold mb-4 text-linear-text">{doc.frontmatter.title}</h1>
              {doc.frontmatter.description && (
                <p className="text-xl text-linear-text-secondary mb-8">{doc.frontmatter.description}</p>
              )}

              <MDXRemote source={doc.content} />
            </article>

            {/* Footer Navigation */}
            <div className="mt-16 pt-8 border-t border-linear-border">
              <div className="text-sm text-linear-text-muted">
                <p>
                  Found an issue?{' '}
                  <a
                    href={`https://github.com/cloudcwfranck/kspec/edit/main/site/content/docs/${slug}.mdx`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-accent hover:text-accent-hover"
                  >
                    Edit this page on GitHub
                  </a>
                </p>
              </div>
            </div>
          </main>
        </div>
      </div>
    </div>
  );
}
