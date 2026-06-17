import { useEffect, useState, useCallback } from 'react';
import {
  View,
  Text,
  ScrollView,
  StyleSheet,
  StatusBar,
  ActivityIndicator,
  TouchableOpacity,
  Share,
  Alert,
  Modal,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { AnnualReview, DailyMood, MoodArcDay, MoodHistoryResponse, StreakInfo, WeeklyReview } from '../../src/types';
import ShareInsightModal from '../../src/components/ShareInsightModal';
import PatternRadar from '../../src/components/PatternRadar';

// ── Milestone data ─────────────────────────────────────────────────────────────

const MILESTONES = [7, 21, 50, 100];

function getMilestoneLabel(streak: number): string | null {
  if (streak === 7) return 'One week. Every single day.';
  if (streak === 21) return '21 days in a row.';
  if (streak === 50) return '50 days of showing up.';
  if (streak === 100) return '100 days. Exceptional.';
  return null;
}

function getMilestoneMessage(streak: number): string {
  if (streak === 7) return "You've journaled every day for a week. Your consistency is building something real.";
  if (streak === 21) return "21 days. You've made this a habit. The research is clear - this is when it becomes part of you.";
  if (streak === 50) return "50 days of showing up. That's not luck or motivation. That's character.";
  if (streak === 100) return "100 days. You are exceptional. What you've built here - this discipline, this self-awareness - is rare.";
  return '';
}

// ── Sub-components ────────────────────────────────────────────────────────────

function StreakCard({
  value,
  label,
  highlight = false,
}: {
  value: number;
  label: string;
  highlight?: boolean;
}) {
  const { colors } = useTheme();
  const styles = getStyles(colors);
  return (
    <View style={[styles.streakCard, highlight && styles.streakCardHL]}>
      <Text style={[styles.streakValue, highlight && styles.streakValueHL]}>{value}</Text>
      <Text style={styles.streakLabel}>{label}</Text>
    </View>
  );
}

function MoodSparkline({ days }: { days: DailyMood[] }) {
  const { colors, moodToColor } = useTheme();
  const styles = getStyles(colors);
  if (days.length === 0) return null;
  const maxScore = 100;
  const chartH = 100;

  return (
    <View style={styles.sparklineWrap}>
      {[25, 50, 75].map((v) => (
        <View key={v} style={[styles.gridLine, { bottom: (v / maxScore) * chartH }]} />
      ))}
      {days.map((day) => {
        const h = Math.max(6, (day.avg_mood / maxScore) * chartH);
        const color = moodToColor(day.avg_mood);
        const label = new Date(day.day + 'T00:00:00Z').toLocaleDateString('en-US', {
          weekday: 'short',
          timeZone: 'UTC',
        });
        return (
          <View key={day.day} style={styles.barCol}>
            <View style={styles.barTrack}>
              <View
                style={[
                  styles.bar,
                  { height: h, backgroundColor: color + '55', borderTopColor: color },
                ]}
              />
            </View>
            <Text style={styles.barLabel}>{label.slice(0, 1)}</Text>
          </View>
        );
      })}
    </View>
  );
}

function WeeklyReviewCard({ review }: { review: WeeklyReview }) {
  const { colors } = useTheme();
  const styles = getStyles(colors);
  const date = new Date(review.week_start + 'T00:00:00Z');
  const weekLabel = date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  });
  return (
    <View style={styles.reviewCard}>
      <Text style={styles.reviewWeekLabel}>Week of {weekLabel}</Text>
      <Text style={styles.reviewNarrative}>{review.narrative}</Text>
      {review.top_emotions.length > 0 && (
        <View style={styles.reviewEmotionRow}>
          {review.top_emotions.map((e) => (
            <View key={e} style={styles.reviewEmotionTag}>
              <Text style={styles.reviewEmotionText}>{e}</Text>
            </View>
          ))}
        </View>
      )}
    </View>
  );
}

