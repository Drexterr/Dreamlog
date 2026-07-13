import { useCallback, useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  Modal,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useFocusEffect } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { T } from '../../src/testIDs';
import type { ExternalClient, ClientSummary } from '../../src/types';

// Client list: the therapist's own (external) clients plus Ode users who
// consented to share their journal (linked clients).
export default function TherapistClientsScreen() {
  const [external, setExternal] = useState<ExternalClient[]>([]);
  const [linked, setLinked] = useState<ClientSummary[]>([]);
  const [showAdd, setShowAdd] = useState(false);
  const [newName, setNewName] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const router = useRouter();
  const { colors } = useTheme();

  const load = useCallback(async () => {
    try {
      const [ext, linkedResp] = await Promise.all([
        api.listExternalClients(),
        api.listClients().catch(() => ({ clients: [] as ClientSummary[] })),
      ]);
      setExternal(ext.clients);
      setLinked(linkedResp.clients);
    } catch {
      // surfaced by empty state
    }
  }, []);

  useFocusEffect(
    useCallback(() => {
      load();
    }, [load]),
  );

  const handleAdd = async () => {
    const trimmed = newName.trim();
    if (!trimmed) {
      setError('A name or initials are required.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await api.createExternalClient(trimmed);
      setNewName('');
      setShowAdd(false);
      await load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not add client.');
    } finally {
      setSaving(false);
    }
  };

  const handleArchive = (client: ExternalClient) => {
    Alert.alert('Archive client?', `${client.name} will be hidden from your list. Their session notes are kept.`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Archive',
        style: 'destructive',
        onPress: async () => {
          await api.updateExternalClient(client.id, { archived: true }).catch(() => {});
          load();
        },
      },
    ]);
  };

  return (
    <SafeAreaView testID={T.therapistPortal.clientsScreen} style={[styles.container, { backgroundColor: colors.bg }]}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.headerRow}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Clients</Text>
          <TouchableOpacity
            testID={T.therapistPortal.clientsAddButton}
            style={[styles.addBtn, { backgroundColor: colors.purple600 }]}
            onPress={() => setShowAdd(true)}
          >
            <Text style={styles.addBtnText}>＋ Add</Text>
          </TouchableOpacity>
        </View>

        <Text style={[styles.privacyHint, { color: colors.textMuted }]}>
          Tip: use a first name or initials — whatever your client is comfortable with. Names are
          stored encrypted.
        </Text>

        {/* External clients */}
        <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>My practice</Text>
        {external.length === 0 ? (
          <View style={[styles.emptyCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <Text style={[styles.emptyText, { color: colors.textMuted }]}>
              Add your first client to start keeping digital session notes.
            </Text>
          </View>
        ) : (
          external.map(c => (
            <TouchableOpacity
              key={c.id}
              style={[styles.clientCard, { backgroundColor: colors.card, borderColor: colors.border }]}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              onPress={() => router.push(`/therapist/client/${c.id}` as any)}
              onLongPress={() => handleArchive(c)}
            >
              <View style={[styles.avatar, { backgroundColor: colors.purple600 + '33' }]}>
                <Text style={[styles.avatarText, { color: colors.textPrimary }]}>
                  {c.name.slice(0, 2).toUpperCase()}
                </Text>
              </View>
              <View style={{ flex: 1 }}>
                <Text style={[styles.clientName, { color: colors.textPrimary }]}>{c.name}</Text>
                <Text style={[styles.clientMeta, { color: colors.textMuted }]}>
                  {c.session_count} session{c.session_count === 1 ? '' : 's'}
                  {c.last_session_at ? ` · last ${new Date(c.last_session_at).toLocaleDateString()}` : ''}
                </Text>
              </View>
              <Text style={[styles.chevron, { color: colors.textMuted }]}>›</Text>
            </TouchableOpacity>
          ))
        )}

        {/* Linked Ode users */}
        <Text style={[styles.sectionTitle, { color: colors.textPrimary, marginTop: 24 }]}>Ode clients</Text>
        <Text style={[styles.privacyHint, { color: colors.textMuted }]}>
          App users who consented to share their journal summaries with you.
        </Text>
        {linked.length === 0 ? (
          <View style={[styles.emptyCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <Text style={[styles.emptyText, { color: colors.textMuted }]}>
              No linked app users yet. Ask a client using Ode to share their user ID with you,
              then link them from the web portal — they approve the request in their app.
            </Text>
          </View>
        ) : (
          linked.map(c => (
            <View
              key={c.client_id}
              style={[styles.clientCard, { backgroundColor: colors.card, borderColor: colors.border }]}
            >
              <View style={[styles.avatar, { backgroundColor: '#34d39933' }]}>
                <Text style={[styles.avatarText, { color: colors.textPrimary }]}>
                  {c.name.slice(0, 2).toUpperCase()}
                </Text>
              </View>
              <View style={{ flex: 1 }}>
                <Text style={[styles.clientName, { color: colors.textPrimary }]}>{c.name}</Text>
                <Text style={[styles.clientMeta, { color: colors.textMuted }]}>
                  {c.entry_count} journal entries
                  {c.avg_mood_30d != null ? ` · mood ${c.avg_mood_30d}/100 (30d)` : ''}
                </Text>
              </View>
              <View style={[styles.linkedBadge, { borderColor: colors.border }]}>
                <Text style={[styles.linkedBadgeText, { color: colors.textMuted }]}>app user</Text>
              </View>
            </View>
          ))
        )}
      </ScrollView>

      {/* Add client modal */}
      <Modal visible={showAdd} transparent animationType="fade" onRequestClose={() => setShowAdd(false)}>
        <View style={styles.modalOverlay}>
          <View style={[styles.modalCard, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}>
            <Text style={[styles.modalTitle, { color: colors.textPrimary }]}>Add a client</Text>
            <TextInput
              testID={T.therapistPortal.clientsAddNameInput}
              style={[styles.input, { backgroundColor: colors.card, borderColor: colors.borderFaint, color: colors.textPrimary }]}
              value={newName}
              onChangeText={v => { setNewName(v); setError(''); }}
              placeholder="First name or initials (e.g. Asha K)"
              placeholderTextColor={colors.textFaint}
              autoFocus
            />
            {!!error && <Text style={styles.errorText}>{error}</Text>}
            <View style={styles.modalActions}>
              <TouchableOpacity style={styles.modalCancel} onPress={() => { setShowAdd(false); setError(''); }}>
                <Text style={[styles.modalCancelText, { color: colors.textMuted }]}>Cancel</Text>
              </TouchableOpacity>
              <TouchableOpacity
                testID={T.therapistPortal.clientsAddSubmit}
                style={[styles.modalSave, { backgroundColor: colors.purple600 }]}
                onPress={handleAdd}
                disabled={saving}
              >
                {saving ? <ActivityIndicator color="#fff" size="small" /> : <Text style={styles.modalSaveText}>Add client</Text>}
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </Modal>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 20, paddingBottom: 48 },
  headerRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 },
  title: { fontSize: 30, fontFamily: 'CormorantGaramond_500Medium' },
  addBtn: { borderRadius: 12, paddingHorizontal: 16, paddingVertical: 10 },
  addBtnText: { color: '#fff', fontSize: 14, fontFamily: 'Nunito_600SemiBold' },
  privacyHint: { fontSize: 12, fontFamily: 'Nunito_400Regular', lineHeight: 17, marginBottom: 16 },
  sectionTitle: { fontSize: 19, fontFamily: 'CormorantGaramond_500Medium', marginBottom: 10 },
  emptyCard: { borderRadius: 16, borderWidth: 1, padding: 18 },
  emptyText: { fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 20 },
  clientCard: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    borderRadius: 16,
    borderWidth: 1,
    padding: 14,
    marginBottom: 10,
  },
  avatar: { width: 42, height: 42, borderRadius: 21, alignItems: 'center', justifyContent: 'center' },
  avatarText: { fontSize: 14, fontFamily: 'Nunito_700Bold' },
  clientName: { fontSize: 15, fontFamily: 'Nunito_600SemiBold' },
  clientMeta: { fontSize: 12, fontFamily: 'Nunito_400Regular', marginTop: 2 },
  chevron: { fontSize: 24, lineHeight: 24 },
  linkedBadge: { borderWidth: 1, borderRadius: 8, paddingHorizontal: 8, paddingVertical: 4 },
  linkedBadgeText: { fontSize: 10, fontFamily: 'Nunito_600SemiBold' },

  modalOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.7)',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 28,
  },
  modalCard: { width: '100%', borderRadius: 20, borderWidth: 1, padding: 22 },
  modalTitle: { fontSize: 20, fontFamily: 'CormorantGaramond_500Medium', marginBottom: 14 },
  input: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 14,
    fontFamily: 'Nunito_400Regular',
    fontSize: 15,
    marginBottom: 10,
  },
  errorText: { fontSize: 13, color: '#ef4444', fontFamily: 'Nunito_400Regular', marginBottom: 8 },
  modalActions: { flexDirection: 'row', justifyContent: 'flex-end', gap: 12, marginTop: 6 },
  modalCancel: { paddingVertical: 12, paddingHorizontal: 14 },
  modalCancelText: { fontSize: 14, fontFamily: 'Nunito_600SemiBold' },
  modalSave: { borderRadius: 12, paddingVertical: 12, paddingHorizontal: 20 },
  modalSaveText: { color: '#fff', fontSize: 14, fontFamily: 'Nunito_600SemiBold' },
});
