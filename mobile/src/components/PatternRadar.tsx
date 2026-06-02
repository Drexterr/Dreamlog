/**
 * PatternRadar — emotional pattern visualization for the Mood screen.
 * Shows the top emotions as scored bars (frequency × intensity) with a
 * mood-distribution summary (high / neutral / low).
 * No external dependencies — pure React Native Views.
 */

import { useState, useCallback, useEffect } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ActivityIndicator } from 'react-native';
import { api } from '../api/client';
import { useTheme } from '../context/ThemeContext';
import type { PatternRadarResponse, EmotionPattern } from '../types';

type Range = '30d' | '90d' | '365d';
const RANGE_LABELS: Record<Range, string> = { '30d': '30D', '90d': '90D', '365d': '1Y' };

// ── Emotion bar row ───────────────────────────────────────────────────────────

function EmotionBar({
  pattern,
  maxScore,
  colors,
}: {
  pattern: EmotionPattern;
  maxScore: number;
  colors: ReturnType<typeof useTheme>['colors'];
}) {
  const fillPct = maxScore > 0 ? pattern.score / maxScore : 0;
  const intensityPct = Math.round(pattern.avg_intensity * 100);

  // Color intensity: low intensity → muted purple, high → bright
  const barColor =
    pattern.avg_intensity >= 0.7
      ? colors.purple400
      : pattern.avg_intensity >= 0.5
      ? colors.purple500
      : colors.purple600;

  return (
    <View style={barStyles.row}>
      <Text style={[barStyles.label, { color: colors.textSecondary }]} numberOfLines={1}>
        {pattern.emotion}
      </Text>
      <View style={[barStyles.track, { backgroundColor: colors.borderFaint }]}>
        <View
          style={[
            barStyles.fill,
            {
              width: `${Math.round(fillPct * 100)}%`,
              backgroundColor: barColor,
            },
          ]}
        />
      </View>
      <Text style={[barStyles.meta, { color: colors.textMuted }]}>
        {pattern.frequency}× · {intensityPct}%
      </Text>
    </View>
  );
}

const barStyles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 10,
  },
  label: {
    width: 90,
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    flexShrink: 0,
    textTransform: 'capitalize',
  },
  track: {
    flex: 1,
    height: 6,
    borderRadius: 3,
    overflow: 'hidden',
  },
  fill: {
    height: '100%',
    borderRadius: 3,
    minWidth: 4,
  },
  meta: {
    width: 60,
    fontSize: 10,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'right',
    flexShrink: 0,
  },
});

// ── Mood distribution pills ───────────────────────────────────────────────────

function DistributionBar({
  high,
  neutral,
  low,
  colors,
}: {
  high: number;
  neutral: number;
  low: number;
  colors: ReturnType<typeof useTheme>['colors'];
}) {
  const total = high + neutral + low;
  if (total === 0) return null;

  const highPct = (high / total) * 100;
  const neutralPct = (neutral / total) * 100;
  const lowPct = (low / total) * 100;

  return (
    <View style={distStyles.wrap}>
      <View style={distStyles.barRow}>
        {highPct > 0 && (
          <View
            style={[
              distStyles.segment,
              { width: `${highPct}%`, backgroundColor: colors.moodGreen },
            ]}
          />
        )}
        {neutralPct > 0 && (
          <View
            style={[
              distStyles.segment,
              { width: `${neutralPct}%`, backgroundColor: colors.moodYellow },
            ]}
          />
        )}
        {lowPct > 0 && (
          <View
            style={[
              distStyles.segment,
              { width: `${lowPct}%`, backgroundColor: colors.moodRed },
            ]}
          />
        )}
      </View>
      <View style={distStyles.legend}>
        <Text style={[distStyles.legendItem, { color: colors.moodGreen }]}>
          ● {high} high
        </Text>
        <Text style={[distStyles.legendItem, { color: colors.moodYellow }]}>
          ● {neutral} neutral
        </Text>
        <Text style={[distStyles.legendItem, { color: colors.moodRed }]}>
          ● {low} low
        </Text>
      </View>
    </View>
  );
}

const distStyles = StyleSheet.create({
  wrap: { marginTop: 12 },
  barRow: {
    flexDirection: 'row',
    height: 6,
    borderRadius: 3,
    overflow: 'hidden',
    marginBottom: 6,
  },
  segment: { height: '100%' },
  legend: { flexDirection: 'row', gap: 12 },
  legendItem: { fontSize: 10, fontFamily: 'Nunito_400Regular' },
});

