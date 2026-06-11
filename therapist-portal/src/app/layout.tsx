import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: {
    default: 'DreamLog — Voice Journaling with AI Reflection',
    template: '%s · DreamLog',
  },
  description:
    'Talk for two minutes or twenty. DreamLog listens, transcribes, and reflects back something worth sitting with — grounded in everything you have shared before.',
  keywords: ['voice journaling', 'AI reflection', 'mental wellness', 'mood tracking', 'therapy companion'],
  openGraph: {
    title: 'DreamLog — Voice Journaling with AI Reflection',
    description: 'Your thoughts, out loud. Voice journaling with AI reflections, mood patterns, and therapist sharing.',
    type: 'website',
    siteName: 'DreamLog',
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Cormorant+Garamond:ital,wght@0,300;0,400;0,600;1,300;1,400;1,600&family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap"
          rel="stylesheet"
        />
      </head>
      <body>{children}</body>
    </html>
  );
}
