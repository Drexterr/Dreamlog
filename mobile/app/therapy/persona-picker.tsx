import { useEffect, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  SafeAreaView,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { type TherapyPersona, PERSONA_META } from '../../src/types';
import { getCachedRegion, THERAPY_SESSION_PRICE } from '../../src/services/region';

const PERSONAS: TherapyPersona[] = ['comforting', 'rational', 'cbt', 'mindful'];

export default function PersonaPickerScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const [selected, setSelected] = useState<TherapyPersona>('comforting');
  const [starting, setStarting] = useState(false);
  const [priceDisplay, setPriceDisplay] = useState(THERAPY_SESSION_PRICE.inr);

  useEffect(() => {
    getCachedRegion().then((r) => {
      setPriceDisplay(THERAPY_SESSION_PRICE[r ?? 'usd']);
    });
  }, []);

  const handleStart = async () => {
    setStarting(true);
    try {
      const session = await api.startTherapySession(selected);
      router.replace({ pathname: '/therapy/session', params: { id: session.id } } as any);
    } catch (err: any) {
      const status = err?.response?.status;
      if (status === 402) {
        // No session credits — send straight to the therapy pricing screen.
        router.replace('/therapy/pricing' as any);
      } else {
        Alert.alert('Could not start session', 'Please try again in a moment.');
      }
    } finally {
      setStarting(false);
    }
  };

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.bg }]}>
      <ScrollView contentContainerStyle={styles.container}>
        <View style={styles.header}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Choose your companion</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
            Each companion has a different style. You can switch next session.
          </Text>
        </View>

        <View style={styles.cards}>
          {PERSONAS.map((p) => {
            const meta = PERSONA_META[p];
            const isSelected = selected === p;
            return (
              <TouchableOpacity
                key={p}
                style={[
                  styles.card,
                  {
                    backgroundColor: colors.card,
                    borderColor: isSelected ? colors.brand : colors.border,
                    borderWidth: isSelected ? 2 : 1,
                  },
                ]}
                onPress={() => setSelected(p)}
                activeOpacity={0.7}
              >
                <View style={styles.cardRow}>
                  <Text style={styles.cardEmoji}>{meta.emoji}</Text>
                  <View style={styles.cardText}>
                    <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>
                      {meta.label}
                    </Text>
                    <Text style={[styles.cardTagline, { color: colors.textSecondary }]}>
                      {meta.tagline}
                    </Text>
                  </View>
                  {isSelected && (
                    <View style={[styles.selectedDot, { backgroundColor: colors.brand }]} />
                  )}
                </View>
              </TouchableOpacity>
            );
          })}
        </View>

        <TouchableOpacity
          style={[styles.startBtn, { backgroundColor: starting ? colors.border : colors.brand }]}
          onPress={handleStart}
          disabled={starting}
          activeOpacity={0.8}
        >
          {starting ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.startBtnText}>
              Begin with {PERSONA_META[selected].label}
            </Text>
          )}
        </TouchableOpacity>

        <Text style={[styles.note, { color: colors.textMuted }]}>
          First session free · {priceDisplay}/session · Included in Pro (2/month)
        </Text>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1 },
  container: { padding: 24, paddingBottom: 48 },
  header: { marginBottom: 28 },
  title: { fontSize: 26, fontFamily: 'CormorantGaramond_600SemiBold', marginBottom: 8 },
  subtitle: { fontSize: 15, fontFamily: 'Nunito_400Regular', lineHeight: 22 },
  cards: { gap: 12, marginBottom: 32 },
  card: {
    borderRadius: 14,
    padding: 16,
  },
  cardRow: { flexDirection: 'row', alignItems: 'center', gap: 14 },
  cardEmoji: { fontSize: 28 },
  cardText: { flex: 1 },
  cardLabel: { fontSize: 16, fontFamily: 'Nunito_700Bold', marginBottom: 2 },
  cardTagline: { fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 18 },
  selectedDot: { width: 10, height: 10, borderRadius: 5 },
  startBtn: {
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 12,
  },
  startBtnText: { color: '#fff', fontSize: 17, fontFamily: 'Nunito_700Bold' },
  note: { fontSize: 12, fontFamily: 'Nunito_400Regular', textAlign: 'center' },
});
