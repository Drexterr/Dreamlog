import { useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  SafeAreaView,
  StatusBar,
  Animated,
  ActivityIndicator,
} from 'react-native';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { TherapySessionSummary } from '../../src/types';
import { PERSONA_META } from '../../src/types';
import { getCachedRegion, THERAPY_SESSION_PRICE } from '../../src/services/region';

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatDuration(sec?: number): string {
  if (!sec) return '—';
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return m > 0 ? `${m}m ${s}s` : `${s}s`;
}

function statusLabel(status: string): { label: string; isActive: boolean } {
  switch (status) {
    case 'active': return { label: 'In Progress', isActive: true };
    case 'completed': return { label: 'Completed', isActive: false };
    case 'expired': return { label: 'Expired', isActive: false };
    case 'crisis_detected': return { label: 'Ended (Support provided)', isActive: false };
    default: return { label: status, isActive: false };
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-IN', {
    day: 'numeric', month: 'short', year: 'numeric',
  });
}

// ── Ambient hero glow ─────────────────────────────────────────────────────────

function AmbientGlow({ colors }: { colors: any }) {
  const pulse = useRef(new Animated.Value(0.3)).current;

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(pulse, { toValue: 0.55, duration: 3500, useNativeDriver: true }),
        Animated.timing(pulse, { toValue: 0.3, duration: 3500, useNativeDriver: true }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, []);

  return (
    <Animated.View
      pointerEvents="none"
      style={[
        glowStyles.blob,
        { backgroundColor: colors.purple600, opacity: pulse },
      ]}
    />
  );
}

const glowStyles = StyleSheet.create({
  blob: {
    position: 'absolute',
    top: -60,
    alignSelf: 'center',
    width: 320,
    height: 320,
    borderRadius: 160,
  },
});

// ── Feature chips ─────────────────────────────────────────────────────────────

const FEATURES = [
  { icon: '🎙', label: 'Voice or text' },
  { icon: '⏱', label: 'Up to 60 min' },
  { icon: '🧠', label: 'Journal-aware AI' },
  { icon: '📝', label: 'Post-session summary' },
  { icon: '🛡', label: 'Crisis detection' },
];

// ── Persona preview strip ─────────────────────────────────────────────────────

const PERSONA_KEYS = ['comforting', 'rational', 'cbt', 'mindful'] as const;

// ── Past session card ─────────────────────────────────────────────────────────

function SessionCard({
  session,
  onPress,
  colors,
}: {
  session: TherapySessionSummary;
  onPress: () => void;
  colors: any;
}) {
  const persona = session.persona ? PERSONA_META[session.persona] : null;
  const { label, isActive } = statusLabel(session.status);

  return (
    <TouchableOpacity
      style={[cardStyles.card, { backgroundColor: colors.card, borderColor: colors.border }]}
      onPress={onPress}
      activeOpacity={0.75}
    >
      {/* Persona accent bar */}
      <View style={[cardStyles.accentBar, { backgroundColor: isActive ? colors.brand : colors.purple700 }]} />

      <View style={cardStyles.body}>
        {/* Top row */}
        <View style={cardStyles.topRow}>
          <View style={cardStyles.dateRow}>
            {persona && <Text style={cardStyles.personaEmoji}>{persona.emoji}</Text>}
            <View>
              <Text style={[cardStyles.personaName, { color: colors.textPrimary }]}>
                {persona?.label ?? 'Session'}
              </Text>
              <Text style={[cardStyles.date, { color: colors.textMuted }]}>
                {formatDate(session.started_at)}
              </Text>
            </View>
          </View>
          <View style={[
            cardStyles.statusBadge,
            { backgroundColor: isActive ? `${colors.brand}22` : `${colors.purple700}44` },
          ]}>
            <Text style={[cardStyles.statusText, { color: isActive ? colors.brand : colors.textMuted }]}>
              {label}
            </Text>
          </View>
        </View>

        {/* Summary preview */}
        {session.post_session_summary ? (
          <Text style={[cardStyles.summary, { color: colors.textSecondary }]} numberOfLines={2}>
            {session.post_session_summary}
          </Text>
        ) : null}

        {/* Meta */}
        <Text style={[cardStyles.meta, { color: colors.textMuted }]}>
          {session.turn_count} turn{session.turn_count !== 1 ? 's' : ''}
          {session.duration_sec ? ` · ${formatDuration(session.duration_sec)}` : ''}
          {isActive ? '  →  Tap to continue' : ''}
        </Text>
      </View>
    </TouchableOpacity>
  );
}

const cardStyles = StyleSheet.create({
  card: {
    borderRadius: 16,
    borderWidth: 1,
    flexDirection: 'row',
    overflow: 'hidden',
    marginBottom: 12,
  },
  accentBar: { width: 4 },
  body: { flex: 1, padding: 14, gap: 6 },
  topRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
  },
  dateRow: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  personaEmoji: { fontSize: 26 },
  personaName: { fontSize: 14, fontFamily: 'Nunito_700Bold', marginBottom: 1 },
  date: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  statusBadge: {
    borderRadius: 8,
    paddingHorizontal: 8,
    paddingVertical: 3,
  },
  statusText: { fontSize: 11, fontFamily: 'Nunito_700Bold' },
  summary: { fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 19 },
  meta: { fontSize: 11, fontFamily: 'Nunito_400Regular' },
});

