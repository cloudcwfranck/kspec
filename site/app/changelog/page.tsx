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
        next: { revalidate: 300 },
      }
    );

    if (!response.ok) {
      console.error(`GitHub API error: ${response.status} ${response.statusText}`);
      return [];
    }

    const releases: GitHubRelease[] = await response.json();
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
    month: 'short',
    day: 'numeric',
  });
}

function parseReleaseBody(body: string): React.ReactNode[] {
  const lines = body.split('\n');
  const elements: React.ReactNode[] = [];

  lines.forEach((line, index) => {
    // Skip empty lines
    if (!line.trim()) {
      elements.push(<br key={`br-${index}`} />);
      return;
    }

    // Headers
    if (line.startsWith('## ')) {
      elements.push(
        <h3 key={index} className="text-lg font-semibold mt-6 mb-3 text-gray-900">
          {line.replace('## ', '')}
        </h3>
      );
    } else if (line.startsWith('### ')) {
      elements.push(
        <h4 key={index} className="text-base font-semibold mt-4 mb-2 text-gray-900">
          {line.replace('### ', '')}
        </h4>
      );
    } else if (line.startsWith('- ')) {
      // Bullet points
      const content = line.replace('- ', '');
      elements.push(
        <div key={index} className="flex gap-2 mb-1">
          <span className="text-gray-400 mt-1">•</span>
          <span className="text-gray-700">{content}</span>
        </div>
      );
    } else if (line.startsWith('**') && line.endsWith('**')) {
      // Bold standalone lines
      elements.push(
        <p key={index} className="font-medium text-gray-900 mt-3 mb-2">
          {line.replace(/\*\*/g, '')}
        </p>
      );
    } else {
      // Regular paragraphs
      elements.push(
        <p key={index} className="text-gray-700 mb-2">
          {line}
        </p>
      );
    }
  });

  return elements;
}

export default async function ChangelogPage() {
  const releases = await getReleases();

  return (
    <div className="min-h-screen bg-white">
      {/* Header - Linear style */}
      <div className="border-b border-gray-200">
        <div className="max-w-3xl mx-auto px-6 py-16">
          <h1 className="text-4xl font-bold mb-2 text-gray-900">Changelog</h1>
          <p className="text-base text-gray-600">
            New features, improvements, and fixes
          </p>
        </div>
      </div>

      {/* Releases - Linear style cards */}
      <div className="max-w-3xl mx-auto px-6 py-8">
        {releases.length === 0 ? (
          <div className="text-center py-20">
            <p className="text-gray-500 text-sm">No releases yet</p>
          </div>
        ) : (
          <div className="space-y-8">
            {releases.map((release, idx) => (
              <article
                key={release.id}
                className="bg-white border border-gray-200 rounded-xl p-8 hover:border-gray-300 transition-colors"
              >
                {/* Release header - minimal */}
                <div className="flex items-center justify-between mb-6">
                  <div>
                    <div className="flex items-baseline gap-3">
                      <h2 className="text-2xl font-bold text-gray-900">
                        {release.tag_name}
                      </h2>
                      <time className="text-sm text-gray-500">
                        {formatDate(release.published_at)}
                      </time>
                    </div>
                    {release.name && release.name !== release.tag_name && (
                      <p className="text-sm text-gray-600 mt-1">{release.name}</p>
                    )}
                  </div>

                  <a
                    href={release.html_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-gray-400 hover:text-gray-600 transition-colors"
                    aria-label="View on GitHub"
                  >
                    <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                      <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
                    </svg>
                  </a>
                </div>

                {/* Release body - clean typography */}
                <div className="prose-sm">
                  {parseReleaseBody(release.body)}
                </div>
              </article>
            ))}
          </div>
        )}

        {/* Footer note - subtle */}
        <div className="mt-16 pt-8 border-t border-gray-200 text-center">
          <p className="text-xs text-gray-500">
            Updates automatically from{' '}
            <a
              href="https://github.com/cloudcwfranck/kspec/releases"
              target="_blank"
              rel="noopener noreferrer"
              className="text-gray-700 hover:text-gray-900"
            >
              GitHub Releases
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}
