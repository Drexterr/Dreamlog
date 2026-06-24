import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  Animated,
  StyleSheet,
  StatusBar,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { useAuth } from '../../src/context/AuthContext';
import { useGuidedTour } from '../../src/context/GuidedTourContext';
import type { DailyMood, StreakInfo } from '../../src/types';

// ── Format date label ─────────────────────────────────────────────────────────
function formatEntryDate(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (d.toDateString() === now.toDateString()) return 'Today';
  if (d.toDateString() === yesterday.toDateString()) return 'Yesterday';
  return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
}

// ── Record button ─────────────────────────────────────────────────────────────
function RecordButton({ onPress, colors }: { onPress: () => void; colors: any }) {
  const scaleAnim = useRef(new Animated.Value(0.84)).current;

  useEffect(() => {
    Animated.spring(scaleAnim, {
      toValue: 1,
      useNativeDriver: true,
      tension: 55,
      friction: 7,
      delay: 250,
    }).start();
  }, []);

  const handlePressIn = useCallback(() => {
    Animated.spring(scaleAnim, { toValue: 0.93, useNativeDriver: true, tension: 140, friction: 9 }).start();
  }, [scaleAnim]);

  const handlePressOut = useCallback(() => {
    Animated.spring(scaleAnim, { toValue: 1, useNativeDriver: true, tension: 55, friction: 7 }).start();
    onPress();
  }, [scaleAnim, onPress]);

  return (
    <TouchableOpacity activeOpacity={1} onPressIn={handlePressIn} onPressOut={handlePressOut}>
      <Animated.View
        style={[
          styles.recBtn,
          { backgroundColor: colors.brand, shadowColor: colors.brand, transform: [{ scale: scaleAnim }] },
        ]}
      >
        <Ionicons name="mic" size={38} color={colors.brandCore} />
      </Animated.View>
    </TouchableOpacity>
  );
}

// ── Weekly mood strip ─────────────────────────────────────────────────────────
function WeekStrip({ days, colors, moodToColor }: { days: DailyMood[]; colors: any; moodToColor: (n: number) => string }) {
  const dayLabels = ['M', 'T', 'W', 'T', 'F', 'S', 'S'];
  const today = new Date();
  const todayDow = today.getDay();
  const slots = Array.from({ length: 7 }, (_, i) => {
    const offset = i - ((todayDow + 6) % 7);
    const d = new Date(today);
    d.setDate(today.getDate() + offset);
    const key = d.toISOString().slice(0, 10);
    return days.find((m) => m.day === key) ?? null;
  });

  const barAnims = useRef(slots.map(() => new Animated.Value(0))).current;

  useEffect(() => {
    Animated.stagger(
      38,
      barAnims.map((a) =>
        Animated.spring(a, { toValue: 1, useNativeDriver: false, tension: 70, friction: 8 }),
      ),
    ).start();
  }, []);

  return (
    <View style={styles.stripWrap}>
      <Text style={[styles.stripLabel, { color: colors.textMuted }]}>this week</Text>
      <View style={styles.stripBars}>
        {slots.map((slot, i) => {
          const targetH = slot ? Math.max(8, (slot.avg_mood / 100) * 38) : 8;
          const color = slot ? moodToColor(slot.avg_mood) : null;
          return (
            <View key={i} style={styles.stripCol}>
              <Animated.View
                style={
                  slot
                    ? {
                        width: '100%',
                        borderRadius: 3,
                        borderTopWidth: 1.5,
                        borderTopColor: color!,
                        backgroundColor: color! + '44',
                        height: barAnims[i].interpolate({ inputRange: [0, 1], outputRange: [0, targetH] }),
                      }
                    : {
                        width: '100%',
                        height: 8,
                        borderRadius: 3,
                        borderWidth: 1,
                        borderStyle: 'dashed',
                        borderColor: colors.borderFaint,
                      }
                }
              />
              <Text style={[styles.stripDay, { color: colors.textMuted }]}>{dayLabels[i]}</Text>
            </View>
          );
        })}
      </View>
    </View>
  );
}

