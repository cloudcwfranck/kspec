export interface Commit {
  sha: string;
  message: string;
  author: string;
  url: string;
  pr?: string;
}

export interface ParsedRelease {
  version: string;
  date: string;
  title: string;
  summary?: string;
  highlights: string[];
  commits: Commit[];
}

/**
 * Parse GitHub release body to extract structured sections
 */
export function parseReleaseBody(body: string, tagName: string): {
  summary?: string;
  highlights: string[];
  otherChanges: string[];
} {
  const lines = body.split('\n');
  const summary: string[] = [];
  const highlights: string[] = [];
  const otherChanges: string[] = [];

  let currentSection: 'summary' | 'highlights' | 'other' | 'skip' = 'summary';
  let foundFirstHeader = false;

  for (const line of lines) {
    const trimmed = line.trim();

    // Skip empty lines
    if (!trimmed) continue;

    // Detect sections
    if (trimmed.startsWith('## ')) {
      foundFirstHeader = true;
      const header = trimmed.toLowerCase();

      if (header.includes('what') || header.includes('new') || header.includes('feature') || header.includes('highlight')) {
        currentSection = 'highlights';
      } else if (header.includes('other') || header.includes('change') || header.includes('commit')) {
        currentSection = 'other';
      } else {
        currentSection = 'skip';
      }
      continue;
    }

    // Parse content based on current section
    if (trimmed.startsWith('- ')) {
      const content = trimmed.substring(2).replace(/\*\*/g, '').trim();

      if (currentSection === 'highlights') {
        highlights.push(content);
      } else if (currentSection === 'other') {
        otherChanges.push(content);
      }
    } else if (!foundFirstHeader && currentSection === 'summary') {
      // Collect summary paragraphs before any headers
      if (!trimmed.startsWith('#') && !trimmed.startsWith('**Full Changelog')) {
        summary.push(trimmed);
      }
    }
  }

  return {
    summary: summary.length > 0 ? summary.join(' ') : undefined,
    highlights,
    otherChanges,
  };
}

/**
 * Parse commit string from "Other Changes" section
 * Format: "- {message} by @{author} in #{pr}" or "- {message} ({hash})"
 */
export function parseCommitLine(line: string, repoUrl: string): Commit | null {
  // Remove leading "- "
  const cleaned = line.replace(/^-\s*/, '').trim();

  // Try to extract PR number
  const prMatch = cleaned.match(/#(\d+)/);
  const pr = prMatch ? prMatch[1] : undefined;

  // Try to extract author
  const authorMatch = cleaned.match(/@(\w+)/);
  const author = authorMatch ? authorMatch[1] : 'unknown';

  // Try to extract hash
  const hashMatch = cleaned.match(/\(([a-f0-9]{7,40})\)/);
  const sha = hashMatch ? hashMatch[1] : 'unknown';

  // Clean message (remove author, PR, hash)
  let message = cleaned
    .replace(/by @\w+/, '')
    .replace(/in #\d+/, '')
    .replace(/\([a-f0-9]{7,40}\)/, '')
    .replace(/\*\*/g, '')
    .trim();

  // If no message, use the whole line
  if (!message) {
    message = cleaned;
  }

  return {
    sha,
    message,
    author,
    url: pr ? `${repoUrl}/pull/${pr}` : `${repoUrl}/commit/${sha}`,
    pr,
  };
}

/**
 * Fetch commits for a release from GitHub API
 */
export async function fetchReleaseCommits(
  owner: string,
  repo: string,
  tagName: string,
  previousTag?: string,
  token?: string
): Promise<Commit[]> {
  try {
    const headers: HeadersInit = {
      'Accept': 'application/vnd.github+json',
      'X-GitHub-Api-Version': '2022-11-28',
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    // Use compare API to get commits between tags
    const base = previousTag || 'HEAD~10'; // Fallback to last 10 commits
    const url = `https://api.github.com/repos/${owner}/${repo}/compare/${base}...${tagName}`;

    const response = await fetch(url, { headers, next: { revalidate: 300 } });

    if (!response.ok) {
      console.error(`GitHub compare API error: ${response.status}`);
      return [];
    }

    const data = await response.json();
    const commits = data.commits || [];

    return commits.map((commit: any) => ({
      sha: commit.sha,
      message: commit.commit.message.split('\n')[0], // First line only
      author: commit.author?.login || commit.commit.author?.name || 'unknown',
      url: commit.html_url,
      pr: undefined, // Would need to fetch PR data separately
    }));
  } catch (error) {
    console.error('Failed to fetch commits:', error);
    return [];
  }
}

/**
 * Format date in Linear.app style
 */
export function formatReleaseDate(dateString: string): string {
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
