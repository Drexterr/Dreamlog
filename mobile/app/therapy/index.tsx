import { useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  StatusBar,
  Animated,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { useAuth } from '../../src/context/AuthContext';
import type { TherapySessionSummary, TherapyPersona } from '../../src/types';
import { PERSONA_META } from '../../src/types';
import { getCachedRegion, THERAPY_SESSION_PRICE } from '../../src/services/region';

// Per-persona accent colors — intentional identity, not themed
const PERSONA_ACCENT: Record<TherapyPersona, string> = {
  comforting: '#C4A06A',
  rational:   '#6A9EC4',
  cbt:        '#7AAA88',
  mindful:    '#9A8AC0',
};

function formatDuration(sec?: number): string {
  if (!sec) return '-';
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return m > 0 ? `${m}m ${s}s` : `${s}s`;
}

function statusLabel(status: string): { label: string; isActive: boolean } {
  switch (status) {
    case 'active':           return { label: 'In progress', isActive: true };
    case 'completed':        return { label: 'Completed', isActive: false };
    case 'expired':          return { label: 'Expired', isActive: false };
    case 'crisis_detected':  return { label: 'Ended', isActive: false };
    default:                 return { label: status, isActive: false };
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-IN', {
    day: 'numeric', month: 'short', year: 'numeric',
  });
}

// ── Ambient background glow ───────────────────────────────────────────────────

function AmbientGlow({ colors }: { colors: any }) {
  const pulse = useRef(new Animated.Value(0.18)).current;

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(pulse, { toValue: 0.32, duration: 4000, useNativeDriver: true }),
        Animated.timing(pulse, { toValue: 0.18, duration: 4000, useNativeDriver: true }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, []);

  return (
    <Animated.View
      pointerEvents="none"
      style={[glowStyles.blob, { backgroundColor: colors.brand, opacity: pulse }]}
    />
  );
}

const glowStyles = StyleSheet.create({
  blob: {
    position: 'absolute',
    top: -80,
    alignSelf: 'center',
    width: 280,
    height: 280,
    borderRadius: 140,
  },
});

// ── Session card ──────────────────────────────────────────────────────────────

function SessionCard({
  session,
  onPress,
  colors,
}: {
  session: TherapySessionSummary;
  onPress: () => void;
  colors: any;
}) {
  const personaKey = (session.persona ?? 'comforting') as TherapyPersona;
  const persona = PERSONA_META[personaKey];
  const accent = PERSONA_ACCENT[personaKey];
  const { label, isActive } = statusLabel(session.status);

  return (
    <TouchableOpacity
      style={[cardStyles.card, { backgroundColor: colors.card, borderColor: colors.border }]}
      onPress={onPress}
      activeOpacity={0.75}
    >
      <View style={[cardStyles.accentBar, { backgroundColor: isActive ? accent : colors.border }]} />
      <View style={cardStyles.body}>
        <View style={cardStyles.topRow}>
          <View style={{ flex: 1 }}>
            <Text style={[cardStyles.personaName, { color: colors.textPrimary }]}>
              {persona?.label ?? 'Session'}
            </Text>
            <Text style={[cardStyles.date, { color: colors.textMuted }]}>
              {formatDate(session.started_at)}
            </Text>
          </View>
          <Text style={[cardStyles.statusText, { color: isActive ? accent : colors.textMuted }]}>
            {label}
          </Text>
        </View>

        {session.post_session_summary ? (
          <Text style={[cardStyles.summary, { color: colors.textSecondary }]} numberOfLines={2}>
            {session.post_session_summary}
          </Text>
        ) : null}

        <Text style={[cardStyles.meta, { color: colors.textMuted }]}>
          {session.turn_count} turn{session.turn_count !== 1 ? 's' : ''}
          {session.duration_sec ? `  ·  ${formatDuration(session.duration_sec)}` : ''}
          {isActive ? '  ·  Tap to continue' : ''}
        </Text>
      </View>
    </TouchableOpacity>
  );
}

const cardStyles = StyleSheet.create({
  card: {
    borderRadius: 14,
    borderWidth: 1,
    flexDirection: 'row',
    overflow: 'hidden',
    marginBottom: 10,
  },
  accentBar: { width: 3 },
  body: { flex: 1, padding: 14, gap: 6 },
  topRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
  },
  personaName: { fontSize: 14, fontFamily: 'Nunito_700Bold', marginBottom: 1 },
  date: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  statusText: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  summary: { fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 19 },
  meta: { fontSize: 11, fontFamily: 'Nunito_400Regular' },
});

// ── Main screen ───────────────────────────────────────────────────────────────

const PERSONA_KEYS: TherapyPersona[] = ['comforting', 'rational', 'cbt', 'mindful'];

