import { useCallback, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  RefreshControl,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useFocusEffect } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { TherapistOverview, ClientSession, ExternalClient, TherapistMeResponse } from '../../src/types';

const STATUS_LABEL: Record<string, string> = {
  pending: 'Extracting…',
  processing: 'Extracting…',
  completed: 'Ready',
  failed: 'Failed',
};

// Therapist dashboard: caseload metrics, recent session notes, quick actions.
export default function TherapistDashboard() {
  const [me, setMe] = useState<TherapistMeResponse | null>(null);
  const [overview, setOverview] = useState<TherapistOverview | null>(null);
  const [recent, setRecent] = useState<ClientSession[]>([]);
  const [clients, setClients] = useState<ExternalClient[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const router = useRouter();
  const { colors } = useTheme();

  const load = useCallback(async () => {
    try {
      const [meResp, ov, sessions, clientList] = await Promise.all([
        api.therapistMe(),
        api.therapistOverview(),
        api.listClientSessions(),
        api.listExternalClients(),
      ]);
      setMe(meResp);
      setOverview(ov);
      setRecent(sessions.sessions.slice(0, 5));
      setClients(clientList.clients);
    } catch {
      // therapistMe 404 → not a therapist; send to registration
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      router.replace('/therapist/register' as any);
    }
  }, [router]);

  useFocusEffect(
    useCallback(() => {
      load();
    }, [load]),
  );

  const onRefresh = async () => {
    setRefreshing(true);
    await load();
    setRefreshing(false);
  };

  const clientName = (s: ClientSession): string => {
    if (s.external_client_id) {
      return clients.find(c => c.id === s.external_client_id)?.name ?? 'Client';
    }
    return 'Linked client';
  };

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]}>
      <ScrollView
        contentContainerStyle={styles.scroll}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={colors.textMuted} />}
      >
        <View style={styles.headerRow}>
          <View style={{ flex: 1 }}>
            <Text style={[styles.eyebrow, { color: colors.textMuted }]}>THERAPIST WORKSPACE</Text>
            <Text style={[styles.title, { color: colors.textPrimary }]}>
              {me?.therapist.name ? `Dr. ${me.therapist.name.split(' ')[0]}` : 'Welcome'}
            </Text>
          </View>
          <TouchableOpacity
            style={[styles.journalBtn, { borderColor: colors.border, backgroundColor: colors.card }]}
            onPress={() => router.replace('/(tabs)')}
          >
            <Text style={[styles.journalBtnText, { color: colors.textPrimary }]}>My journal →</Text>
          </TouchableOpacity>
        </View>

        {/* Metrics */}
        {overview && (
          <View style={styles.statsRow}>
            <View style={[styles.statCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
              <Text style={[styles.statValue, { color: colors.textPrimary }]}>
                {overview.external_clients + overview.linked_clients}
              </Text>
              <Text style={[styles.statLabel, { color: colors.textMuted }]}>clients</Text>
            </View>
            <View style={[styles.statCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
              <Text style={[styles.statValue, { color: colors.textPrimary }]}>{overview.sessions_this_week}</Text>
              <Text style={[styles.statLabel, { color: colors.textMuted }]}>this week</Text>
            </View>
            <View style={[styles.statCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
              <Text style={[styles.statValue, { color: colors.textPrimary }]}>{overview.total_sessions}</Text>
              <Text style={[styles.statLabel, { color: colors.textMuted }]}>total sessions</Text>
            </View>
          </View>
        )}

        {/* Primary actions */}
        <TouchableOpacity
          style={[styles.primaryBtn, { backgroundColor: colors.purple600, shadowColor: colors.purple500 }]}
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          onPress={() => router.push('/therapist/add-session' as any)}
          activeOpacity={0.85}
        >
          <Text style={styles.primaryBtnText}>＋ New session notes</Text>
        </TouchableOpacity>

        <View style={styles.actionRow}>
          <TouchableOpacity
            style={[styles.actionCard, { backgroundColor: colors.card, borderColor: colors.border }]}
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            onPress={() => router.push('/therapist/clients' as any)}
          >
            <Text style={styles.actionEmoji}>👥</Text>
            <Text style={[styles.actionLabel, { color: colors.textPrimary }]}>My clients</Text>
            <Text style={[styles.actionSub, { color: colors.textMuted }]}>
              {clients.length} active
            </Text>
          </TouchableOpacity>
        </View>

        {/* Recent sessions */}
        <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Recent sessions</Text>
        {recent.length === 0 ? (
          <View style={[styles.emptyCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <Text style={[styles.emptyText, { color: colors.textMuted }]}>
              No sessions yet. Photograph your handwritten notes after a consultation and DreamLog
              will turn them into an editable digital record.
            </Text>
          </View>
        ) : (
          recent.map(s => (
            <TouchableOpacity
              key={s.id}
              style={[styles.sessionCard, { backgroundColor: colors.card, borderColor: colors.border }]}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              onPress={() => router.push(`/therapist/session/${s.id}` as any)}
            >
              <View style={styles.sessionHeader}>
                <Text style={[styles.sessionClient, { color: colors.textPrimary }]}>{clientName(s)}</Text>
                <Text
                  style={[
                    styles.sessionStatus,
                    { color: s.status === 'completed' ? '#34d399' : s.status === 'failed' ? '#ef4444' : colors.textMuted },
                  ]}
                >
                  {STATUS_LABEL[s.status]}
                </Text>
              </View>
              <Text style={[styles.sessionDate, { color: colors.textMuted }]}>{s.session_date}</Text>
              {s.bullets.length > 0 && (
                <Text style={[styles.sessionPreview, { color: colors.textSecondary }]} numberOfLines={2}>
                  {s.bullets[0]}
                </Text>
              )}
            </TouchableOpacity>
          ))
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 20, paddingBottom: 48 },
  headerRow: { flexDirection: 'row', alignItems: 'center', marginBottom: 20 },
  eyebrow: { fontSize: 11, fontFamily: 'Nunito_700Bold', letterSpacing: 1.5, marginBottom: 4 },
  title: { fontSize: 30, fontFamily: 'CormorantGaramond_500Medium' },
  journalBtn: { borderWidth: 1, borderRadius: 12, paddingHorizontal: 14, paddingVertical: 10 },
  journalBtnText: { fontSize: 13, fontFamily: 'Nunito_600SemiBold' },

  statsRow: { flexDirection: 'row', gap: 10, marginBottom: 20 },
  statCard: {
    flex: 1,
    borderRadius: 16,
    borderWidth: 1,
    paddingVertical: 14,
    alignItems: 'center',
  },
  statValue: { fontSize: 22, fontFamily: 'Nunito_700Bold' },
  statLabel: { fontSize: 11, fontFamily: 'Nunito_400Regular', marginTop: 2 },

  primaryBtn: {
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 12,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 6,
  },
  primaryBtnText: { color: '#fff', fontSize: 16, fontFamily: 'Nunito_600SemiBold', letterSpacing: 0.3 },

  actionRow: { flexDirection: 'row', gap: 10, marginBottom: 28 },
  actionCard: {
    flex: 1,
    borderRadius: 16,
    borderWidth: 1,
    padding: 16,
  },
  actionEmoji: { fontSize: 22, marginBottom: 8 },
  actionLabel: { fontSize: 15, fontFamily: 'Nunito_600SemiBold' },
  actionSub: { fontSize: 12, fontFamily: 'Nunito_400Regular', marginTop: 2 },

  sectionTitle: { fontSize: 20, fontFamily: 'CormorantGaramond_500Medium', marginBottom: 12 },
  emptyCard: { borderRadius: 16, borderWidth: 1, padding: 20 },
  emptyText: { fontSize: 13.5, fontFamily: 'Nunito_400Regular', lineHeight: 20 },

  sessionCard: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 16,
    marginBottom: 10,
  },
  sessionHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  sessionClient: { fontSize: 15, fontFamily: 'Nunito_700Bold' },
  sessionStatus: { fontSize: 12, fontFamily: 'Nunito_600SemiBold' },
  sessionDate: { fontSize: 12, fontFamily: 'Nunito_400Regular', marginTop: 2, marginBottom: 6 },
  sessionPreview: { fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 19 },
});
