import { useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { useTheme } from '../../src/context/ThemeContext';
import { resetAndDetectRegion } from '../../src/services/region';
import type { RegionCurrency } from '../../src/services/region';

// ── Pricing data ──────────────────────────────────────────────────────────────

type SessionOption = {
  id: string;
  badge?: string;
  title: string;
  subtitle: string;
  priceInr: string;
  priceUsd: string;
  perSessionInr?: string;
  perSessionUsd?: string;
  saving?: string;
  features: string[];
  cta: string;
  highlight: boolean;
  isPlan?: boolean;
};

const SESSION_OPTIONS: SessionOption[] = [
  {
    id: 'single',
    title: 'Single Session',
    subtitle: 'Perfect for trying it out',
    priceInr: '₹499',
    priceUsd: '$4.99',
    features: [
      'Up to 1 hour',
      'Voice or text input',
      'AI companion grounded in your journal',
      'Post-session summary',
      'Crisis detection active',
    ],
    cta: 'Book a Session',
    highlight: false,
  },
  {
    id: 'pack5',
    badge: 'POPULAR',
    title: '5-Session Pack',
    subtitle: 'Commit to your wellbeing',
    priceInr: '₹1,999',
    priceUsd: '$19.99',
    perSessionInr: '₹400 per session',
    perSessionUsd: '$4.00 per session',
    saving: 'Save 20%',
    features: [
      '5 sessions, valid for 3 months',
      'Everything in Single Session',
      'Priority response time',
      'Session history & summaries',
    ],
    cta: 'Get 5 Sessions',
    highlight: true,
  },
  {
    id: 'pack12',
    title: '12-Session Pack',
    subtitle: 'For sustained growth',
    priceInr: '₹3,999',
    priceUsd: '$39.99',
    perSessionInr: '₹333 per session',
    perSessionUsd: '$3.33 per session',
    saving: 'Save 33%',
    features: [
      '12 sessions, valid for 6 months',
      'Everything in 5-Session Pack',
      'Monthly progress report',
      'Dedicated session themes',
    ],
    cta: 'Get 12 Sessions',
    highlight: false,
  },
  {
    id: 'pro',
    badge: 'BEST VALUE',
    title: 'DreamLog Pro',
    subtitle: '2 sessions included every month',
    priceInr: '₹499 / mo',
    priceUsd: '$14.99 / mo',
    perSessionInr: '≈₹250 per session',
    perSessionUsd: '≈$7.50 per session',
    features: [
      '2 therapy sessions per month',
      'Unlimited journal entries',
      'PDF export & Apple Health sync',
      'Full mood history & analytics',
      'All prompt modes',
    ],
    cta: 'Get Pro',
    highlight: false,
    isPlan: true,
  },
];

// ── Persona showcase ──────────────────────────────────────────────────────────

const PERSONAS = [
  { emoji: '🌿', name: 'Comforting', desc: 'Warm and validating' },
  { emoji: '🧠', name: 'Rational', desc: 'Structured & Socratic' },
  { emoji: '🔄', name: 'CBT', desc: 'Pattern-aware' },
  { emoji: '🧘', name: 'Mindful', desc: 'Grounding & present' },
];

// ── Option card ───────────────────────────────────────────────────────────────

function OptionCard({
  option,
  currency,
  colors,
  onPress,
}: {
  option: SessionOption;
  currency: RegionCurrency;
  colors: any;
  onPress: () => void;
}) {
  const price = currency === 'inr' ? option.priceInr : option.priceUsd;
  const perSession = currency === 'inr' ? option.perSessionInr : option.perSessionUsd;

  return (
    <TouchableOpacity
      style={[
        cardStyles.card,
        {
          backgroundColor: option.highlight ? colors.brandGlow : colors.card,
          borderColor: option.highlight ? colors.brand : colors.border,
          borderWidth: option.highlight ? 1.5 : 1,
        },
      ]}
      onPress={onPress}
      activeOpacity={0.85}
    >
      {/* Badge */}
      {option.badge && (
        <View style={[cardStyles.badge, { backgroundColor: option.highlight ? colors.brand : colors.purple600 }]}>
          <Text style={cardStyles.badgeText}>{option.badge}</Text>
        </View>
      )}

      {/* Title row */}
      <Text style={[cardStyles.title, { color: option.highlight ? colors.purple300 : colors.textPrimary }]}>
        {option.title}
      </Text>
      <Text style={[cardStyles.subtitle, { color: colors.textSecondary }]}>{option.subtitle}</Text>

      {/* Price */}
      <View style={cardStyles.priceRow}>
        <Text style={[cardStyles.price, { color: option.highlight ? colors.purple300 : colors.textPrimary }]}>
          {price}
        </Text>
        {option.saving && (
          <View style={[cardStyles.savingBadge, { backgroundColor: `${colors.brand}22` }]}>
            <Text style={[cardStyles.savingText, { color: colors.brand }]}>{option.saving}</Text>
          </View>
        )}
      </View>
      {perSession && (
        <Text style={[cardStyles.perSession, { color: colors.textMuted }]}>{perSession}</Text>
      )}

      {/* Divider */}
      <View style={[cardStyles.divider, { backgroundColor: colors.border }]} />

      {/* Features */}
      {option.features.map((f) => (
        <View key={f} style={cardStyles.featureRow}>
          <Text style={[cardStyles.featureTick, { color: colors.brand }]}>✓</Text>
          <Text style={[cardStyles.featureText, { color: colors.textSecondary }]}>{f}</Text>
        </View>
      ))}

      {/* CTA */}
      <TouchableOpacity
        style={[
          cardStyles.cta,
          {
            backgroundColor: option.highlight ? colors.brand : `${colors.brand}22`,
            borderColor: colors.brand,
            borderWidth: option.highlight ? 0 : 1,
          },
        ]}
        onPress={onPress}
        activeOpacity={0.85}
      >
        <Text style={[cardStyles.ctaText, { color: option.highlight ? '#fff' : colors.brand }]}>
          {option.cta}
        </Text>
      </TouchableOpacity>
    </TouchableOpacity>
  );
}

const cardStyles = StyleSheet.create({
  card: {
    borderRadius: 20,
    padding: 20,
    marginBottom: 16,
    position: 'relative',
    overflow: 'hidden',
  },
  badge: {
    alignSelf: 'flex-start',
    borderRadius: 6,
    paddingHorizontal: 8,
    paddingVertical: 3,
    marginBottom: 10,
  },
  badgeText: {
    fontSize: 10,
    fontFamily: 'Nunito_700Bold',
    color: '#fff',
    letterSpacing: 1,
  },
  title: {
    fontSize: 20,
    fontFamily: 'CormorantGaramond_600SemiBold',
    marginBottom: 2,
  },
  subtitle: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    marginBottom: 14,
  },
  priceRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    marginBottom: 2,
  },
  price: {
    fontSize: 28,
    fontFamily: 'CormorantGaramond_600SemiBold',
  },
  savingBadge: {
    borderRadius: 8,
    paddingHorizontal: 8,
    paddingVertical: 3,
  },
  savingText: {
    fontSize: 12,
    fontFamily: 'Nunito_700Bold',
  },
  perSession: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    marginBottom: 14,
  },
  divider: { height: 1, marginBottom: 12 },
  featureRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 8,
    marginBottom: 6,
  },
  featureTick: { fontSize: 13, fontFamily: 'Nunito_700Bold', marginTop: 1 },
  featureText: { flex: 1, fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 18 },
  cta: {
    borderRadius: 12,
    paddingVertical: 13,
    alignItems: 'center',
    marginTop: 14,
  },
  ctaText: {
    fontSize: 15,
    fontFamily: 'Nunito_700Bold',
  },
});

