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
  prompt?: string;
  tabs: DemoTab[];
}

interface TerminalLine {
  type: 'prompt' | 'command' | 'output';
  content: string;
}

const typedDemoData = demoData as DemoData;
const TYPING_SPEED = 20; // ms per character for commands
const LINE_DELAY = 50; // ms between output lines

export default function DemoTerminal() {
  const [activeTab, setActiveTab] = useState(0);
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [lines, setLines] = useState<TerminalLine[]>([]);
  const [typingCommand, setTypingCommand] = useState('');
  const [isTypingCommand, setIsTypingCommand] = useState(false);
  const [outputLines, setOutputLines] = useState<string[]>([]);
  const [currentOutputLine, setCurrentOutputLine] = useState(0);
  const [copiedCommand, setCopiedCommand] = useState(false);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const [showFollowButton, setShowFollowButton] = useState(false);

  const terminalRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
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
  const prompt = typedDemoData.prompt || '$';

  // Scroll detection with IntersectionObserver
  useEffect(() => {
    if (!sentinelRef.current || !terminalRef.current) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const [entry] = entries;
        const atBottom = entry.isIntersecting;
        setIsAtBottom(atBottom);
        setShowFollowButton(!atBottom && isPlaying);
      },
      {
        root: terminalRef.current,
        threshold: 0.1,
      }
    );

    observer.observe(sentinelRef.current);
    return () => observer.disconnect();
  }, [isPlaying]);

  // Auto-scroll to bottom when content changes (only if user is at bottom)
  useEffect(() => {
    if (isAtBottom && terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
    }
  }, [lines, typingCommand, isAtBottom]);

  const scrollToBottom = useCallback(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
      setIsAtBottom(true);
      setShowFollowButton(false);
    }
  }, []);

  // Type command animation
  useEffect(() => {
    if (!isPlaying || !isTypingCommand || !currentStep) return;

    const command = currentStep.command;
    const speed = prefersReducedMotion.current ? 0 : TYPING_SPEED;

    if (typingCommand.length < command.length) {
      const timeout = setTimeout(() => {
        setTypingCommand(command.slice(0, typingCommand.length + 1));
      }, speed);
      return () => clearTimeout(timeout);
    } else {
      // Command fully typed, add it to lines and start output
      setLines(prev => [...prev, { type: 'command', content: command }]);
      setIsTypingCommand(false);

      // Parse output into lines
      const output = currentStep.output;
      const outputLineArray = output.split('\n');
      setOutputLines(outputLineArray);
      setCurrentOutputLine(0);
    }
  }, [isPlaying, isTypingCommand, typingCommand, currentStep]);

  // Stream output lines
  useEffect(() => {
    if (!isPlaying || isTypingCommand || outputLines.length === 0) return;

    const delay = prefersReducedMotion.current ? 0 : LINE_DELAY;

    if (currentOutputLine < outputLines.length) {
      const timeout = setTimeout(() => {
        setLines(prev => [...prev, { type: 'output', content: outputLines[currentOutputLine] }]);
        setCurrentOutputLine(currentOutputLine + 1);
      }, delay);
      return () => clearTimeout(timeout);
    } else {
      // Output complete, move to next step or stop
      const nextStepDelay = prefersReducedMotion.current ? 300 : 800;
      const timeout = setTimeout(() => {
        if (currentStepIndex < currentTab.steps.length - 1) {
          // Add prompt for next command
          setLines(prev => [...prev, { type: 'prompt', content: prompt }]);
          setCurrentStepIndex(currentStepIndex + 1);
          setTypingCommand('');
          setIsTypingCommand(true);
          setOutputLines([]);
          setCurrentOutputLine(0);
        } else {
          // End of tab
          setIsPlaying(false);
        }
      }, nextStepDelay);
      return () => clearTimeout(timeout);
    }
  }, [isPlaying, isTypingCommand, currentOutputLine, outputLines, currentStepIndex, currentTab, prompt]);

  const handleTabChange = useCallback((index: number) => {
    setActiveTab(index);
    setCurrentStepIndex(0);
    setLines([]);
    setTypingCommand('');
    setIsTypingCommand(false);
    setOutputLines([]);
    setCurrentOutputLine(0);
    setIsPlaying(false);
  }, []);

  const handlePlay = useCallback(() => {
    if (lines.length === 0) {
      // Start from beginning
      setLines([{ type: 'prompt', content: prompt }]);
      setTypingCommand('');
      setIsTypingCommand(true);
    }
    setIsPlaying(true);
  }, [lines, prompt]);

  const handlePause = useCallback(() => {
    setIsPlaying(false);
  }, []);

  const handleReset = useCallback(() => {
    setCurrentStepIndex(0);
    setLines([]);
    setTypingCommand('');
    setIsTypingCommand(false);
    setOutputLines([]);
    setCurrentOutputLine(0);
    setIsPlaying(false);
  }, []);

  const handleNextStep = useCallback(() => {
    if (currentStepIndex < currentTab.steps.length - 1) {
      const nextIndex = currentStepIndex + 1;

      // Add current command and output instantly
      const newLines: TerminalLine[] = [...lines];
      if (currentStep) {
        newLines.push({ type: 'command', content: currentStep.command });
        currentStep.output.split('\n').forEach(line => {
          newLines.push({ type: 'output', content: line });
        });
      }
      newLines.push({ type: 'prompt', content: prompt });

      setLines(newLines);
      setCurrentStepIndex(nextIndex);
      setTypingCommand('');
      setIsTypingCommand(false);
      setOutputLines([]);
      setCurrentOutputLine(0);
    }
  }, [currentStepIndex, currentTab, currentStep, lines, prompt]);

  const handlePrevStep = useCallback(() => {
    if (currentStepIndex > 0) {
      // Rebuild lines up to previous step
      const prevIndex = currentStepIndex - 1;
      const newLines: TerminalLine[] = [];

      for (let i = 0; i <= prevIndex; i++) {
        newLines.push({ type: 'prompt', content: prompt });
        newLines.push({ type: 'command', content: currentTab.steps[i].command });
        currentTab.steps[i].output.split('\n').forEach(line => {
          newLines.push({ type: 'output', content: line });
        });
      }
      newLines.push({ type: 'prompt', content: prompt });

      setLines(newLines);
      setCurrentStepIndex(prevIndex);
      setTypingCommand('');
      setIsTypingCommand(false);
      setOutputLines([]);
      setCurrentOutputLine(0);
    }
  }, [currentStepIndex, currentTab, prompt]);

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
        className="flex gap-2 mb-6 overflow-x-auto pb-2"
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
            className={`px-4 py-2.5 rounded-md font-medium text-sm whitespace-nowrap transition-all duration-150 ${
              activeTab === index
                ? 'bg-gray-900 text-white shadow-sm'
                : 'bg-gray-50 text-gray-700 hover:bg-gray-100 border border-gray-200'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Description */}
      <div className="mb-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <p className="text-gray-600 text-sm">{currentTab.description}</p>
        <Link
          href={currentTab.docsLink}
          className="text-gray-900 hover:text-gray-700 font-medium text-sm whitespace-nowrap flex items-center gap-1.5 transition-colors"
          aria-label={`View documentation for ${currentTab.label}`}
        >
          View docs
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </Link>
      </div>

      {/* Terminal Window */}
      <div
        className="bg-[#0A0A0A] rounded-lg shadow-[0_8px_30px_rgb(0,0,0,0.12)] overflow-hidden border border-gray-800 relative"
        role="region"
        aria-label="Interactive terminal demo"
      >
        {/* Terminal Header */}
        <div className="bg-[#1A1A1A] px-4 py-2.5 flex items-center justify-between border-b border-gray-800">
          <div className="flex items-center gap-3">
            <div className="flex gap-1.5">
              <div className="w-3 h-3 rounded-full bg-[#FF5F57]" aria-hidden="true" />
              <div className="w-3 h-3 rounded-full bg-[#FEBC2E]" aria-hidden="true" />
              <div className="w-3 h-3 rounded-full bg-[#28CA42]" aria-hidden="true" />
            </div>
            <span className="text-gray-500 text-xs font-mono ml-1">
              kspec-demo
            </span>
          </div>

          {/* Control Buttons */}
          <div className="flex items-center gap-1">
            <button
              onClick={handlePrevStep}
              disabled={currentStepIndex === 0}
              aria-label="Previous step"
              className="p-1.5 text-gray-500 hover:text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors rounded hover:bg-gray-800"
              title="Previous step (←)"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M15 19l-7-7 7-7" />
              </svg>
            </button>

            {isPlaying ? (
              <button
                onClick={handlePause}
                aria-label="Pause playback"
                className="p-1.5 text-gray-500 hover:text-gray-300 transition-colors rounded hover:bg-gray-800"
                title="Pause (Space)"
              >
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                </svg>
              </button>
            ) : (
              <button
                onClick={handlePlay}
                aria-label="Play demo"
                className="p-1.5 text-gray-500 hover:text-gray-300 transition-colors rounded hover:bg-gray-800"
                title="Play (Space)"
              >
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z" />
                </svg>
              </button>
            )}

            <button
              onClick={handleNextStep}
              disabled={currentStepIndex === currentTab.steps.length - 1}
              aria-label="Next step"
              className="p-1.5 text-gray-500 hover:text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors rounded hover:bg-gray-800"
              title="Next step (→)"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M9 5l7 7-7 7" />
              </svg>
            </button>

            <div className="w-px h-4 bg-gray-700 mx-1" />

            <button
              onClick={handleReset}
              aria-label="Reset demo"
              className="p-1.5 text-gray-500 hover:text-gray-300 transition-colors rounded hover:bg-gray-800"
              title="Reset (R)"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </button>

            <button
              onClick={handleCopyCommand}
              aria-label="Copy current command"
              className="p-1.5 text-gray-500 hover:text-gray-300 transition-colors rounded hover:bg-gray-800"
              title="Copy command"
            >
              {copiedCommand ? (
                <svg className="w-3.5 h-3.5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                </svg>
              ) : (
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
              )}
            </button>
          </div>
        </div>

        {/* Terminal Content */}
        <div
          ref={terminalRef}
          className="relative p-4 font-mono text-[13px] leading-relaxed text-gray-100 h-[500px] overflow-y-auto"
          role="tabpanel"
          id={`tabpanel-${currentTab.id}`}
          aria-labelledby={`tab-${currentTab.id}`}
          style={{ scrollBehavior: 'smooth' }}
        >
          {lines.map((line, index) => (
            <div key={index} className="mb-0.5">
              {line.type === 'prompt' && (
                <div className="flex items-start gap-1.5 text-emerald-400 mt-2">
                  <span>{line.content}</span>
                </div>
              )}
              {line.type === 'command' && (
                <div className="flex items-start gap-1.5">
                  <span className="text-emerald-400">{prompt}</span>
                  <span className="text-white">{line.content}</span>
                </div>
              )}
              {line.type === 'output' && (
                <pre className="text-gray-400 whitespace-pre-wrap break-words font-mono text-[13px] pl-0">
                  {line.content}
                </pre>
              )}
            </div>
          ))}

          {/* Currently typing command */}
          {isTypingCommand && (
            <div className="flex items-start gap-1.5">
              <span className="text-emerald-400">{prompt}</span>
              <span className="text-white">
                {typingCommand}
                {isPlaying && <span className="animate-pulse text-emerald-400">▊</span>}
              </span>
            </div>
          )}

          {/* Idle cursor - show when not playing or when finished */}
          {!isPlaying && !isTypingCommand && (
            <div className="flex items-center gap-1.5 mt-2">
              <span className="text-emerald-400">{prompt}</span>
              <span className="animate-pulse text-emerald-400">▊</span>
            </div>
          )}

          {/* Sentinel for scroll detection */}
          <div ref={sentinelRef} className="h-px" />
        </div>

        {/* Follow Button */}
        {showFollowButton && (
          <div className="absolute bottom-20 right-8">
            <button
              onClick={scrollToBottom}
              className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-white text-xs font-medium rounded-md shadow-lg border border-gray-700 flex items-center gap-1.5 transition-colors"
              aria-label="Scroll to bottom and resume auto-scroll"
            >
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
              </svg>
              Follow
            </button>
          </div>
        )}

        {/* Progress Footer */}
        <div className="bg-[#1A1A1A] px-4 py-2 border-t border-gray-800">
          <div className="flex items-center justify-between text-xs text-gray-500">
            <span>
              Step {currentStepIndex + 1} of {currentTab.steps.length}
            </span>
            <span className="hidden sm:inline font-mono">
              ← → navigate • Space play/pause • R reset
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
