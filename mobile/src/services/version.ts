// Force-update gate: compares the installed app version against the backend's
// minimum_version (GET /version). Fail-open by design - if the check cannot
// complete (offline, old backend, timeout), the app must launch normally.
import { Linking, Platform } from 'react-native';
import Constants from 'expo-constants';
import { api } from '../api/client';
import { VersionInfo } from '../types';

// Play Store deep link; falls back to the https listing if the store app is missing.
const PLAY_MARKET_URL = 'market://details?id=com.dreamlog.app';

/** Returns true when `current` is strictly below `minimum`. Tolerates uneven
 *  segment counts ("1.2" vs "1.2.0") and non-numeric input (treated as 0). */
export function isVersionBelow(current: string, minimum: string): boolean {
  const a = current.split('.').map((p) => parseInt(p, 10) || 0);
  const b = minimum.split('.').map((p) => parseInt(p, 10) || 0);
  const len = Math.max(a.length, b.length);
  for (let i = 0; i < len; i++) {
    const ai = a[i] ?? 0;
    const bi = b[i] ?? 0;
    if (ai < bi) return true;
    if (ai > bi) return false;
  }
  return false;
}

/** Fetches /version and returns the VersionInfo when the installed app is
 *  below the minimum, or null when the app is up to date or the check fails. */
export async function checkForceUpdate(): Promise<VersionInfo | null> {
  try {
    const installed = Constants.expoConfig?.version;
    if (!installed) return null;
    const info = await api.getVersion();
    return isVersionBelow(installed, info.minimum_version) ? info : null;
  } catch {
    return null; // fail-open: never block the app on a failed check
  }
}

/** Opens the platform's store listing. Prefers the server-provided URL so it
 *  can be corrected after launch without shipping a new binary. */
export async function openStoreListing(info: VersionInfo): Promise<void> {
  try {
    if (Platform.OS === 'ios') {
      if (info.ios_store_url) await Linking.openURL(info.ios_store_url);
      return;
    }
    if (info.android_store_url) {
      // Try the market:// deep link first so the Play Store app opens directly.
      const canOpenMarket = await Linking.canOpenURL(PLAY_MARKET_URL);
      await Linking.openURL(canOpenMarket ? PLAY_MARKET_URL : info.android_store_url);
    }
  } catch {
    // Opening the store is best-effort; the gate stays up either way.
  }
}
