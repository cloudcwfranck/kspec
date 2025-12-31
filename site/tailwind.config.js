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
        // Linear dark theme colors
        primary: {
          400: '#a78bfa',
          500: '#8b5cf6',
          600: '#7c3aed',
          700: '#6d28d9',
        },
        dark: {
          50: '#1a1a1a',   // Slightly lighter than bg
          100: '#0a0a0a',  // Main background
          200: '#050505',  // Darker areas
          300: '#2a2a2a',  // Hover states
          400: '#3a3a3a',  // Borders
        },
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
      backgroundColor: {
        'dark-bg': '#0a0a0a',
        'dark-card': '#1a1a1a',
        'dark-hover': '#2a2a2a',
      },
      borderColor: {
        'dark-border': '#2a2a2a',
      },
      textColor: {
        'dark-primary': '#ffffff',
        'dark-secondary': '#a0a0a0',
        'dark-tertiary': '#707070',
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
};