function YearInReviewCard({ review }: { review: AnnualReview }) {
  const { colors } = useTheme();
  const styles = getStyles(colors);

  const maxMood = review.mood_arc.reduce((m, d) => Math.max(m, d.avg_mood), 1);

  return (
    <View style={styles.reviewCard}>
      <Text style={styles.reviewWeekLabel}>{review.year} in Review</Text>
      <View style={styles.yearReviewMeta}>
        <Text style={styles.yearReviewMetaText}>{review.entry_count} entries</Text>
        {review.avg_mood != null && (
          <Text style={styles.yearReviewMetaText}>avg mood {review.avg_mood}</Text>
        )}
      </View>
      <Text style={styles.reviewNarrative}>{review.narrative}</Text>

      {/* Monthly mood arc - compact bar chart */}
      {review.mood_arc.length > 0 && (
        <View style={styles.yearArcWrap}>
          {review.mood_arc.map((d) => {
            const h = Math.max(4, (d.avg_mood / Math.max(maxMood, 1)) * 28);
            const label = d.month.slice(5, 7);
            return (
              <View key={d.month} style={styles.yearArcCol}>
                <View style={[styles.yearArcBar, { height: h, backgroundColor: colors.purple500 + 'AA' }]} />
                <Text style={[styles.yearArcLabel, { color: colors.textFaint }]}>{label}</Text>
              </View>
            );
          })}
        </View>
      )}

      {/* Top emotions */}
      {review.top_emotions.length > 0 && (
        <View style={styles.reviewEmotionRow}>
          {review.top_emotions.slice(0, 5).map((e) => (
            <View key={e} style={styles.reviewEmotionTag}>
              <Text style={styles.reviewEmotionText}>{e}</Text>
            </View>
          ))}
        </View>
      )}

      {/* Top topics */}
      {review.top_topics.length > 0 && (
        <View style={[styles.reviewEmotionRow, { marginTop: 6 }]}>
          {review.top_topics.slice(0, 5).map((t) => (
            <View key={t} style={[styles.reviewEmotionTag, { backgroundColor: colors.card }]}>
              <Text style={[styles.reviewEmotionText, { color: colors.textMuted }]}>{t}</Text>
            </View>
          ))}
        </View>
      )}
    </View>
  );
}

function MilestoneModal({
  streak,
  onClose,
  onShare,
}: {
  streak: number;
  onClose: () => void;
  onShare: () => void;
}) {
  const { colors } = useTheme();
  const styles = getStyles(colors);
  const label = getMilestoneLabel(streak);
  const message = getMilestoneMessage(streak);
  if (!label) return null;

  return (
    <Modal transparent animationType="fade" onRequestClose={onClose}>
      <View style={styles.milestoneOverlay}>
        <View style={styles.milestoneCard}>
          <Text style={styles.milestoneTitle}>{label}</Text>
          <Text style={styles.milestoneMessage}>{message}</Text>
          <View style={styles.milestoneActions}>
            <TouchableOpacity style={styles.shareBtn} onPress={onShare}>
              <Text style={styles.shareBtnText}>Share</Text>
            </TouchableOpacity>
            <TouchableOpacity style={styles.closeBtn} onPress={onClose}>
              <Text style={styles.closeBtnText}>Continue</Text>
            </TouchableOpacity>
          </View>
        </View>
      </View>
    </Modal>
  );
}

// ── Life Graph component ──────────────────────────────────────────────────────

type HistoryRange = '30d' | '90d' | '365d';

