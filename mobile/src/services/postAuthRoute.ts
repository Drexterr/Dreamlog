import AsyncStorage from '@react-native-async-storage/async-storage';
import { api } from '../api/client';

// The role the user picked on the auth screen ("I'm a user" / "I'm a therapist").
// Persisted so post-auth routing (which also runs from the root layout's
// SIGNED_IN listener and the accept-terms screen) agrees with the pick.
const ROLE_KEY = 'dreamlog_role';

export type AppRole = 'user' | 'therapist';

export const setAppRole = (role: AppRole): Promise<void> => AsyncStorage.setItem(ROLE_KEY, role);

export const getAppRole = async (): Promise<AppRole> =>
  (await AsyncStorage.getItem(ROLE_KEY)) === 'therapist' ? 'therapist' : 'user';

/**
 * Decides where a just-signed-in user should land.
 *
 * Order:
 * 1. Terms gate — if the current ToS version hasn't been accepted, everything
 *    else waits behind /accept-terms (fail-open when the check itself errors,
 *    so a flaky network never locks anyone out).
 * 2. Therapist role — existing therapist profile → dashboard; none yet →
 *    therapist registration.
 * 3. Normal user — onboarding if no goal set, tabs otherwise.
 */
export async function resolvePostAuthRoute(opts?: { skipTermsGate?: boolean }): Promise<string> {
  if (!opts?.skipTermsGate) {
    try {
      const terms = await api.getTerms();
      if (!terms.tos_accepted_at || terms.tos_version !== terms.current_version) {
        return '/accept-terms';
      }
    } catch {
      // fail-open: don't block sign-in on a terms lookup error
    }
  }

  const role = await getAppRole();
  if (role === 'therapist') {
    try {
      await api.therapistMe();
      return '/therapist';
    } catch {
      return '/therapist/register';
    }
  }

  const user = await api.me().catch(() => null);
  return user && !user.goal ? '/onboarding' : '/(tabs)';
}
