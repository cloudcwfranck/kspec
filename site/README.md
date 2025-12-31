# kspec Website

Official marketing and documentation website for kspec policy enforcement platform.

Built with Next.js 14 (App Router), TailwindCSS, and MDX. Features a Linear.app-inspired design with clean typography, subtle gradients, and lots of whitespace.

## Features

- **Homepage**: Hero section, value propositions, how it works, CTAs
- **Documentation**: MDX-based docs with sidebar navigation and syntax highlighting
- **Changelog**: Auto-updates from GitHub Releases every 5 minutes (ISR)
- **Status Page**: Real-time system status from GitHub Actions API (60s refresh)
- **Install Guide**: Step-by-step installation instructions
- **SEO**: Metadata, OpenGraph, sitemap, robots.txt
- **Responsive**: Mobile-first design with Tailwind

## Tech Stack

- **Framework**: Next.js 14 (App Router)
- **Styling**: TailwindCSS + custom Linear-inspired theme
- **Content**: MDX via next-mdx-remote
- **Syntax Highlighting**: rehype-highlight
- **APIs**: GitHub REST API (Releases, Actions, Issues)
- **Deployment**: Vercel (optimized for zero-config deployment)

## Local Development

### Prerequisites

- Node.js 18+
- npm or yarn

### Setup

```bash
# Install dependencies
cd site
npm install

# Run development server
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Environment Variables

Create a `.env.local` file (optional):

```env
# GitHub API (optional - for higher rate limits)
GITHUB_TOKEN=ghp_your_github_personal_access_token

# GitHub repository (defaults shown)
GITHUB_OWNER=cloudcwfranck
GITHUB_REPO=kspec
```

#### Why GITHUB_TOKEN?

- **Without token**: 60 requests/hour (unauthenticated)
- **With token**: 5,000 requests/hour (authenticated)

The changelog and status pages fetch from GitHub API. With ISR caching (300s for changelog, 60s for status), the site works fine without a token for most traffic levels.

**To create a token:**

1. Go to GitHub Settings → Developer settings → Personal access tokens
2. Generate new token (classic)
3. Select scopes: `public_repo` (read-only access)
4. Copy token to `.env.local`

### Project Structure

```
site/
├── app/                    # Next.js App Router pages
│   ├── layout.tsx         # Root layout with nav/footer
│   ├── page.tsx           # Homepage
│   ├── docs/              # Documentation
│   │   ├── page.tsx       # Docs index
│   │   └── [...slug]/     # Dynamic doc pages
│   ├── changelog/         # Auto-updating changelog
│   ├── status/            # Real-time status page
│   ├── install/           # Installation guide
│   ├── sitemap.ts         # Dynamic sitemap
│   └── robots.ts          # robots.txt
├── components/            # React components
│   ├── Navigation.tsx     # Header nav
│   ├── Footer.tsx         # Footer
│   └── DocSidebar.tsx     # Docs sidebar
├── content/
│   └── docs/              # MDX documentation files
│       ├── getting-started.mdx
│       ├── guides/
│       └── api-reference.mdx
├── lib/
│   └── docs.ts            # MDX utilities
├── public/                # Static assets (if any)
├── package.json
├── next.config.js
├── tailwind.config.js
├── tsconfig.json
└── README.md
```

## Adding Documentation

### Create a new doc page

1. Create an MDX file in `content/docs/`:

```mdx
---
title: My New Doc
description: A helpful description
order: 10
---

# My New Doc

Your content here...

## Code Examples

\`\`\`yaml
apiVersion: kspec.io/v1alpha1
kind: ClusterSpecification
# ...
\`\`\`
```

2. The page will automatically appear in the docs sidebar and navigation.

### Organizing Docs

- Place files in subdirectories to create sections: `content/docs/guides/my-guide.mdx`
- Use `order` frontmatter field to control sidebar ordering (lower = higher)
- Section titles are derived from directory names

## Deployment

### Vercel (Recommended)

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https://github.com/cloudcwfranck/kspec/tree/main/site)

**One-click deployment steps:**

1. Click "Deploy to Vercel" button (or manually import from GitHub)
2. Configure project:
   - **Root Directory**: `site`
   - **Framework Preset**: Next.js
   - **Build Command**: `npm run build` (auto-detected)
   - **Output Directory**: `.next` (auto-detected)
3. Add environment variables (optional):
   - `GITHUB_TOKEN`: Your GitHub personal access token
   - `GITHUB_OWNER`: `cloudcwfranck` (default)
   - `GITHUB_REPO`: `kspec` (default)
