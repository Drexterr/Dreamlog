/**
 * End-to-end test mode configuration.
 *
 * When the app is built (or Metro is started) with `EXPO_PUBLIC_E2E=1`, the
 * root layout can skip the two slowest, least-deterministic gates in the E2E
 * suite — the onboarding walk and the interactive sign-in — so feature flows
 * start on an authenticated Home tab in a couple of seconds instead of ~20
 * scripted steps.
 *
 * How it works:
 *  - `E2E` true            → onboarding is treated as already complete
 *                            (no forced redirect to /onboarding).
 *  - `E2E_TOKEN` provided  → that Supabase access token is stored on startup
 *                            and the app boots authenticated as that user.
 *                            Supply a token minted for a seeded test account:
 *
 *      EXPO_PUBLIC_E2E=1 \
 *      EXPO_PUBLIC_E2E_TOKEN=<supabase_access_token> \
 *      npx expo start
 *
 * SAFETY: this is inert unless the env var is present at build/start time.
 * Production builds never set it, so there is no runtime bypass in the wild.
 * The token path only *uses* a token you already own — it does not forge one.
 */
const flag = (v: string | undefined) => v === '1' || v === 'true';

export const E2E: boolean = flag(process.env.EXPO_PUBLIC_E2E);

export const E2E_TOKEN: string | null =
  E2E && process.env.EXPO_PUBLIC_E2E_TOKEN
    ? String(process.env.EXPO_PUBLIC_E2E_TOKEN)
    : null;

/**
 * Deterministic AI note: the *backend* already supports canned reflections via
 * `STUB_AI_ANALYSIS=true` (see docs/ARCHITECTURE.md → "Dev vs Prod"). Point the
 * E2E build's `EXPO_PUBLIC_API_URL` at a backend running with that flag to get
 * stable, assertable reflection/therapy content without calling a live model.
 */
export const E2E_HINT_STUB_AI = 'Run the backend with STUB_AI_ANALYSIS=true';
