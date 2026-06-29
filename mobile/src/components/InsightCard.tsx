/**
 * InsightCard - the shareable weekly insight card.
 * Captured via react-native-view-shot and shared through the native share sheet.
 * Shows only anonymized data: avg mood + mood arc + top emotions + streak.
 * No journal content, names, or transcripts ever appear here.
 *
 * Colors come from the live ThemeContext so the card always matches the user's
 * currently chosen theme (espresso, blue, green, rose, etc.).
 *
 * NOTE: react-native-view-shot requires a dev build (not Expo Go).
 * Run `npx expo prebuild && npx expo run:android` or use EAS Build.
 */

import { forwardRef } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from '../context/ThemeContext';
import type { ThemeColors } from '../theme';
import type { MoodArcDay } from '../types';

export interface InsightCardProps {
  weekLabel: string;      // e.g. "May 26 – Jun 1, 2026"
  moodArc: MoodArcDay[];  // up to 7 days
  topEmotions: string[];  // up to 3 emotions
  streak: number;         // current streak (0 = don't show)
  entryCount: number;     // number of entries this week
}

const CARD_WIDTH = 375;
const CARD_HEIGHT = 560;

type MoodFn = (score: number) => string;

// ── Derived weekly stats ──────────────────────────────────────────────────────

function weekStats(days: MoodArcDay[]): { avg: number | null; bestDay: string | null } {
  const valid = days.filter((d) => d.avg_mood > 0);
  if (valid.length === 0) return { avg: null, bestDay: null };
  const avg = Math.round(valid.reduce((s, d) => s + d.avg_mood, 0) / valid.length);
  const best = valid.reduce((m, d) => (d.avg_mood > m.avg_mood ? d : m), valid[0]);
  const bestDay = new Date(best.date + 'T00:00:00Z').toLocaleDateString('en-US', {
    weekday: 'short',
    timeZone: 'UTC',
  });
  return { avg, bestDay };
}

function moodWord(avg: number): string {
  if (avg >= 71) return 'Bright';
  if (avg >= 46) return 'Steady';
  if (avg >= 26) return 'Tender';
  return 'Heavy';
}

// ── Mini bar chart inside the card ───────────────────────────────────────────

function MiniMoodArc({
  days,
  colors,
  moodToColor,
}: {
  days: MoodArcDay[];
  colors: ThemeColors;
  moodToColor: MoodFn;
}) {
  if (days.length === 0) return null;
  const maxScore = 100;
  const chartH = 80;

  return (
    <View style={arc.wrap}>
      {days.map((d) => {
        const h = Math.max(6, (d.avg_mood / maxScore) * chartH);
        const color = moodToColor(d.avg_mood);
        const label = new Date(d.date + 'T00:00:00Z').toLocaleDateString('en-US', {
          weekday: 'narrow',
          timeZone: 'UTC',
        });
        return (
          <View key={d.date} style={arc.col}>
            <View style={arc.track}>
              <View
                style={[
                  arc.bar,
                  { height: h, backgroundColor: color + '55', borderTopColor: color },
                ]}
              />
            </View>
            <Text style={[arc.label, { color: colors.textMuted }]}>{label}</Text>
          </View>
        );
      })}
    </View>
  );
}

const arc = StyleSheet.create({
  wrap: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    height: 100,
    gap: 6,
  },
  col: { flex: 1, alignItems: 'center', height: '100%', justifyContent: 'flex-end' },
  track: { width: '100%', flex: 1, justifyContent: 'flex-end' },
  bar: { width: '100%', borderRadius: 4, borderTopWidth: 2 },
  label: {
    fontSize: 9,
    fontFamily: 'Nunito_400Regular',
    marginTop: 4,
  },
});

// ── Main card component ───────────────────────────────────────────────────────

const InsightCard = forwardRef<View, InsightCardProps>(
  ({ weekLabel, moodArc, topEmotions, streak, entryCount }, ref) => {
    const { colors, moodToColor } = useTheme();
    const styles = getStyles(colors);
    const { avg, bestDay } = weekStats(moodArc);
    const avgColor = avg != null ? moodToColor(avg) : colors.textMuted;

    return (
      <View ref={ref} style={styles.card}>
        {/* Top accent bar */}
        <View style={styles.accentBar} />

        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.appName}>dreamlog</Text>
          <Text style={styles.weekLabel}>{weekLabel}</Text>
        </View>

        {/* Hero: average mood */}
        <View style={styles.heroRow}>
          <View>
            <Text style={styles.heroLabel}>AVG MOOD</Text>
            <View style={styles.heroValueRow}>
              <Text style={[styles.heroValue, { color: avgColor }]}>{avg != null ? avg : '—'}</Text>
              {avg != null && <Text style={styles.heroOutOf}>/100</Text>}
            </View>
          </View>
          {avg != null && (
            <View style={[styles.heroWordPill, { borderColor: avgColor + '55' }]}>
              <Text style={[styles.heroWord, { color: avgColor }]}>{moodWord(avg)}</Text>
            </View>
          )}
        </View>

        {/* Mood arc */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>MOOD ARC</Text>
          {moodArc.length > 0 ? (
            <MiniMoodArc days={moodArc} colors={colors} moodToColor={moodToColor} />
          ) : (
            <View style={styles.arcEmpty}>
              <Text style={styles.arcEmptyText}>No mood data</Text>
            </View>
          )}
        </View>

        {/* Top emotions */}
        {topEmotions.length > 0 && (
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>THIS WEEK&apos;S EMOTIONS</Text>
            <View style={styles.emotionRow}>
              {topEmotions.slice(0, 3).map((e) => (
                <View key={e} style={styles.emotionPill}>
                  <Text style={styles.emotionText}>{e}</Text>
                </View>
              ))}
            </View>
          </View>
        )}

        {/* Stats row */}
        <View style={styles.statsRow}>
          <View style={styles.statBox}>
            <Text style={styles.statValue}>{entryCount}</Text>
            <Text style={styles.statLabel}>entries</Text>
          </View>
          {bestDay && (
            <View style={styles.statBox}>
              <Text style={styles.statValue}>{bestDay}</Text>
              <Text style={styles.statLabel}>best day</Text>
            </View>
          )}
          {streak > 0 && (
            <View style={[styles.statBox, styles.statBoxHL]}>
              <Text style={[styles.statValue, styles.statValueHL]}>{streak}</Text>
              <Text style={styles.statLabel}>day streak</Text>
            </View>
          )}
        </View>

        {/* Footer */}
        <View style={styles.footer}>
          <Text style={styles.footerText}>Voice journaling for emotional clarity</Text>
          <Text style={styles.footerApp}>dreamlog.app</Text>
        </View>
      </View>
    );
  }
);

