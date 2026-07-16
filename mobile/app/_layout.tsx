import 'react-native-gesture-handler';
import { useCallback, useEffect, useRef, useState } from 'react';
import { View } from 'react-native';
import { Slot, SplashScreen, useRouter, useSegments } from 'expo-router';
import { GoogleSignin } from '@react-native-google-signin/google-signin';
import * as Sentry from '@sentry/react-native';

// Crash reporting. Fail-silent: no DSN (local dev, Expo Go) = fully disabled.
// sendDefaultPii stays false so journal/therapy content and user identity
// never leave the device inside an error event. Errors only — no tracing.
const SENTRY_DSN = process.env.EXPO_PUBLIC_SENTRY_DSN;
if (SENTRY_DSN) {
  Sentry.init({
    dsn: SENTRY_DSN,
    sendDefaultPii: false,
  });
}

GoogleSignin.configure({
  webClientId: process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID,
  scopes: ['email', 'profile'],
});
import { useFonts } from 'expo-font';
import {
  HankenGrotesk_300Light,
  HankenGrotesk_400Regular,
  HankenGrotesk_600SemiBold,
  HankenGrotesk_700Bold,
} from '@expo-google-fonts/hanken-grotesk';
import NetInfo from '@react-native-community/netinfo';
import { api, storeToken, clearToken } from '../src/api/client';
import { supabase, deepLinkReady } from '../src/lib/supabase';
import { ThemeProvider } from '../src/context/ThemeContext';
import { AuthContext } from '../src/context/AuthContext';
import { GuidedTourProvider } from '../src/context/GuidedTourContext';
import GuidedTour from '../src/components/GuidedTour';
import { detectAndCacheRegion, setRegionFromCountry } from '../src/services/region';
import { resolvePostAuthRoute } from '../src/services/postAuthRoute';
import { flush as flushOfflineQueue } from '../src/services/offlineQueue';
import { registerForPushNotifications } from '../src/services/push';
import { setupQuickActions } from '../src/services/quickActions';
import { checkForceUpdate } from '../src/services/version';
import { runStartupUpdateCheck } from '../src/services/updates';
import {
  hasCompletedOnboarding,
  markOnboardingDone,
  loadGuestPreferences,
  clearGuestPreferences,
} from '../src/services/guestStorage';
import ForceUpdateScreen from '../src/components/ForceUpdateScreen';
import AuthSheet from '../src/components/AuthSheet';
import BrandSplash from '../src/components/BrandSplash';
import { E2E, E2E_TOKEN } from '../src/config/e2e';
import type { VersionInfo } from '../src/types';

SplashScreen.preventAutoHideAsync();

