import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  Animated,
  StyleSheet,
  StatusBar,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import NetInfo from '@react-native-community/netinfo';
import { useRecorder } from '../src/hooks/useRecorder';
import { uploadRecording } from '../src/services/upload';
import { enqueue as queueOffline } from '../src/services/offlineQueue';
import { uuid } from '../src/utils/uuid';
import { useTheme } from '../src/context/ThemeContext';
import type { EntryMode } from '../src/types';

const WAVEFORM_BARS = 9;

// ── Animated waveform ────────────────────────────────────────────────────────
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
        <Animated.View
          key={i}
          style={[wvStyles.bar, { height: anim }]}
        />
      ))}
    </View>
  );
}

const wvStyles = StyleSheet.create({
  wrap: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    height: 36,
  },
  bar: {
    width: 3,
    borderRadius: 2,
    backgroundColor: 'rgba(255,255,255,0.85)',
  },
});

// ── Recording orb ─────────────────────────────────────────────────────────────
function RecordingOrb({
  recording,
  onPress,
}: {
  recording: boolean;
  onPress: () => void;
}) {
  const { colors } = useTheme();
  const pulseAnim = useRef(new Animated.Value(1)).current;
  const glowAnim = useRef(new Animated.Value(0.4)).current;

  useEffect(() => {
    if (!recording) { pulseAnim.setValue(1); glowAnim.setValue(0.4); return; }
    const loop = Animated.loop(
      Animated.sequence([
        Animated.parallel([
          Animated.timing(pulseAnim, { toValue: 1.06, duration: 1500, useNativeDriver: true }),
          Animated.timing(glowAnim, { toValue: 0.65, duration: 1500, useNativeDriver: true }),
        ]),
        Animated.parallel([
          Animated.timing(pulseAnim, { toValue: 1, duration: 1500, useNativeDriver: true }),
          Animated.timing(glowAnim, { toValue: 0.4, duration: 1500, useNativeDriver: true }),
        ]),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [recording]);

  return (
    <TouchableOpacity onPress={onPress} activeOpacity={0.85}>
      <View style={styles.orbWrap}>
        <Animated.View style={[styles.orbGlow, { backgroundColor: colors.purple700, opacity: glowAnim }]} />
        <Animated.View
          style={[
            styles.orb,
            {
              backgroundColor: recording ? colors.purple600 : colors.purple700,
              shadowColor: colors.purple500,
              transform: [{ scale: pulseAnim }],
            },
          ]}
        >
          {recording ? <Waveform /> : <View style={styles.micDot} />}
        </Animated.View>
      </View>
    </TouchableOpacity>
  );
}

// ── Time display ──────────────────────────────────────────────────────────────
function Timer({ ms }: { ms: number }) {
  const { colors } = useTheme();
  const totalSec = Math.floor(ms / 1000);
  const m = String(Math.floor(totalSec / 60)).padStart(2, '0');
  const s = String(totalSec % 60).padStart(2, '0');
  return <Text style={[styles.timer, { color: colors.textMuted }]}>{m}:{s}</Text>;
}

// ── Mode picker ───────────────────────────────────────────────────────────────

const MODES: { key: EntryMode; label: string; description: string }[] = [
  { key: 'processing', label: 'Process', description: 'Full emotional analysis' },
  { key: 'rant',       label: 'Rant',    description: 'Just be heard' },
  { key: 'gratitude',  label: 'Gratitude', description: "Notice what's good" },
  { key: 'decision',   label: 'Decide',  description: 'Think it through' },
];

function ModePicker({
  selected,
  onSelect,
}: {
  selected: EntryMode;
  onSelect: (m: EntryMode) => void;
}) {
  const { colors } = useTheme();
  return (
    <View style={mpStyles.row}>
      {MODES.map((m) => {
        const active = m.key === selected;
        return (
          <TouchableOpacity
            key={m.key}
            style={[
              mpStyles.chip,
              {
                backgroundColor: active ? colors.brandGlow : 'transparent',
                borderColor: active ? colors.brand : colors.border,
              },
            ]}
            onPress={() => onSelect(m.key)}
            activeOpacity={0.75}
          >
            <Text style={[mpStyles.chipLabel, { color: active ? colors.textPrimary : colors.textMuted }]}>
              {m.label}
            </Text>
          </TouchableOpacity>
        );
      })}
    </View>
  );
}

const mpStyles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    justifyContent: 'center',
    marginBottom: 32,
  },
  chip: {
    borderWidth: 1,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 7,
  },
  chipLabel: {
    fontSize: 13,
    fontFamily: 'Nunito_600SemiBold',
  },
});

// ── Main screen ───────────────────────────────────────────────────────────────
export default function RecordScreen() {
  const router = useRouter();
  const recorder = useRecorder();
  const { colors } = useTheme();
  const [phase, setPhase] = useState<'idle' | 'recording' | 'uploading'>('idle');
  const [uploadLabel, setUploadLabel] = useState('');
  const [mode, setMode] = useState<EntryMode>('processing');

  // Don't auto-start - wait for user to tap the orb after choosing mode.

  const handleStop = useCallback(async () => {
    if (phase !== 'recording') return;

    const result = await recorder.stopRecording();
    if (!result) return;

    setPhase('uploading');

    const net = await NetInfo.fetch();
    if (!net.isConnected) {
      await queueOffline({
        id: uuid(),
        localUri: result.uri,
        durationSec: result.durationSec,
        sizeBytes: result.sizeBytes,
        createdAt: new Date().toISOString(),
        mode,
      });
      Alert.alert(
        'Saved offline',
        "You're offline. Recording saved - it will upload when you reconnect.",
        [{ text: 'OK', onPress: () => router.back() }],
      );
      return;
    }

    try {
      const { entry } = await uploadRecording(result.uri, result.durationSec, (p) => {
        const labels = {
          presigning: 'Preparing…',
          uploading: 'Uploading…',
          registering: 'Processing…',
          done: 'Done',
        };
        setUploadLabel(labels[p.phase]);
      }, mode);
      recorder.discardRecording();
      router.replace(`/processing/${entry.id}`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Upload failed';
      Alert.alert('Upload failed', msg, [
        { text: 'OK', onPress: () => { setPhase('recording'); recorder.startRecording(); } },
      ]);
    }
  }, [phase, recorder, router, mode]);

  const handleOrbPress = useCallback(async () => {
    if (phase === 'idle') {
      await recorder.startRecording();
      setPhase('recording');
    } else if (phase === 'recording') {
      await handleStop();
    }
  }, [phase, recorder, handleStop]);

  const handleCancel = useCallback(async () => {
    await recorder.discardRecording();
    router.back();
  }, [recorder, router]);

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={styles.safe}>
        {/* Cancel */}
        {phase === 'recording' && (
          <TouchableOpacity onPress={handleCancel} style={styles.cancelBtn}>
            <Text style={[styles.cancelText, { color: colors.textMuted }]}>Cancel</Text>
          </TouchableOpacity>
        )}

        {/* Status label - only while recording/uploading so it doesn't leave a gap in idle */}
        {phase !== 'idle' && (
          <Text style={[styles.listeningLabel, { color: colors.textSecondary }]}>
            {phase === 'recording' ? 'listening' : 'uploading'}
          </Text>
        )}

        {/* Tagline */}
        <Text style={[styles.tagline, { color: colors.textPrimary }]}>
          {phase === 'idle' ? 'What kind of\nsession is this?' : 'Take your time.\nI’m here.'}
        </Text>

        {/* Mode picker - visible only before recording starts */}
        {phase === 'idle' && (
          <ModePicker selected={mode} onSelect={setMode} />
        )}

        {/* Orb */}
        <RecordingOrb
          recording={phase === 'recording'}
          onPress={handleOrbPress}
        />

        {/* Timer / upload label */}
        <View style={styles.timerWrap}>
          {phase === 'recording' ? (
            <Timer ms={recorder.durationMs} />
          ) : phase === 'uploading' ? (
            <Text style={[styles.uploadLabel, { color: colors.textSecondary }]}>{uploadLabel}</Text>
          ) : (
            <Text style={[styles.hint, { color: colors.textMuted }]}>tap to begin</Text>
          )}
        </View>

        {/* Error */}
        {recorder.error && (
          <Text style={[styles.errorText, { color: colors.danger }]}>{recorder.error}</Text>
        )}

        {/* Hint while recording */}
        {phase === 'recording' && (
          <Text style={[styles.bottomHint, { color: colors.textMuted }]}>tap the orb when you're done</Text>
        )}
      </SafeAreaView>
    </View>
  );
}

const ORB = 168;

const styles = StyleSheet.create({
  container: { flex: 1 },
  safe: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: 24,
    position: 'relative',
  },

  cancelBtn: {
    position: 'absolute',
    top: 16,
    left: 20,
    padding: 8,
  },
  cancelText: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
  },

  listeningLabel: {
    fontSize: 12,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 2,
    textTransform: 'uppercase',
    marginBottom: 10,
  },
  tagline: {
    fontSize: 22,
    fontFamily: 'CormorantGaramond_300Light',
    textAlign: 'center',
    lineHeight: 32,
    marginBottom: 48,
    maxWidth: 260,
  },

  orbWrap: {
    width: ORB + 40,
    height: ORB + 40,
    alignItems: 'center',
    justifyContent: 'center',
  },
  orbGlow: {
    position: 'absolute',
    width: ORB + 40,
    height: ORB + 40,
    borderRadius: (ORB + 40) / 2,
  },
  orb: {
    width: ORB,
    height: ORB,
    borderRadius: ORB / 2,
    alignItems: 'center',
    justifyContent: 'center',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.5,
    shadowRadius: 28,
    elevation: 14,
  },
  micDot: {
    width: 24,
    height: 24,
    borderRadius: 12,
    backgroundColor: 'rgba(255,255,255,0.8)',
  },

  timerWrap: {
    marginTop: 36,
    minHeight: 24,
    alignItems: 'center',
    justifyContent: 'center',
    alignSelf: 'stretch',
  },
  timer: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 15,
    letterSpacing: 2,
  },
  uploadLabel: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
  },
  errorText: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    marginTop: 12,
    textAlign: 'center',
  },

  // Inline hint, centered below the orb (idle: "tap to begin").
  hint: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
  },
  // Screen-bottom hint shown while recording.
  bottomHint: {
    position: 'absolute',
    bottom: 48,
    alignSelf: 'center',
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
  },
});
