import type { Metadata, Viewport } from 'next';
import './globals.css';
import '../styles/website.css';
import '../styles/landing.css';
import '../styles/refinements.css';

export const viewport: Viewport = {
  themeColor: '#18150f',
};

export const metadata: Metadata = {
  icons: {
    icon: [
      { url: '/favicon-32.png', sizes: '32x32', type: 'image/png' },
      { url: '/favicon.png', sizes: '48x48', type: 'image/png' },
      { url: '/icon-192.png', sizes: '192x192', type: 'image/png' },
      { url: '/icon-512.png', sizes: '512x512', type: 'image/png' },
    ],
    apple: '/apple-touch-icon.png',
  },
  title: {
    default: 'Ode — Voice Journaling with AI Reflection',
    template: '%s · Ode',
  },
  description:
    'Record or write your journal. Ode offers AI reflections grounded in your recent entries and helps you notice recurring themes.',
  keywords: ['voice journaling', 'AI reflection', 'mental wellness', 'mood tracking', 'therapy companion'],
  openGraph: {
    title: 'Ode — Voice Journaling with AI Reflection',
    description: 'Your thoughts, out loud. Voice journaling with AI reflections, mood patterns, and therapist sharing.',
    type: 'website',
    siteName: 'Ode',
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head><link rel="preload" href="/fonts/Erode-Regular.ttf" as="font" type="font/ttf" crossOrigin="anonymous" /></head>
      <body>{children}</body>
    </html>
  );
}
