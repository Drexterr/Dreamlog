/**
 * Apple Health (HealthKit) and Google Fit integration.
 *
 * After each completed journal entry we write a MindfulSession event so the
 * user gets credit in Apple Health's Mindfulness minutes and Google Fit's
 * mindfulness goals.
 *
 * The integration is opt-in and fails silently - a HealthKit error never
 * interrupts the main app flow.
 */

import { Platform } from 'react-native';

// ── iOS / HealthKit ──────────────────────────────────────────────────────────

let AppleHealthKit: typeof import('react-native-health').default | null = null;

try {
  // Dynamic import so the module is only resolved on iOS native builds.
  // On Android or web this require will throw and we fall back gracefully.
  const pkg = require('react-native-health');
  AppleHealthKit = pkg.default;
} catch {
  // android / web / simulator without native HealthKit
}

let hkInitialized = false;

async function initHealthKit(): Promise<boolean> {
  if (Platform.OS !== 'ios' || !AppleHealthKit) return false;
  if (hkInitialized) return true;

  return new Promise((resolve) => {
    const permissions = {
      permissions: {
        read: [],
        write: ['MindfulSession'],
      },
    };

    AppleHealthKit!.initHealthKit(permissions as any, (err: string) => {
      if (err) {
        resolve(false);
        return;
      }
      hkInitialized = true;
      resolve(true);
    });
  });
}

// ── Public API ────────────────────────────────────────────────────────────────

export interface MindfulSessionOptions {
  startDate: Date;
  endDate: Date;
}

/**
 * Writes a MindfulSession event to Apple Health (iOS) or Google Fit (Android).
 * Always resolves - never throws into the caller.
 */
export async function writeMindfulSession(opts: MindfulSessionOptions): Promise<void> {
  try {
    if (Platform.OS === 'ios') {
      await writeHealthKitSession(opts);
    } else if (Platform.OS === 'android') {
      await writeGoogleFitSession(opts);
    }
  } catch {
    // Health writes are best-effort; never surface errors to the user.
  }
}

async function writeHealthKitSession(opts: MindfulSessionOptions): Promise<void> {
  const ready = await initHealthKit();
  if (!ready || !AppleHealthKit) return;

  return new Promise((resolve) => {
    const payload = {
      startDate: opts.startDate.toISOString(),
      endDate: opts.endDate.toISOString(),
      value: 0,
    };
    AppleHealthKit!.saveMindfulSession(payload, (err: string) => {
      // err is non-null on failure; resolve either way.
      void err;
      resolve();
    });
  });
}

async function writeGoogleFitSession(opts: MindfulSessionOptions): Promise<void> {
  // Google Fit requires @react-native-community/google-fit and Google Sign-In.
  // The pattern is the same: init → save activity session with activityType = MEDITATION.
  // Stubbed for now - implement when adding Google Fit credentials.
  void opts;
}
