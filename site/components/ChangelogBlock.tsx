'use client';

import { useState } from 'react';

interface Commit {
  sha: string;
  message: string;
  author: string;
  url: string;
  pr?: string;
}

interface ChangelogBlockProps {
  version: string;
  date: string;
  title: string;
  summary?: string;
  highlights: string[];
  commits: Commit[];
  githubUrl: string;
}

function CommitRow({ commit }: { commit: Commit }) {
  const shortHash = commit.sha.substring(0, 7);
  const truncatedMessage = commit.message.length > 60
    ? commit.message.substring(0, 60) + '...'
    : commit.message;

  return (
    <a
      href={commit.url}
      target="_blank"
      rel="noopener noreferrer"
      className="flex items-center gap-3 px-4 py-3 hover:bg-linear-surface rounded-lg transition-colors group"
    >
      {/* Hash badge */}
      <div className="flex-shrink-0">
        <code className="text-xs font-mono text-linear-text-muted bg-linear-surface px-2 py-1 rounded border border-linear-border">
          {shortHash}
        </code>
      </div>

      {/* Commit message */}
      <div className="flex-1 min-w-0">
        <p className="text-sm text-linear-text-secondary group-hover:text-linear-text truncate transition-colors">
          {truncatedMessage}
        </p>
      </div>

      {/* Author + PR badges */}
      <div className="flex items-center gap-2 flex-shrink-0">
        {commit.pr && (
          <span className="text-xs text-accent bg-accent/10 px-2 py-1 rounded border border-accent/20">
            #{commit.pr}
          </span>
        )}
        <span className="text-xs text-linear-text-muted bg-linear-surface px-2 py-1 rounded border border-linear-border">
          {commit.author}
        </span>
      </div>
    </a>
  );
}

export default function ChangelogBlock({
  version,
  date,
  title,
  summary,
  highlights,
  commits,
  githubUrl,
}: ChangelogBlockProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <article className="relative">
      {/* Header row */}
      <div className="flex items-center gap-4 mb-6">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-accent" />
          <time className="text-sm text-linear-text-secondary font-medium">{date}</time>
        </div>
        <span className="text-xs text-linear-text-muted bg-linear-surface px-2 py-1 rounded border border-linear-border font-mono">
          {version}
        </span>
      </div>

      {/* Title */}
      <h2 className="text-3xl font-bold text-linear-text mb-4 tracking-tight">
        {title}
      </h2>

      {/* Summary */}
      {summary && (
        <p className="text-lg text-linear-text-secondary mb-8 leading-relaxed">
          {summary}
        </p>
      )}

      {/* Highlights - What's in this release */}
      {highlights.length > 0 && (
        <div className="mb-8">
          <h3 className="text-sm font-semibold text-linear-text-muted uppercase mb-4 tracking-wide">
            What's in this release
          </h3>
          <div className="space-y-3">
            {highlights.map((highlight, idx) => (
              <div key={idx} className="flex gap-3 text-linear-text-secondary leading-relaxed">
                <span className="text-linear-text-muted mt-1.5">•</span>
                <span>{highlight}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* All changes - Collapsible */}
      {commits.length > 0 && (
        <div className="mt-8">
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="flex items-center gap-2 text-sm font-medium text-linear-text-muted hover:text-linear-text transition-colors mb-4"
          >
            <svg
              className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
            All changes ({commits.length})
          </button>

          {isExpanded && (
            <div className="bg-linear-surface border border-linear-border rounded-xl overflow-hidden">
              <div className="max-h-[420px] overflow-y-auto">
                <div className="divide-y divide-linear-border">
                  {commits.map((commit, idx) => (
                    <CommitRow key={idx} commit={commit} />
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* GitHub link */}
      <div className="mt-8">
        <a
          href={githubUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 text-sm text-linear-text-muted hover:text-linear-text-secondary transition-colors"
        >
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path
              fillRule="evenodd"
              d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
              clipRule="evenodd"
            />
          </svg>
          View on GitHub
        </a>
      </div>
    </article>
  );
}
