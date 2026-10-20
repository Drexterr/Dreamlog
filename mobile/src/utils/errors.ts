// Narrowing helpers for caught errors (`catch (err: unknown)`) - mobile/CLAUDE.md
// forbids `any`, and a thrown value can be anything.

/** The error's message, or `fallback` when it has none. */
export function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'object' && err !== null && typeof (err as { message?: unknown }).message === 'string') {
    const msg = (err as { message: string }).message;
    if (msg) return msg;
  }
  return fallback;
}

/** A string `code` property (e.g. 'ERR_REQUEST_CANCELED'), if present. */
export function errorCode(err: unknown): string | undefined {
  if (typeof err === 'object' && err !== null) {
    const code = (err as { code?: unknown }).code;
    if (typeof code === 'string') return code;
  }
  return undefined;
}

/** The HTTP status of an API (axios) error, if present. */
export function httpStatus(err: unknown): number | undefined {
  if (typeof err === 'object' && err !== null) {
    const status = (err as { response?: { status?: unknown } }).response?.status;
    if (typeof status === 'number') return status;
  }
  return undefined;
}
