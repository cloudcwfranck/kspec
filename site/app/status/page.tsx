import { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Status',
  description: 'Real-time system status for kspec',
};

// Revalidate every 60 seconds
export const revalidate = 60;

interface WorkflowRun {
  id: number;
  name: string;
  status: string;
  conclusion: string | null;
  created_at: string;
  updated_at: string;
  html_url: string;
}

interface WorkflowRunsResponse {
  total_count: number;
  workflow_runs: WorkflowRun[];
}

interface StatusIndicator {
  name: string;
  status: 'operational' | 'degraded' | 'outage' | 'unknown';
  message: string;
  lastUpdated: string;
  url?: string;
}

async function getWorkflowStatus(): Promise<StatusIndicator[]> {
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

  const indicators: StatusIndicator[] = [];

  try {
    // Get latest workflow runs
    const response = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/actions/runs?per_page=10`,
      {
        headers,
        next: { revalidate: 60 },
      }
    );

    if (!response.ok) {
      return [
        {
          name: 'GitHub API',
          status: 'degraded',
          message: `API returned ${response.status}`,
          lastUpdated: new Date().toISOString(),
        },
      ];
    }

    const data: WorkflowRunsResponse = await response.json();
    const runs = data.workflow_runs;

    // CI Workflow Status
    const ciRuns = runs.filter(r => r.name.toLowerCase().includes('ci') || r.name.toLowerCase().includes('test'));
    const latestCI = ciRuns[0];
    if (latestCI) {
      indicators.push({
        name: 'CI Pipeline',
        status: latestCI.conclusion === 'success' ? 'operational' : latestCI.conclusion === 'failure' ? 'degraded' : 'unknown',
        message: latestCI.conclusion === 'success' ? 'All tests passing' : latestCI.conclusion === 'failure' ? 'Some tests failing' : 'Running...',
        lastUpdated: latestCI.updated_at,
        url: latestCI.html_url,
      });
    }

    // E2E Workflow Status
    const e2eRuns = runs.filter(r => r.name.toLowerCase().includes('e2e'));
    const latestE2E = e2eRuns[0];
    if (latestE2E) {
      indicators.push({
        name: 'E2E Tests',
        status: latestE2E.conclusion === 'success' ? 'operational' : latestE2E.conclusion === 'failure' ? 'degraded' : 'unknown',
        message: latestE2E.conclusion === 'success' ? 'All scenarios passing' : latestE2E.conclusion === 'failure' ? 'Some scenarios failing' : 'Running...',
        lastUpdated: latestE2E.updated_at,
        url: latestE2E.html_url,
      });
    }

    // Build Status
    const buildRuns = runs.filter(r => r.name.toLowerCase().includes('build') || r.name.toLowerCase().includes('docker'));
    const latestBuild = buildRuns[0];
    if (latestBuild) {
      indicators.push({
        name: 'Container Builds',
        status: latestBuild.conclusion === 'success' ? 'operational' : latestBuild.conclusion === 'failure' ? 'degraded' : 'unknown',
        message: latestBuild.conclusion === 'success' ? 'Building successfully' : latestBuild.conclusion === 'failure' ? 'Build failing' : 'Building...',
        lastUpdated: latestBuild.updated_at,
        url: latestBuild.html_url,
      });
    }

    // Get latest release
    const releaseResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/releases/latest`,
      {
        headers,
        next: { revalidate: 60 },
      }
    );

    if (releaseResponse.ok) {
      const latestRelease = await releaseResponse.json();
      indicators.push({
        name: 'Latest Release',
        status: 'operational',
        message: latestRelease.tag_name || 'Available',
        lastUpdated: latestRelease.published_at,
        url: latestRelease.html_url,
      });
    }

    // Get issue count
    const issuesResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/issues?labels=bug&state=open&per_page=1`,
      {
        headers,
        next: { revalidate: 60 },
      }
    );

    if (issuesResponse.ok) {
      const issuesLink = issuesResponse.headers.get('link');
      let bugCount = 0;

      if (issuesLink) {
        const match = issuesLink.match(/page=(\d+)>; rel="last"/);
        bugCount = match ? parseInt(match[1]) : 1;
      } else {
        const issues = await issuesResponse.json();
        bugCount = issues.length;
      }

      indicators.push({
        name: 'Open Bug Reports',
        status: bugCount === 0 ? 'operational' : bugCount < 5 ? 'operational' : bugCount < 10 ? 'degraded' : 'degraded',
        message: `${bugCount} open bug${bugCount !== 1 ? 's' : ''}`,
        lastUpdated: new Date().toISOString(),
        url: `https://github.com/${owner}/${repo}/issues?q=is:issue+is:open+label:bug`,
      });
    }

    // If no indicators found, add a default
    if (indicators.length === 0) {
      indicators.push({
        name: 'System Status',
        status: 'operational',
        message: 'All systems operational',
        lastUpdated: new Date().toISOString(),
      });
    }

    return indicators;
  } catch (error) {
    console.error('Failed to fetch workflow status:', error);
    return [
      {
        name: 'System Status',
        status: 'unknown',
        message: 'Unable to fetch status',
        lastUpdated: new Date().toISOString(),
      },
    ];
  }
}