function LifeGraph({ history, range, onRangeChange }: {
  history: MoodHistoryResponse | null;
  range: HistoryRange;
  onRangeChange: (r: HistoryRange) => void;
}) {
  const { colors, moodToColor } = useTheme();
  const styles = getStyles(colors);
  const ranges: HistoryRange[] = ['30d', '90d', '365d'];
  const rangeLabels: Record<HistoryRange, string> = { '30d': '30D', '90d': '90D', '365d': '1Y' };

  const days = history?.days ?? [];
  const chartH = 80;
  const maxScore = 100;

  return (
    <View style={styles.lifeGraphCard}>
      <View style={styles.lifeGraphHeader}>
        <Text style={styles.chartTitle}>LIFE GRAPH</Text>
        <View style={styles.rangeSelector}>
          {ranges.map((r) => (
            <TouchableOpacity
              key={r}
              style={[styles.rangeBtn, range === r && styles.rangeBtnActive]}
              onPress={() => onRangeChange(r)}
            >
              <Text style={[styles.rangeBtnText, range === r && styles.rangeBtnTextActive]}>
                {rangeLabels[r]}
              </Text>
            </TouchableOpacity>
          ))}
        </View>
      </View>

      {/* Delta summary */}
      {history && history.avg_mood != null && (
        <View style={styles.lifeGraphSummary}>
          <Text style={styles.lifeGraphAvg}>
            Avg mood <Text style={{ color: moodToColor(history.avg_mood) }}>{history.avg_mood}</Text>
          </Text>
          {history.mood_delta != null && (
            <Text style={[
              styles.lifeGraphDelta,
              { color: history.mood_delta >= 0 ? colors.moodGreen : colors.moodRed },
            ]}>
              {history.mood_delta >= 0 ? '+' : ''}{history.mood_delta} vs prior period
            </Text>
          )}
          <Text style={styles.lifeGraphEntries}>{history.entry_count} entries</Text>
        </View>
      )}

      {/* Trendline chart */}
      {days.length === 0 ? (
        <Text style={[styles.chartEmpty, { paddingVertical: 24 }]}>
          Keep journaling to see your life graph.
        </Text>
      ) : (
        <View style={[styles.sparklineWrap, { height: chartH + 20, marginBottom: 0 }]}>
          {[25, 50, 75].map((v) => (
            <View key={v} style={[styles.gridLine, { bottom: (v / maxScore) * chartH + 4 }]} />
          ))}
          {days.map((day) => {
            const h = Math.max(3, (day.avg_mood / maxScore) * chartH);
            const color = moodToColor(day.avg_mood);
            return (
              <View key={day.day} style={[styles.barCol, { height: chartH + 4 }]}>
                <View style={styles.barTrack}>
                  <View style={[styles.bar, { height: h, backgroundColor: color + '44', borderTopColor: color }]} />
                </View>
              </View>
            );
          })}
        </View>
      )}

      {/* Top emotions */}
      {history && history.top_emotions.length > 0 && (
        <View style={[styles.emotionRow, { marginTop: 12 }]}>
          <Text style={[styles.lifeGraphEntries, { marginRight: 6, alignSelf: 'center' }]}>Top:</Text>
          {history.top_emotions.map((e) => (
            <View key={e} style={styles.reviewEmotionTag}>
              <Text style={styles.reviewEmotionText}>{e}</Text>
            </View>
          ))}
        </View>
      )}
    </View>
  );
}

// ── Main screen ───────────────────────────────────────────────────────────────

