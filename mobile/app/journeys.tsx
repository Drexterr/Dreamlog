import { useCallback, useEffect, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  SafeAreaView,
  StatusBar,
  ActivityIndicator,
} from 'react-native';
import { useRouter } from 'expo-router';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';
import type { JourneySession, JourneyTemplate } from '../src/types';

// ── Tag pill ──────────────────────────────────────────────────────────────────
function TagPill({ tag }: { tag: string }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.tag, { backgroundColor: colors.card, borderColor: colors.border }]}>
      <Text style={[styles.tagText, { color: colors.textMuted }]}>{tag}</Text>
    </View>
  );
}

// ── Journey template card ─────────────────────────────────────────────────────
function TemplateCard({
  template,
  onStart,
  starting,
}: {
  template: JourneyTemplate;
  onStart: (id: string) => void;
  starting: string | null;
}) {
  const { colors } = useTheme();
  const isStarting = starting === template.id;

  return (
    <View style={[styles.card, { backgroundColor: colors.card, borderColor: colors.border }]}>
      <View style={styles.cardHeader}>
        <Text style={[styles.cardTitle, { color: colors.textPrimary }]}>{template.title}</Text>
        <Text style={[styles.cardMeta, { color: colors.textMuted }]}>
          {template.step_count} steps · {template.estimated_minutes} min
        </Text>
      </View>
      <Text style={[styles.cardDesc, { color: colors.textSecondary }]}>{template.description}</Text>
      <View style={styles.cardFooter}>
        <View style={styles.tags}>
          {template.tags.slice(0, 3).map((t) => (
            <TagPill key={t} tag={t} />
          ))}
        </View>
        <TouchableOpacity
          style={[styles.startBtn, { backgroundColor: colors.purple600 }]}
          onPress={() => onStart(template.id)}
          disabled={!!starting}
          activeOpacity={0.8}
        >
          {isStarting ? (
            <ActivityIndicator size="small" color="#fff" />
          ) : (
            <Text style={styles.startBtnText}>Begin</Text>
          )}
        </TouchableOpacity>
      </View>
    </View>
  );
}

// ── Active session card ───────────────────────────────────────────────────────
function SessionCard({ session }: { session: JourneySession }) {
  const { colors } = useTheme();
  const router = useRouter();
  const pct = session.total_steps > 0 ? session.current_step / session.total_steps : 0;
  const isComplete = session.status === 'completed';

  return (
    <TouchableOpacity
      style={[styles.sessionCard, { backgroundColor: colors.card, borderColor: isComplete ? colors.moodGreen + '60' : colors.border }]}
      onPress={() => router.push({ pathname: '/journeys/[sessionId]', params: { sessionId: session.id } })}
      activeOpacity={0.8}
    >
      <View style={styles.sessionHeader}>
        <Text style={[styles.sessionTitle, { color: colors.textPrimary }]}>{session.journey_title}</Text>
        {isComplete ? (
          <Text style={[styles.sessionBadge, { color: colors.moodGreen }]}>Done</Text>
        ) : (
          <Text style={[styles.sessionBadge, { color: colors.purple300 }]}>
            {session.current_step}/{session.total_steps}
          </Text>
        )}
      </View>
      {/* Progress bar */}
      <View style={[styles.progressTrack, { backgroundColor: colors.border }]}>
        <View
          style={[
            styles.progressFill,
            { width: `${pct * 100}%`, backgroundColor: isComplete ? colors.moodGreen : colors.purple500 },
          ]}
        />
      </View>
      {!isComplete && (
        <Text style={[styles.sessionHint, { color: colors.textMuted }]}>
          Next: {(session.steps ?? [])[session.current_step]?.prompt ?? 'Continue'}
        </Text>
      )}
    </TouchableOpacity>
  );
}