InsightCard.displayName = 'InsightCard';
export default InsightCard;

// ── Styles ────────────────────────────────────────────────────────────────────

const getStyles = (colors: ThemeColors) =>
  StyleSheet.create({
    card: {
      width: CARD_WIDTH,
      height: CARD_HEIGHT,
      backgroundColor: colors.bg,
      borderRadius: 0,
      padding: 28,
      justifyContent: 'space-between',
    },

    accentBar: {
      position: 'absolute',
      top: 0,
      left: 0,
      right: 0,
      height: 3,
      backgroundColor: colors.brand,
    },

    header: {
      marginTop: 10,
      marginBottom: 4,
    },
    appName: {
      fontSize: 13,
      color: colors.brand,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 2,
      marginBottom: 6,
      textTransform: 'lowercase',
    },
    weekLabel: {
      fontSize: 22,
      color: colors.textPrimary,
      fontFamily: 'CormorantGaramond_300Light',
      lineHeight: 28,
    },

    // Hero average-mood block
    heroRow: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'space-between',
      backgroundColor: colors.card,
      borderRadius: 16,
      borderWidth: 1,
      borderColor: colors.borderFaint,
      paddingVertical: 16,
      paddingHorizontal: 20,
    },
    heroLabel: {
      fontSize: 9,
      color: colors.textMuted,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 2,
      marginBottom: 4,
    },
    heroValueRow: { flexDirection: 'row', alignItems: 'flex-end' },
    heroValue: {
      fontSize: 46,
      fontFamily: 'CormorantGaramond_300Light',
      lineHeight: 48,
    },
    heroOutOf: {
      fontSize: 14,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      marginBottom: 7,
      marginLeft: 3,
    },
    heroWordPill: {
      borderWidth: 1,
      borderRadius: 100,
      paddingHorizontal: 16,
      paddingVertical: 7,
    },
    heroWord: {
      fontSize: 14,
      fontFamily: 'CormorantGaramond_400Regular',
      letterSpacing: 0.5,
    },

    section: {
      marginBottom: 2,
    },
    sectionTitle: {
      fontSize: 9,
      color: colors.textSecondary,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 2,
      marginBottom: 10,
    },

    arcEmpty: {
      height: 80,
      justifyContent: 'center',
      alignItems: 'center',
    },
    arcEmptyText: {
      fontSize: 12,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
    },

    emotionRow: {
      flexDirection: 'row',
      gap: 8,
      flexWrap: 'wrap',
    },
    emotionPill: {
      backgroundColor: colors.brandGlow,
      borderRadius: 20,
      paddingHorizontal: 14,
      paddingVertical: 5,
    },
    emotionText: {
      fontSize: 12,
      color: colors.purple300,
      fontFamily: 'Nunito_400Regular',
    },

    statsRow: {
      flexDirection: 'row',
      gap: 10,
    },
    statBox: {
      flex: 1,
      backgroundColor: colors.card,
      borderRadius: 12,
      borderWidth: 1,
      borderColor: colors.borderFaint,
      padding: 14,
      alignItems: 'center',
    },
    statBoxHL: {
      backgroundColor: colors.brandGlow,
      borderColor: colors.border,
    },
    statValue: {
      fontSize: 26,
      fontFamily: 'CormorantGaramond_400Regular',
      color: colors.textSecondary,
      marginBottom: 2,
    },
    statValueHL: { color: colors.brand },
    statLabel: {
      fontSize: 10,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      textAlign: 'center',
    },

    footer: {
      borderTopWidth: 1,
      borderTopColor: colors.borderFaint,
      paddingTop: 14,
      flexDirection: 'row',
      justifyContent: 'space-between',
      alignItems: 'center',
    },
    footerText: {
      fontSize: 10,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      flex: 1,
    },
    footerApp: {
      fontSize: 10,
      color: colors.brand,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 0.5,
    },
  });

export { CARD_WIDTH, CARD_HEIGHT };
