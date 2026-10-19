import { useCallback, useState } from 'react';
import { ActivityIndicator, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useFocusEffect } from 'expo-router';
import { api } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useStreak } from '../context/StreakContext';
import { useTheme } from '../context/ThemeContext';
import { Fonts } from '../theme';
import { T } from '../testIDs';
import type { AnnualReview, WeeklyReview } from '../types';
import { WeeklyReviewCard, YearInReviewCard } from './ReviewCards';
import ShareInsightModal from './ShareInsightModal';

function isMissing(reason: unknown): boolean {
  return typeof reason === 'object' && reason !== null && 'response' in reason
    && typeof reason.response === 'object' && reason.response !== null
    && 'status' in reason.response && reason.response.status === 404;
}

export default function ReflectionsSection() {
  const { colors } = useTheme();
  const { isAuthenticated, requestAuth } = useAuth();
  const { streak } = useStreak();
  const [weekly, setWeekly] = useState<WeeklyReview | null>(null);
  const [annual, setAnnual] = useState<AnnualReview | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  const [sharing, setSharing] = useState(false);

  useFocusEffect(useCallback(() => {
    let active = true;
    setSharing(false); setWeekly(null); setAnnual(null); setFailed(false);
    if (!isAuthenticated) { setLoading(false); return; }
    setLoading(true);
    Promise.allSettled([api.getLatestWeeklyReview(), api.getLatestAnnualReview()]).then(([week, year]) => {
      if (!active) return;
      setWeekly(week.status === 'fulfilled' && week.value?.status === 'completed' ? week.value : null);
      setAnnual(year.status === 'fulfilled' && year.value?.status === 'completed' ? year.value : null);
      setFailed([week, year].some(result => result.status === 'rejected' && !isMissing(result.reason)));
      setLoading(false);
    });
    return () => { active = false; };
  }, [isAuthenticated, retry]));

  const start = weekly ? new Date(weekly.week_start + 'T00:00:00Z') : null;
  const end = start ? new Date(start) : null;
  if (end) end.setUTCDate(end.getUTCDate() + 6);
  const weekLabel = start && end
    ? start.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' }) + ' – ' + end.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' })
    : '';
  return (
    <View testID={T.explore.reflections} style={styles.section}>
      <Text accessibilityRole="header" style={[styles.title, { color: colors.textPrimary }]}>Reflections</Text>
      <Text style={[styles.description, { color: colors.textSecondary }]}>Your weekly and yearly reviews, together in one place.</Text>
      {!isAuthenticated ? (
        <TouchableOpacity accessibilityRole="button" onPress={() => requestAuth(() => setRetry(value => value + 1))} style={styles.action}>
          <Text style={[styles.link, { color: colors.purple300 }]}>Sign in to see your reflections</Text>
        </TouchableOpacity>
      ) : loading ? <ActivityIndicator testID="explore-reviews-loading" color={colors.purple400} style={{ marginVertical: 20 }} /> : (
        <>
          {weekly && <View testID={T.explore.weeklyReview}>
            <View style={styles.headingRow}>
              <Text style={[styles.label, { color: colors.textSecondary }]}>WEEKLY REVIEW</Text>
              <TouchableOpacity testID={T.explore.shareReview} accessibilityRole="button" onPress={() => setSharing(true)} style={styles.action}>
                <Text style={[styles.link, { color: colors.purple300 }]}>Share card</Text>
              </TouchableOpacity>
            </View>
            <WeeklyReviewCard review={weekly} />
          </View>}
          {annual && <View testID={T.explore.annualReview}>
            <Text style={[styles.label, { color: colors.textSecondary, marginBottom: 12 }]}>YEAR IN REVIEW</Text>
            <YearInReviewCard review={annual} />
          </View>}
          {failed ? <View>
            <Text accessibilityRole="alert" style={[styles.description, { color: colors.textSecondary }]}>Some reflections could not be loaded.</Text>
            <TouchableOpacity testID={T.explore.retryReviews} accessibilityRole="button" onPress={() => setRetry(value => value + 1)} style={styles.action}>
              <Text style={[styles.link, { color: colors.purple300 }]}>Retry</Text>
            </TouchableOpacity>
          </View> : !weekly && !annual && <Text style={[styles.description, { color: colors.textSecondary }]}>Your reflections will appear here as you keep journaling.</Text>}
        </>
      )}
      {isAuthenticated && weekly && <ShareInsightModal visible={sharing} onClose={() => setSharing(false)} weekLabel={weekLabel} weekStart={weekly.week_start} moodArc={weekly.mood_arc} topEmotions={weekly.top_emotions} streak={streak?.current_streak ?? 0} entryCount={weekly.entry_count} />}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { marginBottom: 28 },
  title: { fontFamily: Fonts.serif, fontSize: 22, marginBottom: 8 },
  description: { fontFamily: Fonts.sans, fontSize: 13, lineHeight: 20, marginBottom: 16 },
  headingRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 },
  label: { fontFamily: Fonts.sansSB, fontSize: 10, letterSpacing: 1.5 },
  action: { minHeight: 44, justifyContent: 'center', alignSelf: 'flex-start', paddingHorizontal: 4 },
  link: { fontFamily: Fonts.sans, fontSize: 13, textDecorationLine: 'underline' },
});
