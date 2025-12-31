import { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Changelog',
  description: 'Latest releases and updates for kspec',
};

// Revalidate every 5 minutes (300 seconds)
export const revalidate = 300;

interface GitHubRelease {
  id: number;
  name: string;
  tag_name: string;
  published_at: string;
  body: string;
  html_url: string;
  prerelease: boolean;
  draft: boolean;
}

async function getReleases(): Promise<GitHubRelease[]> {
  const owner = process.env.GITHUB_OWNER || 'cloudcwfranck';
  const repo = process.env.GITHUB_REPO || 'kspec';
  const token = process.env.GITHUB_TOKEN;

  const headers: HeadersInit = {
    'Accept': 'application/vnd.github+json',
    'X-GitHub-Api-Version': '2022-11-28',
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const response = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/releases`,
      {
        headers,
        next: { revalidate: 300 }, // ISR: revalidate every 5 minutes
      }
    );

    if (!response.ok) {
      console.error(`GitHub API error: ${response.status} ${response.statusText}`);
      return [];
    }

    const releases: GitHubRelease[] = await response.json();
    // Filter out drafts and prereleases
    return releases.filter(r => !r.draft && !r.prerelease);
  } catch (error) {
    console.error('Failed to fetch releases:', error);
    return [];
  }
}

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

function parseReleaseBody(body: string): string {
  // Convert GitHub-flavored markdown links and basic formatting
  return body
    .replace(/\*\*/g, '') // Remove bold markers for simplicity
    .replace(/```(\w+)?\n([\s\S]*?)```/g, (_, lang, code) => {
      return `<pre class="bg-gray-900 text-gray-100 rounded-lg p-4 overflow-x-auto my-4"><code>${code.trim()}</code></pre>`;
    })
    .replace(/`([^`]+)`/g, '<code class="bg-gray-100 text-primary-700 px-2 py-1 rounded text-sm">$1</code>')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-primary-600 hover:text-primary-700 underline" target="_blank" rel="noopener noreferrer">$1</a>');
}

export default async function ChangelogPage() {
  const releases = await getReleases();

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <div className="bg-gradient-to-b from-gray-50 to-white border-b border-gray-200 py-16">
        <div className="max-w-4xl mx-auto px-6">
          <h1 className="text-5xl font-bold mb-4">Changelog</h1>
          <p className="text-xl text-gray-600">
            Latest releases and updates for kspec. Automatically updated from GitHub Releases.
          </p>
        </div>
      </div>

      {/* Releases */}
      <div className="max-w-4xl mx-auto px-6 py-12">
        {releases.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-500">No releases found. Check back later.</p>
          </div>
        ) : (
          <div className="space-y-12">
            {releases.map((release) => (
              <article key={release.id} className="border-b border-gray-200 pb-12 last:border-0">
                {/* Release header */}
                <div className="flex items-baseline gap-4 mb-4">
                  <h2 className="text-3xl font-bold">
                    <a
                      href={release.html_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-gray-900 hover:text-primary-600 transition-colors"
                    >
                      {release.name || release.tag_name}
                    </a>
                  </h2>
                  <span className="text-sm text-gray-500 font-medium">
                    {release.tag_name}
                  </span>
                </div>

                <div className="text-sm text-gray-500 mb-6">
                  Released on {formatDate(release.published_at)}
                </div>

                {/* Release body */}
                <div
                  className="prose prose-gray max-w-none"
                  dangerouslySetInnerHTML={{ __html: parseReleaseBody(release.body) }}
                />

                {/* View on GitHub link */}
                <div className="mt-6">
                  <a
                    href={release.html_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary-600 hover:text-primary-700 text-sm font-medium inline-flex items-center gap-1"
                  >
                    View on GitHub
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                    </svg>
                  </a>
                </div>
              </article>
            ))}
          </div>
        )}

        {/* Footer note */}
        <div className="mt-12 text-center text-sm text-gray-500">
          <p>This page automatically updates every 5 minutes from GitHub Releases.</p>
          <p className="mt-2">
            <a
              href="https://github.com/cloudcwfranck/kspec/releases"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary-600 hover:text-primary-700"
            >
              View all releases on GitHub →
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}
