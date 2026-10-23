/**
 * Sentry breadcrumbs for the upload path (presign → PUT → register, plus the
 * offline queue). Breadcrumbs ride along with the next error event, so a
 * crash or failed upload report shows which step failed and how.
 *
 * Privacy: breadcrumb data is restricted to the allowlisted keys below, each
 * a number, boolean or short enum-like string. Never pass audio keys (they
 * contain user ids), URLs (presigned URLs carry signatures), file paths,
 * transcripts or error messages.
 *
 * Fail-silent: a no-op when Sentry is not configured (no
 * EXPO_PUBLIC_SENTRY_DSN), when the native module is missing (Expo Go, tests),
 * or when Sentry itself throws.
 */

export type UploadCrumbLevel = 'info' | 'warning' | 'error';

export interface UploadCrumbData {
  phase?: 'presign' | 'put' | 'register' | 'note_put' | 'queue';
  attempt?: number;
  status?: number;        // HTTP status of the failing call
  size?: string;          // sizeBucket() output, never exact bytes
  mode?: string;          // entry mode (processing | rant | ...)
  resumed?: boolean;      // retry continued with audio already in storage
  permanent?: boolean;    // offline item stopped auto-retrying
  count?: number;
  uploaded?: number;
  remaining?: number;
}

const ALLOWED_KEYS: ReadonlySet<string> = new Set([
  'phase', 'attempt', 'status', 'size', 'mode', 'resumed', 'permanent', 'count', 'uploaded', 'remaining',
]);
// Short identifier-like strings only: anything with a slash, colon, dot or
// space (paths, URLs, keys, messages) is dropped.
const SAFE_STRING = /^[a-z0-9_<>=+-]{1,24}$/i;

interface SentryLike {
  addBreadcrumb: (crumb: { category: string; message: string; level: UploadCrumbLevel; data?: Record<string, unknown> }) => void;
}

let sentry: SentryLike | null | undefined;

function loadSentry(): SentryLike | null {
  if (sentry !== undefined) return sentry;
  sentry = null;
  if (!process.env.EXPO_PUBLIC_SENTRY_DSN) return sentry;
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const mod = require('@sentry/react-native') as Partial<SentryLike>;
    if (mod && typeof mod.addBreadcrumb === 'function') sentry = mod as SentryLike;
  } catch {
    // Native module absent - stay disabled.
  }
  return sentry;
}

function sanitize(data?: UploadCrumbData): Record<string, unknown> | undefined {
  if (!data) return undefined;
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(data)) {
    if (!ALLOWED_KEYS.has(key)) continue;
    if (typeof value === 'number' && Number.isFinite(value)) out[key] = value;
    else if (typeof value === 'boolean') out[key] = value;
    else if (typeof value === 'string' && SAFE_STRING.test(value)) out[key] = value;
  }
  return Object.keys(out).length > 0 ? out : undefined;
}

/** Records an upload breadcrumb. Never throws. */
export function uploadBreadcrumb(message: string, data?: UploadCrumbData, level: UploadCrumbLevel = 'info'): void {
  try {
    loadSentry()?.addBreadcrumb({ category: 'upload', message, level, data: sanitize(data) });
  } catch {
    // Telemetry must never break an upload.
  }
}

/** Coarse size bucket for breadcrumbs - exact sizes are not needed. */
export function sizeBucket(bytes: number | null | undefined): string {
  if (typeof bytes !== 'number' || !Number.isFinite(bytes) || bytes < 0) return 'unknown';
  const mb = bytes / (1024 * 1024);
  if (mb < 0.1) return '<100KB';
  if (mb < 1) return '100KB-1MB';
  if (mb < 5) return '1-5MB';
  if (mb < 20) return '5-20MB';
  if (mb < 50) return '20-50MB';
  return '>=50MB';
}

/** HTTP status carried by an Axios-style error, if any. */
export function errorStatus(err: unknown): number | undefined {
  const status = (err as { response?: { status?: unknown } } | null)?.response?.status;
  return typeof status === 'number' ? status : undefined;
}

/** Test hook: forget the cached Sentry module. */
export function __resetTelemetryForTests(): void {
  sentry = undefined;
}
