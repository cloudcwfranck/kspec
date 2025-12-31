/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './app/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
    './content/**/*.{md,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        // Linear.app official accent color
        accent: {
          DEFAULT: '#5E6AD2',
          hover: '#6B76DB',
        },
        // Linear.app dark mode colors (exact spec)
        linear: {
          bg: '#0E0F12',              // App background
          surface: '#16181D',         // Cards/panels
          'surface-secondary': '#1C1F26', // Secondary surface
          text: '#EDEEF0',            // Primary text
          'text-secondary': '#A8ADB7', // Secondary text
          'text-muted': '#8A8F98',    // Muted text
          border: '#262A33',          // Borders
        },
        // Semantic colors (Linear spec)
        success: '#4CB782',
        warning: '#F2C94C',
        error: '#EB5757',
      },
      fontFamily: {
        sans: [
          'Inter',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'sans-serif',
        ],
        mono: [
          'JetBrains Mono',
          'Menlo',
          'Monaco',
          'Courier New',
          'monospace',
        ],
      },
      borderRadius: {
        'lg': '10px',
        'xl': '12px',
      },
      spacing: {
        // 8px spacing system (Linear spec)
        '1': '4px',
        '2': '8px',
        '3': '12px',
        '4': '16px',
        '6': '24px',
        '8': '32px',
        '12': '48px',
      },
      transitionDuration: {
        fast: '120ms',
        normal: '180ms',
        slow: '220ms',
      },
      transitionTimingFunction: {
        'linear-ease': 'cubic-bezier(0.2, 0, 0, 1)',
      },
      lineHeight: {
        'tight': '1.45',
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
};
