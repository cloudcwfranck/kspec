import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import Navigation from '@/components/Navigation';
import Footer from '@/components/Footer';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: {
    default: 'kspec - Policy Enforcement for Kubernetes',
    template: '%s | kspec',
  },
  description: 'Declarative policy enforcement platform for Kubernetes clusters. Define security, compliance, and operational policies as code with automated enforcement.',
  keywords: ['kubernetes', 'policy', 'compliance', 'security', 'enforcement', 'governance', 'kyverno'],
  authors: [{ name: 'kspec team' }],
  creator: 'kspec',
  openGraph: {
    type: 'website',
    locale: 'en_US',
    url: 'https://kspec.dev',
    title: 'kspec - Policy Enforcement for Kubernetes',
    description: 'Declarative policy enforcement platform for Kubernetes clusters',
    siteName: 'kspec',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'kspec - Policy Enforcement for Kubernetes',
    description: 'Declarative policy enforcement platform for Kubernetes clusters',
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-video-preview': -1,
      'max-image-preview': 'large',
      'max-snippet': -1,
    },
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="scroll-smooth">
      <body className={`${inter.className} antialiased bg-linear-bg text-linear-text`}>
        <Navigation />
        <main className="min-h-screen">{children}</main>
        <Footer />
      </body>
    </html>
  );
}
