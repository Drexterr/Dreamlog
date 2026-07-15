// Sentry browser-side error reporting for the therapist portal.
// The portal is a static export (no server runtime), so this client config is
// the entire Sentry surface. Enabled only in production builds - `next dev`
// stays Sentry-free. The DSN is a public identifier, not a secret.
//
// Privacy: sendDefaultPii stays false and Session Replay is deliberately NOT
// enabled - the portal renders decrypted client session notes, which must
// never appear in an error event or replay recording.
import * as Sentry from '@sentry/nextjs';

Sentry.init({
  dsn: 'https://4d1ae72d9ab5249412e8b538805f9f8a@o4511737812287488.ingest.de.sentry.io/4511737948602448',
  enabled: process.env.NODE_ENV === 'production',
  sendDefaultPii: false,
});
