import 'react-native-gesture-handler';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Animated, StyleSheet, Text, View } from 'react-native';
import { Slot, SplashScreen, useRouter, useSegments } from 'expo-router';
import { StripeProvider } from '@stripe/stripe-react-native';
import {
  useFonts,
  CormorantGaramond_300Light,
  CormorantGaramond_400Regular,
  CormorantGaramond_500Medium,
  CormorantGaramond_600SemiBold,
} from '@expo-google-fonts/cormorant-garamond';
import {
  Nunito_300Light,
  Nunito_400Regular,
  Nunito_600SemiBold,
  Nunito_700Bold,
} from '@expo-google-fonts/nunito';
import NetInfo from '@react-native-community/netinfo';
import { api, storeToken, clearToken } from '../src/api/client';
import { supabase, deepLinkReady } from '../src/lib/supabase';
import { ThemeProvider } from '../src/context/ThemeContext';
import { AuthContext } from '../src/context/AuthContext';
import { detectAndCacheRegion, setRegionFromCountry } from '../src/services/region';
import { flush as flushOfflineQueue } from '../src/services/offlineQueue';
import { registerForPushNotifications } from '../src/services/push';
import { checkForceUpdate } from '../src/services/version';
import {
  hasCompletedOnboarding,
  markOnboardingDone,
  loadGuestPreferences,
  clearGuestPreferences,
} from '../src/services/guestStorage';
import ForceUpdateScreen from '../src/components/ForceUpdateScreen';
import AuthSheet from '../src/components/AuthSheet';
import type { VersionInfo } from '../src/types';

const STRIPE_PK = process.env.EXPO_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? '';

SplashScreen.preventAutoHideAsync();

export default function RootLayout() {
  const [ready, setReady] = useState(false);
  const [hasToken, setHasToken] = useState(false);
  const [needsOnboarding, setNeedsOnboarding] = useState(false);
  const [greetingName, setGreetingName] = useState<string | null>(null);
  const [showGreeting, setShowGreeting] = useState(false);
  const [forceUpdate, setForceUpdate] = useState<VersionInfo | null>(null);
  const [showAuthSheet, setShowAuthSheet] = useState(false);
  const greetingOpacity = useRef(new Animated.Value(0)).current;
  const afterAuthCallback = useRef<(() => void) | null>(null);
  const router = useRouter();
  const segments = useSegments();
  const redirected = useRef(false);

  const [fontsLoaded, fontError] = useFonts({
    CormorantGaramond_300Light,
    CormorantGaramond_400Regular,
    CormorantGaramond_500Medium,
    CormorantGaramond_600SemiBold,
    Nunito_300Light,
    Nunito_400Regular,
    Nunito_600SemiBold,
    Nunito_700Bold,
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

  // Check Supabase session on startup. Never blocks on missing session.
  useEffect(() => {
    (async () => {
      try {
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
        setNeedsOnboarding(!user.goal);
        if (user.goal) {
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

      const seg0 = segments[0] as string;
      const inAuth        = seg0 === 'auth';
      const inOnboarding  = seg0 === 'onboarding';
      const onboardingDone = await hasCompletedOnboarding();

      if (!hasToken) {
        // Guest user path
        if (!onboardingDone) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          router.replace('/onboarding' as any);
        } else if (inAuth || inOnboarding) {
          // Returning guest lands on tabs
          router.replace('/(tabs)');
        }
        // Otherwise they're already navigating freely
        return;
      }

      // Authenticated user path
      if (!onboardingDone) await markOnboardingDone();

      if (needsOnboarding && !inOnboarding) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        router.replace('/onboarding' as any);
      } else if (!needsOnboarding && (inAuth || inOnboarding)) {
        router.replace('/(tabs)');
      } else if (!needsOnboarding && greetingName) {
        setShowGreeting(true);
        Animated.sequence([
          Animated.timing(greetingOpacity, { toValue: 1, duration: 400, useNativeDriver: true }),
          Animated.delay(1400),
          Animated.timing(greetingOpacity, { toValue: 0, duration: 400, useNativeDriver: true }),
        ]).start(() => setShowGreeting(false));
      }
    })();
  }, [ready, fontsLoaded, fontError]);

  // FCM push registration
  useEffect(() => {
    if (!ready || !hasToken) return;
    registerForPushNotifications();
  }, [ready, hasToken]);

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
            // Deep link / normal sign-in: navigate to tabs
            const user = await api.me().catch(() => null);
            if (user && !user.goal) {
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              router.replace('/onboarding' as any);
            } else {
              router.replace('/(tabs)');
            }
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
    return <View style={{ flex: 1, backgroundColor: '#0f0c1e' }} />;
  }

  return (
    <StripeProvider publishableKey={STRIPE_PK} merchantIdentifier="merchant.com.dreamlog">
      <ThemeProvider>
        <AuthContext.Provider value={{ isAuthenticated: hasToken, requestAuth }}>
          <Slot />
          {showGreeting && greetingName ? (
            <Animated.View style={[styles.greetingOverlay, { opacity: greetingOpacity }]}>
              <Text style={styles.greetingText}>Hello, {greetingName}</Text>
            </Animated.View>
          ) : null}
          {forceUpdate ? <ForceUpdateScreen info={forceUpdate} /> : null}
          <AuthSheet
            visible={showAuthSheet}
            prompt="Sign in to continue"
            onClose={closeAuthSheet}
          />
        </AuthContext.Provider>
      </ThemeProvider>
    </StripeProvider>
  );
}

const styles = StyleSheet.create({
  greetingOverlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: '#0f0c1e',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 999,
  },
  greetingText: {
    fontFamily: 'CormorantGaramond_300Light',
    fontSize: 36,
    color: '#e2d9f3',
    letterSpacing: 1,
  },
});