export default function MoodScreen() {
  const { colors } = useTheme();
  const styles = getStyles(colors);
  const [moods, setMoods] = useState<DailyMood[]>([]);
  const [streak, setStreak] = useState<StreakInfo | null>(null);
  const [weeklyReview, setWeeklyReview] = useState<WeeklyReview | null>(null);
  const [annualReview, setAnnualReview] = useState<AnnualReview | null>(null);
  const [loading, setLoading] = useState(true);
  const [showMilestone, setShowMilestone] = useState(false);
  const [showShareModal, setShowShareModal] = useState(false);
  const [usingFreeze, setUsingFreeze] = useState(false);
  const [historyRange, setHistoryRange] = useState<HistoryRange>('30d');
  const [moodHistory, setMoodHistory] = useState<MoodHistoryResponse | null>(null);

  const emotionColors = [colors.moodYellow, colors.moodGreen, colors.moodRed, colors.info, colors.moodOrange];

  const loadData = useCallback(() => {
    setLoading(true);
    Promise.all([
      api.weeklyMood(),
      api.streak(),
      api.getLatestWeeklyReview().catch(() => null),
      api.moodHistory(historyRange).catch(() => null),
      api.getLatestAnnualReview().catch(() => null),
    ])
      .then(([m, s, rv, hist, annual]) => {
        setMoods(m.days ?? []);
        setStreak(s);
        setWeeklyReview(rv);
        setMoodHistory(hist);
        setAnnualReview(annual);
        if (s && MILESTONES.includes(s.current_streak)) {
          setShowMilestone(true);
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [historyRange]);

  const handleRangeChange = useCallback((r: HistoryRange) => {
    setHistoryRange(r);
    api.moodHistory(r).then(setMoodHistory).catch(() => {});
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleUseFreeze = async () => {
    if (!streak || streak.freeze_count <= 0) {
      Alert.alert('No Freezes', 'You have no streak freezes available this week.');
      return;
    }
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const dateStr = yesterday.toISOString().split('T')[0];

    setUsingFreeze(true);
    try {
      const result = await api.useStreakFreeze(dateStr);
      setStreak((prev) =>
        prev ? { ...prev, freeze_count: result.freeze_count } : prev
      );
      Alert.alert('Freeze Applied', 'Yesterday is protected. Your streak lives on.');
      // Re-fetch streak to get updated current_streak.
      const updated = await api.streak();
      setStreak(updated);
    } catch {
      Alert.alert('Error', 'Could not apply freeze. Try again.');
    } finally {
      setUsingFreeze(false);
    }
  };

  const handleShare = async () => {
    if (!streak) return;
    const message = streak.current_streak > 0
      ? `I've journaled for ${streak.current_streak} days in a row on DreamLog. Building emotional awareness, one day at a time.`
      : `I've logged ${streak.total_days} journal entries on DreamLog. Voice journaling is changing how I understand myself.`;
    try {
      await Share.share({ message });
    } catch {
      // Ignore share cancellation.
    }
  };

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor: colors.bg }]}>
        <SafeAreaView style={styles.center}>
          <ActivityIndicator color={colors.purple400} />
        </SafeAreaView>
      </View>
    );
  }

  const streakBroken = streak && streak.current_streak === 0 && streak.total_days > 0;

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
          <Text style={styles.title}>Mood Map</Text>
          <Text style={styles.subtitle}>Last 7 days</Text>

          {/* Streak cards */}
          {streak && (
            <>
              <View style={styles.streakRow}>
                <StreakCard
                  value={streak.current_streak}
                  label="day streak"
                  highlight={streak.current_streak >= 3}
                />
                <StreakCard value={streak.longest_streak} label="best streak" />
                <StreakCard value={streak.total_days} label="days logged" />
              </View>

              {/* Comeback / milestone / freeze row */}
              {streakBroken ? (
                <View style={styles.comebackCard}>
                  <Text style={styles.comebackTitle}>Every streak has a pause.</Text>
                  <Text style={styles.comebackBody}>
                    You've journaled {streak.total_days} days total. That doesn't disappear.
                    Today is just day one of the next streak.
                  </Text>
                  {streak.freeze_count > 0 && (
                    <TouchableOpacity
                      style={styles.freezeBtn}
                      onPress={handleUseFreeze}
                      disabled={usingFreeze}
                    >
                      <Text style={styles.freezeBtnText}>
                        {usingFreeze ? 'Applying…' : `Use Freeze (${streak.freeze_count} left)`}
                      </Text>
                    </TouchableOpacity>
                  )}
                </View>
              ) : streak.current_streak > 0 ? (
                <View style={styles.streakFooter}>
                  {streak.next_milestone > 0 && (
                    <Text style={styles.milestoneHint}>
                      {streak.next_milestone - streak.current_streak} days to {streak.next_milestone}-day milestone
                    </Text>
                  )}
                  <View style={styles.streakActions}>
                    {streak.freeze_count > 0 && (
                      <Text style={styles.freezeAvail}>
                        ❄ {streak.freeze_count} freeze{streak.freeze_count !== 1 ? 's' : ''} available
                      </Text>
                    )}
                    <TouchableOpacity onPress={handleShare}>
                      <Text style={styles.shareLink}>Share streak</Text>
                    </TouchableOpacity>
                  </View>
                </View>
              ) : null}
            </>
          )}

          {/* Mood chart */}
          <View style={styles.chartCard}>
            <Text style={styles.chartTitle}>MOOD TREND</Text>
            {moods.length === 0 ? (
              <Text style={styles.chartEmpty}>Keep journaling to see your mood trend.</Text>
            ) : (
              <MoodSparkline days={moods} />
            )}
            {moods.length > 0 && (
              <View style={styles.emotionRow}>
                {moods.slice(0, 5).map((d, i) => (
                  <View key={d.day} style={[styles.emotionTag, { borderColor: emotionColors[i % emotionColors.length] + '40' }]}>
                    <View style={[styles.emotionDot, { backgroundColor: emotionColors[i % emotionColors.length] }]} />
                    <Text style={[styles.emotionLabel, { color: emotionColors[i % emotionColors.length] + 'cc' }]}>
                      mood {d.avg_mood}
                    </Text>
                  </View>
                ))}
              </View>
            )}
          </View>

          {/* Weekly review */}
          {weeklyReview && (
            <>
              <View style={styles.sectionHeader}>
                <Text style={styles.sectionLabel}>WEEKLY REVIEW</Text>
                <TouchableOpacity onPress={() => setShowShareModal(true)}>
                  <Text style={styles.shareInsightLink}>Share card</Text>
                </TouchableOpacity>
              </View>
              <WeeklyReviewCard review={weeklyReview} />
            </>
          )}

          {/* Share insight button when there's mood data but no weekly review */}
          {!weeklyReview && moods.length > 0 && (
            <TouchableOpacity
              style={styles.shareInsightBtn}
              onPress={() => setShowShareModal(true)}
            >
              <Text style={styles.shareInsightBtnText}>Share This Week's Insight</Text>
            </TouchableOpacity>
          )}

          {/* Emotion Pattern Radar */}
          <PatternRadar />

          {/* Year in Review */}
          {annualReview && (
            <>
              <Text style={styles.sectionLabel}>YEAR IN REVIEW</Text>
              <YearInReviewCard review={annualReview} />
            </>
          )}

          {/* Life Graph */}
          <LifeGraph
            history={moodHistory}
            range={historyRange}
            onRangeChange={handleRangeChange}
          />
        </ScrollView>
      </SafeAreaView>

      {/* Milestone celebration modal */}
      {showMilestone && streak && MILESTONES.includes(streak.current_streak) && (
        <MilestoneModal
          streak={streak.current_streak}
          onClose={() => setShowMilestone(false)}
          onShare={() => {
            setShowMilestone(false);
            handleShare();
          }}
        />
      )}

      {/* Shareable insight card modal */}
      <ShareInsightModal
        visible={showShareModal}
        onClose={() => setShowShareModal(false)}
        weekLabel={weeklyReview?.week_start
          ? formatWeekLabel(weeklyReview.week_start)
          : formatCurrentWeekLabel()}
        weekStart={weeklyReview?.week_start}
        moodArc={weeklyReview?.mood_arc ?? moodsToArc(moods)}
        topEmotions={weeklyReview?.top_emotions ?? []}
        streak={streak?.current_streak ?? 0}
        entryCount={weeklyReview?.entry_count ?? moods.reduce((s, d) => s + d.entry_count, 0)}
      />
    </View>
  );
}

