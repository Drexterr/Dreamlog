import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { api } from '../../../src/api/client';
import { useTheme } from '../../../src/context/ThemeContext';
import type { ClientSession } from '../../../src/types';

const POLL_INTERVAL_MS = 3000;
const MAX_POLLS = 60; // give OCR up to ~3 minutes before surfacing a hint

// Session detail: live OCR status → editable bullet list → AI summary.
export default function SessionDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const [session, setSession] = useState<ClientSession | null>(null);
  const [bullets, setBullets] = useState<string[]>([]);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [summarizing, setSummarizing] = useState(false);
  const [showRaw, setShowRaw] = useState(false);
  const [error, setError] = useState('');
  const pollCount = useRef(0);
  const router = useRouter();
  const { colors } = useTheme();

  const applySession = useCallback((s: ClientSession) => {
    setSession(s);
    setBullets(s.bullets.length > 0 ? s.bullets : ['']);
  }, []);

  // Load + poll while OCR is running.
  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;

    const tick = async () => {
      try {
        const s = await api.getClientSession(id);
        if (cancelled) return;
        applySession(s);
        if ((s.status === 'pending' || s.status === 'processing') && pollCount.current < MAX_POLLS) {
          pollCount.current += 1;
          timer = setTimeout(tick, POLL_INTERVAL_MS);
        }
      } catch {
        if (!cancelled) setError('Could not load this session.');
      }
    };
    tick();
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [id, applySession]);

  const updateBullet = (index: number, value: string) => {
    setBullets(prev => prev.map((b, i) => (i === index ? value : b)));
    setDirty(true);
  };

  const addBullet = () => {
    setBullets(prev => [...prev, '']);
    setDirty(true);
  };

  const removeBullet = (index: number) => {
    setBullets(prev => prev.filter((_, i) => i !== index));
    setDirty(true);
  };

  const handleSave = async () => {
    if (!session) return;
    const cleaned = bullets.map(b => b.trim()).filter(Boolean);
    if (cleaned.length === 0) {
      setError('Notes cannot be empty.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      const updated = await api.updateClientSessionBullets(session.id, cleaned);
      applySession(updated);
      setDirty(false);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Save failed.');
    } finally {
      setSaving(false);
    }
  };

  const handleSummarize = async () => {
    if (!session) return;
    if (dirty) {
      setError('Save your edits first, then summarize.');
      return;
    }
    setSummarizing(true);
    setError('');
    try {
      const updated = await api.summarizeClientSession(session.id);
      applySession(updated);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Summarization failed.');
    } finally {
      setSummarizing(false);
    }
  };

  const handleDelete = () => {
    if (!session) return;
    Alert.alert('Delete session?', 'These notes will be permanently deleted.', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: async () => {
          await api.deleteClientSession(session.id).catch(() => {});
          router.back();
        },
      },
    ]);
  };

  if (!session) {
    return (
      <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]}>
        <View style={styles.centerFill}>
          {error ? (
            <Text style={styles.errorText}>{error}</Text>
          ) : (
            <ActivityIndicator color={colors.textMuted} />
          )}
        </View>
      </SafeAreaView>
    );
  }

  const isExtracting = session.status === 'pending' || session.status === 'processing';

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]}>
      <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <View style={styles.headerRow}>
            <View style={{ flex: 1 }}>
              <Text style={[styles.title, { color: colors.textPrimary }]}>Session notes</Text>
              <Text style={[styles.meta, { color: colors.textMuted }]}>{session.session_date}</Text>
            </View>
            <TouchableOpacity onPress={handleDelete}>
              <Text style={styles.deleteText}>Delete</Text>
            </TouchableOpacity>
          </View>

          {isExtracting && (
            <View style={[styles.extractingCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
              <ActivityIndicator color={colors.textMuted} size="small" />
              <Text style={[styles.extractingText, { color: colors.textSecondary }]}>
                Reading your handwritten notes… The photo is deleted as soon as extraction finishes.
              </Text>
            </View>
          )}

          {session.status === 'failed' && (
            <View style={[styles.failedCard, { borderColor: '#ef444455' }]}>
              <Text style={[styles.failedTitle, { color: '#ef4444' }]}>Extraction failed</Text>
              <Text style={[styles.failedText, { color: colors.textSecondary }]}>
                {session.error_msg || 'The photo could not be read.'} You can delete this session and
                retake the photo, or type the notes instead.
              </Text>
            </View>
          )}

          {session.status === 'completed' && (
            <>
              {/* AI summary */}
              {session.summary ? (
                <View style={[styles.summaryCard, { backgroundColor: colors.card, borderColor: colors.purple600 + '66' }]}>
                  <Text style={[styles.summaryLabel, { color: colors.textMuted }]}>✦ AI SESSION SUMMARY</Text>
                  <Text style={[styles.summaryText, { color: colors.textSecondary }]}>{session.summary}</Text>
                </View>
              ) : (
                <TouchableOpacity
                  style={[styles.summarizeBtn, { borderColor: colors.purple600 }]}
                  onPress={handleSummarize}
                  disabled={summarizing}
                >
                  {summarizing ? (
                    <ActivityIndicator color={colors.textMuted} size="small" />
                  ) : (
                    <Text style={[styles.summarizeBtnText, { color: colors.textPrimary }]}>✦ Summarize with AI</Text>
                  )}
                </TouchableOpacity>
              )}

              {/* Bullet editor */}
              <Text style={[styles.label, { color: colors.textMuted }]}>NOTES</Text>
              {bullets.map((b, i) => (
                <View key={i} style={styles.bulletRow}>
                  <Text style={[styles.bulletDot, { color: colors.textMuted }]}>•</Text>
                  <TextInput
                    style={[styles.bulletInput, { backgroundColor: colors.card, borderColor: colors.border, color: colors.textPrimary }]}
                    value={b}
                    onChangeText={v => updateBullet(i, v)}
                    multiline
                    placeholder="Note…"
                    placeholderTextColor={colors.textFaint}
                  />
                  <TouchableOpacity onPress={() => removeBullet(i)} style={styles.bulletRemove}>
                    <Text style={{ color: colors.textMuted, fontSize: 16 }}>✕</Text>
                  </TouchableOpacity>
                </View>
              ))}
              <TouchableOpacity onPress={addBullet} style={styles.addBulletBtn}>
                <Text style={[styles.addBulletText, { color: colors.textMuted }]}>＋ Add a note</Text>
              </TouchableOpacity>

              {/* Raw OCR text reference */}
              {!!session.raw_text && (
                <TouchableOpacity onPress={() => setShowRaw(s => !s)} style={styles.rawToggle}>
                  <Text style={[styles.rawToggleText, { color: colors.textMuted }]}>
                    {showRaw ? '▾ Hide original extracted text' : '▸ Show original extracted text'}
                  </Text>
                </TouchableOpacity>
              )}
              {showRaw && !!session.raw_text && (
                <View style={[styles.rawCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
                  <Text style={[styles.rawText, { color: colors.textMuted }]}>{session.raw_text}</Text>
                </View>
              )}

              {!!error && <Text style={styles.errorText}>{error}</Text>}

              {dirty && (
                <TouchableOpacity
                  style={[styles.saveBtn, { backgroundColor: colors.purple600, shadowColor: colors.purple500 }, saving && { opacity: 0.6 }]}
                  onPress={handleSave}
                  disabled={saving}
                >
                  {saving ? <ActivityIndicator color="#fff" size="small" /> : <Text style={styles.saveBtnText}>Save changes</Text>}
                </TouchableOpacity>
              )}
            </>
          )}
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centerFill: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  scroll: { padding: 20, paddingBottom: 60 },
  headerRow: { flexDirection: 'row', alignItems: 'center', marginBottom: 18 },
  title: { fontSize: 26, fontFamily: 'CormorantGaramond_500Medium' },
  meta: { fontSize: 12.5, fontFamily: 'Nunito_400Regular', marginTop: 2 },
  deleteText: { color: '#ef4444', fontSize: 13, fontFamily: 'Nunito_600SemiBold' },

  extractingCard: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    borderRadius: 16,
    borderWidth: 1,
    padding: 18,
  },
  extractingText: { flex: 1, fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 19 },

  failedCard: { borderRadius: 16, borderWidth: 1, padding: 18 },
  failedTitle: { fontSize: 15, fontFamily: 'Nunito_700Bold', marginBottom: 6 },
  failedText: { fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 19 },

  summaryCard: { borderRadius: 16, borderWidth: 1, padding: 18, marginBottom: 20 },
  summaryLabel: { fontSize: 10.5, fontFamily: 'Nunito_700Bold', letterSpacing: 1.2, marginBottom: 8 },
  summaryText: { fontSize: 14, fontFamily: 'Nunito_400Regular', lineHeight: 21 },
  summarizeBtn: {
    borderRadius: 14,
    borderWidth: 1.5,
    paddingVertical: 14,
    alignItems: 'center',
    marginBottom: 20,
  },
  summarizeBtnText: { fontSize: 14.5, fontFamily: 'Nunito_600SemiBold' },

  label: { fontSize: 11, fontFamily: 'Nunito_700Bold', letterSpacing: 1.5, marginBottom: 10 },
  bulletRow: { flexDirection: 'row', alignItems: 'flex-start', gap: 8, marginBottom: 8 },
  bulletDot: { fontSize: 18, lineHeight: 24, marginTop: 10 },
  bulletInput: {
    flex: 1,
    borderRadius: 12,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 10,
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 20,
  },
  bulletRemove: { padding: 10, marginTop: 2 },
  addBulletBtn: { paddingVertical: 10 },
  addBulletText: { fontSize: 13.5, fontFamily: 'Nunito_600SemiBold' },

  rawToggle: { paddingVertical: 12 },
  rawToggleText: { fontSize: 13, fontFamily: 'Nunito_600SemiBold' },
  rawCard: { borderRadius: 12, borderWidth: 1, padding: 14 },
  rawText: { fontSize: 12.5, fontFamily: 'Nunito_400Regular', lineHeight: 19 },

  errorText: { fontSize: 13, color: '#ef4444', fontFamily: 'Nunito_400Regular', marginTop: 12 },
  saveBtn: {
    borderRadius: 14,
    paddingVertical: 15,
    alignItems: 'center',
    marginTop: 16,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 6,
  },
  saveBtnText: { color: '#fff', fontSize: 15, fontFamily: 'Nunito_600SemiBold' },
});