// ── Main component ────────────────────────────────────────────────────────────

interface PatternRadarProps {
  initialData?: PatternRadarResponse | null;
}

export default function PatternRadar({ initialData }: PatternRadarProps) {
  const { colors } = useTheme();
  const styles = getStyles(colors);
  const [range, setRange] = useState<Range>('30d');
  const [data, setData] = useState<PatternRadarResponse | null>(initialData ?? null);
  const [loading, setLoading] = useState(!initialData);
  const [error, setError] = useState(false);

  const load = useCallback((r: Range) => {
    setLoading(true);
    setError(false);
    api
      .getPatternRadar(r)
      .then(setData)
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, []);

  const handleRangeChange = (r: Range) => {
    setRange(r);
    load(r);
  };

  useEffect(() => {
    if (!initialData) {
      load('30d');
    }
  }, [load, initialData]);

  const emotions = data?.emotions ?? [];
  const maxScore = emotions.reduce((m, e) => Math.max(m, e.score), 0);
  const dist = data?.mood_distribution;

  return (
    <View style={styles.card}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.title}>EMOTION PATTERNS</Text>
        <View style={styles.rangeSelector}>
          {(['30d', '90d', '365d'] as Range[]).map((r) => (
            <TouchableOpacity
              key={r}
              style={[styles.rangeBtn, range === r && styles.rangeBtnActive]}
              onPress={() => handleRangeChange(r)}
              disabled={loading}
            >
              <Text style={[styles.rangeBtnText, range === r && styles.rangeBtnTextActive]}>
                {RANGE_LABELS[r]}
              </Text>
            </TouchableOpacity>
          ))}
        </View>
      </View>

      {/* Entry count subtitle */}
      {data && (
        <Text style={styles.subtitle}>
          From {data.total_entries} {data.total_entries === 1 ? 'entry' : 'entries'}
        </Text>
      )}

      {/* Content */}
      {loading ? (
        <View style={styles.center}>
          <ActivityIndicator color={colors.purple400} />
        </View>
      ) : error ? (
        <Text style={styles.empty}>Could not load patterns. Tap a range to retry.</Text>
      ) : emotions.length === 0 ? (
        <Text style={styles.empty}>
          Keep journaling to reveal your emotional patterns.
        </Text>
      ) : (
        <>
          {/* Emotion bars */}
          <View style={styles.barSection}>
            {emotions.map((ep) => (
              <EmotionBar key={ep.emotion} pattern={ep} maxScore={maxScore} colors={colors} />
            ))}
          </View>

          {/* Mood distribution */}
          {dist && (
            <>
              <Text style={styles.distLabel}>MOOD DISTRIBUTION</Text>
              <DistributionBar
                high={dist.high}
                neutral={dist.neutral}
                low={dist.low}
                colors={colors}
              />
            </>
          )}
        </>
      )}
    </View>
  );
}

const getStyles = (colors: ReturnType<typeof useTheme>['colors']) =>
  StyleSheet.create({
    card: {
      backgroundColor: colors.card,
      borderRadius: 20,
      borderWidth: 1,
      borderColor: colors.borderFaint,
      padding: 20,
      marginBottom: 24,
    },
    header: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      alignItems: 'center',
      marginBottom: 6,
    },
    title: {
      fontSize: 10,
      color: colors.textSecondary,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 1.5,
    },
    subtitle: {
      fontSize: 11,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      marginBottom: 16,
    },
    rangeSelector: { flexDirection: 'row', gap: 4 },
    rangeBtn: {
      paddingHorizontal: 10,
      paddingVertical: 4,
      borderRadius: 8,
    },
    rangeBtnActive: { backgroundColor: colors.brandGlow },
    rangeBtnText: {
      fontSize: 10,
      color: colors.textMuted,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 0.5,
    },
    rangeBtnTextActive: { color: colors.purple300 },
    center: { paddingVertical: 32, alignItems: 'center' },
    empty: {
      fontSize: 13,
      color: colors.textSecondary,
      fontFamily: 'Nunito_400Regular',
      textAlign: 'center',
      paddingVertical: 24,
    },
    barSection: { marginBottom: 4 },
    distLabel: {
      fontSize: 9,
      color: colors.textMuted,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 1.5,
      marginBottom: 6,
    },
  });
