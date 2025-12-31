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
        // Vercel brand colors
        accent: {
          DEFAULT: '#0070F3',         // Vercel blue
          hover: '#0761D1',           // Darker blue on hover
        },
        // Vercel light mode design system
        vercel: {
          bg: '#FFFFFF',              // Pure white background
          'bg-subtle': '#FAFAFA',     // Off-white background
          surface: '#FFFFFF',         // Cards/panels (white)
          text: '#000000',            // Primary text (black)
          'text-secondary': '#666666', // Secondary text (gray)
          'text-muted': '#999999',    // Muted text (light gray)
          border: '#EAEAEA',          // Primary borders
          'border-light': '#F0F0F0',  // Lighter borders
        },
        // Semantic colors
        success: '#0070F3',           // Blue for success
        warning: '#F5A623',           // Orange for warnings
        error: '#E00',                // Red for errors
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
        'lg': '8px',
        'xl': '12px',
        '2xl': '16px',
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