// ── Main screen ───────────────────────────────────────────────────────────────

export default function TherapyPricingScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const [currency, setCurrency] = useState<RegionCurrency | null>(null);

  useEffect(() => {
    resetAndDetectRegion().then(setCurrency).catch(() => setCurrency('usd'));
  }, []);

  const activeCurrency: RegionCurrency = currency ?? 'usd';

  const handleOptionPress = (option: SessionOption) => {
    if (option.isPlan) {
      router.push('/upgrade' as any);
    } else {
      // Navigate to persona picker to start a session
      router.push('/therapy/persona-picker' as any);
    }
  };

  if (currency === null) {
    return (
      <View style={[styles.container, { backgroundColor: colors.bg }]}>
        <SafeAreaView style={styles.center}>
          <ActivityIndicator color={colors.purple400} />
        </SafeAreaView>
      </View>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

          {/* Back */}
          <TouchableOpacity style={styles.backBtn} onPress={() => router.back()} activeOpacity={0.7}>
            <Text style={[styles.backText, { color: colors.textMuted }]}>← Back</Text>
          </TouchableOpacity>

          {/* Header */}
          <Text style={[styles.wordmark, { color: colors.purple300 }]}>Therapy Sessions</Text>
          <Text style={[styles.heading, { color: colors.textPrimary }]}>
            A space to talk,{'\n'}grounded in your journal
          </Text>
          <Text style={[styles.subheading, { color: colors.textSecondary }]}>
            AI-assisted voice conversations that know your emotional history. Not a therapy replacement - a thoughtful companion that meets you where you are.
          </Text>

          {/* Free trial badge */}
          <View style={[styles.freeBadge, { backgroundColor: colors.brandGlow, borderColor: colors.brand }]}>
            <Text style={[styles.freeBadgeText, { color: colors.purple300 }]}>
              ✦  Your first session is free
            </Text>
          </View>

          {/* Persona chips */}
          <Text style={[styles.sectionLabel, { color: colors.textSecondary }]}>CHOOSE YOUR COMPANION STYLE</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.personaScroll}>
            <View style={styles.personaRow}>
              {PERSONAS.map((p) => (
                <View key={p.name} style={[styles.personaChip, { backgroundColor: colors.card, borderColor: colors.border }]}>
                  <Text style={styles.personaEmoji}>{p.emoji}</Text>
                  <Text style={[styles.personaName, { color: colors.textPrimary }]}>{p.name}</Text>
                  <Text style={[styles.personaDesc, { color: colors.textMuted }]}>{p.desc}</Text>
                </View>
              ))}
            </View>
          </ScrollView>

          {/* Options */}
          {SESSION_OPTIONS.map((option) => (
            <OptionCard
              key={option.id}
              option={option}
              currency={activeCurrency}
              colors={colors}
              onPress={() => handleOptionPress(option)}
            />
          ))}

          {/* What's included */}
          <View style={[styles.infoBox, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
            <Text style={[styles.infoTitle, { color: colors.textPrimary }]}>Every session includes</Text>
            {[
              'Up to 60 minutes of conversation',
              'AI companion aware of your journal history',
              'Voice or text - your choice, any turn',
              'Two-stage crisis detection throughout',
              'Post-session summary you can revisit',
            ].map((item) => (
              <View key={item} style={styles.infoRow}>
                <Text style={[styles.infoTick, { color: colors.brand }]}>✓</Text>
                <Text style={[styles.infoText, { color: colors.textSecondary }]}>{item}</Text>
              </View>
            ))}
          </View>

          {/* Footer disclaimer */}
          <Text style={[styles.disclaimer, { color: colors.textMuted }]}>
            DreamLog Therapy Sessions are AI-assisted and are not a substitute for professional mental health care. If you are in crisis, please contact a crisis helpline immediately.
          </Text>

        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  scroll: {
    paddingHorizontal: 20,
    paddingTop: 16,
    paddingBottom: 60,
  },

  backBtn: { marginBottom: 20 },
  backText: { fontSize: 14, fontFamily: 'Nunito_400Regular' },

  wordmark: {
    fontFamily: 'CormorantGaramond_300Light',
    fontSize: 13,
    letterSpacing: 2,
    textTransform: 'uppercase',
    marginBottom: 10,
  },
  heading: {
    fontFamily: 'CormorantGaramond_300Light',
    fontSize: 34,
    lineHeight: 42,
    marginBottom: 12,
  },
  subheading: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 14,
    lineHeight: 22,
    marginBottom: 20,
  },

  freeBadge: {
    alignSelf: 'flex-start',
    borderWidth: 1,
    borderRadius: 10,
    paddingHorizontal: 14,
    paddingVertical: 7,
    marginBottom: 28,
  },
  freeBadgeText: {
    fontFamily: 'Nunito_700Bold',
    fontSize: 13,
  },

  sectionLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_700Bold',
    letterSpacing: 1.5,
    marginBottom: 12,
  },
  personaScroll: { marginBottom: 28 },
  personaRow: { flexDirection: 'row', gap: 10, paddingRight: 20 },
  personaChip: {
    borderWidth: 1,
    borderRadius: 14,
    padding: 14,
    alignItems: 'center',
    width: 100,
    gap: 4,
  },
  personaEmoji: { fontSize: 24, marginBottom: 2 },
  personaName: { fontSize: 13, fontFamily: 'Nunito_700Bold', textAlign: 'center' },
  personaDesc: { fontSize: 11, fontFamily: 'Nunito_400Regular', textAlign: 'center', lineHeight: 15 },

  infoBox: {
    borderWidth: 1,
    borderRadius: 16,
    padding: 18,
    marginBottom: 20,
    gap: 8,
  },
  infoTitle: {
    fontSize: 15,
    fontFamily: 'Nunito_700Bold',
    marginBottom: 6,
  },
  infoRow: { flexDirection: 'row', alignItems: 'flex-start', gap: 8 },
  infoTick: { fontSize: 13, fontFamily: 'Nunito_700Bold', marginTop: 1 },
  infoText: { flex: 1, fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 19 },

  disclaimer: {
    fontSize: 11,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 17,
    textAlign: 'center',
    marginTop: 4,
  },
});
