import { useCallback, useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { useTheme } from '../../src/context/ThemeContext';
import { api } from '../../src/api/client';
import {
  purchaseTherapyCredit,
  finishPurchase,
  isUserCancelled,
  THERAPY_CREDIT_PRODUCT_IDS,
} from '../../src/services/iap';
import type { TherapyCreditKind } from '../../src/services/iap';
import type { BillingPlanResponse } from '../../src/types';
import {
  resetAndDetectRegion,
  THERAPY_SESSION_PRICE,
  THERAPY_MEMBER_SESSION_PRICE,
  PLAN_PRICE_SHORT,
} from '../../src/services/region';
import type { RegionCurrency } from '../../src/services/region';
import { T, therapyPricingCtaID } from '../../src/testIDs';
import { Palette } from '../../src/theme';
import type { ThemeColors } from '../../src/theme';

// ── Pricing data ──────────────────────────────────────────────────────────────

type SessionOption = {
  id: string;
  badge?: string;
  title: string;
  subtitle: string;
  price: string;
  perSession?: string;
  saving?: string;
  features: string[];
  cta: string;
  highlight: boolean;
  isPlan?: boolean;
  // Set for a pay-per-use session: which consumable SKU it buys.
  creditKind?: TherapyCreditKind;
};

// All prices render in the user's currency only (location asked at account
// creation: India → INR, Europe → EUR, else USD). Canonical: docs/PRICING.md.
// Pay-per-use sessions are single prepaid credits (no packs - see PRICING.md):
// Pro members buy at the member price (its own SKU, redeemable only on Pro),
// everyone else at the standalone price.
function buildSessionOptions(currency: RegionCurrency, isPro: boolean): SessionOption[] {
  const memberPrice = THERAPY_MEMBER_SESSION_PRICE[currency];
  const single: SessionOption = isPro
    ? {
        id: 'single',
        badge: 'MEMBER PRICE',
        title: 'Extra session',
        subtitle: 'For when you want more than your included session',
        price: memberPrice,
        perSession: `Member price - ${THERAPY_SESSION_PRICE[currency]} without Pro`,
        features: [
          'One session, up to 60 minutes',
          'Use it whenever you like - it does not expire',
          'One-time purchase, no subscription',
        ],
        cta: `Buy a session · ${memberPrice}`,
        highlight: false,
        creditKind: 'member',
      }
    : {
        id: 'single',
        title: 'Single session',
        subtitle: 'Pay once, talk when you are ready',
        price: THERAPY_SESSION_PRICE[currency],
        perSession: `Pro members pay ${memberPrice}`,
        features: [
          'One session, up to 60 minutes',
          'Use it whenever you like - it does not expire',
          'One-time purchase, no subscription',
        ],
        cta: `Buy a session · ${THERAPY_SESSION_PRICE[currency]}`,
        highlight: false,
        creditKind: 'single',
      };
  if (isPro) return [single];
  return [
    single,
    {
      id: 'pro',
      badge: 'BEST VALUE',
      title: 'Ode Pro',
      subtitle: '1 session included every month',
      price: `${PLAN_PRICE_SHORT.pro[currency]} / mo`,
      perSession: `Extra sessions at ${memberPrice} member price`,
      saving: 'A session/month + the full journal',
      features: [
        '1 therapy session included per month',
        `Extra sessions at member price (${memberPrice})`,
        'Everything in Ode+ - unlimited entries, all modes',
        'PDF export & Apple Health sync',
        'Full mood history & analytics',
      ],
      cta: 'Get Pro',
      highlight: true,
      isPlan: true,
    },
  ];
}

// ── Persona showcase ──────────────────────────────────────────────────────────

const PERSONAS = [
  { name: 'Comforting', desc: 'Warm and validating' },
  { name: 'Rational', desc: 'Structured & Socratic' },
  { name: 'CBT', desc: 'Pattern-aware' },
  { name: 'Mindful', desc: 'Grounding & present' },
];

// ── Option card ───────────────────────────────────────────────────────────────

function OptionCard({
  option,
  colors,
  onPress,
  busy = false,
}: {
  option: SessionOption;
  colors: ThemeColors;
  onPress: () => void;
  busy?: boolean;
}) {
  const { price, perSession } = option;

  return (
    <TouchableOpacity
      testID={therapyPricingCtaID(option.id)}
      disabled={busy}
      accessibilityState={{ busy, disabled: busy }}
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
        disabled={busy}
        activeOpacity={0.85}
      >
        {busy ? (
          <ActivityIndicator color={option.highlight ? Palette.onBrand : colors.brand} />
        ) : (
          <Text style={[cardStyles.ctaText, { color: option.highlight ? Palette.onBrand : colors.brand }]}>
            {option.cta}
          </Text>
        )}
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
    fontFamily: 'HankenGrotesk_700Bold',
    color: Palette.onBrand,
    letterSpacing: 1,
  },
  title: {
    fontSize: 20,
    fontFamily: 'Erode_600SemiBold',
    marginBottom: 2,
  },
  subtitle: {
    fontSize: 13,
    fontFamily: 'HankenGrotesk_400Regular',
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
    fontFamily: 'Erode_600SemiBold',
  },
  savingBadge: {
    borderRadius: 8,
    paddingHorizontal: 8,
    paddingVertical: 3,
  },
  savingText: {
    fontSize: 12,
    fontFamily: 'HankenGrotesk_700Bold',
  },
  perSession: {
    fontSize: 12,
    fontFamily: 'HankenGrotesk_400Regular',
    marginBottom: 14,
  },
  divider: { height: 1, marginBottom: 12 },
  featureRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 8,
    marginBottom: 6,
  },
  featureTick: { fontSize: 13, fontFamily: 'HankenGrotesk_700Bold', marginTop: 1 },
  featureText: { flex: 1, fontSize: 13, fontFamily: 'HankenGrotesk_400Regular', lineHeight: 18 },
  cta: {
    borderRadius: 12,
    paddingVertical: 13,
    alignItems: 'center',
    marginTop: 14,
  },
  ctaText: {
    fontSize: 15,
    fontFamily: 'HankenGrotesk_700Bold',
  },
});

