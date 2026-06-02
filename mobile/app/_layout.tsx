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
import { api, getToken } from '../src/api/client';
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

  // Check stored JWT and, if present, fetch profile to determine onboarding state.
  useEffect(() => {
    (async () => {
      try {
        const token = await getToken();
        if (!token) {
          setHasToken(false);
          return;
        }
        setHasToken(true);
        const [user] = await Promise.all([
          api.me(),
          detectAndCacheRegion(), // fire-and-forget; result cached in AsyncStorage
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
