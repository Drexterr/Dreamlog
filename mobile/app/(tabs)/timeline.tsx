import { useCallback } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  StatusBar,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { useTheme } from '../../src/context/ThemeContext';
import { useAuth } from '../../src/context/AuthContext';

// ── Feature card ──────────────────────────────────────────────────────────────
function FeatureCard({
  icon,
  title,
  description,
  cta,
  onPress,
  badge,
}: {
  icon: string;
  title: string;
  description: string;
  cta: string;
  onPress: () => void;
  badge?: string;
}) {
  const { colors } = useTheme();
  return (
    <TouchableOpacity
      style={[styles.featureCard, { backgroundColor: colors.card, borderColor: colors.border }]}
      onPress={onPress}
      activeOpacity={0.8}
    >
      <View style={styles.featureTop}>
        <View style={styles.featureTitleRow}>
          <Text style={[styles.featureIcon, { color: colors.purple300 }]}>{icon}</Text>
          <Text style={[styles.featureTitle, { color: colors.textPrimary }]}>{title}</Text>
        </View>
        {badge && (
          <View style={[styles.badge, { borderColor: colors.purple600 }]}>
            <Text style={[styles.badgeText, { color: colors.purple300 }]}>{badge}</Text>
          </View>
        )}
      </View>
      <Text style={[styles.featureDesc, { color: colors.textSecondary }]}>{description}</Text>
      <View style={styles.featureCTARow}>
        <Text style={[styles.featureCTA, { color: colors.purple300 }]}>{cta}</Text>
        <Text style={[styles.featureArrow, { color: colors.purple300 }]}>→</Text>
      </View>
    </TouchableOpacity>
  );
}

// ── Entries card (compact link) ───────────────────────────────────────────────
function EntriesCard({ onPress }: { onPress: () => void }) {
  const { colors } = useTheme();
  return (
    <TouchableOpacity
      style={[styles.entriesCard, { backgroundColor: colors.card, borderColor: colors.border }]}
      onPress={onPress}
      activeOpacity={0.8}
    >
      <View style={styles.entriesLeft}>
        <Text style={[styles.entriesTitle, { color: colors.textPrimary }]}>Journal entries</Text>
        <Text style={[styles.entriesDesc, { color: colors.textMuted }]}>Search and browse your full history</Text>
      </View>
      <Text style={[styles.featureArrow, { color: colors.textMuted }]}>→</Text>
    </TouchableOpacity>
  );
}

// ── Explore screen ────────────────────────────────────────────────────────────
export default function ExploreScreen() {
  const router = useRouter();
  const { colors } = useTheme();
  const { isAuthenticated, requestAuth } = useAuth();

  const handleTherapy = useCallback(() => {
    router.push('/therapy');
  }, [router]);

  const handleDream = useCallback(() => {
    const go = () => router.push({ pathname: '/record', params: { mode: 'dream' } } as any);
    if (isAuthenticated) go();
    else requestAuth(go);
  }, [isAuthenticated, requestAuth, router]);

  const handleJourneys = useCallback(() => {
    router.push('/journeys');
  }, [router]);

  const handleEntries = useCallback(() => {
    router.push('/entries' as any);
  }, [router]);

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={styles.safe}>
        <ScrollView
          contentContainerStyle={styles.scroll}
          showsVerticalScrollIndicator={false}
        >
          {/* Header */}
          <View style={styles.header}>
            <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Explore</Text>
            <Text style={[styles.headerSub, { color: colors.textMuted }]}>tools & your journal</Text>
          </View>

          {/* Entries */}
          <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>YOUR JOURNAL</Text>
          <EntriesCard onPress={handleEntries} />

          {/* Features */}
          <Text style={[styles.sectionLabel, { color: colors.textMuted, marginTop: 28 }]}>TOOLS</Text>

          <FeatureCard
            icon="◎"
            title="Therapy Mode"
            description="A real-time AI conversation grounded in your journal history. Voice or text, up to 60 minutes."
            cta="Start a session"
            onPress={handleTherapy}
          />

          <FeatureCard
            icon="◐"
            title="Dream Decoder"
            description="Record a dream and receive dual interpretations — Jungian depth psychology and Vedic Svapna Shastra."
            cta="Record a dream"
            onPress={handleDream}
          />

          <FeatureCard
            icon="⊹"
            title="Guided Journeys"
            description="Structured multi-step reflections for stress, grief, decisions, and self-compassion."
            cta="Browse journeys"
            onPress={handleJourneys}
          />
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  safe: { flex: 1 },
  scroll: { paddingHorizontal: 22, paddingBottom: 48 },

  header: {
    paddingTop: 20,
    paddingBottom: 24,
  },
  headerTitle: {
    fontSize: 30,
    fontFamily: 'CormorantGaramond_300Light',
    marginBottom: 2,
  },
  headerSub: {
    fontSize: 11,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.4,
  },

  sectionLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.5,
    marginBottom: 12,
  },

  // Entries compact card
  entriesCard: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    borderRadius: 14,
    borderWidth: 1,
    paddingHorizontal: 18,
    paddingVertical: 16,
  },
  entriesLeft: { gap: 2 },
  entriesTitle: {
    fontSize: 15,
    fontFamily: 'CormorantGaramond_500Medium',
    letterSpacing: 0.2,
  },
  entriesDesc: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
  },

  // Feature cards
  featureCard: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 20,
    marginBottom: 12,
    gap: 10,
  },
  featureTop: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
  },
  featureTitleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    flex: 1,
  },
  featureIcon: {
    fontSize: 18,
    lineHeight: 22,
  },
  featureTitle: {
    fontSize: 17,
    fontFamily: 'CormorantGaramond_500Medium',
    letterSpacing: 0.2,
  },
  featureDesc: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 20,
    paddingLeft: 28,
  },
  featureCTARow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: 6,
    paddingTop: 2,
  },
  featureCTA: {
    fontSize: 12,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 0.3,
  },
  featureArrow: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
  },

  badge: {
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 8,
    paddingVertical: 2,
  },
  badgeText: {
    fontSize: 9,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 0.8,
  },
});
