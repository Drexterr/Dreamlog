import 'react-native-gesture-handler';
import { useEffect, useRef, useState } from 'react';
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
import { api, storeToken } from '../src/api/client';
import { supabase, deepLinkReady } from '../src/lib/supabase';
import { ThemeProvider } from '../src/context/ThemeContext';
import { detectAndCacheRegion } from '../src/services/region';

const STRIPE_PK = process.env.EXPO_PUBLIC_STRIPE_PUBLISHABLE_KEY ?? '';

SplashScreen.preventAutoHideAsync();

export default function RootLayout() {
  const [ready, setReady] = useState(false);
  const [hasToken, setHasToken] = useState(false);
  const [needsOnboarding, setNeedsOnboarding] = useState(false);
  const [greetingName, setGreetingName] = useState<string | null>(null);
  const [showGreeting, setShowGreeting] = useState(false);
  const greetingOpacity = useRef(new Animated.Value(0)).current;
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

  // Check Supabase session and, if present, fetch profile to determine onboarding state.
  useEffect(() => {
    (async () => {
      try {
        // Wait for any deep link to be processed first so setSession() completes
        // before we read the session - avoids the race where a confirmation link
        // opens the app but getSession() runs before the tokens are stored.
        await deepLinkReady;
        const { data: { session } } = await supabase.auth.getSession();
        if (!session) {
          setHasToken(false);
          return;
        }
        // Sync the JWT into SecureStore so the axios interceptor picks it up.
        await storeToken(session.access_token);
        setHasToken(true);
        const [user] = await Promise.all([
          api.me(),
          detectAndCacheRegion(),
        ]);
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

  // Once fonts + auth check done, hide splash and redirect exactly once.
  useEffect(() => {
    if (!ready || (!fontsLoaded && !fontError)) return;
    SplashScreen.hideAsync();
    if (redirected.current) return;
    redirected.current = true;

    const seg0 = segments[0] as string;
    const inAuth = seg0 === 'auth';
    const inOnboarding = seg0 === 'onboarding';

    if (!hasToken && !inAuth) {
      router.replace('/auth');
    } else if (hasToken && needsOnboarding && !inOnboarding) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      router.replace('/onboarding' as any);
    } else if (hasToken && !needsOnboarding && (inAuth || inOnboarding)) {
      router.replace('/(tabs)');
    } else if (hasToken && !needsOnboarding && greetingName) {
      // Show greeting overlay for 2 seconds, then fade out.
      setShowGreeting(true);
      Animated.sequence([
        Animated.timing(greetingOpacity, { toValue: 1, duration: 400, useNativeDriver: true }),
        Animated.delay(1400),
        Animated.timing(greetingOpacity, { toValue: 0, duration: 400, useNativeDriver: true }),
      ]).start(() => setShowGreeting(false));
    }
  }, [ready, fontsLoaded, fontError]);

  // After startup, listen for auth state changes triggered by deep links (e.g. email
  // confirmation). This handles the case where the user taps the confirmation link and
  // Supabase fires SIGNED_IN after the initial redirect logic has already run.
  useEffect(() => {
    if (!ready) return;
    const { data: { subscription } } = supabase.auth.onAuthStateChange(async (event, session) => {
      if (event === 'SIGNED_IN' && session) {
        try {
          await storeToken(session.access_token);
          const user = await api.me();
          if (!user.goal) {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            router.replace('/onboarding' as any);
          } else {
            router.replace('/(tabs)');
          }
        } catch {
          // ignore - startup effect will handle recovery on next launch
        }
      } else if (event === 'SIGNED_OUT') {
        router.replace('/auth');
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
        <Slot />
        {showGreeting && greetingName ? (
          <Animated.View style={[styles.greetingOverlay, { opacity: greetingOpacity }]}>
            <Text style={styles.greetingText}>Hello, {greetingName}</Text>
          </Animated.View>
        ) : null}
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
