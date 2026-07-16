import { useCallback, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useLocalSearchParams, useFocusEffect } from 'expo-router';
import { api } from '../../../src/api/client';
import { useTheme } from '../../../src/context/ThemeContext';
import { T } from '../../../src/testIDs';
import type { ExternalClient, ClientSession } from '../../../src/types';

const STATUS_LABEL: Record<string, string> = {
  pending: 'Extracting…',
  processing: 'Extracting…',
  completed: 'Ready',
  failed: 'Failed',
};

// External client detail: profile header + full session history.
export default function ClientDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const [client, setClient] = useState<ExternalClient | null>(null);
  const [sessions, setSessions] = useState<ClientSession[]>([]);
  const router = useRouter();
  const { colors } = useTheme();

  const load = useCallback(async () => {
    if (!id) return;
    try {
      const [c, s] = await Promise.all([
        api.getExternalClient(id),
        api.listClientSessions({ external_client_id: id }),
      ]);
      setClient(c);
      setSessions(s.sessions);
    } catch {
      // deleted or not ours - go back
      router.back();
    }
  }, [id, router]);

  useFocusEffect(
    useCallback(() => {
      load();
    }, [load]),
  );

  const handleDelete = () => {
    if (!client) return;
    Alert.alert(
      'Delete client?',
      `This permanently deletes ${client.name} and ALL their session notes. This cannot be undone.`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete everything',
          style: 'destructive',
          onPress: async () => {
            await api.deleteExternalClient(client.id).catch(() => {});
            router.back();
          },
        },
      ],
    );
  };

  if (!client) return <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]} />;

  return (
    <SafeAreaView testID={T.therapistPortal.clientDetailScreen} style={[styles.container, { backgroundColor: colors.bg }]}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.headerRow}>
          <View style={[styles.avatar, { backgroundColor: colors.purple600 + '33' }]}>
            <Text style={[styles.avatarText, { color: colors.textPrimary }]}>
              {client.name.slice(0, 2).toUpperCase()}
            </Text>
          </View>
          <View style={{ flex: 1 }}>
            <Text style={[styles.title, { color: colors.textPrimary }]}>{client.name}</Text>
            <Text style={[styles.meta, { color: colors.textMuted }]}>
              {client.session_count} session{client.session_count === 1 ? '' : 's'} · added{' '}
              {new Date(client.created_at).toLocaleDateString()}
            </Text>
          </View>
        </View>

        <TouchableOpacity
          testID={T.therapistPortal.clientDetailAddSession}
          style={[styles.primaryBtn, { backgroundColor: colors.purple600, shadowColor: colors.purple500 }]}
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          onPress={() => router.push({ pathname: '/therapist/add-session', params: { clientId: client.id } } as any)}
          activeOpacity={0.85}
        >
          <Text style={styles.primaryBtnText}>＋ Add session notes</Text>
        </TouchableOpacity>

        <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Session history</Text>
        {sessions.length === 0 ? (
          <View style={[styles.emptyCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <Text style={[styles.emptyText, { color: colors.textMuted }]}>No sessions recorded yet.</Text>
          </View>
        ) : (
          sessions.map(s => (
            <TouchableOpacity
              key={s.id}
              style={[styles.sessionCard, { backgroundColor: colors.card, borderColor: colors.border }]}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              onPress={() => router.push(`/therapist/session/${s.id}` as any)}
            >
              <View style={styles.sessionHeader}>
                <Text style={[styles.sessionDate, { color: colors.textPrimary }]}>{s.session_date}</Text>
                <Text
                  style={[
                    styles.sessionStatus,
                    { color: s.status === 'completed' ? '#34d399' : s.status === 'failed' ? '#ef4444' : colors.textMuted },
                  ]}
                >
                  {STATUS_LABEL[s.status]}
                </Text>
              </View>
              {s.bullets.length > 0 && (
                <Text style={[styles.sessionPreview, { color: colors.textSecondary }]} numberOfLines={2}>
                  • {s.bullets[0]}
                </Text>
              )}
              {!!s.summary && (
                <Text style={[styles.summaryBadge, { color: colors.textMuted }]}>✦ AI summary available</Text>
              )}
            </TouchableOpacity>
          ))
        )}

        <TouchableOpacity testID={T.therapistPortal.clientDetailDelete} style={styles.deleteBtn} onPress={handleDelete}>
          <Text style={styles.deleteBtnText}>Delete client and all notes</Text>
        </TouchableOpacity>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 20, paddingBottom: 48 },
  headerRow: { flexDirection: 'row', alignItems: 'center', gap: 14, marginBottom: 20 },
  avatar: { width: 56, height: 56, borderRadius: 28, alignItems: 'center', justifyContent: 'center' },
  avatarText: { fontSize: 18, fontFamily: 'HankenGrotesk_700Bold' },
  title: { fontSize: 26, fontFamily: 'Erode_500Medium' },
  meta: { fontSize: 12, fontFamily: 'HankenGrotesk_400Regular', marginTop: 2 },
  primaryBtn: {
    borderRadius: 14,
    paddingVertical: 15,
    alignItems: 'center',
    marginBottom: 26,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 6,
  },
  primaryBtnText: { color: '#fff', fontSize: 15, fontFamily: 'HankenGrotesk_600SemiBold' },
  sectionTitle: { fontSize: 20, fontFamily: 'Erode_500Medium', marginBottom: 12 },
  emptyCard: { borderRadius: 16, borderWidth: 1, padding: 18 },
  emptyText: { fontSize: 13, fontFamily: 'HankenGrotesk_400Regular' },
  sessionCard: { borderRadius: 16, borderWidth: 1, padding: 16, marginBottom: 10 },
  sessionHeader: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 6 },
  sessionDate: { fontSize: 14, fontFamily: 'HankenGrotesk_700Bold' },
  sessionStatus: { fontSize: 12, fontFamily: 'HankenGrotesk_600SemiBold' },
  sessionPreview: { fontSize: 13, fontFamily: 'HankenGrotesk_400Regular', lineHeight: 19 },
  summaryBadge: { fontSize: 11.5, fontFamily: 'HankenGrotesk_600SemiBold', marginTop: 8 },
  deleteBtn: { alignItems: 'center', paddingVertical: 22 },
  deleteBtnText: { color: '#ef4444', fontSize: 13, fontFamily: 'HankenGrotesk_600SemiBold' },
});
