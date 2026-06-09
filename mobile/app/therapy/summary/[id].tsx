import { useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  
  ActivityIndicator,
  Share,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { api } from '../../../src/api/client';
import { useTheme } from '../../../src/context/ThemeContext';
import type { TherapySession } from '../../../src/types';

function formatDuration(sec?: number): string {
  if (!sec) return '-';
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${m} min ${s} sec`;
}

export default function TherapySummaryScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();

  const [session, setSession] = useState<TherapySession | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    api.getTherapySession(id)
      .then(setSession)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [id]);

  const handleShare = async () => {
    if (!session?.post_session_summary) return;
    await Share.share({
      message: `My reflection session summary:\n\n${session.post_session_summary}\n\n- via DreamLog`,
    });
  };

  if (loading) {
    return (
      <SafeAreaView style={[styles.safe, { backgroundColor: colors.bg }]}>
        <ActivityIndicator color={colors.brand} style={{ flex: 1 }} />
      </SafeAreaView>
    );
  }

  if (!session) {
    return (
      <SafeAreaView style={[styles.safe, { backgroundColor: colors.bg }]}>
        <Text style={[styles.errorText, { color: colors.textSecondary }]}>Session not found.</Text>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.bg }]}>
      <ScrollView contentContainerStyle={styles.container}>
        {/* Header */}
        <Text style={[styles.title, { color: colors.textPrimary }]}>Session Complete</Text>
        <Text style={[styles.date, { color: colors.textMuted }]}>
          {new Date(session.started_at).toLocaleDateString('en-IN', {
            weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
          })}
        </Text>

        {/* Stats row */}
        <View style={[styles.statsRow, { borderColor: colors.border }]}>
          <View style={styles.stat}>
            <Text style={[styles.statValue, { color: colors.textPrimary }]}>{session.turn_count}</Text>
            <Text style={[styles.statLabel, { color: colors.textMuted }]}>Exchanges</Text>
          </View>
          <View style={[styles.statDivider, { backgroundColor: colors.border }]} />
          <View style={styles.stat}>
            <Text style={[styles.statValue, { color: colors.textPrimary }]}>{formatDuration(session.duration_sec)}</Text>
            <Text style={[styles.statLabel, { color: colors.textMuted }]}>Duration</Text>
          </View>
        </View>

        {/* Summary */}
        {session.post_session_summary ? (
          <View style={[styles.summaryCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <Text style={[styles.summaryLabel, { color: colors.textMuted }]}>SESSION SUMMARY</Text>
            <Text style={[styles.summaryText, { color: colors.textPrimary }]}>
              {session.post_session_summary}
            </Text>
            <TouchableOpacity onPress={handleShare} style={styles.shareBtn}>
              <Text style={[styles.shareBtnText, { color: colors.brand }]}>Share this reflection</Text>
            </TouchableOpacity>
          </View>
        ) : (
          <View style={[styles.summaryCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <Text style={[styles.summaryText, { color: colors.textSecondary }]}>
              No summary was generated for this session.
            </Text>
          </View>
        )}

        {/* Disclaimer */}
        <Text style={[styles.disclaimer, { color: colors.textFaint }]}>
          This session was an AI-assisted reflection, not clinical therapy.
          If you're struggling, please reach out to a mental health professional.
        </Text>

        {/* Actions */}
        <TouchableOpacity
          style={[styles.primaryBtn, { backgroundColor: colors.brand }]}
          onPress={() => router.replace('/therapy' as any)}
          activeOpacity={0.8}
        >
          <Text style={styles.primaryBtnText}>Start a New Session</Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.secondaryBtn}
          onPress={() => router.replace('/(tabs)' as any)}
          activeOpacity={0.7}
        >
          <Text style={[styles.secondaryBtnText, { color: colors.textSecondary }]}>Back to Journal</Text>
        </TouchableOpacity>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1 },
  container: { padding: 24, paddingBottom: 48 },
  title: { fontSize: 28, fontFamily: 'CormorantGaramond_600SemiBold', marginBottom: 6 },
  date: { fontSize: 14, fontFamily: 'Nunito_400Regular', marginBottom: 28 },
  statsRow: {
    flexDirection: 'row',
    borderWidth: 1,
    borderRadius: 12,
    padding: 20,
    marginBottom: 24,
    justifyContent: 'space-around',
    alignItems: 'center',
  },
  stat: { alignItems: 'center', gap: 4 },
  statValue: { fontSize: 22, fontFamily: 'Nunito_700Bold' },
  statLabel: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  statDivider: { width: 1, height: 36 },
  summaryCard: {
    borderWidth: 1,
    borderRadius: 12,
    padding: 20,
    marginBottom: 20,
    gap: 12,
  },
  summaryLabel: { fontSize: 11, fontFamily: 'Nunito_700Bold', letterSpacing: 1 },
  summaryText: { fontSize: 16, fontFamily: 'Nunito_400Regular', lineHeight: 26 },
  shareBtn: { alignSelf: 'flex-start' },
  shareBtnText: { fontSize: 14, fontFamily: 'Nunito_600SemiBold' },
  disclaimer: { fontSize: 12, fontFamily: 'Nunito_400Regular', lineHeight: 18, textAlign: 'center', marginBottom: 32 },
  primaryBtn: { borderRadius: 12, paddingVertical: 16, alignItems: 'center', marginBottom: 12 },
  primaryBtnText: { color: '#fff', fontSize: 17, fontFamily: 'Nunito_700Bold' },
  secondaryBtn: { alignItems: 'center', paddingVertical: 12 },
  secondaryBtnText: { fontSize: 15, fontFamily: 'Nunito_400Regular' },
  errorText: { textAlign: 'center', marginTop: 100, fontSize: 16, fontFamily: 'Nunito_400Regular' },
});