// ── Journeys screen ───────────────────────────────────────────────────────────
export default function JourneysScreen() {
  const router = useRouter();
  const { colors } = useTheme();

  const [templates, setTemplates] = useState<JourneyTemplate[]>([]);
  const [sessions, setSessions] = useState<JourneySession[]>([]);
  const [loading, setLoading] = useState(true);
  const [starting, setStarting] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    Promise.all([api.listJourneys(), api.listJourneySessions()])
      .then(([tmplRes, sessRes]) => {
        setTemplates(tmplRes.journeys ?? []);
        setSessions(sessRes.sessions ?? []);
      })
      .catch(() => setError('Could not load journeys.'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleStart = useCallback(async (journeyID: string) => {
    setStarting(journeyID);
    try {
      const session = await api.startJourney(journeyID);
      setSessions((prev) => [session, ...prev]);
      router.push({ pathname: '/journeys/[sessionId]', params: { sessionId: session.id } });
    } catch {
      setError('Could not start journey. Try again.');
    } finally {
      setStarting(null);
    }
  }, [router]);

  const activeSessions = sessions.filter((s) => s.status === 'in_progress');
  const completedSessions = sessions.filter((s) => s.status === 'completed');

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={styles.safe}>
        {/* Header */}
        <View style={styles.header}>
          <TouchableOpacity onPress={() => router.back()} style={styles.backBtn} hitSlop={{ top: 8, bottom: 8, left: 8, right: 8 }}>
            <Text style={[styles.backArrow, { color: colors.textSecondary }]}>←</Text>
          </TouchableOpacity>
          <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Guided Journeys</Text>
          <View style={{ width: 32 }} />
        </View>

        {loading ? (
          <View style={styles.center}>
            <ActivityIndicator color={colors.purple400} />
          </View>
        ) : error ? (
          <View style={styles.center}>
            <Text style={[styles.errorText, { color: colors.textMuted }]}>{error}</Text>
            <TouchableOpacity onPress={load} style={[styles.retryBtn, { borderColor: colors.border }]}>
              <Text style={[styles.retryText, { color: colors.purple300 }]}>Try again</Text>
            </TouchableOpacity>
          </View>
        ) : (
          <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
            {/* Active sessions */}
            {activeSessions.length > 0 && (
              <View style={styles.section}>
                <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>IN PROGRESS</Text>
                {activeSessions.map((s) => (
                  <SessionCard key={s.id} session={s} />
                ))}
              </View>
            )}

            {/* Templates */}
            <View style={styles.section}>
              <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>START A JOURNEY</Text>
              {templates.map((t) => (
                <TemplateCard key={t.id} template={t} onStart={handleStart} starting={starting} />
              ))}
            </View>

            {/* Completed */}
            {completedSessions.length > 0 && (
              <View style={styles.section}>
                <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>COMPLETED</Text>
                {completedSessions.map((s) => (
                  <SessionCard key={s.id} session={s} />
                ))}
              </View>
            )}
          </ScrollView>
        )}
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  safe: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 16 },

  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingTop: 12,
    paddingBottom: 8,
  },
  backBtn: { width: 32, alignItems: 'flex-start' },
  backArrow: { fontSize: 22 },
  headerTitle: {
    fontSize: 18,
    fontFamily: 'CormorantGaramond_500Medium',
    letterSpacing: 0.3,
  },

  scroll: { paddingHorizontal: 20, paddingBottom: 40 },

  section: { marginBottom: 28 },
  sectionLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.5,
    marginBottom: 12,
    marginTop: 8,
  },

  // Template card
  card: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 18,
    marginBottom: 12,
    gap: 10,
  },
  cardHeader: { gap: 2 },
  cardTitle: {
    fontSize: 17,
    fontFamily: 'CormorantGaramond_500Medium',
    letterSpacing: 0.2,
  },
  cardMeta: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
  },
  cardDesc: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 19,
  },
  cardFooter: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: 4,
  },
  tags: { flexDirection: 'row', gap: 6, flexWrap: 'wrap', flex: 1 },
  tag: {
    borderRadius: 8,
    borderWidth: 1,
    paddingHorizontal: 8,
    paddingVertical: 3,
  },
  tagText: { fontSize: 10, fontFamily: 'Nunito_400Regular' },
  startBtn: {
    paddingHorizontal: 18,
    paddingVertical: 9,
    borderRadius: 20,
    minWidth: 70,
    alignItems: 'center',
  },
  startBtnText: {
    color: '#fff',
    fontSize: 13,
    fontFamily: 'Nunito_600SemiBold',
  },

  // Session card
  sessionCard: {
    borderRadius: 14,
    borderWidth: 1,
    padding: 16,
    marginBottom: 10,
    gap: 10,
  },
  sessionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
  },
  sessionTitle: {
    fontSize: 15,
    fontFamily: 'CormorantGaramond_500Medium',
    flex: 1,
    marginRight: 8,
  },
  sessionBadge: {
    fontSize: 12,
    fontFamily: 'Nunito_600SemiBold',
  },
  progressTrack: {
    height: 3,
    borderRadius: 2,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    borderRadius: 2,
  },
  sessionHint: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 17,
  },

  errorText: { fontSize: 14, fontFamily: 'Nunito_400Regular', textAlign: 'center' },
  retryBtn: { paddingHorizontal: 20, paddingVertical: 10, borderRadius: 20, borderWidth: 1 },
  retryText: { fontSize: 14, fontFamily: 'Nunito_600SemiBold' },
});