function RootLayout() {
  const [ready, setReady] = useState(false);
  const [hasToken, setHasToken] = useState(false);
  const [needsOnboarding, setNeedsOnboarding] = useState(false);
  const [greetingName, setGreetingName] = useState<string | null>(null);
  const [showSplash, setShowSplash] = useState(false);
  const [forceUpdate, setForceUpdate] = useState<VersionInfo | null>(null);
  const [showAuthSheet, setShowAuthSheet] = useState(false);
  const afterAuthCallback = useRef<(() => void) | null>(null);
  const router = useRouter();
  const segments = useSegments();
  const redirected = useRef(false);

  const [fontsLoaded, fontError] = useFonts({
    // Erode (Fontshare, ITF FFL) — matches the website's heading face
    Erode_300Light: require('../assets/fonts/Erode-Light.ttf'),
    Erode_400Regular: require('../assets/fonts/Erode-Regular.ttf'),
    Erode_500Medium: require('../assets/fonts/Erode-Medium.ttf'),
    Erode_600SemiBold: require('../assets/fonts/Erode-Semibold.ttf'),
    HankenGrotesk_300Light,
    HankenGrotesk_400Regular,
    HankenGrotesk_600SemiBold,
    HankenGrotesk_700Bold,
  });

  // requestAuth: called by protected screens when a guest user tries an action.
  // Opens the auth sheet; on sign-in success the callback runs.
  const requestAuth = useCallback((afterAuth: () => void) => {
    afterAuthCallback.current = afterAuth;
    setShowAuthSheet(true);
  }, []);

  const closeAuthSheet = useCallback(() => {
    afterAuthCallback.current = null;
    setShowAuthSheet(false);
  }, []);

  // Force-update gate. Fail-open — checkForceUpdate resolves null on any error.
  useEffect(() => {
    checkForceUpdate().then(setForceUpdate);
  }, []);

  // OTA: on cold start, pull and apply the newest JS bundle (one reload max).
  // Fail-silent; no-op in dev/Expo Go. Logs under "[OTA]" for adb logcat/Metro.
  useEffect(() => {
    runStartupUpdateCheck();
  }, []);

  // Check Supabase session on startup. Never blocks on missing session.
  useEffect(() => {
    (async () => {
      try {
        // ── E2E bypass ──────────────────────────────────────────────────────
        // When built with EXPO_PUBLIC_E2E=1, skip onboarding routing. If a test
        // token is supplied, boot straight into an authenticated session so
        // feature flows don't have to script the sign-in each run. Inert unless
        // the env var is set (never true in production builds). See src/config/e2e.ts.
        if (E2E) {
          if (E2E_TOKEN) {
            await storeToken(E2E_TOKEN);
            setHasToken(true);
          }
          setNeedsOnboarding(false);
          await markOnboardingDone();
          setReady(true);
          return;
        }

        await deepLinkReady;
        const { data: { session } } = await supabase.auth.getSession();
        if (!session) {
          setHasToken(false);
          return;
        }
        await storeToken(session.access_token);
        setHasToken(true);
        const user = await api.me();
        if (user.country) {
          setRegionFromCountry(user.country).catch(() => {});
        } else {
          detectAndCacheRegion().catch(() => {});
        }
        // Sync any guest preferences saved during onboarding but not yet sent to the
        // backend. This happens when the user is already authenticated during onboarding —
        // no SIGNED_IN event fires, so the SIGNED_IN handler never syncs them.
        let effectiveGoal = user.goal;
        if (!effectiveGoal) {
          const prefs = await loadGuestPreferences();
          if (prefs.goal) {
            api.updateMe({
              goal: prefs.goal,
              ...(prefs.name     ? { preferred_name: prefs.name }  : {}),
              ...(prefs.ageRange ? { age_range: prefs.ageRange }   : {}),
              ...(prefs.country && prefs.country !== 'OTHER' ? { country: prefs.country } : {}),
            }).then(() => clearGuestPreferences()).catch(() => {});
            effectiveGoal = prefs.goal;
          }
        }
        setNeedsOnboarding(!effectiveGoal);
        if (effectiveGoal) {
          setGreetingName(user.preferred_name || user.name || null);
        }
      } catch {
        setHasToken(false);
      } finally {
        setReady(true);
      }
    })();
  }, []);

  // After fonts + auth check: hide splash, route once.
  useEffect(() => {
    if (!ready || (!fontsLoaded && !fontError)) return;
    (async () => {
      SplashScreen.hideAsync();
      if (redirected.current) return;
      redirected.current = true;

      const seg0 = segments[0] as string | undefined;
      const inAuth        = seg0 === 'auth';
      const inOnboarding  = seg0 === 'onboarding';
      // Root index = app hasn't navigated anywhere yet (blank screen on second restart)
      const atRootIndex   = !seg0 || seg0 === 'index';
      const onboardingDone = await hasCompletedOnboarding();

      if (!hasToken) {
        // Guest user path
        if (!onboardingDone) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          router.replace('/onboarding' as any);
        } else {
          if (inAuth || inOnboarding || atRootIndex) {
            // Returning guest lands on tabs
            router.replace('/(tabs)');
          }
          // Returning guests get the brand splash without the greeting line
          if (!E2E) setShowSplash(true);
        }
        // Otherwise they're already navigating freely
        return;
      }

      // Authenticated user path
      if (!onboardingDone) await markOnboardingDone();

      if (needsOnboarding && !onboardingDone && !inOnboarding) {
        // Only route to onboarding if the user hasn't locally completed it yet.
        // If onboardingDone=true but backend has no goal, the startup sync above will
        // fix it asynchronously — don't loop them back through onboarding.
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        router.replace('/onboarding' as any);
      } else if (!needsOnboarding && (inAuth || inOnboarding || atRootIndex)) {
        // Also redirect from root index — covers blank screen on second restart
        router.replace('/(tabs)');
      }

      // Brand splash — the Breath Line draws itself, "ode" rises, and for a
      // returning user the greeting fades in beneath. Skipped when heading to
      // onboarding (the brand means nothing yet) and in E2E runs.
      if (!E2E && !needsOnboarding) setShowSplash(true);
    })();
  }, [ready, fontsLoaded, fontError]);

  // FCM push registration
  useEffect(() => {
    if (!ready || !hasToken) return;
    registerForPushNotifications();
  }, [ready, hasToken]);

  // Home-screen quick action ("Record a moment") - registered for everyone,
  // guests included; the record flow itself handles auth gating.
  useEffect(() => {
    if (!ready) return;
    setupQuickActions();
  }, [ready]);

  // Flush offline queue on reconnect
  useEffect(() => {
    if (!ready || !hasToken) return;
    const unsubscribe = NetInfo.addEventListener((state) => {
      if (state.isConnected) flushOfflineQueue();
    });
    return () => unsubscribe();
  }, [ready, hasToken]);

  // Auth state changes (deep links, email confirmation, in-sheet sign-in, sign-out)
  useEffect(() => {
    if (!ready) return;
    const { data: { subscription } } = supabase.auth.onAuthStateChange(async (event, session) => {
      if (event === 'SIGNED_IN' && session) {
        try {
          await storeToken(session.access_token);
          setHasToken(true);
          registerForPushNotifications();

          // Sync any preferences collected during guest onboarding
          const prefs = await loadGuestPreferences();
          const hasPrefs = prefs.goal || prefs.name || prefs.ageRange || prefs.country;
          if (hasPrefs) {
            api.updateMe({
              ...(prefs.goal                                   ? { goal: prefs.goal }                : {}),
              ...(prefs.name                                   ? { preferred_name: prefs.name }      : {}),
              ...(prefs.ageRange                               ? { age_range: prefs.ageRange }       : {}),
              ...(prefs.country && prefs.country !== 'OTHER'  ? { country: prefs.country }          : {}),
            }).catch(() => {});
            await clearGuestPreferences();
          }

          if (afterAuthCallback.current) {
            // In-sheet sign-in: run the pending action, close sheet
            const cb = afterAuthCallback.current;
            afterAuthCallback.current = null;
            setShowAuthSheet(false);
            cb();
          } else {
            // Deep link / normal sign-in: terms gate → therapist routing → onboarding/tabs
            const dest = await resolvePostAuthRoute();
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            router.replace(dest as any);
          }
        } catch {
          // ignore
        }
      } else if (event === 'SIGNED_OUT') {
        setHasToken(false);
        clearToken().catch(() => {});
        // Stay on tabs as guest — don't force redirect to /auth
      }
    });
    return () => subscription.unsubscribe();
  }, [ready]);

  if (!ready || (!fontsLoaded && !fontError)) {
    return <View style={{ flex: 1, backgroundColor: '#18150f' }} />;
  }

  return (
    <ThemeProvider>
      <AuthContext.Provider value={{ isAuthenticated: hasToken, requestAuth }}>
        <GuidedTourProvider>
          <Slot />
          <GuidedTour />
          {showSplash ? (
            <BrandSplash name={greetingName} onDone={() => setShowSplash(false)} />
          ) : null}
          {forceUpdate ? <ForceUpdateScreen info={forceUpdate} /> : null}
          <AuthSheet
            visible={showAuthSheet}
            prompt="Sign in to continue"
            onClose={closeAuthSheet}
          />
        </GuidedTourProvider>
      </AuthContext.Provider>
    </ThemeProvider>
  );
}

// Sentry.wrap adds the error boundary + native crash context around the app.
// Skipped entirely when Sentry is not configured.
export default SENTRY_DSN ? Sentry.wrap(RootLayout) : RootLayout;