// ── Home screen ───────────────────────────────────────────────────────────────
export default function HomeScreen() {
  const router = useRouter();
  const { colors, moodToColor } = useTheme();
  const { isAuthenticated, requestAuth } = useAuth();
  const { registerRef, checkAndStartTour } = useGuidedTour();

  // Refs for guided tour spotlight targets
  const recordRef    = useRef<View>(null);
  const weekStripRef = useRef<View>(null);

  const [userName, setUserName] = useState('');
  const [streak, setStreak] = useState<StreakInfo | null>(null);
  const [weekMoods, setWeekMoods] = useState<DailyMood[]>([]);
  const [lastEntry, setLastEntry] = useState<{
    id: string;
    dateLabel: string;
    emotionLabel: string;
    moodScore: number;
    quote: string;
    topic: string;
  } | null>(null);

  const topAnim    = useRef(new Animated.Value(0)).current;
  const lastAnim   = useRef(new Animated.Value(0)).current;
  const centerAnim = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    registerRef('record', recordRef);
    registerRef('week_strip', weekStripRef);
    checkAndStartTour();

    Animated.stagger(70, [
      Animated.spring(topAnim,    { toValue: 1, useNativeDriver: true, tension: 70, friction: 8 }),
      Animated.spring(lastAnim,   { toValue: 1, useNativeDriver: true, tension: 70, friction: 8 }),
      Animated.spring(centerAnim, { toValue: 1, useNativeDriver: true, tension: 70, friction: 8 }),
    ]).start();

    if (!isAuthenticated) return;

    api.me().then((u) => setUserName(u.preferred_name || u.name || '')).catch(() => {});
    api.streak().then(setStreak).catch(() => {});
    api.weeklyMood().then((r) => setWeekMoods(r.days ?? [])).catch(() => {});

    api.getTimeline(1, 1).then((res) => {
      const item = res.entries?.[0];
      if (item?.analysis && item.entry.status === 'completed') {
        const a = item.analysis;
        setLastEntry({
          id:           item.entry.id,
          dateLabel:    formatEntryDate(item.entry.created_at),
          emotionLabel: a.emotional_tone?.[0]?.emotion ?? '',
          moodScore:    a.mood_score,
          quote:        a.key_quotes?.[0] ?? '',
          topic:        a.topics?.[0] ?? '',
        });
      }
    }).catch(() => {});
  }, [isAuthenticated]);

  const handleRecord = useCallback(() => {
    if (isAuthenticated) {
      router.push('/record');
    } else {
      requestAuth(() => router.push('/record'));
    }
  }, [isAuthenticated, requestAuth, router]);

  const openLastReflection = useCallback(() => {
    if (lastEntry) router.push(`/reflection/${lastEntry.id}` as any);
  }, [lastEntry, router]);

  const greeting = (() => {
    const h = new Date().getHours();
    if (h < 12) return 'Good morning';
    if (h < 17) return 'Good afternoon';
    if (h < 21) return 'Good evening';
    return 'Hey there';
  })();

  const todayLabel = new Date().toLocaleDateString('en-US', {
    weekday: 'long', day: 'numeric', month: 'long',
  });

  const topTranslate    = topAnim.interpolate({ inputRange: [0, 1], outputRange: [10, 0] });
  const lastTranslate   = lastAnim.interpolate({ inputRange: [0, 1], outputRange: [10, 0] });
  const centerTranslate = centerAnim.interpolate({ inputRange: [0, 1], outputRange: [12, 0] });

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />

      <SafeAreaView style={styles.safe}>

        {/* ── Top row: greeting + streak ── */}
        <Animated.View style={[styles.topRow, { opacity: topAnim, transform: [{ translateY: topTranslate }] }]}>
          <View>
            <Text style={[styles.dateSub, { color: colors.textMuted }]}>{todayLabel}</Text>
            <Text style={[styles.greetingName, { color: colors.textPrimary }]}>
              {greeting}{userName ? `,\n${userName}.` : '.'}
            </Text>
          </View>
          {streak && streak.current_streak > 0 && (
            <View style={styles.streakBlock}>
              <Text style={[styles.streakNum, { color: colors.textPrimary }]}>{streak.current_streak}</Text>
              <Text style={[styles.streakLabel, { color: colors.textMuted }]}>day streak</Text>
            </View>
          )}
        </Animated.View>

        {/* ── Last entry snippet (authenticated users only) — taps into the reflection ── */}
        {lastEntry && isAuthenticated && (
          <Animated.View
            style={[
              styles.lastWrap,
              { borderTopColor: colors.borderFaint, opacity: lastAnim, transform: [{ translateY: lastTranslate }] },
            ]}
          >
            <TouchableOpacity onPress={openLastReflection} activeOpacity={0.7}>
              <View style={styles.lastHeaderRow}>
                <Text style={[styles.lastMeta, { color: colors.textMuted }]}>
                  {lastEntry.dateLabel}{lastEntry.emotionLabel ? ` · ${lastEntry.emotionLabel}` : ''}
                </Text>
                <Text style={[styles.lastChevron, { color: colors.textMuted }]}>›</Text>
              </View>
              {lastEntry.topic ? (
                <View style={styles.lastRow}>
                  <View style={[styles.lastDot, { backgroundColor: moodToColor(lastEntry.moodScore) }]} />
                  <Text style={[styles.lastScore, { color: colors.textSecondary }]}>
                    {lastEntry.moodScore}{lastEntry.topic ? ` · ${lastEntry.topic}` : ''}
                  </Text>
                </View>
              ) : null}
              {lastEntry.quote ? (
                <Text style={[styles.lastQuote, { color: colors.textSecondary }]} numberOfLines={2}>
                  "{lastEntry.quote}"
                </Text>
              ) : null}
            </TouchableOpacity>
          </Animated.View>
        )}

        {/* ── Guest hint (shown only to unauthenticated users) ── */}
        {!isAuthenticated && (
          <Animated.View
            style={[
              styles.lastWrap,
              { borderTopColor: colors.borderFaint, opacity: lastAnim, transform: [{ translateY: lastTranslate }] },
            ]}
          >
            <Text style={[styles.guestHint, { color: colors.textMuted }]}>
              Tap record to start your first entry
            </Text>
          </Animated.View>
        )}

        {/* ── Center: record button ── */}
        <Animated.View
          style={[styles.centerWrap, { opacity: centerAnim, transform: [{ translateY: centerTranslate }] }]}
        >
          <View ref={recordRef} collapsable={false}>
            <RecordButton onPress={handleRecord} colors={colors} />
          </View>
          <Text style={[styles.recHint, { color: colors.textMuted }]}>record</Text>
        </Animated.View>

        {/* ── Mood strip ── */}
        <View ref={weekStripRef} collapsable={false} style={[styles.stripSection, { borderTopColor: colors.borderFaint }]}>
          <WeekStrip days={weekMoods} colors={colors} moodToColor={moodToColor} />
        </View>

      </SafeAreaView>
    </View>
  );
}

