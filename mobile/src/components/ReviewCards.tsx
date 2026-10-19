import { Fonts } from '../theme';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from '../context/ThemeContext';
import type { AnnualReview, WeeklyReview } from '../types';
import type { ThemeColors } from '../theme';
import WeeklyNarrative from './WeeklyNarrative';

export function WeeklyReviewCard({ review }: { review: WeeklyReview }) {
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
      <WeeklyNarrative text={review.narrative} style={styles.reviewNarrative} linkStyle={styles.shareLink}/>
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

export function YearInReviewCard({ review }: { review: AnnualReview }) {
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


const getStyles = (colors: ThemeColors) => StyleSheet.create({
  shareLink: {
    fontSize: 11,
    color: colors.purple300,
    fontFamily: Fonts.sans,
    textDecorationLine: 'underline',
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
    fontFamily: Fonts.sansSB,
    letterSpacing: 1.2,
    marginBottom: 10,
    textTransform: 'uppercase',
  },
  reviewNarrative: {
    fontSize: 15,
    color: colors.textPrimary,
    fontFamily: Fonts.serifRegular,
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
    fontFamily: Fonts.sans,
  },
  yearReviewMeta: { flexDirection: 'row', gap: 12, marginBottom: 8 },
  yearReviewMetaText: {
    fontSize: 11,
    color: colors.textMuted,
    fontFamily: Fonts.sans,
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
  yearArcBar: { width: 3, borderRadius: 1.5, minHeight: 4 },
  yearArcLabel: { fontSize: 8, fontFamily: Fonts.sans },
});
