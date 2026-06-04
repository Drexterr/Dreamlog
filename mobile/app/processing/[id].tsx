import { useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  Animated,
  StyleSheet,
  
  StatusBar,
  TouchableOpacity,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { writeMindfulSession } from '../../src/services/health';

const MESSAGES = [
  'Listening to your words…',
  'Finding the threads…',
  'Sitting with what you shared…',
  'Almost ready…',
];

const POLL_INTERVAL_MS = 3000;
const MAX_POLLS = 60;

export default function ProcessingScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { colors } = useTheme();
  const [msgIdx, setMsgIdx] = useState(0);
  const [failed, setFailed] = useState(false);
  const pollCount = useRef(0);

  const scaleAnim = useRef(new Animated.Value(1)).current;
  const opacityAnim = useRef(new Animated.Value(0.35)).current;

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.parallel([
          Animated.timing(scaleAnim, { toValue: 1.35, duration: 3000, useNativeDriver: true }),
          Animated.timing(opacityAnim, { toValue: 0.85, duration: 3000, useNativeDriver: true }),
        ]),
        Animated.parallel([
          Animated.timing(scaleAnim, { toValue: 1, duration: 3000, useNativeDriver: true }),
          Animated.timing(opacityAnim, { toValue: 0.35, duration: 3000, useNativeDriver: true }),
        ]),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, []);

  useEffect(() => {
    const iv = setInterval(() => {
      setMsgIdx((i) => (i + 1) % MESSAGES.length);
    }, 4000);
    return () => clearInterval(iv);
  }, []);

  useEffect(() => {
    if (!id) return;

    const iv = setInterval(async () => {
      pollCount.current += 1;
      if (pollCount.current > MAX_POLLS) {
        clearInterval(iv);
        setFailed(true);
        return;
      }
      try {
        const entry = await api.getEntry(id);
        if (entry.status === 'completed') {
          clearInterval(iv);
          const endDate = new Date();
          const durationSec = entry.duration_sec ?? 300;
          const startDate = new Date(endDate.getTime() - durationSec * 1000);
          writeMindfulSession({ startDate, endDate });
          router.replace(`/reflection/${id}`);
        } else if (entry.status === 'failed') {
          clearInterval(iv);
          setFailed(true);
        }
      } catch {
        // network blip — keep trying
      }
    }, POLL_INTERVAL_MS);

    return () => clearInterval(iv);
  }, [id]);

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={styles.safe}>
        {failed ? (
          <View style={styles.center}>
            <Text style={styles.errorIcon}>⚠</Text>
            <Text style={[styles.errorTitle, { color: colors.textPrimary }]}>Something went wrong</Text>
            <Text style={[styles.errorSub, { color: colors.textSecondary }]}>Transcription failed. Please try recording again.</Text>
            <TouchableOpacity
              style={[styles.retryBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
              onPress={() => router.replace('/(tabs)')}
              activeOpacity={0.8}
            >
              <Text style={[styles.retryText, { color: colors.purple300 }]}>Back home</Text>
            </TouchableOpacity>
          </View>
        ) : (
          <View style={styles.center}>
            <View style={styles.orbWrap}>
              <Animated.View
                style={[
                  styles.orbOuter,
                  { backgroundColor: colors.purple700, transform: [{ scale: scaleAnim }], opacity: opacityAnim },
                ]}
              />
              <View style={[styles.orbInner, { backgroundColor: colors.purple500 }]} />
            </View>

            <Text style={[styles.message, { color: colors.textPrimary }]}>{MESSAGES[msgIdx]}</Text>
            <Text style={[styles.sub, { color: colors.textMuted }]}>This usually takes 10–20 seconds</Text>
          </View>
        )}
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  safe: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 32, padding: 40 },

  orbWrap: { width: 130, height: 130, alignItems: 'center', justifyContent: 'center' },
  orbOuter: {
    position: 'absolute',
    width: 110,
    height: 110,
    borderRadius: 55,
  },
  orbInner: {
    width: 40,
    height: 40,
    borderRadius: 20,
    opacity: 0.9,
  },

  message: {
    fontSize: 17,
    fontFamily: 'CormorantGaramond_300Light',
    textAlign: 'center',
    paddingHorizontal: 20,
  },
  sub: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
  },

  errorIcon: { fontSize: 32 },
  errorTitle: {
    fontSize: 20,
    fontFamily: 'CormorantGaramond_400Regular',
    textAlign: 'center',
  },
  errorSub: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
    lineHeight: 22,
  },
  retryBtn: {
    borderRadius: 14,
    borderWidth: 1,
    paddingVertical: 14,
    paddingHorizontal: 32,
  },
  retryText: {
    fontSize: 15,
    fontFamily: 'Nunito_600SemiBold',
  },
});
