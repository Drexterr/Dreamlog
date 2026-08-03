import { Platform } from 'react-native';
import { router } from 'expo-router';
import { api } from '../api/client';

// Nudge types whose tap should land the user straight on the record screen -
// the shortest possible path from impulse to speaking.
const RECORD_NUDGE_TYPES = new Set(['morning_nudge', 'reengagement', 'streak_risk', 'checkin']);

// Plan-expiry nudges should land on the upgrade screen, not the recorder -
// the whole point of the push is to prompt a renewal.
const UPGRADE_NUDGE_TYPES = new Set(['plan_expiring_soon', 'plan_expired']);

function handleNotificationOpen(data?: Record<string, unknown>): void {
  const type = typeof data?.type === 'string' ? (data.type as string) : '';
  try {
    if (RECORD_NUDGE_TYPES.has(type)) {
      router.push('/record');
    } else if (UPGRADE_NUDGE_TYPES.has(type)) {
      router.push('/upgrade');
    }
  } catch {
    // Router not mounted yet (cold start) - the auth guard will land the user
    // on home; losing the deep link is acceptable, crashing is not.
  }
}

/**
 * Registers this device for push notifications and stores its FCM token
 * on the backend (POST /devices). Safe to call on every app start for an
 * authenticated user - the backend upserts on fcm_token.
 *
 * Best-effort by design: missing native module (Expo Go), missing iOS
 * Firebase config, denied permission, or no Play Services must never
 * break the app. Mirrors the fail-silent pattern of services/health.ts.
 */
export async function registerForPushNotifications(): Promise<boolean> {
  if (Platform.OS !== 'android' && Platform.OS !== 'ios') return false;

  try {
    const { getApp } = await import('@react-native-firebase/app');
    const messagingModule = await import('@react-native-firebase/messaging');
    const messaging = messagingModule.getMessaging(getApp());

    // Deep link: tapping a nudge opens the record screen directly.
    messagingModule.onNotificationOpenedApp(messaging, (msg) =>
      handleNotificationOpen(msg?.data as Record<string, unknown> | undefined),
    );
    messagingModule
      .getInitialNotification(messaging)
      .then((msg) => {
        if (!msg) return;
        // Cold start: give the router a moment to mount before navigating.
        setTimeout(() => handleNotificationOpen(msg.data as Record<string, unknown> | undefined), 600);
      })
      .catch(() => undefined);

    // iOS prompts the user; Android 13+ prompts for POST_NOTIFICATIONS,
    // older Android resolves as authorized without a prompt.
    const status = await messagingModule.requestPermission(messaging);
    const enabled =
      status === messagingModule.AuthorizationStatus.AUTHORIZED ||
      status === messagingModule.AuthorizationStatus.PROVISIONAL;
    if (!enabled) return false;

    if (Platform.OS === 'ios') {
      await messagingModule.registerDeviceForRemoteMessages(messaging);
    }

    const token = await messagingModule.getToken(messaging);
    if (!token) return false;

    await api.registerDevice(token, Platform.OS);

    // FCM rotates tokens occasionally; keep the backend in sync.
    messagingModule.onTokenRefresh(messaging, (newToken: string) => {
      api.registerDevice(newToken, Platform.OS as 'ios' | 'android').catch(() => undefined);
    });

    return true;
  } catch {
    return false;
  }
}
