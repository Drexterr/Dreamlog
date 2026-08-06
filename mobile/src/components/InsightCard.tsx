/**
 * InsightCard - the shareable weekly insight card.
 * Captured via react-native-view-shot and shared through the native share sheet.
 * Shows only anonymized data: avg mood + mood arc + top emotions + streak.
 * No journal content, names, or transcripts ever appear here.
 *
 * Colors come from the live ThemeContext so the card always matches the user's
 * currently chosen theme (espresso, blue, green, rose, etc.).
 *
 * Design: the Breath Line mark (see BrandOrbGlyph.tsx, BrandSplash.tsx) sits as
 * a quiet watermark and the mood arc is rendered in that same flowing-line
 * language, instead of a bordered stats-dashboard look.
 *
 * NOTE: react-native-view-shot requires a dev build (not Expo Go).
 * Run `npx expo prebuild && npx expo run:android` or use EAS Build.
 */

import { forwardRef } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import Svg, { Path, Circle, Polyline } from 'react-native-svg';
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
const CONTENT_WIDTH = CARD_WIDTH - 28 * 2; // matches card padding

// Icon crop of the Breath Line mark - same path as the web nav's OdeMark
// (docs/brand/ode-breath-line.html). Kept local to this file, matching how
// BrandSplash.tsx / BrandOrbGlyph.tsx each hold their own copy.
const MARK_D = 'M 8 48 C 17 24, 27 72, 37 48 C 45 30, 54 64, 62 48 C 67 42, 72 52, 76 48';

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

// ── Watermark: the Breath Line mark, quiet, top-right ────────────────────────

function BreathWatermark({ color }: { color: string }) {
  return (
    <Svg width={84} height={28} viewBox="0 24 96 48" style={watermarkStyle.svg} pointerEvents="none">
      <Path d={MARK_D} fill="none" stroke={color} strokeWidth={5.5} strokeLinecap="round" opacity={0.4} />
      <Circle cx={86} cy={48} r={4.5} fill={color} opacity={0.55} />
    </Svg>
  );
}

const watermarkStyle = StyleSheet.create({
  svg: {
    position: 'absolute',
    top: 24,
    right: 24,
  },
});

// ── Mood arc, rendered as one flowing line (the Breath Line's own language)
//    instead of bars - a lightly-smoothed curve through each day's score. ────

function MoodArcLine({
  days,
  colors,
  lineColor,
}: {
  days: MoodArcDay[];
  colors: ThemeColors;
  lineColor: string;
}) {
  if (days.length === 0) return null;
  const w = CONTENT_WIDTH;
  const h = 56;
  const pad = 6;
  const n = days.length;

  const pts = days.map((d, i) => {
    const x = n === 1 ? w / 2 : (i / (n - 1)) * (w - pad * 2) + pad;
    const y = h - pad - (Math.max(0, Math.min(100, d.avg_mood)) / 100) * (h - pad * 2);
    return { x, y };
  });

  // Quadratic midpoint smoothing - same technique as BrandOrbGlyph's morph line,
  // so the arc reads as the same signature curve used elsewhere in the app.
  let d = `M ${pts[0].x.toFixed(1)} ${pts[0].y.toFixed(1)} `;
  for (let i = 1; i < pts.length; i++) {
    const midX = (pts[i - 1].x + pts[i].x) / 2;
    const midY = (pts[i - 1].y + pts[i].y) / 2;
    d += `Q ${pts[i - 1].x.toFixed(1)} ${pts[i - 1].y.toFixed(1)}, ${midX.toFixed(1)} ${midY.toFixed(1)} `;
  }
  d += `L ${pts[pts.length - 1].x.toFixed(1)} ${pts[pts.length - 1].y.toFixed(1)}`;

  const dayLabels = days.map((day) =>
    new Date(day.date + 'T00:00:00Z').toLocaleDateString('en-US', { weekday: 'narrow', timeZone: 'UTC' })
  );

  return (
    <View>
      <Svg width={w} height={h}>
        <Path d={d} fill="none" stroke={lineColor} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />
        <Circle cx={pts[pts.length - 1].x} cy={pts[pts.length - 1].y} r={3} fill={lineColor} />
      </Svg>
      <View style={arc.labelRow}>
        {dayLabels.map((label, i) => (
          <Text key={i} style={[arc.label, { color: colors.textMuted }]}>
            {label}
          </Text>
        ))}
      </View>
    </View>
  );
}

