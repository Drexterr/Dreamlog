import { useEffect, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { type TherapyPersona, PERSONA_META } from '../../src/types';
import { getCachedRegion, THERAPY_SESSION_PRICE } from '../../src/services/region';

const PERSONAS: TherapyPersona[] = ['comforting', 'rational', 'cbt', 'mindful'];

// Per-persona accent colors — each companion has its own visual identity
const PERSONA_ACCENT: Record<TherapyPersona, string> = {
  comforting: '#C4A06A',
  rational:   '#6A9EC4',
  cbt:        '#7AAA88',
  mindful:    '#9A8AC0',
};

// Short descriptive lines that feel more human than a feature spec
const PERSONA_DESC: Record<TherapyPersona, string> = {
  comforting: 'Starts with how you feel, not what you think.',
  rational:   'Asks the questions you haven\'t asked yourself.',
  cbt:        'Spots the story you\'re telling yourself.',
  mindful:    'Brings you back to right now.',
};

export default function PersonaPickerScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const [selected, setSelected] = useState<TherapyPersona>('comforting');
  const [starting, setStarting] = useState(false);
  const [priceDisplay, setPriceDisplay] = useState(THERAPY_SESSION_PRICE.inr);

  useEffect(() => {
    getCachedRegion().then((r) => setPriceDisplay(THERAPY_SESSION_PRICE[r ?? 'usd']));
  }, []);

  const handleStart = async () => {
    setStarting(true);
    try {
      const session = await api.startTherapySession(selected);
      router.replace({ pathname: '/therapy/session', params: { id: session.id } } as any);
    } catch (err: any) {
      const status = err?.response?.status;
      if (status === 402) {
        router.push('/therapy/pricing' as any);
      } else {
        Alert.alert('Could not start session', 'Please try again in a moment.');
      }
    } finally {
      setStarting(false);
    }
  };

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.bg }]}>
      <TouchableOpacity style={styles.backBtn} onPress={() => router.back()} activeOpacity={0.7}>
        <Text style={[styles.backBtnText, { color: colors.textMuted }]}>Back</Text>
      </TouchableOpacity>

      <ScrollView contentContainerStyle={styles.container} showsVerticalScrollIndicator={false}>
        <View style={styles.header}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Who do you want{'\n'}with you today?</Text>
          <Text style={[styles.subtitle, { color: colors.textMuted }]}>
            You can change this next time.
          </Text>
        </View>

        <View style={styles.cards}>
          {PERSONAS.map((p) => {
            const meta = PERSONA_META[p];
            const accent = PERSONA_ACCENT[p];
            const isSelected = selected === p;
            return (
              <TouchableOpacity
                key={p}
                style={[
                  styles.card,
                  {
                    backgroundColor: isSelected ? `${accent}12` : colors.card,
                    borderColor: isSelected ? accent : colors.border,
                  },
                ]}
                onPress={() => setSelected(p)}
                activeOpacity={0.75}
              >
                {/* Left accent bar */}
                <View style={[
                  styles.cardBar,
                  { backgroundColor: isSelected ? accent : 'transparent' },
                ]} />

                <View style={styles.cardBody}>
                  <View style={styles.cardTop}>
                    <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>
                      {meta.label}
                    </Text>
                    {isSelected && (
                      <View style={[styles.checkmark, { borderColor: accent }]}>
                        <View style={[styles.checkmarkInner, { backgroundColor: accent }]} />
                      </View>
                    )}
                  </View>
                  <Text style={[styles.cardDesc, { color: colors.textSecondary }]}>
                    {PERSONA_DESC[p]}
                  </Text>
                </View>
              </TouchableOpacity>
            );
          })}
        </View>

        <TouchableOpacity
          style={[styles.startBtn, {
            backgroundColor: starting ? colors.border : PERSONA_ACCENT[selected],
          }]}
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
          First session free  ·  {priceDisplay}/session
        </Text>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1 },
  backBtn: { paddingHorizontal: 20, paddingTop: 12, paddingBottom: 4 },
  backBtnText: { fontSize: 13, fontFamily: 'Nunito_400Regular' },
  container: { paddingHorizontal: 20, paddingBottom: 48, paddingTop: 8 },

  header: { marginBottom: 28 },
  title: {
    fontSize: 28,
    fontFamily: 'CormorantGaramond_300Light',
    lineHeight: 36,
    marginBottom: 8,
  },
  subtitle: { fontSize: 13, fontFamily: 'Nunito_400Regular' },

  cards: { gap: 10, marginBottom: 32 },
  card: {
    borderRadius: 12,
    borderWidth: 1,
    flexDirection: 'row',
    overflow: 'hidden',
  },
  cardBar: {
    width: 3,
  },
  cardBody: {
    flex: 1,
    padding: 16,
    gap: 5,
  },
  cardTop: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  cardLabel: { fontSize: 15, fontFamily: 'Nunito_700Bold' },
  checkmark: {
    width: 18,
    height: 18,
    borderRadius: 9,
    borderWidth: 1.5,
    alignItems: 'center',
    justifyContent: 'center',
  },
  checkmarkInner: {
    width: 9,
    height: 9,
    borderRadius: 4.5,
  },
  cardDesc: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 19,
  },

  startBtn: {
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 12,
  },
  startBtnText: { color: '#fff', fontSize: 16, fontFamily: 'Nunito_700Bold' },
  note: { fontSize: 12, fontFamily: 'Nunito_400Regular', textAlign: 'center' },
});
