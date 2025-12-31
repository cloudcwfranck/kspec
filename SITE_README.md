# kspec Website (`/site`)

This directory contains the official marketing and documentation website for kspec.

## Isolation Guarantee

The `/site` directory is **100% isolated** from the operator codebase:

- ✅ No changes to Go modules required
- ✅ No changes to operator Docker builds
- ✅ No changes to existing Makefile targets
- ✅ No impact on operator tests or CI/CD
- ✅ Can be safely removed without affecting the operator

## Technology

- **Framework**: Next.js 14 (App Router)
- **Styling**: TailwindCSS (Linear.app aesthetic)
- **Content**: MDX for documentation
- **Deployment**: Vercel (zero-config)

## Features

- **Homepage**: Product marketing with value props
- **Docs**: MDX-based documentation with sidebar
- **Changelog**: Auto-updates from GitHub Releases (ISR 5min)
- **Status**: Real-time GitHub Actions/API status (ISR 60s)
- **Install**: Step-by-step installation guide
- **SEO**: Full metadata, sitemap, robots.txt

## Quick Start

```bash
cd site
npm install
npm run dev
```

Visit http://localhost:3000

## Deployment

### Vercel (Recommended)

1. Import repository to Vercel
2. Set Root Directory: `site`
3. Framework: Next.js (auto-detected)
4. Deploy

### Environment Variables (Optional)

```env
GITHUB_TOKEN=ghp_xxx  # For higher API rate limits
GITHUB_OWNER=cloudcwfranck
GITHUB_REPO=kspec
```

## Documentation

See `/site/README.md` for complete documentation including:

- Adding documentation pages
- Customizing design
- GitHub API integration
- Troubleshooting

## Vercel Configuration

The site includes `vercel.json` for one-click deployment:

- Build command: `npm run build`
- Framework preset: Next.js
- Output directory: `.next`

Just point Vercel to the `/site` directory and it works out of the box.

## Contributing

To add documentation:

1. Create MDX file in `/site/content/docs/`
2. Add frontmatter (title, description, order)
3. Commit and push

The page will automatically appear in navigation.

---

**For full details, see `/site/README.md`**