function getStatusColor(status: string) {
  switch (status) {
    case 'operational':
      return 'bg-emerald-500';
    case 'degraded':
      return 'bg-amber-500';
    case 'outage':
      return 'bg-red-500';
    default:
      return 'bg-gray-400';
  }
}

function getStatusText(status: string) {
  switch (status) {
    case 'operational':
      return 'Operational';
    case 'degraded':
      return 'Degraded';
    case 'outage':
      return 'Outage';
    default:
      return 'Unknown';
  }
}

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}

export default async function StatusPage() {
  const indicators = await getWorkflowStatus();
  const allOperational = indicators.every(i => i.status === 'operational');

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <div className="bg-gradient-to-b from-gray-50 to-white border-b border-gray-200 py-16">
        <div className="max-w-4xl mx-auto px-6">
          <div className="flex items-center gap-4 mb-4">
            <div className={`w-3 h-3 rounded-full ${allOperational ? 'bg-emerald-500' : 'bg-amber-500'} animate-pulse`} />
            <h1 className="text-5xl font-bold">System Status</h1>
          </div>
          <p className="text-xl text-gray-600">
            Real-time status of kspec infrastructure and services
          </p>
        </div>
      </div>

      {/* Overall Status */}
      <div className="max-w-4xl mx-auto px-6 py-12">
        <div className={`rounded-xl p-8 mb-12 ${allOperational ? 'bg-emerald-50 border-2 border-emerald-200' : 'bg-amber-50 border-2 border-amber-200'}`}>
          <div className="flex items-center gap-3 mb-2">
            <div className={`w-4 h-4 rounded-full ${allOperational ? 'bg-emerald-500' : 'bg-amber-500'}`} />
            <h2 className="text-2xl font-bold">
              {allOperational ? 'All Systems Operational' : 'Some Systems Degraded'}
            </h2>
          </div>
          <p className="text-gray-600">
            {allOperational
              ? 'All monitored systems are functioning normally.'
              : 'One or more systems are experiencing issues.'}
          </p>
        </div>

        {/* Status Indicators */}
        <div className="space-y-4">
          {indicators.map((indicator) => (
            <div
              key={indicator.name}
              className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <div className={`w-2.5 h-2.5 rounded-full ${getStatusColor(indicator.status)}`} />
                    <h3 className="font-semibold text-lg">{indicator.name}</h3>
                  </div>
                  <p className="text-gray-600 text-sm mb-1">{indicator.message}</p>
                  <p className="text-gray-400 text-xs">
                    Updated {formatRelativeTime(indicator.lastUpdated)}
                  </p>
                </div>
                <div className="flex items-center gap-4">
                  <span
                    className={`text-sm font-medium px-3 py-1 rounded-full ${
                      indicator.status === 'operational'
                        ? 'bg-emerald-100 text-emerald-700'
                        : indicator.status === 'degraded'
                        ? 'bg-amber-100 text-amber-700'
                        : 'bg-gray-100 text-gray-700'
                    }`}
                  >
                    {getStatusText(indicator.status)}
                  </span>
                  {indicator.url && (
                    <a
                      href={indicator.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary-600 hover:text-primary-700"
                    >
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                      </svg>
                    </a>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Footer */}
        <div className="mt-12 text-center text-sm text-gray-500">
          <p>Status data automatically refreshes every 60 seconds.</p>
          <p className="mt-2">
            Data sourced from{' '}
            <a
              href="https://github.com/cloudcwfranck/kspec/actions"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary-600 hover:text-primary-700"
            >
              GitHub Actions
            </a>
            {' '}and{' '}
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
