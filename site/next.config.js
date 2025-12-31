/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,

  // Ensure static export works, but also support ISR
  // We'll use revalidate for changelog and status pages

  env: {
    GITHUB_OWNER: process.env.GITHUB_OWNER || 'cloudcwfranck',
    GITHUB_REPO: process.env.GITHUB_REPO || 'kspec',
    GITHUB_TOKEN: process.env.GITHUB_TOKEN || '',
  },

  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin',
          },
        ],
      },
    ];
  },
};

module.exports = nextConfig;
