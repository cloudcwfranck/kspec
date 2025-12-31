import { Metadata } from 'next';
import ChangelogBlock from '@/components/ChangelogBlock';
import {
  parseReleaseBody,
  parseCommitLine,
  formatReleaseDate,
  type Commit,
} from '@/lib/changelog';

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

export default async function ChangelogPage() {
  const releases = await getReleases();
  const owner = process.env.GITHUB_OWNER || 'cloudcwfranck';
  const repo = process.env.GITHUB_REPO || 'kspec';
  const repoUrl = `https://github.com/${owner}/${repo}`;

  return (
    <div className="min-h-screen bg-linear-bg">
      {/* Header */}
      <div className="border-b border-linear-border">
        <div className="max-w-5xl mx-auto px-6 py-20">
          <h1 className="text-5xl font-bold mb-3 text-linear-text tracking-tight">Changelog</h1>
          <p className="text-lg text-linear-text-secondary">
            New features, improvements, and fixes
          </p>
        </div>
      </div>

      {/* Releases */}
      <div className="max-w-5xl mx-auto px-6 py-12">
        {releases.length === 0 ? (
          <div className="text-center py-20">
            <p className="text-linear-text-muted">No releases yet</p>
          </div>
        ) : (
          <div className="space-y-16">
            {releases.map((release) => {
              const { summary, highlights, otherChanges } = parseReleaseBody(
                release.body,
                release.tag_name
              );

              // Parse commits from "Other Changes" section
              const commits: Commit[] = otherChanges
                .map(line => parseCommitLine(line, repoUrl))
                .filter((c): c is Commit => c !== null);

              return (
                <ChangelogBlock
                  key={release.id}
                  version={release.tag_name}
                  date={formatReleaseDate(release.published_at)}
                  title={release.name || release.tag_name}
                  summary={summary}
                  highlights={highlights}
                  commits={commits}
                  githubUrl={release.html_url}
                />
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
