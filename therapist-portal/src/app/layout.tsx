import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'DreamLog Therapist Portal',
  description: 'Client dashboard for DreamLog therapists',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