export default function TherapyIndexScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const { isAuthenticated, requestAuth } = useAuth();
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

  const handleStart = () => {
    if (isAuthenticated) {
      router.push('/therapy/persona-picker' as any);
    } else {
      requestAuth(() => router.push('/therapy/persona-picker' as any));
    }
  };

  const handleResume = (s: TherapySessionSummary) => {
    if (s.status === 'active') {
      router.push({ pathname: '/therapy/session', params: { id: s.id } } as any);
    } else {
      router.push({ pathname: '/therapy/summary/[id]', params: { id: s.id } } as any);
    }
  };

  const totalSessions = sessions.length;
  const totalTurns = sessions.reduce((a, s) => a + (s.turn_count ?? 0), 0);
  const completedCount = sessions.filter((s) => s.status === 'completed').length;
  const activeSession = sessions.find((s) => s.status === 'active');

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

          <AmbientGlow colors={colors} />

          <TouchableOpacity style={styles.backBtn} onPress={() => router.back()} activeOpacity={0.7}>
            <Text style={[styles.backText, { color: colors.textMuted }]}>Journal</Text>
          </TouchableOpacity>

          {/* Hero */}
          <View style={styles.hero}>
            <Text style={[styles.heroHeading, { color: colors.textPrimary }]}>
              A space to talk,{'\n'}grounded in your{'\n'}journal
            </Text>
            <Text style={[styles.heroSub, { color: colors.textSecondary }]}>
              Your AI companion already knows your emotional history. No need to start from scratch.
            </Text>
          </View>

          {/* Resume active session */}
          {activeSession && (
            <TouchableOpacity
              style={[styles.resumeBanner, {
                backgroundColor: `${PERSONA_ACCENT[(activeSession.persona ?? 'comforting') as TherapyPersona]}18`,
                borderColor: PERSONA_ACCENT[(activeSession.persona ?? 'comforting') as TherapyPersona],
              }]}
              onPress={() => handleResume(activeSession)}
              activeOpacity={0.85}
            >
              <View style={{ flex: 1 }}>
                <Text style={[styles.resumeTitle, {
                  color: PERSONA_ACCENT[(activeSession.persona ?? 'comforting') as TherapyPersona],
                }]}>
                  Session in progress
                </Text>
                <Text style={[styles.resumeSub, { color: colors.textSecondary }]}>
                  {activeSession.persona
                    ? (PERSONA_META[activeSession.persona as TherapyPersona]?.label ?? 'Reflection')
                    : 'Reflection'}
                  {'  ·  '}
                  {activeSession.turn_count} turn{activeSession.turn_count !== 1 ? 's' : ''}
                </Text>
              </View>
              <Text style={[styles.resumeArrow, {
                color: PERSONA_ACCENT[(activeSession.persona ?? 'comforting') as TherapyPersona],
              }]}>Continue</Text>
            </TouchableOpacity>
          )}

          {/* Companion styles */}
          <Text style={[styles.quietLabel, { color: colors.textMuted }]}>Choose a companion style</Text>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={styles.personaRow}
          >
            {PERSONA_KEYS.map((key) => {
              const meta = PERSONA_META[key];
              const accent = PERSONA_ACCENT[key];
              return (
                <TouchableOpacity
                  key={key}
                  style={[styles.personaCard, {
                    backgroundColor: colors.card,
                    borderColor: colors.border,
                    borderTopColor: accent,
                    borderTopWidth: 2,
                  }]}
                  onPress={handleStart}
                  activeOpacity={0.75}
                >
                  <Text style={[styles.personaName, { color: colors.textPrimary }]}>{meta.label}</Text>
                  <Text style={[styles.personaTagline, { color: colors.textMuted }]} numberOfLines={2}>
                    {meta.tagline}
                  </Text>
                </TouchableOpacity>
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

          <View style={styles.pricingRow}>
            <Text style={[styles.pricingNote, { color: colors.textMuted }]}>
              First session free  ·  {priceDisplay}/session
            </Text>
            <TouchableOpacity onPress={() => router.push('/therapy/pricing' as any)} activeOpacity={0.7}>
              <Text style={[styles.pricingLink, { color: colors.brand }]}>Plans</Text>
            </TouchableOpacity>
          </View>

          {/* Past sessions */}
          {loading ? (
            <ActivityIndicator color={colors.brand} style={{ marginTop: 40 }} />
          ) : sessions.length > 0 ? (
            <View style={styles.historySection}>
              <View style={[styles.divider, { backgroundColor: colors.borderFaint }]} />

              {totalSessions > 0 && (
                <Text style={[styles.statLine, { color: colors.textMuted }]}>
                  {totalSessions} session{totalSessions !== 1 ? 's' : ''}
                  {'  ·  '}
                  {totalTurns} turns
                  {'  ·  '}
                  {completedCount} completed
                </Text>
              )}

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
  backText: { fontSize: 13, fontFamily: 'Nunito_400Regular' },

  hero: { marginBottom: 32 },
  heroHeading: {
    fontSize: 36,
    fontFamily: 'CormorantGaramond_300Light',
    lineHeight: 44,
    marginBottom: 14,
  },
  heroSub: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 22,
    maxWidth: 300,
  },

  resumeBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    borderRadius: 12,
    padding: 14,
    marginBottom: 28,
    gap: 10,
  },
  resumeTitle: { fontSize: 13, fontFamily: 'Nunito_700Bold', marginBottom: 2 },
  resumeSub: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  resumeArrow: { fontSize: 13, fontFamily: 'Nunito_600SemiBold' },

  quietLabel: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    marginBottom: 12,
  },
  personaRow: { gap: 10, paddingRight: 20, marginBottom: 28 },
  personaCard: {
    width: 130,
    borderRadius: 12,
    borderWidth: 1,
    padding: 14,
    gap: 5,
  },
  personaName: { fontSize: 14, fontFamily: 'Nunito_700Bold' },
  personaTagline: { fontSize: 11, fontFamily: 'Nunito_400Regular', lineHeight: 15 },

  startBtn: {
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 12,
  },
  startBtnText: { color: '#fff', fontSize: 16, fontFamily: 'Nunito_700Bold' },

  pricingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
    marginBottom: 32,
  },
  pricingNote: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  pricingLink: { fontSize: 12, fontFamily: 'Nunito_600SemiBold' },

  historySection: { marginTop: 4 },
  divider: { height: 1, marginBottom: 16 },
  statLine: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    marginBottom: 16,
  },
});
