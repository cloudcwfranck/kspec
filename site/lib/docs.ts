import fs from 'fs';
import path from 'path';
import matter from 'gray-matter';

const docsDirectory = path.join(process.cwd(), 'content/docs');

export interface DocFrontmatter {
  title: string;
  description?: string;
  order?: number;
}

export interface DocPage {
  slug: string;
  frontmatter: DocFrontmatter;
  content: string;
}

export interface DocSection {
  title: string;
  pages: { title: string; slug: string }[];
}

export function getAllDocs(): DocPage[] {
  const files = getAllMDXFiles(docsDirectory);

  return files.map((filePath) => {
    const fileContents = fs.readFileSync(filePath, 'utf8');
    const { data, content } = matter(fileContents);

    const slug = filePath
      .replace(docsDirectory, '')
      .replace(/\.mdx?$/, '')
      .replace(/^\//, '');

    return {
      slug,
      frontmatter: data as DocFrontmatter,
      content,
    };
  }).sort((a, b) => {
    const orderA = a.frontmatter.order || 999;
    const orderB = b.frontmatter.order || 999;
    return orderA - orderB;
  });
}

export function getDocBySlug(slug: string): DocPage | null {
  const fullPath = path.join(docsDirectory, `${slug}.mdx`);

  if (!fs.existsSync(fullPath)) {
    return null;
  }

  const fileContents = fs.readFileSync(fullPath, 'utf8');
  const { data, content } = matter(fileContents);

  return {
    slug,
    frontmatter: data as DocFrontmatter,
    content,
  };
}

export function getDocSidebar(): DocSection[] {
  const docs = getAllDocs();

  // Group by directory
  const grouped: Record<string, DocPage[]> = {};

  docs.forEach((doc) => {
    const parts = doc.slug.split('/');
    const section = parts.length > 1 ? parts[0] : 'getting-started';

    if (!grouped[section]) {
      grouped[section] = [];
    }

    grouped[section].push(doc);
  });

  // Convert to DocSection array
  return Object.entries(grouped).map(([key, pages]) => ({
    title: formatSectionTitle(key),
    pages: pages.map((p) => ({
      title: p.frontmatter.title,
      slug: p.slug,
    })),
  }));
}

function getAllMDXFiles(dir: string): string[] {
  if (!fs.existsSync(dir)) {
    return [];
  }

  const files: string[] = [];
  const items = fs.readdirSync(dir);

  items.forEach((item) => {
    const fullPath = path.join(dir, item);
    const stat = fs.statSync(fullPath);

    if (stat.isDirectory()) {
      files.push(...getAllMDXFiles(fullPath));
    } else if (item.endsWith('.mdx') || item.endsWith('.md')) {
      files.push(fullPath);
    }
  });

  return files;
}

function formatSectionTitle(slug: string): string {
  return slug
    .split('-')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}
