import { useCallback, useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  SafeAreaView,
  StatusBar,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { useRouter } from 'expo-router';
import { useStripe } from '@stripe/stripe-react-native';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';
import { resetAndDetectRegion } from '../src/services/region';
import type { Plan, BillingPlanResponse } from '../src/types';

type Currency = 'inr' | 'usd';

interface PlanCard {
  plan: Plan;
  emoji: string;
  title: string;
  subtitle?: string;
  priceInr: string;
  priceUsd: string;
  features: string[];
  highlight: boolean;
}

const PLANS: PlanCard[] = [
  {
    plan: 'free',
    emoji: '🌱',
    title: 'Free',
    priceInr: 'Free forever',
    priceUsd: 'Free forever',
    features: [
      '10 entries per month',
      'Basic AI reflection',
      '7-day mood chart',
      '3-turn follow-up conversation',
    ],
    highlight: false,
  },
  {
    plan: 'plus',
    emoji: '⭐',
    title: 'DreamLog+',
    subtitle: 'Most popular',
    priceInr: '₹199 / month',
    priceUsd: '$7.99 / month',
    features: [
      'Unlimited entries',
      'Hindi + Hinglish support',
      'Life Graph (30 / 90 / 365 days)',
      'Weekly emotional review',
      'All prompt modes — Rant, Gratitude, Decision',
      'Streak freeze (up to 3)',
      'Therapist sharing (5 links / month)',
    ],
    highlight: true,
  },
  {
    plan: 'pro',
    emoji: '🔮',
    title: 'DreamLog Pro',
    priceInr: '₹499 / month',
    priceUsd: '$14.99 / month',
    features: [
      'Everything in DreamLog+',
      'PDF journal export',
      'Apple Health sync',
      '2 Therapy sessions / month',
      'Unlimited therapist sharing',
      'Priority processing',
    ],
    highlight: false,
  },
];

export default function UpgradeScreen() {
  const router = useRouter();
  const { colors } = useTheme();
  const { initPaymentSheet, presentPaymentSheet } = useStripe();

  const [currency, setCurrency] = useState<Currency | null>(null); // set once from device locale
  const [billing, setBilling] = useState<BillingPlanResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [purchasing, setPurchasing] = useState<Plan | null>(null);

  useEffect(() => {
    Promise.all([
      api.getBillingPlan(),
      resetAndDetectRegion(),
    ]).then(([plan, region]) => {
      setBilling(plan);
      setCurrency(region);
    }).catch(() => {
      setCurrency('usd');
    }).finally(() => setLoading(false));
  }, []);

  const currentPlan = billing?.plan ?? 'free';
  const activeCurrency: Currency = currency ?? 'usd';

  const handleUpgrade = useCallback(async (targetPlan: 'plus' | 'pro') => {
    if (purchasing) return;
    setPurchasing(targetPlan);

    try {
      // Step 1: create a Stripe PaymentIntent on the backend
      const intent = await api.createPaymentIntent(targetPlan, activeCurrency);

      // Step 2: init the Stripe payment sheet
      const { error: initError } = await initPaymentSheet({
        paymentIntentClientSecret: intent.client_secret,
        merchantDisplayName: 'DreamLog',
        style: 'alwaysDark',
        primaryButtonLabel: `Pay ${activeCurrency === 'inr' ? (targetPlan === 'plus' ? '₹199' : '₹499') : (targetPlan === 'plus' ? '$7.99' : '$14.99')}`,
      });
      if (initError) {
        Alert.alert('Payment error', initError.message);
        return;
      }

      // Step 3: present the payment sheet
      const { error: presentError } = await presentPaymentSheet();
      if (presentError) {
        if (presentError.code !== 'Canceled') {
          Alert.alert('Payment failed', presentError.message);
        }
        return;
      }

      // Step 4: payment confirmed — update plan on backend
      const expiresAt = new Date();
      expiresAt.setDate(expiresAt.getDate() + 30);

      const updated = await api.upgradePlan(targetPlan, expiresAt.toISOString());
      setBilling(updated);

      Alert.alert(
        'Welcome to ' + (targetPlan === 'plus' ? 'DreamLog+' : 'DreamLog Pro') + '!',
        'Your subscription is now active.',
        [{ text: 'Continue', onPress: () => router.back() }],
      );
    } catch {
      Alert.alert('Something went wrong', 'Please try again.');
    } finally {
      setPurchasing(null);
    }
  }, [activeCurrency, purchasing, initPaymentSheet, presentPaymentSheet, router]);

  if (loading) {
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
        {/* Header */}
        <View style={[styles.header, { borderBottomColor: colors.borderFaint }]}>
          <TouchableOpacity onPress={() => router.back()} style={styles.backBtn}>
            <Text style={[styles.backText, { color: colors.textMuted }]}>← Back</Text>
          </TouchableOpacity>
          <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Choose a Plan</Text>
          <View style={styles.backBtn} />
        </View>

        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

          {/* Plan cards */}
          {PLANS.map((card) => {
            const isCurrent = currentPlan === card.plan;
            const isUpgradeable = card.plan !== 'free' && !isCurrent && currentPlan !== 'pro' || (card.plan === 'pro' && currentPlan === 'plus');
            const isPurchasing = purchasing === card.plan;

            return (
              <View
                key={card.plan}
                style={[
                  styles.card,
                  { backgroundColor: colors.card, borderColor: isCurrent ? colors.brand : card.highlight ? colors.purple600 : colors.borderFaint },
                  card.highlight && { borderWidth: 1.5 },
                ]}
              >
                {/* Card header */}
                <View style={styles.cardHeader}>
                  <View>
                    <View style={styles.titleRow}>
                      <Text style={[styles.cardTitle, { color: colors.textPrimary }]}>
                        {card.emoji}  {card.title}
                      </Text>
                      {card.subtitle && (
                        <View style={[styles.badge, { backgroundColor: colors.purple600 }]}>
                          <Text style={styles.badgeText}>{card.subtitle}</Text>
                        </View>
                      )}
                      {isCurrent && (
                        <View style={[styles.badge, { backgroundColor: colors.brandGlow, borderWidth: 1, borderColor: colors.brand }]}>
                          <Text style={[styles.badgeText, { color: colors.brand }]}>Current plan</Text>
                        </View>
                      )}
                    </View>
                    <Text style={[styles.cardPrice, { color: colors.purple300 }]}>
                      {activeCurrency === 'inr' ? card.priceInr : card.priceUsd}
                    </Text>
                  </View>
                </View>

                {/* Feature list */}
                <View style={styles.featureList}>
                  {card.features.map((f) => (
                    <View key={f} style={styles.featureRow}>
                      <Text style={[styles.featureCheck, { color: colors.brand }]}>✓</Text>
                      <Text style={[styles.featureText, { color: colors.textSecondary }]}>{f}</Text>
                    </View>
                  ))}
                </View>

                {/* CTA button */}
                {isUpgradeable && (
                  <TouchableOpacity
                    style={[styles.ctaBtn, { backgroundColor: card.highlight ? colors.brand : colors.cardSolid, borderColor: colors.brand, borderWidth: card.highlight ? 0 : 1 }]}
                    onPress={() => handleUpgrade(card.plan as 'plus' | 'pro')}
                    disabled={!!purchasing}
                    activeOpacity={0.8}
                  >
                    {isPurchasing ? (
                      <ActivityIndicator color={card.highlight ? '#fff' : colors.brand} size="small" />
                    ) : (
                      <Text style={[styles.ctaText, { color: card.highlight ? '#fff' : colors.brand }]}>
                        Get {card.title} →
                      </Text>
                    )}
                  </TouchableOpacity>
                )}

                {isCurrent && card.plan !== 'free' && (
                  <View style={[styles.currentBanner, { backgroundColor: colors.brandGlow }]}>
                    <Text style={[styles.currentBannerText, { color: colors.brand }]}>
                      Active · renews {billing?.plan_expires_at
                        ? new Date(billing.plan_expires_at).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })
                        : 'monthly'}
                    </Text>
                  </View>
                )}
              </View>
            );
          })}

          {/* Footer */}
          <View style={styles.footer}>
            <Text style={[styles.footerText, { color: colors.textFaint }]}>
              Secured by Stripe · Cancel anytime · No hidden fees
            </Text>
            <Text style={[styles.footerText, { color: colors.textFaint }]}>
              Subscriptions auto-renew monthly. Manage in Settings.
            </Text>
          </View>
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },

  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderBottomWidth: 1,
  },
  backBtn: { width: 60 },
  backText: {
    fontSize: 15,
    fontFamily: 'Nunito_400Regular',
  },
  headerTitle: {
    fontSize: 17,
    fontFamily: 'Nunito_600SemiBold',
  },

  scroll: { padding: 20, paddingBottom: 48 },

  card: {
    borderRadius: 20,
    borderWidth: 1,
    padding: 20,
    marginBottom: 16,
    gap: 16,
  },

  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
  },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 4,
  },
  cardTitle: {
    fontSize: 18,
    fontFamily: 'CormorantGaramond_500Medium',
  },
  cardPrice: {
    fontSize: 15,
    fontFamily: 'Nunito_600SemiBold',
  },

  badge: {
    borderRadius: 8,
    paddingHorizontal: 8,
    paddingVertical: 3,
  },
  badgeText: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    color: '#fff',
    letterSpacing: 0.5,
  },

  featureList: { gap: 8 },
  featureRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 10,
  },
  featureCheck: {
    fontSize: 14,
    fontFamily: 'Nunito_600SemiBold',
    lineHeight: 20,
    width: 14,
  },
  featureText: {
    flex: 1,
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 20,
  },

  ctaBtn: {
    paddingVertical: 14,
    borderRadius: 14,
    alignItems: 'center',
  },
  ctaText: {
    fontSize: 15,
    fontFamily: 'Nunito_600SemiBold',
  },

  currentBanner: {
    borderRadius: 10,
    paddingVertical: 8,
    paddingHorizontal: 14,
    alignItems: 'center',
  },
  currentBannerText: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
  },

  footer: {
    marginTop: 8,
    gap: 6,
    alignItems: 'center',
  },
  footerText: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
    lineHeight: 18,
  },
});
