// OTA update diagnostics + manual apply, wrapping the expo-updates JS API.
//
// Why this exists: the app ships with `checkAutomatically: ON_LOAD` +
// `fallbackToCacheTimeout: 0` (see app.json), which downloads a new update in
// the background and only swaps it in on the *next* cold start. That makes it
// look like "nothing happened" after an `eas update`. These helpers (a) log
// exactly what expo-updates is doing so it can be verified from the device /
// `adb logcat` / Metro, and (b) let us apply an update immediately.
//
// IMPORTANT: expo-updates is DISABLED in dev/debug builds and in Expo Go.
// `Updates.isEnabled` is false there, and OTA can never apply — that alone
// explains most "updates not arriving" reports. The diagnostic surfaces it.
import * as Updates from 'expo-updates';

const TAG = '[OTA]';

export interface UpdateSnapshot {
  /** false in dev/debug builds and Expo Go — OTA cannot work when false. */
  enabled: boolean;
  /** EAS channel this binary listens on (preview / production / …). */
  channel: string | null;
  runtimeVersion: string | null;
  /** ID of the update currently running, or null when running the embedded bundle. */
  updateId: string | null;
  createdAt: string | null;
  /** true when running the JS baked into the build (no OTA applied yet). */
  isEmbeddedLaunch: boolean;
  isEmergencyLaunch: boolean;
}

/** Reads the running app's update state. Safe to call anywhere; never throws. */
export function getUpdateSnapshot(): UpdateSnapshot {
  const read = <T,>(fn: () => T, fallback: T): T => {
    try { return fn(); } catch { return fallback; }
  };
  return {
    enabled:          read(() => Updates.isEnabled, false),
    channel:          read(() => Updates.channel ?? null, null),
    runtimeVersion:   read(() => Updates.runtimeVersion ?? null, null),
    updateId:         read(() => Updates.updateId ?? null, null),
    createdAt:        read(() => (Updates.createdAt ? Updates.createdAt.toISOString() : null), null),
    isEmbeddedLaunch: read(() => Updates.isEmbeddedLaunch, true),
    isEmergencyLaunch: read(() => Updates.isEmergencyLaunch ?? false, false),
  };
}

export type UpdateResult = 'applied' | 'up-to-date' | 'disabled' | 'error';
type LogFn = (line: string) => void;

/**
 * Manually check, download, and apply an update, narrating each step.
 * On success the app restarts via `reloadAsync()` (so this call won't return
 * when result === 'applied'). Used by the in-app diagnostic button.
 */
export async function checkAndApplyUpdate(log: LogFn = () => {}): Promise<UpdateResult> {
  const emit = (line: string) => { console.log(TAG, line); log(line); };

  if (!Updates.isEnabled) {
    emit('expo-updates is DISABLED in this build (dev / debug / Expo Go).');
    emit('OTA cannot apply here — you need a release build to receive updates.');
    return 'disabled';
  }

  emit(`channel = ${Updates.channel ?? '(none)'}`);
  emit(`runtime = ${Updates.runtimeVersion ?? '(none)'}`);
  emit(`running = ${Updates.updateId ?? 'embedded bundle'}`);

  try {
    emit('Checking for a new update…');
    const check = await Updates.checkForUpdateAsync();
    if (!check.isAvailable) {
      emit('✓ Already on the latest update. Nothing to download.');
      return 'up-to-date';
    }

    emit('New update found. Downloading…');
    const fetched = await Updates.fetchUpdateAsync();
    if (!fetched.isNew) {
      emit('Downloaded, but it matches what is already cached.');
      return 'up-to-date';
    }

    emit('Downloaded. Restarting to apply…');
    await Updates.reloadAsync();
    return 'applied'; // unreachable in practice — the app reloads above
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    emit(`✗ Update failed: ${msg}`);
    return 'error';
  }
}

/**
 * Startup check: download a newer update in the background and let
 * expo-updates apply it on the NEXT cold start (under the splash screen).
 *
 * We deliberately do NOT call `reloadAsync()` here. Reloading after the UI is
 * already on screen restarts the JS mid-session and briefly exposes the native
 * window background — the white flash users see on the launch after an OTA.
 * With `fallbackToCacheTimeout: 0` + `checkAutomatically: ON_LOAD` (see app.json)
 * a staged update is swapped in automatically on the next cold start, with no
 * flash. The trade-off (update lands one launch later) is invisible to users.
 *
 * Console-only logging. Fail-silent — never blocks app start.
 * Called from app/_layout.tsx after the auth/fonts gate.
 */
export async function runStartupUpdateCheck(): Promise<void> {
  try {
    if (!Updates.isEnabled) {
      console.log(TAG, 'startup: disabled in this build — skipping');
      return;
    }
    const check = await Updates.checkForUpdateAsync();
    console.log(
      TAG,
      `startup: channel=${Updates.channel} runtime=${Updates.runtimeVersion} available=${check.isAvailable}`,
    );
    if (!check.isAvailable) return;

    const fetched = await Updates.fetchUpdateAsync();
    if (fetched.isNew) {
      // Downloaded + staged. Do NOT reload now — it applies on the next cold
      // start under the splash, avoiding the mid-session white flash.
      console.log(TAG, 'startup: new update downloaded — will apply on next cold start');
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    console.log(TAG, `startup: check failed — ${msg}`);
  }
}
