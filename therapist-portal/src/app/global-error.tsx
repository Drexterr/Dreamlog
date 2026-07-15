'use client';

// Catches React render errors at the root of the app tree and reports them
// to Sentry (unhandled runtime errors are captured automatically; render
// errors need this boundary). Must render its own <html>/<body>.
import * as Sentry from '@sentry/nextjs';
import { useEffect } from 'react';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <html>
      <body className="flex min-h-screen items-center justify-center bg-slate-950 text-slate-100">
        <div className="text-center">
          <h2 className="mb-2 text-xl font-semibold">Something went wrong</h2>
          <p className="mb-6 text-sm text-slate-400">
            The error has been reported. Please try again.
          </p>
          <button
            onClick={reset}
            className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium hover:bg-violet-500"
          >
            Try again
          </button>
        </div>
      </body>
    </html>
  );
}