4. Deploy!

The site will auto-deploy on every push to `main`.

### Manual Deployment

```bash
# Build for production
npm run build

# Start production server
npm start
```

Deploy the `.next` directory and `public` folder to any Node.js hosting platform.

### Static Export (Alternative)

If you need a fully static site (no ISR):

1. Update `next.config.js`:

```js
module.exports = {
  output: 'export',
  // ... rest of config
};
```

2. Build:

```bash
npm run build
```

3. Deploy the `out` directory to any static host (Netlify, Cloudflare Pages, S3, etc.)

**Note**: Static export disables ISR, so changelog/status won't auto-refresh.

## GitHub API Integration

### Changelog (`/changelog`)

- **Source**: `GET /repos/{owner}/{repo}/releases`
- **Caching**: ISR with 300s revalidation (5 minutes)
- **Behavior**: Fetches on first request, then serves cached data. Revalidates in background every 5 minutes.

### Status Page (`/status`)

- **Sources**:
  - `GET /repos/{owner}/{repo}/actions/runs` - Workflow status
  - `GET /repos/{owner}/{repo}/releases/latest` - Latest release
  - `GET /repos/{owner}/{repo}/issues?labels=bug` - Open bugs
- **Caching**: ISR with 60s revalidation
- **Behavior**: Real-time-ish updates (1-minute freshness)

### Rate Limiting

GitHub API limits:
- **Unauthenticated**: 60 requests/hour
- **Authenticated** (with GITHUB_TOKEN): 5,000 requests/hour

With ISR caching, the site makes minimal requests:
- Changelog: Max 12 requests/hour (60min ÷ 5min)
- Status: Max 60 requests/hour

Even without authentication, this stays well under the 60 req/hour limit for moderate traffic.

## Customization

### Changing Colors

Edit `tailwind.config.js`:

```js
theme: {
  extend: {
    colors: {
      primary: {
        // Your brand colors here
        600: '#4f46e5',
        // ...
      },
    },
  },
},
```

### Changing Fonts

Update `app/layout.tsx`:

```tsx
import { YourFont } from 'next/font/google';

const yourFont = YourFont({ subsets: ['latin'] });
```

### Adding Pages

Create a new file in `app/`:

```tsx
// app/about/page.tsx
export default function AboutPage() {
  return <div>About kspec...</div>;
}
```

## Maintenance

### Updating Dependencies

```bash
npm update
npm audit fix
```

### Testing Changelog/Status Locally

The GitHub API pages work in development mode. Just run `npm run dev` and visit:

- http://localhost:3000/changelog
- http://localhost:3000/status

They'll fetch real data from the GitHub API.

## Troubleshooting

### "Failed to fetch releases"

- Check `GITHUB_OWNER` and `GITHUB_REPO` env vars
- Verify repository is public (or add `GITHUB_TOKEN` for private repos)
- Check GitHub API status: https://www.githubstatus.com/

### Docs not showing up

- Ensure MDX files are in `content/docs/`
- Check frontmatter syntax (YAML between `---` markers)
- Verify file extension is `.mdx` or `.md`

### Build errors

- Clear `.next` folder: `rm -rf .next`
- Delete `node_modules` and reinstall: `rm -rf node_modules && npm install`
- Check Next.js version compatibility

### Rate limit errors

- Add `GITHUB_TOKEN` environment variable
- Increase ISR revalidation times in page components

## Performance

- **Lighthouse Score**: 95-100 (target)
- **ISR**: Automatic static optimization with background revalidation
- **Image Optimization**: Next.js automatic image optimization (if using next/image)
- **Code Splitting**: Automatic per-route code splitting
- **Caching**: Aggressive caching for API responses

## Contributing

### Adding Docs

1. Create MDX file in `content/docs/`
2. Add frontmatter (title, description, order)
3. Write content with code examples
4. Commit and push

### Fixing Bugs

1. Create an issue: https://github.com/cloudcwfranck/kspec/issues
2. Submit a PR with fixes
3. Ensure `npm run build` succeeds

### Design Updates

- Follow Linear.app aesthetic: minimal, clean, lots of whitespace
- Use Tailwind utilities (avoid custom CSS)
- Test on mobile and desktop

## License

Apache 2.0 - Same as the kspec project

## Support

- GitHub Issues: https://github.com/cloudcwfranck/kspec/issues
- Discussions: https://github.com/cloudcwfranck/kspec/discussions

---

Built with ❤️ for the kspec community
