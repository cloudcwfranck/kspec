import { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Changelog',
  description: 'Latest releases and updates for kspec',
};

export const revalidate = 300;

interface GitHubRelease {
  id: number;
  name: string;
  tag_name: string;
  published_at: string;
  body: string;
  html_url: string;
  draft: boolean;
  prerelease: boolean;
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
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

interface ParsedChanges {
  features: string[];
  improvements: string[];
  fixes: string[];
  other: string[];
}

function parseReleaseBody(body: string): ParsedChanges {
  const changes: ParsedChanges = {
    features: [],
    improvements: [],
    fixes: [],
    other: [],
  };

  // Split by lines and process
  const lines = body.split('\n');
  let currentSection: keyof ParsedChanges = 'other';

  for (const line of lines) {
    const trimmed = line.trim();

    // Skip empty lines, commit hashes, and very technical lines
    if (!trimmed ||
        trimmed.match(/^[a-f0-9]{7,40}:/) ||
        trimmed.match(/^\(@[\w-]+\)/) ||
        trimmed.startsWith('*') && trimmed.includes('(@')) {
      continue;
    }

    // Detect sections
    if (trimmed.toLowerCase().includes('what\'s in this release') ||
        trimmed.toLowerCase().includes('### new') ||
        trimmed.toLowerCase().includes('### features')) {
      currentSection = 'features';
      continue;
    } else if (trimmed.toLowerCase().includes('### improvements') ||
               trimmed.toLowerCase().includes('### enhancements')) {
      currentSection = 'improvements';
      continue;
    } else if (trimmed.toLowerCase().includes('### fixes') ||
               trimmed.toLowerCase().includes('### bug fixes')) {
      currentSection = 'fixes';
      continue;
    } else if (trimmed.startsWith('##') || trimmed.startsWith('###')) {
      currentSection = 'other';
      continue;
    }

    // Parse bullet points
    if (trimmed.startsWith('- ') || trimmed.startsWith('* ')) {
      let text = trimmed.substring(2).trim();

      // Remove emoji characters and clean up
      text = text.replace(/[✅🔧📊🛡️⚡🎯📝🔍]/g, '').trim();

      // Remove markdown bold
      text = text.replace(/\*\*/g, '');

      // Remove PR/commit references
      text = text.replace(/\(#\d+\)/g, '').trim();
      text = text.replace(/\([a-f0-9]{7,40}\)/g, '').trim();

      // Categorize based on keywords if not in a specific section
      if (currentSection === 'other') {
        const lowerText = text.toLowerCase();
        if (lowerText.startsWith('add') || lowerText.startsWith('implement') ||
            lowerText.includes('new feature')) {
          changes.features.push(text);
        } else if (lowerText.startsWith('fix') || lowerText.includes('bug fix')) {
          changes.fixes.push(text);
        } else if (lowerText.startsWith('improve') || lowerText.startsWith('update') ||
                   lowerText.startsWith('enhance') || lowerText.startsWith('optimize')) {
          changes.improvements.push(text);
        } else if (text.length > 10) { // Only add substantial items
          changes.other.push(text);
        }
      } else if (text.length > 10) {
        changes[currentSection].push(text);
      }
    }
  }

  return changes;
}

export default async function ChangelogPage() {
  const releases = await getReleases();

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <div className="bg-gradient-to-b from-gray-50 to-white border-b border-gray-200 py-16">
        <div className="max-w-4xl mx-auto px-6">
          <h1 className="text-5xl font-bold mb-4 text-gray-900">Changelog</h1>
          <p className="text-xl text-gray-600">
            Track the latest features, improvements, and fixes in kspec
          </p>
        </div>
      </div>

      {/* Releases */}
      <div className="max-w-4xl mx-auto px-6 py-12">
        {releases.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-500">No releases found</p>
          </div>
        ) : (
          <div className="space-y-16">
            {releases.map((release) => {
              const changes = parseReleaseBody(release.body);
              const hasChanges = changes.features.length > 0 ||
                                changes.improvements.length > 0 ||
                                changes.fixes.length > 0 ||
                                changes.other.length > 0;

              return (
                <article key={release.id} className="relative">
                  {/* Version badge and date */}
                  <div className="flex items-center gap-4 mb-6">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-primary-600" />
                      <h2 className="text-3xl font-bold text-gray-900">
                        {release.tag_name}
                      </h2>
                    </div>
                    <time className="text-sm text-gray-500 font-medium">
                      {formatDate(release.published_at)}
                    </time>
                  </div>

                  {release.name && release.name !== release.tag_name && (
                    <p className="text-lg text-gray-700 mb-6">{release.name}</p>
                  )}

                  {hasChanges ? (
                    <div className="space-y-8">
                      {/* Features */}
                      {changes.features.length > 0 && (
                        <div>
                          <h3 className="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-4 flex items-center gap-2">
                            <span className="w-1 h-4 bg-emerald-500 rounded" />
                            New Features
                          </h3>
                          <ul className="space-y-3">
                            {changes.features.map((item, idx) => (
                              <li key={idx} className="flex gap-3 text-gray-700 leading-relaxed">
                                <span className="text-emerald-600 mt-1.5">●</span>
                                <span>{item}</span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}

                      {/* Improvements */}
                      {changes.improvements.length > 0 && (
                        <div>
                          <h3 className="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-4 flex items-center gap-2">
                            <span className="w-1 h-4 bg-blue-500 rounded" />
                            Improvements
                          </h3>
                          <ul className="space-y-3">
                            {changes.improvements.map((item, idx) => (
                              <li key={idx} className="flex gap-3 text-gray-700 leading-relaxed">
                                <span className="text-blue-600 mt-1.5">●</span>
                                <span>{item}</span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}

                      {/* Bug Fixes */}
                      {changes.fixes.length > 0 && (
                        <div>
                          <h3 className="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-4 flex items-center gap-2">
                            <span className="w-1 h-4 bg-amber-500 rounded" />
                            Bug Fixes
                          </h3>
                          <ul className="space-y-3">
                            {changes.fixes.map((item, idx) => (
                              <li key={idx} className="flex gap-3 text-gray-700 leading-relaxed">
                                <span className="text-amber-600 mt-1.5">●</span>
                                <span>{item}</span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}

                      {/* Other Changes */}
                      {changes.other.length > 0 && (
                        <div>
                          <h3 className="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-4 flex items-center gap-2">
                            <span className="w-1 h-4 bg-gray-400 rounded" />
                            Other Changes
                          </h3>
                          <ul className="space-y-3">
                            {changes.other.map((item, idx) => (
                              <li key={idx} className="flex gap-3 text-gray-700 leading-relaxed">
                                <span className="text-gray-400 mt-1.5">●</span>
                                <span>{item}</span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </div>
                  ) : (
                    <p className="text-gray-600">See full release notes for details</p>
                  )}

                  {/* View on GitHub */}
                  <div className="mt-8">
                    <a
                      href={release.html_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 text-sm text-primary-600 hover:text-primary-700 font-medium"
                    >
                      View release on GitHub
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                      </svg>
                    </a>
                  </div>
                </article>
              );
            })}
          </div>
        )}

        {/* Footer */}
        <div className="mt-16 text-center">
          <p className="text-sm text-gray-500">
            Updates published automatically from{' '}
            <a
              href="https://github.com/cloudcwfranck/kspec/releases"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary-600 hover:text-primary-700"
            >
              GitHub Releases
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}
