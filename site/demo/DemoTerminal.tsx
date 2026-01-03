'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import Link from 'next/link';
import demoData from './demoSteps.json';

interface DemoStep {
  command: string;
  output: string;
}

interface DemoTab {
  id: string;
  label: string;
  description: string;
  steps: DemoStep[];
  docsLink: string;
}

interface DemoData {
  version: string;
  tabs: DemoTab[];
}

const typedDemoData = demoData as DemoData;

const TYPING_SPEED = 30; // ms per character
const STEP_DELAY = 1500; // ms between steps

export default function DemoTerminal() {
  const [activeTab, setActiveTab] = useState(0);
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [displayedCommand, setDisplayedCommand] = useState('');
  const [displayedOutput, setDisplayedOutput] = useState('');
  const [showOutput, setShowOutput] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState(false);

  const terminalRef = useRef<HTMLDivElement>(null);
  const prefersReducedMotion = useRef(false);

  // Check for reduced motion preference
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    prefersReducedMotion.current = mediaQuery.matches;

    const handleChange = (e: MediaQueryListEvent) => {
      prefersReducedMotion.current = e.matches;
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  const currentTab = typedDemoData.tabs[activeTab];
  const currentStep = currentTab.steps[currentStepIndex];

  // Type command animation
  useEffect(() => {
    if (!isPlaying || !currentStep) return;

    const command = currentStep.command;
    const speed = prefersReducedMotion.current ? 0 : TYPING_SPEED;

    if (displayedCommand.length < command.length) {
      const timeout = setTimeout(() => {
        setDisplayedCommand(command.slice(0, displayedCommand.length + 1));
      }, speed);
      return () => clearTimeout(timeout);
    } else {
      // Command finished typing, show output
      setShowOutput(true);
    }
  }, [isPlaying, displayedCommand, currentStep]);

  // Show output and advance to next step
  useEffect(() => {
    if (!isPlaying || !showOutput || !currentStep) return;

    const output = currentStep.output;
    const speed = prefersReducedMotion.current ? 0 : TYPING_SPEED;

    if (displayedOutput.length < output.length) {
      const timeout = setTimeout(() => {
        setDisplayedOutput(output.slice(0, displayedOutput.length + 1));
      }, speed);
      return () => clearTimeout(timeout);
    } else {
      // Output finished, wait then advance to next step
      const delay = prefersReducedMotion.current ? 500 : STEP_DELAY;
      const timeout = setTimeout(() => {
        if (currentStepIndex < currentTab.steps.length - 1) {
          setCurrentStepIndex(currentStepIndex + 1);
          setDisplayedCommand('');
          setDisplayedOutput('');
          setShowOutput(false);
        } else {
          // Reached end of tab
          setIsPlaying(false);
        }
      }, delay);
      return () => clearTimeout(timeout);
    }
  }, [isPlaying, showOutput, displayedOutput, currentStep, currentStepIndex, currentTab]);

  // Auto-scroll terminal to bottom
  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
    }
  }, [displayedCommand, displayedOutput]);

  const handleTabChange = useCallback((index: number) => {
    setActiveTab(index);
    setCurrentStepIndex(0);
    setDisplayedCommand('');
    setDisplayedOutput('');
    setShowOutput(false);
    setIsPlaying(false);
  }, []);

  const handlePlay = useCallback(() => {
    setIsPlaying(true);
  }, []);

  const handlePause = useCallback(() => {
    setIsPlaying(false);
  }, []);

  const handleReset = useCallback(() => {
    setCurrentStepIndex(0);
    setDisplayedCommand('');
    setDisplayedOutput('');
    setShowOutput(false);
    setIsPlaying(false);
  }, []);

  const handleNextStep = useCallback(() => {
    if (currentStepIndex < currentTab.steps.length - 1) {
      setCurrentStepIndex(currentStepIndex + 1);
      setDisplayedCommand('');
      setDisplayedOutput('');
      setShowOutput(false);
    }
  }, [currentStepIndex, currentTab]);

  const handlePrevStep = useCallback(() => {
    if (currentStepIndex > 0) {
      setCurrentStepIndex(currentStepIndex - 1);
      setDisplayedCommand('');
      setDisplayedOutput('');
      setShowOutput(false);
    }
  }, [currentStepIndex]);

  const handleCopyCommand = useCallback(async () => {
    if (!currentStep) return;

    try {
      await navigator.clipboard.writeText(currentStep.command);
      setCopiedCommand(true);
      setTimeout(() => setCopiedCommand(false), 2000);
    } catch (err) {
      console.error('Failed to copy command:', err);
    }
  }, [currentStep]);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return;
      }

      switch (e.key) {
        case ' ':
          e.preventDefault();
          if (isPlaying) {
            handlePause();
          } else {
            handlePlay();
          }
          break;
        case 'ArrowRight':
          e.preventDefault();
          handleNextStep();
          break;
        case 'ArrowLeft':
          e.preventDefault();
          handlePrevStep();
          break;
        case 'r':
        case 'R':
          e.preventDefault();
          handleReset();
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isPlaying, handlePlay, handlePause, handleNextStep, handlePrevStep, handleReset]);

  return (
    <div className="w-full max-w-6xl mx-auto">
      {/* Tab Navigation */}
      <div
        className="flex gap-2 mb-4 overflow-x-auto pb-2 scrollbar-thin"
        role="tablist"
        aria-label="Demo capabilities"
      >
        {typedDemoData.tabs.map((tab, index) => (
          <button
            key={tab.id}
            onClick={() => handleTabChange(index)}
            role="tab"
            aria-selected={activeTab === index}
            aria-controls={`tabpanel-${tab.id}`}
            id={`tab-${tab.id}`}
            className={`px-4 py-2 rounded-lg font-medium text-sm whitespace-nowrap transition-all duration-200 ${
              activeTab === index
                ? 'bg-primary-600 text-white shadow-lg'
                : 'bg-white text-gray-700 hover:bg-gray-100 border border-gray-200'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Description */}
      <div className="mb-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <p className="text-gray-600">{currentTab.description}</p>
        <Link
          href={currentTab.docsLink}
          className="text-primary-600 hover:text-primary-700 font-medium text-sm whitespace-nowrap flex items-center gap-1"
          aria-label={`View documentation for ${currentTab.label}`}
        >
          View docs
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </Link>
      </div>

      {/* Terminal Window */}
      <div
        className="bg-gray-900 rounded-lg shadow-2xl overflow-hidden border border-gray-700"
        role="region"
        aria-label="Interactive terminal demo"
      >
        {/* Terminal Header */}
        <div className="bg-gray-800 px-4 py-3 flex items-center justify-between border-b border-gray-700">
          <div className="flex items-center gap-2">
            <div className="flex gap-2">
              <div className="w-3 h-3 rounded-full bg-red-500" aria-hidden="true" />
              <div className="w-3 h-3 rounded-full bg-yellow-500" aria-hidden="true" />
              <div className="w-3 h-3 rounded-full bg-green-500" aria-hidden="true" />
            </div>
            <span className="text-gray-400 text-sm ml-2 font-mono">
              kspec-demo
            </span>
          </div>

          {/* Control Buttons */}
          <div className="flex items-center gap-2">
            <button
              onClick={handlePrevStep}
              disabled={currentStepIndex === 0}
              aria-label="Previous step"
              className="p-1.5 text-gray-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              title="Previous step (←)"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>

            {isPlaying ? (
              <button
                onClick={handlePause}
                aria-label="Pause playback"
                className="p-1.5 text-gray-400 hover:text-white transition-colors"
                title="Pause (Space)"
              >
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                </svg>
              </button>
            ) : (
              <button
                onClick={handlePlay}
                aria-label="Play demo"
                className="p-1.5 text-gray-400 hover:text-white transition-colors"
                title="Play (Space)"
              >
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z" />
                </svg>
              </button>
            )}

            <button
              onClick={handleNextStep}
              disabled={currentStepIndex === currentTab.steps.length - 1}
              aria-label="Next step"
              className="p-1.5 text-gray-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              title="Next step (→)"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>

            <button
              onClick={handleReset}
              aria-label="Reset demo"
              className="p-1.5 text-gray-400 hover:text-white transition-colors"
              title="Reset (R)"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </button>

            <button
              onClick={handleCopyCommand}
              aria-label="Copy current command"
              className="p-1.5 text-gray-400 hover:text-white transition-colors"
              title="Copy command"
            >
              {copiedCommand ? (
                <svg className="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              ) : (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
              )}
            </button>
          </div>
        </div>

        {/* Terminal Content */}
        <div
          ref={terminalRef}
          className="p-6 font-mono text-sm text-gray-100 h-96 overflow-y-auto"
          role="tabpanel"
          id={`tabpanel-${currentTab.id}`}
          aria-labelledby={`tab-${currentTab.id}`}
        >
          {/* Display previous steps */}
          {currentTab.steps.slice(0, currentStepIndex).map((step, index) => (
            <div key={index} className="mb-4">
              <div className="flex items-start gap-2">
                <span className="text-emerald-400" aria-hidden="true">$</span>
                <span className="text-gray-300">{step.command}</span>
              </div>
              <pre className="mt-2 text-gray-400 whitespace-pre-wrap break-words">
                {step.output}
              </pre>
            </div>
          ))}

          {/* Current step being typed */}
          {currentStep && (
            <div className="mb-4">
              <div className="flex items-start gap-2">
                <span className="text-emerald-400" aria-hidden="true">$</span>
                <span className="text-gray-300">
                  {displayedCommand}
                  {isPlaying && displayedCommand.length < currentStep.command.length && (
                    <span className="animate-pulse">▊</span>
                  )}
                </span>
              </div>
              {showOutput && (
                <pre className="mt-2 text-gray-400 whitespace-pre-wrap break-words">
                  {displayedOutput}
                  {isPlaying && displayedOutput.length < currentStep.output.length && (
                    <span className="animate-pulse">▊</span>
                  )}
                </pre>
              )}
            </div>
          )}

          {/* Cursor when idle */}
          {!isPlaying && displayedCommand === currentStep?.command && displayedOutput === currentStep?.output && (
            <div className="flex items-center gap-2">
              <span className="text-emerald-400" aria-hidden="true">$</span>
              <span className="animate-pulse text-gray-300">▊</span>
            </div>
          )}
        </div>

        {/* Progress Indicator */}
        <div className="bg-gray-800 px-4 py-2 border-t border-gray-700">
          <div className="flex items-center justify-between text-xs text-gray-400">
            <span>
              Step {currentStepIndex + 1} of {currentTab.steps.length}
            </span>
            <span className="hidden sm:inline">
              Use ← → to navigate, Space to play/pause, R to reset
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