const arc = StyleSheet.create({
  labelRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 4,
  },
  label: {
    fontSize: 9,
    fontFamily: 'HankenGrotesk_400Regular',
    opacity: 0.6,
  },
});

// ── Main card component ───────────────────────────────────────────────────────

const InsightCard = forwardRef<View, InsightCardProps>(
  ({ weekLabel, moodArc, topEmotions, streak, entryCount }, ref) => {
    const { colors, moodToColor } = useTheme();
    const styles = getStyles(colors);
    const { avg, bestDay } = weekStats(moodArc);
    const moodColor = avg != null ? moodToColor(avg) : colors.textMuted;

    return (
      <View ref={ref} style={styles.card}>
        <BreathWatermark color={colors.brand} />

        {/* Header */}
        <Text style={styles.weekLabel}>{weekLabel}</Text>

        {/* Hero: average mood */}
        <View style={styles.heroRow}>
          <Text style={[styles.heroValue, { color: moodColor }]}>{avg != null ? avg : '—'}</Text>
          {avg != null && <Text style={styles.heroOutOf}>/100</Text>}
        </View>
        {avg != null && <Text style={[styles.moodWordLine, { color: moodColor }]}>{moodWord(avg)} this week</Text>}

        {/* Mood arc */}
        <View style={styles.arcSection}>
          {moodArc.length > 0 ? (
            <MoodArcLine days={moodArc} colors={colors} lineColor={moodColor} />
          ) : (
            <View style={styles.arcEmpty}>
              <Text style={styles.arcEmptyText}>No mood data</Text>
            </View>
          )}
        </View>

        {/* Top emotions */}
        {topEmotions.length > 0 && (
          <Text style={styles.emotionsLine}>{topEmotions.slice(0, 3).join('  ·  ')}</Text>
        )}

        {/* Stats */}
        <View style={styles.statsLine}>
          <Text style={styles.statText}>{entryCount} {entryCount === 1 ? 'entry' : 'entries'}</Text>
          {bestDay && <Text style={styles.statText}>best {bestDay}</Text>}
          {streak > 0 && <Text style={styles.statText}>{streak}-day streak</Text>}
        </View>

        {/* Footer */}
        <View style={styles.footer}>
          <Text style={styles.footerApp}>ode</Text>
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
      padding: 28,
      justifyContent: 'flex-start',
    },

    weekLabel: {
      fontSize: 11,
      color: colors.textMuted,
      fontFamily: 'HankenGrotesk_600SemiBold',
      letterSpacing: 2,
      textTransform: 'uppercase',
      marginTop: 10,
      marginBottom: 28,
    },

    // Hero average-mood block
    heroRow: {
      flexDirection: 'row',
      alignItems: 'flex-end',
      gap: 6,
      marginBottom: 6,
    },
    heroValue: {
      fontSize: 52,
      fontFamily: 'Erode_300Light',
      lineHeight: 54,
    },
    heroOutOf: {
      fontSize: 13,
      color: colors.textMuted,
      fontFamily: 'HankenGrotesk_400Regular',
      marginBottom: 9,
    },
    moodWordLine: {
      fontSize: 15,
      fontFamily: 'Erode_400Regular',
      fontStyle: 'italic',
      marginBottom: 26,
    },

    arcSection: {
      marginBottom: 24,
    },
    arcEmpty: {
      height: 60,
      justifyContent: 'center',
      alignItems: 'center',
    },
    arcEmptyText: {
      fontSize: 12,
      color: colors.textMuted,
      fontFamily: 'HankenGrotesk_400Regular',
    },

    emotionsLine: {
      fontSize: 13,
      color: colors.textSecondary,
      fontFamily: 'HankenGrotesk_400Regular',
      marginBottom: 22,
    },

    statsLine: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      marginBottom: 20,
    },
    statText: {
      fontSize: 11,
      color: colors.textMuted,
      fontFamily: 'HankenGrotesk_400Regular',
    },

    footer: {
      flexDirection: 'row',
      justifyContent: 'flex-end',
    },
    footerApp: {
      fontSize: 11,
      color: colors.brand,
      fontFamily: 'HankenGrotesk_700Bold',
      letterSpacing: 0.5,
    },
  });

export { CARD_WIDTH, CARD_HEIGHT };
