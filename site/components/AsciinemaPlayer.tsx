'use client';

import { useEffect, useRef } from 'react';

interface AsciinemaPlayerProps {
  src: string;
  cols?: number;
  rows?: number;
  autoPlay?: boolean;
  preload?: boolean;
  loop?: boolean;
  startAt?: number | string;
  speed?: number;
  idleTimeLimit?: number;
  theme?: string;
  poster?: string;
  fit?: 'width' | 'height' | 'both' | 'none';
  controls?: boolean | 'auto';
  className?: string;
}

export default function AsciinemaPlayer({
  src,
  cols = 120,
  rows = 30,
  autoPlay = false,
  preload = true,
  loop = false,
  startAt,
  speed = 1,
  idleTimeLimit = 2,
  theme = 'asciinema',
  poster,
  fit = 'width',
  controls = true,
  className = '',
}: AsciinemaPlayerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const playerRef = useRef<any>(null);

  useEffect(() => {
    // Load asciinema-player script and CSS
    const loadAsciinema = async () => {
      // Check if already loaded
      if (window.AsciinemaPlayer) {
        createPlayer();
        return;
      }

      // Load CSS
      const css = document.createElement('link');
      css.rel = 'stylesheet';
      css.type = 'text/css';
      css.href = 'https://cdn.jsdelivr.net/npm/asciinema-player@3.7.0/dist/bundle/asciinema-player.min.css';
      document.head.appendChild(css);

      // Load JS
      const script = document.createElement('script');
      script.src = 'https://cdn.jsdelivr.net/npm/asciinema-player@3.7.0/dist/bundle/asciinema-player.min.js';
      script.async = true;
      script.onload = () => {
        createPlayer();
      };
      document.head.appendChild(script);
    };

    const createPlayer = () => {
      if (!containerRef.current || playerRef.current) return;

      // Clear container
      containerRef.current.innerHTML = '';

      const options: any = {
        cols,
        rows,
        autoPlay,
        preload,
        loop,
        speed,
        idleTimeLimit,
        theme,
        fit,
        controls,
      };

      if (startAt !== undefined) options.startAt = startAt;
      if (poster !== undefined) options.poster = poster;

      playerRef.current = window.AsciinemaPlayer.create(
        src,
        containerRef.current,
        options
      );
    };

    loadAsciinema();

    return () => {
      if (playerRef.current && playerRef.current.dispose) {
        playerRef.current.dispose();
        playerRef.current = null;
      }
    };
  }, [src, cols, rows, autoPlay, preload, loop, startAt, speed, idleTimeLimit, theme, poster, fit, controls]);

  return (
    <div
      ref={containerRef}
      className={className}
      style={{ maxWidth: '100%' }}
    />
  );
}

// Extend Window interface for TypeScript
declare global {
  interface Window {
    AsciinemaPlayer?: any;
  }
}
