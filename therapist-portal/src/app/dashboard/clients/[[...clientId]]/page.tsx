import ClientBriefContent from './ClientBriefContent';

export function generateStaticParams() {
  // Placeholder satisfies Next.js static export build requirement.
  // Real client IDs are runtime values; Firebase rewrites serve the dashboard
  // shell for /dashboard/clients/** paths, and the client component handles routing.
  return [{ clientId: ['_'] }];
}

export default function ClientBriefPage() {
  return <ClientBriefContent />;
}