// ── Main screen ───────────────────────────────────────────────────────────────

export default function TherapyPricingScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const [currency, setCurrency] = useState<RegionCurrency | null>(null);
  const [billing, setBilling] = useState<BillingPlanResponse | null>(null);
  const [credits, setCredits] = useState(0);
  const [purchasing, setPurchasing] = useState(false);

  useEffect(() => {
    resetAndDetectRegion().then(setCurrency).catch(() => setCurrency('usd'));
  }, []);

  // Plan (for the member price), prepaid balance and the purchase account
  // token. Fail-soft: without it the standalone price is offered.
  useEffect(() => {
    let alive = true;
    api.getBillingPlan()
      .then((b) => {
        if (!alive) return;
        setBilling(b);
        setCredits(b.therapy_credits ?? 0);
      })
      .catch(() => {});
    return () => { alive = false; };
  }, []);

  const activeCurrency: RegionCurrency = currency ?? 'usd';
  const isPro = billing?.plan === 'pro' || billing?.plan === 'b2b';

  const buySession = useCallback(async (kind: TherapyCreditKind) => {
    if (purchasing) return;
    setPurchasing(true);
    try {
      // Step 1: native store purchase sheet. null = IAP unavailable (Expo Go /
      // dev) - the dev backend then grants without verification.
      let purchase;
      try {
        purchase = await purchaseTherapyCredit(kind, billing?.purchase_account_token);
      } catch (err) {
        if (!isUserCancelled(err)) {
          Alert.alert('Purchase failed', 'The store could not complete the purchase. You have not been charged.');
        }
        return;
      }

      // Step 2: the backend verifies the purchase and adds the credit.
      const res = await api.buyTherapyCredits(
        purchase
          ? { platform: purchase.platform, product_id: purchase.productId, purchase_token: purchase.purchaseToken, transaction_id: purchase.transactionId }
          : { product_id: THERAPY_CREDIT_PRODUCT_IDS[kind] },
      );
      setCredits(res.therapy_credits);

      // Step 3: only now consume the store transaction. If the app dies
      // before this, the store re-delivers it and launch recovery redeems it.
      if (purchase) await finishPurchase(purchase);

      router.push('/therapy/persona-picker');
    } catch {
      Alert.alert(
        'Something went wrong',
        'Your purchase could not be confirmed. If you were charged, reopen the app to retry - your purchase is saved with the store.',
      );
    } finally {
      setPurchasing(false);
    }
  }, [billing, purchasing, router]);

  const handleOptionPress = (option: SessionOption) => {
    if (option.isPlan) {
      router.push('/upgrade');
    } else if (option.creditKind) {
      buySession(option.creditKind);
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
    <View testID={T.therapyPricing.screen} style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

          {/* Back */}
          <TouchableOpacity testID={T.therapyPricing.back} style={styles.backBtn} onPress={() => router.back()} activeOpacity={0.7}>
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
                  <Text style={[styles.personaName, { color: colors.textPrimary }]}>{p.name}</Text>
                  <Text style={[styles.personaDesc, { color: colors.textMuted }]}>{p.desc}</Text>
                </View>
              ))}
            </View>
          </ScrollView>

          {/* Prepaid balance */}
          {credits > 0 && (
            <View
              testID={T.therapyPricing.credits}
              style={[styles.creditsBox, { backgroundColor: colors.card, borderColor: colors.brand }]}
            >
              <Text style={[styles.creditsText, { color: colors.textPrimary }]}>
                {credits === 1 ? 'You have 1 prepaid session' : `You have ${credits} prepaid sessions`}
              </Text>
              <TouchableOpacity
                testID={T.therapyPricing.useCredit}
                onPress={() => router.push('/therapy/persona-picker')}
                activeOpacity={0.85}
                style={[styles.creditsCta, { backgroundColor: colors.brand }]}
              >
                <Text style={[styles.creditsCtaText, { color: Palette.onBrand }]}>Start a session</Text>
              </TouchableOpacity>
            </View>
          )}

          {/* Options */}
          {buildSessionOptions(activeCurrency, isPro).map((option) => (
            <OptionCard
              key={option.id}
              option={option}
              colors={colors}
              busy={purchasing && option.creditKind !== undefined}
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
            Ode Therapy Sessions are AI-assisted and are not a substitute for professional mental health care. If you are in crisis, please contact a crisis helpline immediately.
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
  backText: { fontSize: 14, fontFamily: 'HankenGrotesk_400Regular' },

  wordmark: {
    fontFamily: 'Erode_300Light',
    fontSize: 13,
    letterSpacing: 2,
    textTransform: 'uppercase',
    marginBottom: 10,
  },
  heading: {
    fontFamily: 'Erode_300Light',
    fontSize: 34,
    lineHeight: 42,
    marginBottom: 12,
  },
  subheading: {
    fontFamily: 'HankenGrotesk_400Regular',
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
    fontFamily: 'HankenGrotesk_700Bold',
    fontSize: 13,
  },

  creditsBox: {
    borderWidth: 1,
    borderRadius: 16,
    padding: 16,
    marginBottom: 20,
    gap: 12,
  },
  creditsText: { fontSize: 15, fontFamily: 'HankenGrotesk_700Bold' },
  creditsCta: { borderRadius: 12, paddingVertical: 12, alignItems: 'center' },
  creditsCtaText: { fontSize: 15, fontFamily: 'HankenGrotesk_700Bold' },

  sectionLabel: {
    fontSize: 10,
    fontFamily: 'HankenGrotesk_700Bold',
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
  personaName: { fontSize: 13, fontFamily: 'HankenGrotesk_700Bold', textAlign: 'center' },
  personaDesc: { fontSize: 11, fontFamily: 'HankenGrotesk_400Regular', textAlign: 'center', lineHeight: 15 },

  infoBox: {
    borderWidth: 1,
    borderRadius: 16,
    padding: 18,
    marginBottom: 20,
    gap: 8,
  },
  infoTitle: {
    fontSize: 15,
    fontFamily: 'HankenGrotesk_700Bold',
    marginBottom: 6,
  },
  infoRow: { flexDirection: 'row', alignItems: 'flex-start', gap: 8 },
  infoTick: { fontSize: 13, fontFamily: 'HankenGrotesk_700Bold', marginTop: 1 },
  infoText: { flex: 1, fontSize: 13, fontFamily: 'HankenGrotesk_400Regular', lineHeight: 19 },

  disclaimer: {
    fontSize: 11,
    fontFamily: 'HankenGrotesk_400Regular',
    lineHeight: 17,
    textAlign: 'center',
    marginTop: 4,
  },
});
