'use client';

import { useState } from 'react';
import AsciinemaPlayer from './AsciinemaPlayer';
import demoData from '@/demo/demoSteps.json';

interface DemoTab {
  id: string;
  label: string;
  description: string;
  docsLink: string;
}

export default function AsciinemaDemo() {
  const [activeTabId, setActiveTabId] = useState(demoData.tabs[0].id);

  const currentTab = demoData.tabs.find(tab => tab.id === activeTabId) as DemoTab;

  return (
    <div className="w-full">
      {/* Tab Navigation */}
      <div className="border-b border-gray-200 mb-6">
        <nav className="-mb-px flex gap-8 overflow-x-auto" aria-label="Demo tabs">
          {demoData.tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTabId(tab.id)}
              className={`
                whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm transition-colors
                ${activeTabId === tab.id
                  ? 'border-primary-600 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }
              `}
              role="tab"
              aria-selected={activeTabId === tab.id}
              aria-controls={`panel-${tab.id}`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div
        role="tabpanel"
        id={`panel-${activeTabId}`}
        aria-labelledby={`tab-${activeTabId}`}
        className="space-y-6"
      >
        {/* Description and Docs Link */}
        <div className="flex items-center justify-between gap-4 flex-wrap">
          <p className="text-gray-700 text-base">
            {currentTab.description}
          </p>
          <a
            href={currentTab.docsLink}
            target="_blank"
            rel="noopener noreferrer"
            className="text-gray-900 hover:text-gray-700 font-medium text-sm whitespace-nowrap transition-colors inline-flex items-center gap-1"
            aria-label={`View documentation for ${currentTab.label} (opens in new tab)`}
          >
            View docs →
          </a>
        </div>

        {/* Asciinema Player Container */}
        <div className="bg-[#0A0A0A] rounded-lg shadow-[0_8px_30px_rgb(0,0,0,0.12)] overflow-hidden border border-gray-800 relative">
          {/* macOS-style traffic lights */}
          <div className="flex items-center gap-2 px-4 py-3 bg-[#1A1A1A] border-b border-gray-800">
            <div className="flex gap-2">
              <div className="w-3 h-3 rounded-full bg-[#FF5F57]" />
              <div className="w-3 h-3 rounded-full bg-[#FEBC2E]" />
              <div className="w-3 h-3 rounded-full bg-[#28CA42]" />
            </div>
            <div className="flex-1 text-center text-xs text-gray-400 font-medium">
              {currentTab.label} Demo
            </div>
          </div>

          {/* Player */}
          <div className="p-4">
            <AsciinemaPlayer
              key={activeTabId}
              src={`/demos/asciinema/${activeTabId}.cast`}
              cols={120}
              rows={30}
              autoPlay={false}
              preload={true}
              loop={false}
              idleTimeLimit={2}
              theme="asciinema"
              fit="width"
              controls={true}
              className="w-full"
            />
          </div>
        </div>

        {/* Help text */}
        <div className="text-sm text-gray-500 text-center">
          Click play to start the demo, or use the controls to pause and scrub through the recording
        </div>
      </div>
    </div>
  );
}
