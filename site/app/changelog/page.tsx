import { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Changelog',
  description: 'New features, improvements, and fixes',
};

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
      console.error(`GitHub API error: ${response.status}`);
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
  const now = new Date();
  const diffTime = Math.abs(now.getTime() - date.getTime());
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays === 0) return 'Today';
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) return `${diffDays} days ago`;

  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

export default async function ChangelogPage() {
  const releases = await getReleases();

  return (
    <div className="min-h-screen bg-[#0a0a0a]">
      {/* Header */}
      <div className="border-b border-[#2a2a2a]">
        <div className="max-w-5xl mx-auto px-6 py-20">
          <h1 className="text-5xl font-bold mb-3 text-white tracking-tight">Changelog</h1>
          <p className="text-lg text-[#a0a0a0]">
            New features, improvements, and fixes
          </p>
        </div>
      </div>

      {/* Releases */}
      <div className="max-w-5xl mx-auto px-6 py-12">
        {releases.length === 0 ? (
          <div className="text-center py-20">
            <p className="text-[#707070]">No releases yet</p>
          </div>
        ) : (
          <div className="space-y-16">
            {releases.map((release) => {
              const lines = release.body.split('\n').filter(l => l.trim());

              return (
                <article key={release.id} className="relative">
                  {/* Date badge - Linear style */}
                  <div className="flex items-center gap-3 mb-8">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-primary-500" />
                      <time className="text-sm text-[#a0a0a0] font-medium">
                        {formatDate(release.published_at)}
                      </time>
                    </div>
                  </div>

                  {/* Title */}
                  <h2 className="text-3xl font-bold text-white mb-8 tracking-tight">
                    {release.name || release.tag_name}
                  </h2>

                  {/* Content */}
                  <div className="space-y-6">
                    {lines.map((line, idx) => {
                      // Section headers
                      if (line.startsWith('## ')) {
                        return (
                          <h3 key={idx} className="text-xl font-semibold text-white mt-10 mb-4">
                            {line.replace('## ', '')}
                          </h3>
                        );
                      }

                      // Subsection headers
                      if (line.startsWith('### ')) {
                        return (
                          <h4 key={idx} className="text-base font-semibold text-[#e0e0e0] mt-6 mb-3">
                            {line.replace('### ', '')}
                          </h4>
                        );
                      }

                      // Bullet points
                      if (line.startsWith('- ')) {
                        const content = line.replace('- ', '').replace(/\*\*/g, '');
                        return (
                          <div key={idx} className="flex gap-3 text-[#a0a0a0] leading-relaxed">
                            <span className="text-[#707070] mt-1.5">•</span>
                            <span>{content}</span>
                          </div>
                        );
                      }

                      // Bold lines
                      if (line.match(/^\*\*.*\*\*$/)) {
                        return (
                          <p key={idx} className="text-white font-medium mt-6">
                            {line.replace(/\*\*/g, '')}
                          </p>
                        );
                      }

                      // Regular paragraphs
                      if (line.trim() && !line.startsWith('#')) {
                        return (
                          <p key={idx} className="text-[#a0a0a0] leading-relaxed">
                            {line}
                          </p>
                        );
                      }

                      return null;
                    })}
                  </div>

                  {/* GitHub link */}
                  <div className="mt-8">
                    <a
                      href={release.html_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 text-sm text-[#707070] hover:text-[#a0a0a0] transition-colors"
                    >
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                        <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
                      </svg>
                      View on GitHub
                    </a>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
