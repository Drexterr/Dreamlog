const { withSentryConfig } = require('@sentry/nextjs');

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  trailingSlash: true,
};

// Source maps upload to Sentry at build time when SENTRY_AUTH_TOKEN is set
// (therapist-portal/.env.local, gitignored). Without the token the upload is
// skipped with a warning and the build still succeeds.
module.exports = withSentryConfig(nextConfig, {
  org: 'bharat-jain-i5',
  project: 'ode-portal',
  sentryUrl: 'https://de.sentry.io/',
  silent: !process.env.CI,
  widenClientFileUpload: true,
  sourcemaps: {
    deleteSourcemapsAfterUpload: true,
  },
});