// ── Main screen ───────────────────────────────────────────────────────────────

export default function TherapyIndexScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const [sessions, setSessions] = useState<TherapySessionSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [priceDisplay, setPriceDisplay] = useState(THERAPY_SESSION_PRICE.inr);

  useEffect(() => {
    getCachedRegion().then((r) => setPriceDisplay(THERAPY_SESSION_PRICE[r ?? 'usd']));
    api.listTherapySessions()
      .then((r) => setSessions(r.sessions ?? []))
      .catch(() => setSessions([]))
      .finally(() => setLoading(false));
  }, []);

  const handleStart = () => router.push('/therapy/persona-picker' as any);

  const handleResume = (s: TherapySessionSummary) => {
    if (s.status === 'active') {
      router.push({ pathname: '/therapy/session', params: { id: s.id } } as any);
    } else {
      router.push({ pathname: '/therapy/summary/[id]', params: { id: s.id } } as any);
    }
  };

  // Aggregate stats
  const totalSessions = sessions.length;
  const totalTurnCount = sessions.reduce((a, s) => a + (s.turn_count ?? 0), 0);
  const activeSession = sessions.find((s) => s.status === 'active');

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

          {/* Ambient background */}
          <AmbientGlow colors={colors} />

          {/* Back */}
          <TouchableOpacity
            style={styles.backBtn}
            onPress={() => router.replace('/(tabs)')}
            activeOpacity={0.7}
          >
            <Text style={[styles.backText, { color: colors.textMuted }]}>← Journal</Text>
          </TouchableOpacity>

          {/* Hero */}
          <View style={styles.hero}>
            <Text style={[styles.eyebrow, { color: colors.purple300 }]}>Reflection Session</Text>
            <Text style={[styles.heroHeading, { color: colors.textPrimary }]}>
              A space to talk,{'\n'}grounded in your{'\n'}journal
            </Text>
            <Text style={[styles.heroSub, { color: colors.textSecondary }]}>
              An AI companion that already knows your emotional history — so you don't have to start from scratch.
            </Text>
          </View>

          {/* Feature chips */}
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            style={styles.featureScroll}
            contentContainerStyle={styles.featureRow}
          >
            {FEATURES.map((f) => (
              <View key={f.label} style={[styles.featureChip, { backgroundColor: colors.card, borderColor: colors.border }]}>
                <Text style={styles.featureIcon}>{f.icon}</Text>
                <Text style={[styles.featureLabel, { color: colors.textSecondary }]}>{f.label}</Text>
              </View>
            ))}
          </ScrollView>

          {/* Resume active session banner */}
          {activeSession && (
            <TouchableOpacity
              style={[styles.resumeBanner, { backgroundColor: `${colors.brand}18`, borderColor: colors.brand }]}
              onPress={() => handleResume(activeSession)}
              activeOpacity={0.85}
            >
              <View style={{ flex: 1 }}>
                <Text style={[styles.resumeTitle, { color: colors.purple300 }]}>Session in progress</Text>
                <Text style={[styles.resumeSub, { color: colors.textSecondary }]}>
                  {activeSession.persona ? `${PERSONA_META[activeSession.persona]?.emoji} ${PERSONA_META[activeSession.persona]?.label}` : 'Reflection Session'}
                  {' · '}
                  {activeSession.turn_count} turn{activeSession.turn_count !== 1 ? 's' : ''}
                </Text>
              </View>
              <Text style={[styles.resumeArrow, { color: colors.brand }]}>Continue →</Text>
            </TouchableOpacity>
          )}

          {/* Persona preview */}
          <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>COMPANION STYLES</Text>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={styles.personaRow}
          >
            {PERSONA_KEYS.map((key) => {
              const meta = PERSONA_META[key];
              return (
                <View key={key} style={[styles.personaCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
                  <Text style={styles.personaEmoji}>{meta.emoji}</Text>
                  <Text style={[styles.personaName, { color: colors.textPrimary }]}>{meta.label}</Text>
                  <Text style={[styles.personaTagline, { color: colors.textMuted }]} numberOfLines={2}>
                    {meta.tagline}
                  </Text>
                </View>
              );
            })}
          </ScrollView>

          {/* CTA */}
          <TouchableOpacity
            style={[styles.startBtn, { backgroundColor: colors.brand }]}
            onPress={handleStart}
            activeOpacity={0.85}
          >
            <Text style={styles.startBtnText}>Start a Session</Text>
          </TouchableOpacity>

          {/* Pricing note */}
          <View style={styles.pricingRow}>
            <Text style={[styles.pricingNote, { color: colors.textMuted }]}>
              First session free · {priceDisplay}/session
            </Text>
            <TouchableOpacity onPress={() => router.push('/therapy/pricing' as any)} activeOpacity={0.7}>
              <Text style={[styles.pricingLink, { color: colors.brand }]}>See all plans →</Text>
            </TouchableOpacity>
          </View>

          {/* Stats bar (only when user has sessions) */}
          {totalSessions > 0 && (
            <View style={[styles.statsBar, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
              <View style={styles.statItem}>
                <Text style={[styles.statValue, { color: colors.textPrimary }]}>{totalSessions}</Text>
                <Text style={[styles.statLabel, { color: colors.textMuted }]}>sessions</Text>
              </View>
              <View style={[styles.statDivider, { backgroundColor: colors.borderFaint }]} />
              <View style={styles.statItem}>
                <Text style={[styles.statValue, { color: colors.textPrimary }]}>{totalTurnCount}</Text>
                <Text style={[styles.statLabel, { color: colors.textMuted }]}>total turns</Text>
              </View>
              <View style={[styles.statDivider, { backgroundColor: colors.borderFaint }]} />
              <View style={styles.statItem}>
                <Text style={[styles.statValue, { color: colors.textPrimary }]}>
                  {sessions.filter((s) => s.status === 'completed').length}
                </Text>
                <Text style={[styles.statLabel, { color: colors.textMuted }]}>completed</Text>
              </View>
            </View>
          )}

          {/* Past sessions */}
          {loading ? (
            <ActivityIndicator color={colors.brand} style={{ marginTop: 32 }} />
          ) : sessions.length > 0 ? (
            <View style={styles.historySection}>
              <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>PAST SESSIONS</Text>
              {sessions.map((s) => (
                <SessionCard
                  key={s.id}
                  session={s}
                  onPress={() => handleResume(s)}
                  colors={colors}
                />
              ))}
            </View>
          ) : null}

        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { paddingHorizontal: 20, paddingTop: 16, paddingBottom: 60, overflow: 'visible' },

  backBtn: { marginBottom: 28 },
  backText: { fontSize: 14, fontFamily: 'Nunito_400Regular' },

  // Hero
  hero: { marginBottom: 28 },
  eyebrow: {
    fontSize: 11,
    fontFamily: 'Nunito_700Bold',
    letterSpacing: 2,
    textTransform: 'uppercase',
    marginBottom: 10,
  },
  heroHeading: {
    fontSize: 38,
    fontFamily: 'CormorantGaramond_300Light',
    lineHeight: 46,
    marginBottom: 12,
  },
  heroSub: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 22,
    maxWidth: 300,
  },

  // Feature chips
  featureScroll: { marginBottom: 24 },
  featureRow: { gap: 8, paddingRight: 20 },
  featureChip: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    borderWidth: 1,
    borderRadius: 20,
    paddingHorizontal: 12,
    paddingVertical: 7,
  },
  featureIcon: { fontSize: 13 },
  featureLabel: { fontSize: 12, fontFamily: 'Nunito_600SemiBold' },

  // Resume banner
  resumeBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    borderRadius: 14,
    padding: 14,
    marginBottom: 28,
    gap: 10,
  },
  resumeTitle: { fontSize: 13, fontFamily: 'Nunito_700Bold', marginBottom: 2 },
  resumeSub: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  resumeArrow: { fontSize: 13, fontFamily: 'Nunito_700Bold' },

  // Persona preview
  sectionLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_700Bold',
    letterSpacing: 1.5,
    marginBottom: 12,
  },
  personaRow: { gap: 10, paddingRight: 20, marginBottom: 28 },
  personaCard: {
    width: 120,
    borderRadius: 14,
    borderWidth: 1,
    padding: 14,
    gap: 4,
    alignItems: 'center',
  },
  personaEmoji: { fontSize: 28, marginBottom: 4 },
  personaName: { fontSize: 13, fontFamily: 'Nunito_700Bold', textAlign: 'center' },
  personaTagline: { fontSize: 11, fontFamily: 'Nunito_400Regular', textAlign: 'center', lineHeight: 15 },

  // CTA
  startBtn: {
    borderRadius: 14,
    paddingVertical: 17,
    alignItems: 'center',
    marginBottom: 12,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.3,
    shadowRadius: 12,
    elevation: 6,
  },
  startBtnText: { color: '#fff', fontSize: 17, fontFamily: 'Nunito_700Bold' },

  // Pricing
  pricingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
    marginBottom: 28,
  },
  pricingNote: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  pricingLink: { fontSize: 12, fontFamily: 'Nunito_700Bold' },

  // Stats bar
  statsBar: {
    flexDirection: 'row',
    borderRadius: 14,
    borderWidth: 1,
    marginBottom: 28,
    overflow: 'hidden',
  },
  statItem: { flex: 1, alignItems: 'center', paddingVertical: 14 },
  statValue: { fontSize: 20, fontFamily: 'CormorantGaramond_600SemiBold', marginBottom: 2 },
  statLabel: { fontSize: 11, fontFamily: 'Nunito_400Regular' },
  statDivider: { width: 1 },

  // Past sessions
  historySection: { marginTop: 4 },
});
