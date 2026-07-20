import type { Metadata, Viewport } from 'next';
import './globals.css';

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
    'Talk for two minutes or twenty. Ode listens, transcribes, and reflects back something worth sitting with — grounded in everything you have shared before.',
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
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link rel="preconnect" href="https://cdn.fontshare.com" crossOrigin="anonymous" />
        {/* Hanken Grotesk: body/UI · Newsreader Italic: quotes & reflections */}
        <link
          href="https://fonts.googleapis.com/css2?family=Hanken+Grotesk:wght@400;500;600;700&family=Newsreader:ital,opsz,wght@1,6..72,400&display=swap"
          rel="stylesheet"
        />
        {/* Erode: display & headings (Fontshare · ITF Free Font License) */}
        <link
          href="https://api.fontshare.com/v2/css?f[]=erode@300,400,401,500,600,700&display=swap"
          rel="stylesheet"
        />
      </head>
      <body>{children}</body>
    </html>
  );
}
