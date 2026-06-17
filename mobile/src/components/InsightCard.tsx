/**
 * InsightCard - the shareable weekly insight card.
 * Captured via react-native-view-shot and shared through the native share sheet.
 * Shows only anonymized data: mood arc + top emotions + streak. No journal content.
 *
 * NOTE: react-native-view-shot requires a dev build (not Expo Go).
 * Run `npx expo prebuild && npx expo run:android` or use EAS Build.
 */

import { forwardRef } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Colors, moodToColor } from '../theme';
import type { MoodArcDay } from '../types';

export interface InsightCardProps {
  weekLabel: string;      // e.g. "May 26 – Jun 1, 2026"
  moodArc: MoodArcDay[];  // up to 7 days
  topEmotions: string[];  // up to 3 emotions
  streak: number;         // current streak (0 = don't show)
  entryCount: number;     // number of entries this week
}

const CARD_WIDTH = 375;
const CARD_HEIGHT = 500;

// ── Mini bar chart inside the card ───────────────────────────────────────────

function MiniMoodArc({ days }: { days: MoodArcDay[] }) {
  if (days.length === 0) return null;
  const maxScore = 100;
  const chartH = 90;

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
                  { height: h, backgroundColor: color + '60', borderTopColor: color },
                ]}
              />
            </View>
            <Text style={arc.label}>{label}</Text>
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
    height: 110,
    gap: 6,
  },
  col: { flex: 1, alignItems: 'center', height: '100%', justifyContent: 'flex-end' },
  track: { width: '100%', flex: 1, justifyContent: 'flex-end' },
  bar: { width: '100%', borderRadius: 4, borderTopWidth: 2 },
  label: {
    fontSize: 9,
    color: Colors.textMuted,
    fontFamily: 'Nunito_400Regular',
    marginTop: 4,
  },
});

// ── Main card component ───────────────────────────────────────────────────────

const InsightCard = forwardRef<View, InsightCardProps>(
  ({ weekLabel, moodArc, topEmotions, streak, entryCount }, ref) => {
    return (
      <View ref={ref} style={styles.card}>
        {/* Top accent bar */}
        <View style={styles.accentBar} />

        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.appName}>dreamlog</Text>
          <Text style={styles.weekLabel}>{weekLabel}</Text>
        </View>

        {/* Mood arc */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>MOOD ARC</Text>
          {moodArc.length > 0 ? (
            <MiniMoodArc days={moodArc} />
          ) : (
            <View style={styles.arcEmpty}>
              <Text style={styles.arcEmptyText}>No mood data</Text>
            </View>
          )}
        </View>

        {/* Top emotions */}
        {topEmotions.length > 0 && (
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>THIS WEEK'S EMOTIONS</Text>
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

const styles = StyleSheet.create({
  card: {
    width: CARD_WIDTH,
    height: CARD_HEIGHT,
    backgroundColor: '#0d0b1a',
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
    backgroundColor: Colors.purple400,
    borderTopLeftRadius: 0,
    borderTopRightRadius: 0,
  },

  header: {
    marginTop: 10,
    marginBottom: 8,
  },
  appName: {
    fontSize: 13,
    color: Colors.purple400,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 2,
    marginBottom: 6,
    textTransform: 'lowercase',
  },
  weekLabel: {
    fontSize: 22,
    color: Colors.textPrimary,
    fontFamily: 'CormorantGaramond_300Light',
    lineHeight: 28,
  },

  section: {
    marginBottom: 4,
  },
  sectionTitle: {
    fontSize: 9,
    color: Colors.textSecondary,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 2,
    marginBottom: 10,
  },

  arcEmpty: {
    height: 90,
    justifyContent: 'center',
    alignItems: 'center',
  },
  arcEmptyText: {
    fontSize: 12,
    color: Colors.textMuted,
    fontFamily: 'Nunito_400Regular',
  },

  emotionRow: {
    flexDirection: 'row',
    gap: 8,
    flexWrap: 'wrap',
  },
  emotionPill: {
    backgroundColor: 'rgba(139,92,246,0.15)',
    borderRadius: 20,
    paddingHorizontal: 14,
    paddingVertical: 5,
    borderWidth: 1,
    borderColor: 'rgba(139,92,246,0.3)',
  },
  emotionText: {
    fontSize: 12,
    color: Colors.purple300,
    fontFamily: 'Nunito_400Regular',
  },

  statsRow: {
    flexDirection: 'row',
    gap: 10,
  },
  statBox: {
    flex: 1,
    backgroundColor: 'rgba(255,255,255,0.04)',
    borderRadius: 12,
    padding: 14,
    alignItems: 'center',
  },
  statBoxHL: {
    backgroundColor: 'rgba(93,155,245,0.08)',
    borderWidth: 1,
    borderColor: 'rgba(93,155,245,0.18)',
  },
  statValue: {
    fontSize: 28,
    fontFamily: 'CormorantGaramond_400Regular',
    color: Colors.textSecondary,
    marginBottom: 2,
  },
  statValueHL: { color: Colors.info },
  statLabel: {
    fontSize: 10,
    color: Colors.textMuted,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
  },

  footer: {
    borderTopWidth: 1,
    borderTopColor: 'rgba(139,92,246,0.12)',
    paddingTop: 14,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  footerText: {
    fontSize: 10,
    color: Colors.textMuted,
    fontFamily: 'Nunito_400Regular',
    flex: 1,
  },
  footerApp: {
    fontSize: 10,
    color: Colors.purple400,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 0.5,
  },
});

export { CARD_WIDTH, CARD_HEIGHT };
