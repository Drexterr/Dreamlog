import { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Alert, BackHandler, KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import NetInfo from '@react-native-community/netinfo';
import { api } from '../api/client';
import { useTheme } from '../context/ThemeContext';
import { Fonts } from '../theme';
import { T } from '../testIDs';
import type { EntryMode } from '../types';
import { uuid } from '../utils/uuid';

interface Props {
  mode: EntryMode | null;
  modes: { key: EntryMode; label: string }[];
  onSelectMode: (mode: EntryMode) => void;
  onVoice: () => void;
  onCancel: () => void;
  onSaved: (id: string) => void;
  enabled: boolean;
  prompt?: string | null;
}
const MAX_LENGTH = 20000;

export default function TextJournalComposer({ mode, modes, onSelectMode, onVoice, onCancel, onSaved, enabled, prompt }: Props) {
  const { colors } = useTheme();
  const [draft, setDraft] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const busy = useRef(false);
  const active = useRef(enabled);
  const generation = useRef(0);
  const submission = useRef<{ id: string; content: string; mode: EntryMode } | null>(null);
  useEffect(() => {
    generation.current++;
    active.current = enabled; busy.current = false; setSending(false);
    if (!enabled) { setDraft(''); setError(''); submission.current = null; }
    return () => { active.current = false; generation.current++; };
  }, [enabled]);

  const leave = (action: () => void) => {
    if (busy.current) return;
    if (!draft.trim()) { action(); return; }
    Alert.alert('Discard this draft?', 'Your unsent journal will be discarded.', [
      { text: 'Keep writing', style: 'cancel' }, { text: 'Discard', style: 'destructive', onPress: action },
    ]);
  };
  useEffect(() => {
    const listener = BackHandler.addEventListener('hardwareBackPress', () => {
      if (busy.current || draft.trim()) { leave(onCancel); return true; }
      return false;
    });
    return () => listener.remove();
  }, [draft, onCancel]);

  const send = async () => {
    const content = draft.trim();
    if (!enabled || !active.current || busy.current || !mode || !content || content.length > MAX_LENGTH) return;
    busy.current = true; setSending(true); setError('');
    if (submission.current?.content !== content || submission.current.mode !== mode) submission.current = { id: uuid(), content, mode };
    const request = submission.current;
    const requestGeneration = generation.current;
    try {
      const network = await NetInfo.fetch();
      if (!active.current || generation.current !== requestGeneration) return;
      if (!network.isConnected) { setError('You’re offline. Keep this screen open and retry when you reconnect.'); return; }
      const entry = await api.createTextEntry({ content: request.content, request_id: request.id, mode: request.mode });
      if (active.current && generation.current === requestGeneration) onSaved(entry.id);
    } catch {
      if (active.current && generation.current === requestGeneration) setError('Could not send your journal. Your text is still here. Please try again.');
    } finally {
      if (generation.current === requestGeneration) { busy.current = false; if (active.current) setSending(false); }
    }
  };
  const canSend = enabled && !!mode && !!draft.trim() && !sending;
  return (
    <SafeAreaView testID={T.record.textScreen} style={[styles.safe, { backgroundColor: colors.bg }]}>
      <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={styles.header}>
          <TouchableOpacity accessibilityRole="button" testID={T.record.textCancel} disabled={sending} onPress={() => leave(onCancel)} style={styles.action}><Text style={[styles.link, { color: colors.textSecondary }]}>Cancel</Text></TouchableOpacity>
          <TouchableOpacity accessibilityRole="button" testID={T.record.voiceMode} disabled={sending} onPress={() => leave(onVoice)} style={styles.action}><Text style={[styles.link, { color: colors.purple300 }]}>Use voice</Text></TouchableOpacity>
        </View>
        <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={styles.content}>
          <Text accessibilityRole="header" style={[styles.title, { color: colors.textPrimary }]}>Write your journal</Text>
          <Text style={[styles.guidance, { color: colors.textSecondary }]}>Write your whole journal in one message. After your reflection, you can chat with Ode for up to 3 follow-up exchanges.</Text>
          {!!prompt && <Text style={[styles.prompt, { color: colors.textPrimary }]}>{prompt}</Text>}
          <View style={styles.modes}>{modes.map(item => <TouchableOpacity key={item.key} accessibilityRole="button" accessibilityState={{ selected: mode === item.key, disabled: sending }} disabled={sending} testID={'text-journal-mode-' + item.key} onPress={() => onSelectMode(item.key)} style={[styles.mode, { backgroundColor: mode === item.key ? colors.brandGlow : colors.card, borderColor: colors.border }]}><Text style={[styles.modeText, { color: mode === item.key ? colors.textPrimary : colors.textSecondary }]}>{item.label}</Text></TouchableOpacity>)}</View>
          <TextInput testID={T.record.textInput} accessibilityLabel="Your complete journal" value={draft} onChangeText={setDraft} editable={!sending && enabled} multiline maxLength={MAX_LENGTH} placeholder="What’s on your mind?" placeholderTextColor={colors.textSecondary} textAlignVertical="top" style={[styles.input, { backgroundColor: colors.card, borderColor: colors.border, color: colors.textPrimary }]} />
          <Text style={[styles.count, { color: colors.textSecondary }]}>{draft.length.toLocaleString()} / 20,000</Text>
          {!!error && <Text accessibilityRole="alert" testID={T.record.textError} style={[styles.error, { color: colors.danger }]}>{error}</Text>}
          <TouchableOpacity testID={T.record.textSend} accessibilityRole="button" accessibilityState={{ disabled: !canSend, busy: sending }} disabled={!canSend} onPress={send} style={[styles.send, { backgroundColor: colors.brandGlow, borderColor: colors.border, opacity: canSend || sending ? 1 : 0.45 }]}>{sending ? <ActivityIndicator color={colors.textPrimary} /> : <Text style={[styles.sendText, { color: colors.textPrimary }]}>Send journal</Text>}</TouchableOpacity>
          {!mode && <Text style={[styles.count, { color: colors.textSecondary }]}>Choose a journal mode before sending.</Text>}
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1 }, fill: { flex: 1 }, header: { flexDirection: 'row', justifyContent: 'space-between', paddingHorizontal: 24, paddingTop: 8 },
  action: { minHeight: 44, justifyContent: 'center' }, link: { fontFamily: Fonts.sans, fontSize: 14 },
  content: { padding: 24, paddingTop: 12, paddingBottom: 40 }, title: { fontFamily: Fonts.serif, fontSize: 30, marginBottom: 12 },
  guidance: { fontFamily: Fonts.sans, fontSize: 14, lineHeight: 22, marginBottom: 20 }, prompt: { fontFamily: Fonts.serifRegular, fontSize: 18, lineHeight: 26, marginBottom: 20 },
  modes: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginBottom: 20 }, mode: { minHeight: 44, justifyContent: 'center', paddingHorizontal: 12, borderWidth: 1, borderRadius: 12 }, modeText: { fontFamily: Fonts.sans, fontSize: 13 },
  input: { minHeight: 220, padding: 16, borderWidth: 1, borderRadius: 16, fontFamily: Fonts.sans, fontSize: 16, lineHeight: 25 },
  count: { fontFamily: Fonts.sans, fontSize: 12, marginVertical: 10 }, error: { fontFamily: Fonts.sans, fontSize: 13, lineHeight: 20, marginBottom: 14 },
  send: { minHeight: 48, borderWidth: 1, borderRadius: 12, justifyContent: 'center', alignItems: 'center', marginTop: 6 }, sendText: { fontFamily: Fonts.sansSB, fontSize: 15 },
});
