import { Platform } from 'react-native';
import { router } from 'expo-router';

/**
 * Registers home-screen quick actions (long-press the app icon → "Record").
 * On Android the action can be dragged out and pinned as a standalone
 * one-tap record icon on the home screen; on iOS it lives in the icon's
 * context menu.
 *
 * Best-effort by design, mirroring services/push.ts: the native module is
 * absent in Expo Go and in builds made before it was installed, and that
 * must never crash the app.
 */
export async function setupQuickActions(): Promise<void> {
  if (Platform.OS !== 'android' && Platform.OS !== 'ios') return;

  try {
    const QuickActions = await import('expo-quick-actions');

    if (!(await QuickActions.isSupported())) return;

    await QuickActions.setItems([
      {
        id: 'record',
        title: 'Record a moment',
        subtitle: 'One quiet minute is enough',
        icon: Platform.OS === 'ios' ? 'symbol:mic.fill' : undefined,
        params: { href: '/record' },
      },
    ]);

    const handle = (action?: { params?: Record<string, unknown> | null }) => {
      const href = action?.params?.href;
      if (typeof href !== 'string') return;
      try {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        router.push(href as any);
      } catch {
        // Router not mounted yet - losing the shortcut hop is acceptable.
      }
    };

    // Cold start via a quick action: give the router a moment to mount.
    if (QuickActions.initial) {
      const initial = QuickActions.initial;
      setTimeout(() => handle(initial), 600);
    }
    QuickActions.addListener(handle);
  } catch {
    // Native module unavailable (Expo Go / old build) - silently skip.
  }
}