const REC_SIZE = 108;

const styles = StyleSheet.create({
  container: { flex: 1 },
  safe: {
    flex: 1,
    paddingHorizontal: 26,
    paddingBottom: 16,
  },

  topRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    paddingTop: 8,
    marginBottom: 20,
  },
  dateSub: {
    fontSize: 10,
    fontFamily: 'Nunito_400Regular',
    fontWeight: '300',
    marginBottom: 4,
    letterSpacing: 0.3,
  },
  greetingName: {
    fontSize: 24,
    fontFamily: 'CormorantGaramond_300Light',
    fontWeight: '300',
    lineHeight: 30,
  },
  streakBlock: { alignItems: 'flex-end' },
  streakNum: {
    fontSize: 26,
    fontFamily: 'CormorantGaramond_300Light',
    fontWeight: '300',
    lineHeight: 28,
  },
  streakLabel: {
    fontSize: 9,
    fontFamily: 'Nunito_400Regular',
    fontWeight: '300',
    marginTop: 2,
  },

  lastWrap: {
    paddingTop: 16,
    marginBottom: 4,
    borderTopWidth: 1,
  },
  lastHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  lastChevron: {
    fontSize: 16,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 16,
  },
  lastMeta: {
    fontSize: 9.5,
    fontFamily: 'Nunito_400Regular',
    fontWeight: '300',
    marginBottom: 5,
  },
  lastRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 7,
    marginBottom: 5,
  },
  lastDot: { width: 5, height: 5, borderRadius: 3 },
  lastScore: {
    fontSize: 10.5,
    fontFamily: 'Nunito_400Regular',
    fontWeight: '300',
  },
  lastQuote: {
    fontSize: 13,
    fontFamily: 'CormorantGaramond_300Light',
    fontStyle: 'italic',
    fontWeight: '300',
    lineHeight: 20,
  },
  guestHint: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    fontWeight: '300',
    fontStyle: 'italic',
  },

  centerWrap: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 14,
  },
  recBtn: {
    width: REC_SIZE,
    height: REC_SIZE,
    borderRadius: REC_SIZE / 2,
    alignItems: 'center',
    justifyContent: 'center',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.18,
    shadowRadius: 16,
    elevation: 8,
  },
  recHint: {
    fontSize: 11,
    fontFamily: 'Nunito_400Regular',
    fontWeight: '300',
  },

  stripSection: {
    borderTopWidth: 1,
    paddingTop: 12,
  },
  stripWrap: {},
  stripLabel: {
    fontSize: 9,
    fontFamily: 'Nunito_400Regular',
    fontWeight: '300',
    marginBottom: 8,
    letterSpacing: 0.3,
  },
  stripBars: {
    flexDirection: 'row',
    gap: 5,
    alignItems: 'flex-end',
    height: 44,
  },
  stripCol: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: 4,
  },
  stripDay: {
    fontSize: 8,
    fontFamily: 'Nunito_400Regular',
  },
});
