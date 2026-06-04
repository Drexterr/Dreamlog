import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  ScrollView,
  Animated,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  Alert,
  Linking,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import * as FileSystem from 'expo-file-system/legacy';
import { Audio } from 'expo-av';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { useRecorder } from '../../src/hooks/useRecorder';
import type { TherapySession, TherapySessionMessage, TherapySessionStatus } from '../../src/types';
import { PERSONA_META } from '../../src/types';

const CRISIS_HOTLINES = [
  { name: 'iCall', tel: 'tel:9152987821', info: '9152987821 · India' },
  { name: 'Vandrevala Foundation', tel: 'tel:18602662345', info: '1860-2662-345 · India · 24/7' },
  { name: '988 Lifeline', tel: 'tel:988', info: '988 · US · 24/7' },
];

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatTimeRemaining(sec: number): string {
  if (sec <= 0) return '0:00';
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

function formatDuration(ms: number): string {
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  return `${m}:${(s % 60).toString().padStart(2, '0')}`;
}

// ── Animated waveform (reuses record.tsx pattern) ─────────────────────────────

const WAVEFORM_BARS = 9;

function Waveform() {
  const anims = useRef(
    Array.from({ length: WAVEFORM_BARS }, () => new Animated.Value(6)),
  ).current;

  useEffect(() => {
    const loops = anims.map((anim, i) => {
      const targetH = 8 + Math.random() * 20;
      const loop = Animated.loop(
        Animated.sequence([
          Animated.delay(i * 80),
          Animated.timing(anim, { toValue: targetH, duration: 400 + i * 40, useNativeDriver: false }),
          Animated.timing(anim, { toValue: 6, duration: 400 + i * 40, useNativeDriver: false }),
        ]),
      );
      loop.start();
      return loop;
    });
    return () => loops.forEach((l) => l.stop());
  }, []);

  return (
    <View style={wvStyles.wrap}>
      {anims.map((anim, i) => (
        <Animated.View key={i} style={[wvStyles.bar, { height: anim }]} />
      ))}
    </View>
  );
}

const wvStyles = StyleSheet.create({
  wrap: { flexDirection: 'row', alignItems: 'center', gap: 4, height: 36 },
  bar: { width: 3, borderRadius: 2, backgroundColor: 'rgba(255,255,255,0.85)' },
});

// ── Pulsating orb ─────────────────────────────────────────────────────────────

const ORB = 160;

function SessionOrb({
  recording,
  thinking,
  onPress,
  colors,
}: {
  recording: boolean;
  thinking: boolean;
  onPress: () => void;
  colors: any;
}) {
  const pulseAnim = useRef(new Animated.Value(1)).current;
  const glowAnim = useRef(new Animated.Value(0.35)).current;
  const idleAnim = useRef(new Animated.Value(1)).current;

  // Idle breathing pulse (always on, subtle)
  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(idleAnim, { toValue: 1.03, duration: 2400, useNativeDriver: true }),
        Animated.timing(idleAnim, { toValue: 1, duration: 2400, useNativeDriver: true }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, []);

  // Strong pulse when recording
  useEffect(() => {
    if (!recording) { pulseAnim.setValue(1); glowAnim.setValue(0.35); return; }
    const loop = Animated.loop(
      Animated.sequence([
        Animated.parallel([
          Animated.timing(pulseAnim, { toValue: 1.08, duration: 1200, useNativeDriver: true }),
          Animated.timing(glowAnim, { toValue: 0.7, duration: 1200, useNativeDriver: true }),
        ]),
        Animated.parallel([
          Animated.timing(pulseAnim, { toValue: 1, duration: 1200, useNativeDriver: true }),
          Animated.timing(glowAnim, { toValue: 0.35, duration: 1200, useNativeDriver: true }),
        ]),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [recording]);

  const scale = recording ? pulseAnim : idleAnim;

  return (
    <TouchableOpacity onPress={onPress} activeOpacity={0.85} disabled={thinking}>
      <View style={orbStyles.wrap}>
        {/* Outer glow ring */}
        <Animated.View
          style={[
            orbStyles.glow,
            {
              backgroundColor: recording ? colors.danger : colors.purple600,
              opacity: glowAnim,
            },
          ]}
        />
        {/* Orb */}
        <Animated.View
          style={[
            orbStyles.orb,
            {
              backgroundColor: recording ? colors.danger : colors.purple700,
              shadowColor: recording ? colors.danger : colors.purple500,
              transform: [{ scale }],
            },
          ]}
        >
          {thinking ? (
            <ActivityIndicator color="rgba(255,255,255,0.9)" size="large" />
          ) : recording ? (
            <Waveform />
          ) : (
            <Text style={orbStyles.micIcon}>🎙</Text>
          )}
        </Animated.View>
      </View>
    </TouchableOpacity>
  );
}

const orbStyles = StyleSheet.create({
  wrap: {
    width: ORB + 48,
    height: ORB + 48,
    alignItems: 'center',
    justifyContent: 'center',
  },
  glow: {
    position: 'absolute',
    width: ORB + 48,
    height: ORB + 48,
    borderRadius: (ORB + 48) / 2,
  },
  orb: {
    width: ORB,
    height: ORB,
    borderRadius: ORB / 2,
    alignItems: 'center',
    justifyContent: 'center',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.55,
    shadowRadius: 32,
    elevation: 16,
  },
  micIcon: { fontSize: 36 },
});

// ── Message bubble ────────────────────────────────────────────────────────────

function MessageBubble({ msg, colors }: { msg: TherapySessionMessage; colors: any }) {
  const isUser = msg.role === 'user';
  return (
    <View
      style={[
        bubbleStyles.bubble,
        isUser
          ? [bubbleStyles.user, { backgroundColor: colors.brand }]
          : [bubbleStyles.assistant, { backgroundColor: colors.card, borderColor: colors.border }],
      ]}
    >
      <Text style={[bubbleStyles.text, { color: isUser ? '#fff' : colors.textPrimary }]}>
        {msg.content}
      </Text>
      {msg.input_mode === 'voice' && isUser && (
        <Text style={bubbleStyles.voicePip}>🎙</Text>
      )}
    </View>
  );
}

const bubbleStyles = StyleSheet.create({
  bubble: {
    maxWidth: '82%',
    borderRadius: 18,
    padding: 13,
    marginBottom: 6,
  },
  user: {
    alignSelf: 'flex-end',
    borderBottomRightRadius: 4,
  },
  assistant: {
    alignSelf: 'flex-start',
    borderBottomLeftRadius: 4,
    borderWidth: 1,
  },
  text: {
    fontSize: 15,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 22,
  },
  voicePip: { fontSize: 10, marginTop: 4, textAlign: 'right' },
});

// ── Crisis screen ─────────────────────────────────────────────────────────────

function CrisisScreen({ colors }: { colors: any }) {
  return (
    <View style={{ gap: 14, padding: 24 }}>
      <Text style={[{ fontSize: 26, fontFamily: 'CormorantGaramond_600SemiBold', color: colors.textPrimary }]}>
        You're not alone.
      </Text>
      <Text style={[{ fontSize: 15, fontFamily: 'Nunito_400Regular', color: colors.textSecondary, lineHeight: 22 }]}>
        This session has been paused. Please reach out to one of these resources right now.
      </Text>
      {CRISIS_HOTLINES.map((h) => (
        <TouchableOpacity
          key={h.name}
          style={[{ borderWidth: 1, borderRadius: 12, padding: 16, backgroundColor: colors.card, borderColor: colors.danger }]}
          onPress={() => Linking.openURL(h.tel)}
          activeOpacity={0.8}
        >
          <Text style={[{ fontSize: 16, fontFamily: 'Nunito_700Bold', color: colors.textPrimary, marginBottom: 4 }]}>{h.name}</Text>
          <Text style={[{ fontSize: 13, fontFamily: 'Nunito_400Regular', color: colors.textSecondary }]}>{h.info}</Text>
        </TouchableOpacity>
      ))}
    </View>
  );
}

// ── Main screen ───────────────────────────────────────────────────────────────

export default function TherapySessionScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();

  const [session, setSession] = useState<TherapySession | null>(null);
  const [messages, setMessages] = useState<TherapySessionMessage[]>([]);
  const [status, setStatus] = useState<TherapySessionStatus>('active');
  const [timeRemaining, setTimeRemaining] = useState(3600);
  const [crisisWarnings, setCrisisWarnings] = useState(0);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [uploadingVoice, setUploadingVoice] = useState(false);
  const [draft, setDraft] = useState('');
  const [inputMode, setInputMode] = useState<'voice' | 'text'>('voice');

  const scrollRef = useRef<ScrollView>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const ttsSound = useRef<Audio.Sound | null>(null);
  const recorder = useRecorder();
  const isRecording = recorder.state === 'recording';

  // Unload TTS sound on unmount to avoid memory leaks.
  useEffect(() => {
    return () => {
      ttsSound.current?.unloadAsync().catch(() => null);
    };
  }, []);

  const playTTS = useCallback(async (url: string | undefined) => {
    if (!url) return;
    try {
      await ttsSound.current?.unloadAsync().catch(() => null);
      const { sound } = await Audio.Sound.createAsync({ uri: url }, { shouldPlay: true });
      ttsSound.current = sound;
    } catch {
      // TTS playback failures are silent — text is always visible as fallback.
    }
  }, []);

  // Load session on mount
  useEffect(() => {
    if (!id) return;
    api.getTherapySession(id)
      .then((s) => {
        setSession(s);
        setMessages(s.messages ?? []);
        setStatus(s.status);
        setTimeRemaining(s.time_remaining_sec);
        setCrisisWarnings(s.crisis_warnings ?? 0);
      })
      .catch(() => Alert.alert('Error', 'Could not load session.'))
      .finally(() => setLoading(false));
  }, [id]);

  // Countdown timer
  useEffect(() => {
    if (status !== 'active') return;
    timerRef.current = setInterval(() => {
      setTimeRemaining((t) => {
        if (t <= 1) { clearInterval(timerRef.current!); setStatus('expired'); return 0; }
        return t - 1;
      });
    }, 1000);
    return () => clearInterval(timerRef.current!);
  }, [status]);

  useEffect(() => {
    scrollRef.current?.scrollToEnd({ animated: true });
  }, [messages]);

  const handleSend = useCallback(async () => {
    const content = draft.trim();
    if (!content || !id || sending || status !== 'active') return;
    setDraft('');
    setSending(true);
    try {
      const resp = await api.sendTherapyMessage(id, { content, input_mode: 'text' });
      setMessages((prev) => [...prev, resp.user_message, resp.assistant_message]);
      setStatus(resp.session_state.status);
      setTimeRemaining(resp.session_state.time_remaining_sec);
      setCrisisWarnings(resp.session_state.crisis_warnings ?? 0);
      playTTS(resp.assistant_message.tts_url);
      // Switch back to voice mode after sending so orb is the focus again
      setInputMode('voice');
    } catch (err: any) {
      const s = err?.response?.status;
      if (s === 410) setStatus('expired');
      else if (s === 409) Alert.alert('Session ended', 'This session is no longer active.');
      else Alert.alert('Error', 'Could not send message. Please try again.');
    } finally {
      setSending(false);
    }
  }, [draft, id, sending, status]);

  const handleVoicePress = useCallback(async () => {
    if (!id || status !== 'active') return;

    if (isRecording) {
      const result = await recorder.stopRecording();
      if (!result) return;
      setUploadingVoice(true);
      try {
        const filename = `therapy-${id}-${Date.now()}.m4a`;
        const { upload_url, audio_key } = await api.presignTherapyAudio(id, filename, 'audio/aac');
        const uploadResult = await FileSystem.uploadAsync(upload_url, result.uri, {
          httpMethod: 'PUT',
          uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
          headers: { 'Content-Type': 'audio/aac' },
        });
        if (uploadResult.status < 200 || uploadResult.status >= 300) throw new Error(`Upload failed: ${uploadResult.status}`);
        setSending(true);
        const resp = await api.sendTherapyMessage(id, { audio_key, input_mode: 'voice' });
        setMessages((prev) => [...prev, resp.user_message, resp.assistant_message]);
        setStatus(resp.session_state.status);
        setTimeRemaining(resp.session_state.time_remaining_sec);
        setCrisisWarnings(resp.session_state.crisis_warnings ?? 0);
        playTTS(resp.assistant_message.tts_url);
      } catch (err: any) {
        if (err?.response?.status === 410) setStatus('expired');
        else Alert.alert('Error', 'Could not send voice message. Please try again.');
        await recorder.discardRecording();
      } finally {
        setUploadingVoice(false);
        setSending(false);
      }
    } else {
      await recorder.startRecording();
    }
  }, [id, isRecording, recorder, status]);

  const handleEnd = useCallback(() => {
    Alert.alert('End Session', 'Are you sure you want to end this session?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'End Session',
        style: 'destructive',
        onPress: async () => {
          if (!id) return;
          if (isRecording) await recorder.discardRecording();
          try {
            await api.endTherapySession(id);
            router.replace({ pathname: '/therapy/summary/[id]', params: { id } } as any);
          } catch {
            Alert.alert('Error', 'Could not end session. Please try again.');
          }
        },
      },
    ]);
  }, [id, isRecording, recorder, router]);

  if (loading) {
    return (
      <SafeAreaView style={[styles.safe, { backgroundColor: colors.bg }]}>
        <ActivityIndicator color={colors.brand} style={{ flex: 1 }} />
      </SafeAreaView>
    );
  }

  const isCrisis = status === 'crisis_detected';
  const isDeEscalating = crisisWarnings === 1 && status === 'active';
  const isEnded = status === 'completed' || status === 'expired' || isCrisis;
  const isBusy = sending || uploadingVoice;
  const personaMeta = session?.persona ? PERSONA_META[session.persona] : null;
  const timeIsLow = timeRemaining < 300;

  // ── Crisis view ───────────────────────────────────────────────────────────
  if (isCrisis) {
    return (
      <SafeAreaView style={[styles.safe, { backgroundColor: colors.bg }]}>
        <ScrollView><CrisisScreen colors={colors} /></ScrollView>
      </SafeAreaView>
    );
  }

  // ── Text input mode ───────────────────────────────────────────────────────
  if (inputMode === 'text' && !isEnded) {
    return (
      <SafeAreaView style={[styles.safe, { backgroundColor: colors.bg }]}>
        {/* Header */}
        <View style={[styles.header, { borderBottomColor: colors.border }]}>
          <TouchableOpacity
            style={styles.backToVoice}
            onPress={() => setInputMode('voice')}
            activeOpacity={0.7}
          >
            <Text style={[styles.backToVoiceText, { color: colors.brand }]}>🎙 Voice</Text>
          </TouchableOpacity>
          <View style={{ alignItems: 'center' }}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 5 }}>
              {personaMeta && <Text style={{ fontSize: 14 }}>{personaMeta.emoji}</Text>}
              <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>
                {personaMeta ? personaMeta.label : 'Reflection Session'}
              </Text>
            </View>
            <Text style={[styles.headerTimer, { color: timeIsLow ? colors.danger : colors.textMuted }]}>
              {formatTimeRemaining(timeRemaining)} remaining
            </Text>
          </View>
          <TouchableOpacity onPress={handleEnd} style={[styles.endBtn, { borderColor: colors.border }]}>
            <Text style={[styles.endBtnText, { color: colors.textMuted }]}>End</Text>
          </TouchableOpacity>
        </View>

        <KeyboardAvoidingView
          style={{ flex: 1 }}
          behavior={Platform.OS === 'ios' ? 'padding' : undefined}
          keyboardVerticalOffset={90}
        >
          {isDeEscalating && (
            <View style={[styles.deEscBanner, { backgroundColor: `${colors.danger}18`, borderColor: `${colors.danger}44` }]}>
              <Text style={[styles.deEscText, { color: colors.danger }]}>
                It sounds like you might be struggling. Please reach out to someone you trust or call a crisis line.
              </Text>
            </View>
          )}

          <ScrollView
            ref={scrollRef}
            contentContainerStyle={styles.messageList}
            keyboardShouldPersistTaps="handled"
          >
            {messages.length === 0 && (
              <Text style={[styles.emptyHint, { color: colors.textMuted }]}>
                Start by sharing what's on your mind.
              </Text>
            )}
            {messages.map((m) => <MessageBubble key={m.id} msg={m} colors={colors} />)}
            {isBusy && (
              <View style={[bubbleStyles.bubble, bubbleStyles.assistant, { backgroundColor: colors.card, borderColor: colors.border, borderWidth: 1 }]}>
                <ActivityIndicator size="small" color={colors.brand} />
              </View>
            )}
          </ScrollView>

          {/* Text input bar */}
          <View style={[styles.inputBar, { borderTopColor: colors.border, backgroundColor: colors.bg }]}>
            <TextInput
              style={[styles.textInput, { color: colors.textPrimary, borderColor: colors.border, backgroundColor: colors.card }]}
              placeholder="Type a message…"
              placeholderTextColor={colors.textMuted}
              value={draft}
              onChangeText={setDraft}
              multiline
              maxLength={2000}
              returnKeyType="send"
              blurOnSubmit={false}
              editable={!isBusy}
              autoFocus
            />
            <TouchableOpacity
              style={[styles.sendBtn, { backgroundColor: draft.trim() && !isBusy ? colors.brand : colors.cardSolid }]}
              onPress={handleSend}
              disabled={!draft.trim() || isBusy}
            >
              <Text style={styles.sendBtnText}>↑</Text>
            </TouchableOpacity>
          </View>
        </KeyboardAvoidingView>
      </SafeAreaView>
    );
  }

  // ── Voice / orb mode (primary) ────────────────────────────────────────────
  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <SafeAreaView style={{ flex: 1 }}>

        {/* Header */}
        <View style={[styles.header, { borderBottomColor: colors.border }]}>
          <View style={{ width: 60 }} />
          <View style={{ alignItems: 'center' }}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 5 }}>
              {personaMeta && <Text style={{ fontSize: 14 }}>{personaMeta.emoji}</Text>}
              <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>
                {personaMeta ? personaMeta.label : 'Reflection Session'}
              </Text>
            </View>
            {!isEnded && (
              <Text style={[styles.headerTimer, { color: timeIsLow ? colors.danger : colors.textMuted }]}>
                {formatTimeRemaining(timeRemaining)} remaining
              </Text>
            )}
            {status === 'expired' && (
              <Text style={[styles.headerTimer, { color: colors.danger }]}>Session expired</Text>
            )}
          </View>
          {!isEnded ? (
            <TouchableOpacity onPress={handleEnd} style={[styles.endBtn, { borderColor: colors.border }]}>
              <Text style={[styles.endBtnText, { color: colors.textMuted }]}>End</Text>
            </TouchableOpacity>
          ) : (
            <View style={{ width: 60 }} />
          )}
        </View>

        {/* De-escalation banner */}
        {isDeEscalating && (
          <View style={[styles.deEscBanner, { backgroundColor: `${colors.danger}18`, borderColor: `${colors.danger}44` }]}>
            <Text style={[styles.deEscText, { color: colors.danger }]}>
              It sounds like you might be struggling. Please reach out to someone you trust or call a crisis line.
            </Text>
          </View>
        )}

        {/* Recent messages (compact, above orb) */}
        {messages.length > 0 && (
          <ScrollView
            ref={scrollRef}
            style={styles.messageScroll}
            contentContainerStyle={styles.messageList}
            showsVerticalScrollIndicator={false}
          >
            {messages.map((m) => <MessageBubble key={m.id} msg={m} colors={colors} />)}
            {isBusy && (
              <View style={[bubbleStyles.bubble, bubbleStyles.assistant, { backgroundColor: colors.card, borderColor: colors.border, borderWidth: 1 }]}>
                <ActivityIndicator size="small" color={colors.brand} />
              </View>
            )}
          </ScrollView>
        )}

        {/* Central orb area */}
        <View style={styles.orbArea}>
          {/* Tagline / status text */}
          {!isEnded && (
            <Text style={[styles.orbTagline, { color: colors.textSecondary }]}>
              {isRecording
                ? 'listening…'
                : isBusy
                ? 'thinking…'
                : messages.length === 0
                ? personaMeta?.tagline ?? 'I\'m here to listen'
                : 'tap to speak'}
            </Text>
          )}

          {/* Orb — hidden when ended */}
          {!isEnded && (
            <SessionOrb
              recording={isRecording}
              thinking={isBusy}
              onPress={handleVoicePress}
              colors={colors}
            />
          )}

          {/* Recording timer */}
          {isRecording && (
            <Text style={[styles.recTimer, { color: colors.textMuted }]}>
              {formatDuration(recorder.durationMs)}
            </Text>
          )}

          {/* "tap to speak" hint when idle */}
          {!isRecording && !isBusy && !isEnded && (
            <Text style={[styles.tapHint, { color: colors.textMuted }]}>
              {isRecording ? 'tap to stop' : 'tap to speak'}
            </Text>
          )}

          {/* Ended state */}
          {isEnded && (
            <View style={styles.endedBox}>
              <Text style={[styles.endedLabel, { color: colors.textSecondary }]}>
                {status === 'expired' ? 'Session expired.' : 'Session complete.'}
              </Text>
              <TouchableOpacity
                style={[styles.summaryBtn, { backgroundColor: colors.brand }]}
                onPress={() => router.replace({ pathname: '/therapy/summary/[id]', params: { id } } as any)}
                activeOpacity={0.85}
              >
                <Text style={styles.summaryBtnText}>View Summary →</Text>
              </TouchableOpacity>
            </View>
          )}
        </View>

        {/* Bottom action row */}
        {!isEnded && (
          <View style={[styles.bottomBar, { borderTopColor: colors.borderFaint }]}>
            {/* Chat toggle */}
            <TouchableOpacity
              style={[styles.chatToggle, { backgroundColor: colors.card, borderColor: colors.border }]}
              onPress={() => setInputMode('text')}
              activeOpacity={0.75}
            >
              <Text style={[styles.chatToggleIcon, { color: colors.textMuted }]}>⌨️</Text>
              <Text style={[styles.chatToggleLabel, { color: colors.textMuted }]}>Chat</Text>
            </TouchableOpacity>

            {/* Spacer / recorder control hint */}
            <View style={styles.bottomCenter}>
              {isRecording && (
                <TouchableOpacity
                  style={[styles.stopPill, { backgroundColor: `${colors.danger}22`, borderColor: colors.danger }]}
                  onPress={handleVoicePress}
                >
                  <View style={[styles.stopDot, { backgroundColor: colors.danger }]} />
                  <Text style={[styles.stopLabel, { color: colors.danger }]}>tap to stop</Text>
                </TouchableOpacity>
              )}
            </View>

            {/* Spacer to balance chat toggle */}
            <View style={{ width: 80 }} />
          </View>
        )}
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1 },
  container: { flex: 1 },

  // Header
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
  },
  headerTitle: { fontSize: 16, fontFamily: 'Nunito_700Bold' },
  headerTimer: { fontSize: 11, fontFamily: 'Nunito_400Regular', marginTop: 2 },
  endBtn: { borderWidth: 1, borderRadius: 8, paddingHorizontal: 12, paddingVertical: 6, width: 60, alignItems: 'center' },
  endBtnText: { fontSize: 13, fontFamily: 'Nunito_600SemiBold' },
  backToVoice: { paddingHorizontal: 4 },
  backToVoiceText: { fontSize: 13, fontFamily: 'Nunito_600SemiBold' },

  // De-escalation
  deEscBanner: { borderWidth: 1, paddingHorizontal: 16, paddingVertical: 10 },
  deEscText: { fontSize: 13, fontFamily: 'Nunito_400Regular', lineHeight: 18 },

  // Messages
  messageScroll: { maxHeight: '38%' },
  messageList: { padding: 16, paddingBottom: 8, gap: 4 },
  emptyHint: { textAlign: 'center', marginTop: 40, fontSize: 14, fontFamily: 'Nunito_400Regular' },

  // Orb area
  orbArea: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 16,
    gap: 8,
  },
  orbTagline: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.5,
    marginBottom: 8,
  },
  recTimer: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 15,
    letterSpacing: 2,
    marginTop: 4,
  },
  tapHint: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.5,
    marginTop: 4,
  },

  // Ended
  endedBox: { alignItems: 'center', gap: 16, marginTop: 8 },
  endedLabel: { fontSize: 15, fontFamily: 'Nunito_400Regular' },
  summaryBtn: {
    borderRadius: 14,
    paddingVertical: 14,
    paddingHorizontal: 32,
  },
  summaryBtnText: {
    color: '#fff',
    fontSize: 15,
    fontFamily: 'Nunito_700Bold',
  },

  // Bottom bar
  bottomBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 24,
    paddingVertical: 16,
    borderTopWidth: 1,
  },
  chatToggle: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    borderWidth: 1,
    borderRadius: 20,
    paddingHorizontal: 14,
    paddingVertical: 8,
    width: 80,
  },
  chatToggleIcon: { fontSize: 14 },
  chatToggleLabel: { fontSize: 13, fontFamily: 'Nunito_600SemiBold' },
  bottomCenter: { flex: 1, alignItems: 'center' },
  stopPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    borderWidth: 1,
    borderRadius: 16,
    paddingHorizontal: 14,
    paddingVertical: 8,
  },
  stopDot: { width: 8, height: 8, borderRadius: 4 },
  stopLabel: { fontSize: 13, fontFamily: 'Nunito_600SemiBold' },

  // Text input (text mode)
  inputBar: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    padding: 12,
    gap: 8,
    borderTopWidth: 1,
  },
  textInput: {
    flex: 1,
    borderWidth: 1,
    borderRadius: 14,
    paddingHorizontal: 14,
    paddingVertical: 10,
    fontSize: 15,
    fontFamily: 'Nunito_400Regular',
    maxHeight: 120,
  },
  sendBtn: {
    width: 44,
    height: 44,
    borderRadius: 22,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendBtnText: { color: '#fff', fontSize: 18, fontFamily: 'Nunito_700Bold' },
});
