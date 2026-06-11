import { Platform } from 'react-native';
import { api } from '../api/client';

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