// ── Week label helpers ─────────────────────────────────────────────────────────

function formatWeekLabel(weekStart: string): string {
  const start = new Date(weekStart + 'T00:00:00Z');
  const end = new Date(start);
  end.setUTCDate(end.getUTCDate() + 6);
  const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric', timeZone: 'UTC' };
  const s = start.toLocaleDateString('en-US', opts);
  const e = end.toLocaleDateString('en-US', { ...opts, year: 'numeric' });
  return `${s} – ${e}`;
}

function formatCurrentWeekLabel(): string {
  const now = new Date();
  const day = now.getUTCDay(); // 0=Sun
  const sun = new Date(now);
  sun.setUTCDate(now.getUTCDate() - day);
  return formatWeekLabel(sun.toISOString().split('T')[0]);
}

function moodsToArc(moods: DailyMood[]): MoodArcDay[] {
  return moods.map((d) => ({ date: d.day, avg_mood: d.avg_mood }));
}

// ── Styles ────────────────────────────────────────────────────────────────────

const getStyles = (colors: any) => StyleSheet.create({
  container: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  scroll: { padding: 20, paddingBottom: 60 },

  title: {
    fontSize: 26,
    color: colors.textPrimary,
    fontFamily: 'CormorantGaramond_300Light',
    marginBottom: 2,
  },
  subtitle: {
    fontSize: 11,
    color: colors.textMuted,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.5,
    marginBottom: 24,
  },

  streakRow: { flexDirection: 'row', gap: 10, marginBottom: 12 },
  streakCard: {
    flex: 1,
    backgroundColor: colors.cardSolid,
    borderRadius: 14,
    padding: 16,
    alignItems: 'center',
  },
  streakCardHL: {
    backgroundColor: colors.brandGlow,
    borderWidth: 1,
    borderColor: colors.border,
  },
  streakValue: {
    fontSize: 28,
    fontFamily: 'CormorantGaramond_400Regular',
    color: colors.textMuted,
    marginBottom: 2,
  },
  streakValueHL: { color: colors.info },
  streakLabel: {
    fontSize: 10,
    color: colors.textMuted,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
  },

  streakFooter: { marginBottom: 20 },
  milestoneHint: {
    fontSize: 12,
    color: colors.purple300,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
    marginBottom: 6,
  },
  streakActions: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    gap: 16,
  },
  freezeAvail: {
    fontSize: 11,
    color: colors.info,
    fontFamily: 'Nunito_400Regular',
  },
  shareLink: {
    fontSize: 11,
    color: colors.purple300,
    fontFamily: 'Nunito_400Regular',
    textDecorationLine: 'underline',
  },

  comebackCard: {
    backgroundColor: colors.brandGlow,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: colors.borderFaint,
    padding: 18,
    marginBottom: 20,
  },
  comebackTitle: {
    fontSize: 16,
    color: colors.textPrimary,
    fontFamily: 'CormorantGaramond_400Regular',
    marginBottom: 6,
  },
  comebackBody: {
    fontSize: 13,
    color: colors.textSecondary,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 20,
    marginBottom: 14,
  },
  freezeBtn: {
    backgroundColor: colors.brandGlow,
    borderRadius: 10,
    paddingVertical: 10,
    paddingHorizontal: 16,
    alignSelf: 'flex-start',
    borderWidth: 1,
    borderColor: colors.border,
  },
  freezeBtnText: {
    fontSize: 13,
    color: colors.info,
    fontFamily: 'Nunito_600SemiBold',
  },

  chartCard: {
    backgroundColor: colors.card,
    borderRadius: 20,
    borderWidth: 1,
    borderColor: colors.borderFaint,
    padding: 20,
    marginBottom: 24,
  },
  chartTitle: {
    fontSize: 10,
    color: colors.textSecondary,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.5,
    marginBottom: 16,
  },
  chartEmpty: {
    fontSize: 13,
    color: colors.textSecondary,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
    paddingVertical: 32,
  },

  sparklineWrap: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    height: 110,
    gap: 4,
    position: 'relative',
    marginBottom: 16,
  },
  gridLine: {
    position: 'absolute',
    left: 0,
    right: 0,
    height: 1,
    backgroundColor: 'rgba(139,92,246,0.06)',
  },
  barCol: { flex: 1, alignItems: 'center', height: '100%', justifyContent: 'flex-end' },
  barTrack: { width: '100%', flex: 1, justifyContent: 'flex-end' },
  bar: { width: '100%', borderRadius: 4, borderTopWidth: 2 },
  barLabel: {
    fontSize: 9,
    color: colors.textMuted,
    fontFamily: 'Nunito_400Regular',
    marginTop: 4,
  },

  emotionRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 6, marginTop: 8 },
  emotionTag: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    borderWidth: 1,
    borderRadius: 10,
    paddingHorizontal: 10,
    paddingVertical: 3,
    flexShrink: 0,
  },
  emotionDot: { width: 5, height: 5, borderRadius: 2.5 },
  emotionLabel: { fontSize: 10, fontFamily: 'Nunito_400Regular', flexShrink: 0 },

  sectionLabel: {
    fontSize: 10,
    color: colors.textSecondary,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.5,
    marginBottom: 12,
  },

  reviewCard: {
    backgroundColor: colors.card,
    borderRadius: 20,
    borderWidth: 1,
    borderColor: colors.borderFaint,
    padding: 20,
    marginBottom: 24,
  },
  reviewWeekLabel: {
    fontSize: 10,
    color: colors.textSecondary,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.2,
    marginBottom: 10,
    textTransform: 'uppercase',
  },
  reviewNarrative: {
    fontSize: 15,
    color: colors.textPrimary,
    fontFamily: 'CormorantGaramond_400Regular',
    lineHeight: 24,
    marginBottom: 14,
  },
  reviewEmotionRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 6 },
  reviewEmotionTag: {
    backgroundColor: colors.brandGlow,
    borderRadius: 10,
    paddingHorizontal: 10,
    paddingVertical: 3,
  },
  reviewEmotionText: {
    fontSize: 11,
    color: colors.purple300,
    fontFamily: 'Nunito_400Regular',
  },

  // Year in review card extras
  yearReviewMeta: { flexDirection: 'row', gap: 12, marginBottom: 8 },
  yearReviewMetaText: {
    fontSize: 11,
    color: colors.textMuted,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.4,
  },
  yearArcWrap: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    gap: 3,
    height: 40,
    marginBottom: 12,
  },
  yearArcCol: { flex: 1, alignItems: 'center', justifyContent: 'flex-end', gap: 3 },
  yearArcBar: { width: '100%', borderRadius: 2, minHeight: 4 },
  yearArcLabel: { fontSize: 8, fontFamily: 'Nunito_400Regular' },

  lifeGraphCard: {
    backgroundColor: colors.card,
    borderRadius: 20,
    borderWidth: 1,
    borderColor: colors.borderFaint,
    padding: 20,
    marginBottom: 24,
  },
  lifeGraphHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  rangeSelector: { flexDirection: 'row', gap: 4 },
  rangeBtn: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 8,
    backgroundColor: 'transparent',
  },
  rangeBtnActive: { backgroundColor: colors.brandGlow },
  rangeBtnText: {
    fontSize: 10,
    color: colors.textMuted,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 0.5,
  },
  rangeBtnTextActive: { color: colors.purple300 },
  lifeGraphSummary: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginBottom: 12,
  },
  lifeGraphAvg: {
    fontSize: 13,
    color: colors.textSecondary,
    fontFamily: 'Nunito_400Regular',
  },
  lifeGraphDelta: {
    fontSize: 12,
    fontFamily: 'Nunito_600SemiBold',
  },
  lifeGraphEntries: {
    fontSize: 11,
    color: colors.textMuted,
    fontFamily: 'Nunito_400Regular',
  },

  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  shareInsightLink: {
    fontSize: 11,
    color: colors.purple300,
    fontFamily: 'Nunito_400Regular',
    textDecorationLine: 'underline',
  },
  shareInsightBtn: {
    backgroundColor: colors.brandGlow,
    borderRadius: 14,
    paddingVertical: 14,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: colors.border,
    marginBottom: 24,
  },
  shareInsightBtnText: {
    color: colors.purple300,
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 14,
  },

  milestoneOverlay: {
    flex: 1,
    backgroundColor: 'rgba(10,5,20,0.85)',
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  milestoneCard: {
    backgroundColor: colors.cardSolid,
    borderRadius: 24,
    padding: 28,
    width: '100%',
    borderWidth: 1,
    borderColor: colors.border,
  },
  milestoneTitle: {
    fontSize: 22,
    color: colors.textPrimary,
    fontFamily: 'CormorantGaramond_400Regular',
    marginBottom: 12,
    textAlign: 'center',
  },
  milestoneMessage: {
    fontSize: 14,
    color: colors.textSecondary,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 22,
    textAlign: 'center',
    marginBottom: 24,
  },
  milestoneActions: { flexDirection: 'row', gap: 12 },
  shareBtn: {
    flex: 1,
    backgroundColor: colors.brandGlow,
    borderRadius: 12,
    paddingVertical: 12,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: colors.border,
  },
  shareBtnText: {
    color: colors.purple300,
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 14,
  },
  closeBtn: {
    flex: 1,
    backgroundColor: colors.card,
    borderRadius: 12,
    paddingVertical: 12,
    alignItems: 'center',
  },
  closeBtnText: {
    color: colors.textSecondary,
    fontFamily: 'Nunito_400Regular',
    fontSize: 14,
  },
});
