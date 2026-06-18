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
import { Ionicons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import NetInfo from '@react-native-community/netinfo';
import { useRecorder } from '../src/hooks/useRecorder';
import { uploadRecording } from '../src/services/upload';
import { enqueue as queueOffline } from '../src/services/offlineQueue';
import { uuid } from '../src/utils/uuid';
import { useTheme } from '../src/context/ThemeContext';
import { useAuth } from '../src/context/AuthContext';
import type { EntryMode } from '../src/types';

const WAVEFORM_BARS = 9;

// ── Waveform ──────────────────────────────────────────────────────────────────
function Waveform({ colors }: { colors: any }) {
  const anims = useRef(
    Array.from({ length: WAVEFORM_BARS }, () => new Animated.Value(4)),
  ).current;

  useEffect(() => {
    const loops = anims.map((anim, i) => {
      const loop = Animated.loop(
        Animated.sequence([
          Animated.delay(i * 70),
          Animated.timing(anim, { toValue: 4 + Math.random() * 18, duration: 380 + i * 35, useNativeDriver: false }),
          Animated.timing(anim, { toValue: 4, duration: 380 + i * 35, useNativeDriver: false }),
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
        <Animated.View key={i} style={[wvStyles.bar, { height: anim, backgroundColor: colors.brandCore }]} />
      ))}
    </View>
  );
}

const wvStyles = StyleSheet.create({
  wrap: { flexDirection: 'row', alignItems: 'center', gap: 3.5, height: 32 },
  bar:  { width: 2.5, borderRadius: 2 },
});

// ── Ripple rings ──────────────────────────────────────────────────────────────
function RippleRings({ colors }: { colors: any }) {
  const ring1 = useRef(new Animated.Value(0)).current;
  const ring2 = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    const animate = (anim: Animated.Value, delay: number) =>
      Animated.loop(
        Animated.sequence([
          Animated.delay(delay),
          Animated.timing(anim, { toValue: 1, duration: 1900, useNativeDriver: true }),
          Animated.timing(anim, { toValue: 0, duration: 0, useNativeDriver: true }),
        ]),
      );
    const l1 = animate(ring1, 0);
    const l2 = animate(ring2, 950);
    l1.start(); l2.start();
    return () => { l1.stop(); l2.stop(); };
  }, []);

  const ringStyle = (anim: Animated.Value) => ({
    position: 'absolute' as const,
    width: ORB + 10,
    height: ORB + 10,
    borderRadius: (ORB + 10) / 2,
    borderWidth: 1,
    borderColor: colors.brand,
    opacity: anim.interpolate({ inputRange: [0, 0.3, 1], outputRange: [0, 0.5, 0] }),
    transform: [{ scale: anim.interpolate({ inputRange: [0, 1], outputRange: [1, 1.55] }) }],
  });

  return (
    <>
      <Animated.View style={ringStyle(ring1)} />
      <Animated.View style={ringStyle(ring2)} />
    </>
  );
}

// ── Recording orb — amber throughout, waveform when recording ────────────────
function RecordingOrb({
  recording,
  modeSelected,
  onPress,
  colors,
}: {
  recording: boolean;
  modeSelected: boolean;
  onPress: () => void;
  colors: any;
}) {
  const scaleAnim = useRef(new Animated.Value(1)).current;

  const handlePressIn = () => {
    if (!modeSelected) return;
    Animated.spring(scaleAnim, { toValue: 0.93, useNativeDriver: true, tension: 140, friction: 9 }).start();
  };

  const handlePressOut = () => {
    if (!modeSelected) return;
    Animated.spring(scaleAnim, { toValue: 1, useNativeDriver: true, tension: 55, friction: 7 }).start();
    onPress();
  };

  return (
    <TouchableOpacity activeOpacity={1} onPressIn={handlePressIn} onPressOut={handlePressOut}>
      <View style={styles.orbWrap}>
        {recording && <RippleRings colors={colors} />}
        <Animated.View
          style={[
            styles.orb,
            {
              backgroundColor: colors.brand,
              shadowColor: colors.brand,
              opacity: modeSelected ? 1 : 0.45,
              transform: [{ scale: scaleAnim }],
            },
          ]}
        >
          {recording ? (
            <Waveform colors={colors} />
          ) : (
            <Ionicons name="mic" size={48} color={colors.brandCore} />
          )}
        </Animated.View>
      </View>
    </TouchableOpacity>
  );
}

// ── Timer ─────────────────────────────────────────────────────────────────────
function Timer({ ms, colors }: { ms: number; colors: any }) {
  const totalSec = Math.floor(ms / 1000);
  const m = String(Math.floor(totalSec / 60)).padStart(2, '0');
  const s = String(totalSec % 60).padStart(2, '0');
  return <Text style={[styles.timer, { color: colors.textMuted }]}>{m}:{s}</Text>;
}

// ── Mode cards — stagger in, require selection before recording ───────────────
const MODES: { key: EntryMode; label: string; description: string }[] = [
  { key: 'processing', label: 'Process',   description: 'Full emotional analysis' },
  { key: 'rant',       label: 'Rant',      description: 'Just be heard, no advice' },
  { key: 'gratitude',  label: 'Gratitude', description: "Notice what's going right" },
  { key: 'decision',   label: 'Decide',    description: 'Think a choice through' },
];

function ModeGrid({
  selected,
  onSelect,
  disabled,
  colors,
}: {
  selected: EntryMode | null;
  onSelect: (m: EntryMode) => void;
  disabled: boolean;
  colors: any;
}) {
  const anims = useRef(MODES.map(() => new Animated.Value(0))).current;

  useEffect(() => {
    Animated.stagger(
      55,
      anims.map((a) =>
        Animated.spring(a, { toValue: 1, useNativeDriver: true, tension: 70, friction: 8 }),
      ),
    ).start();
  }, []);

  return (
    <View style={[styles.modeGrid, disabled && { opacity: 0.28, pointerEvents: 'none' }]}>
      {MODES.map((m, i) => {
        const active = m.key === selected;
        const translateY = anims[i].interpolate({ inputRange: [0, 1], outputRange: [10, 0] });
        return (
          <Animated.View
            key={m.key}
            style={{ opacity: anims[i], transform: [{ translateY }], flex: 1, minWidth: '47%' }}
          >
            <TouchableOpacity
              disabled={disabled}
              onPress={() => onSelect(m.key)}
              activeOpacity={0.8}
              style={[
                styles.modeCard,
                {
                  backgroundColor: active ? colors.brandGlow : 'rgba(232,221,208,0.02)',
                  borderColor: active ? colors.brand + '55' : colors.borderFaint,
                },
              ]}
            >
              <Text style={[styles.modeName, { color: active ? colors.brand : colors.textSecondary }]}>
                {m.label}
              </Text>
              <Text style={[styles.modeDesc, { color: active ? colors.textSecondary : colors.textMuted }]}>
                {m.description}
              </Text>
            </TouchableOpacity>
          </Animated.View>
        );
      })}
    </View>
  );
}

// ── Main screen ───────────────────────────────────────────────────────────────
export default function RecordScreen() {
  const router = useRouter();
  const { isAuthenticated, requestAuth } = useAuth();
  const recorder = useRecorder();

  // Safety net: if a guest somehow lands here, prompt auth and go back.
  useEffect(() => {
    if (!isAuthenticated) {
      router.back();
      requestAuth(() => router.push('/record'));
    }
  }, [isAuthenticated]);
  const { colors } = useTheme();
  const [phase, setPhase] = useState<'idle' | 'recording' | 'uploading'>('idle');
  const [uploadLabel, setUploadLabel] = useState('');
  const [mode, setMode] = useState<EntryMode | null>(null);

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
        mode: mode ?? 'processing',
      });
      Alert.alert(
        "Saved for later",
        "You're offline. Your recording will upload when you reconnect.",
        [{ text: 'OK', onPress: () => router.back() }],
      );
      return;
    }

    try {
      const { entry } = await uploadRecording(
        result.uri,
        result.durationSec,
        (p) => {
          const labels = { presigning: 'Preparing…', uploading: 'Uploading…', registering: 'Processing…', done: 'Done' };
          setUploadLabel(labels[p.phase]);
        },
        mode ?? 'processing',
      );
      recorder.discardRecording();
      router.replace(`/processing/${entry.id}`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Upload failed';
      Alert.alert('Something went wrong', msg, [
        { text: 'Try again', onPress: () => { setPhase('recording'); recorder.startRecording(); } },
      ]);
    }
  }, [phase, recorder, router, mode]);

  const handleOrbPress = useCallback(async () => {
    if (!mode) return; // mode must be selected first
    if (phase === 'idle') {
      await recorder.startRecording();
      setPhase('recording');
    } else if (phase === 'recording') {
      await handleStop();
    }
  }, [phase, mode, recorder, handleStop]);

  const handleCancel = useCallback(async () => {
    await recorder.discardRecording();
    router.back();
  }, [recorder, router]);

  const tagline = phase === 'idle'
    ? (mode ? 'Take your time.\nI\'m listening.' : 'What kind of\nsession is this?')
    : phase === 'recording'
    ? 'Take your time.\nI\'m here.'
    : 'Almost done…';

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={styles.safe}>

        {/* Header */}
        <View style={styles.header}>
          <TouchableOpacity onPress={handleCancel} style={styles.cancelBtn}>
            <Text style={[styles.cancelText, { color: colors.textMuted }]}>
              {phase === 'uploading' ? '' : 'Cancel'}
            </Text>
          </TouchableOpacity>
          {phase === 'recording' && (
            <Text style={[styles.listeningLabel, { color: colors.textMuted }]}>listening</Text>
          )}
        </View>

        {/* Tagline */}
        <Text style={[styles.tagline, { color: colors.textPrimary }]}>{tagline}</Text>

        {/* Mode grid — shown until recording starts */}
        {phase === 'idle' && (
          <ModeGrid
            selected={mode}
            onSelect={setMode}
            disabled={false}
            colors={colors}
          />
        )}

        {/* Orb */}
        <View style={styles.orbSection}>
          <RecordingOrb
            recording={phase === 'recording'}
            modeSelected={mode !== null}
            onPress={handleOrbPress}
            colors={colors}
          />

          {/* Timer / upload label / hint */}
          <View style={styles.belowOrb}>
            {phase === 'recording' ? (
              <Timer ms={recorder.durationMs} colors={colors} />
            ) : phase === 'uploading' ? (
              <Text style={[styles.uploadLabel, { color: colors.textSecondary }]}>{uploadLabel}</Text>
            ) : mode ? (
              <Text style={[styles.hint, { color: colors.textMuted }]}>tap to begin</Text>
            ) : (
              <Text style={[styles.hint, { color: colors.textMuted }]}>choose a mode above</Text>
            )}
          </View>
        </View>

        {/* Error */}
        {recorder.error ? (
          <Text style={[styles.errorText, { color: colors.danger }]}>{recorder.error}</Text>
        ) : null}

        {/* Bottom hint while recording */}
        {phase === 'recording' && (
          <Text style={[styles.bottomHint, { color: colors.textMuted }]}>tap the button when you're done</Text>
        )}

      </SafeAreaView>
    </View>
  );
}

const ORB = 148;

const styles = StyleSheet.create({
  container: { flex: 1 },
  safe: {
    flex: 1,
    alignItems: 'center',
    padding: 24,
  },

  header: {
    width: '100%',
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  cancelBtn: { padding: 6 },
  cancelText: { fontSize: 14, fontFamily: 'Nunito_400Regular' },
  listeningLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 2,
    textTransform: 'lowercase',
  },

  tagline: {
    fontSize: 22,
    fontFamily: 'CormorantGaramond_300Light',
    textAlign: 'center',
    lineHeight: 32,
    marginBottom: 28,
    maxWidth: 260,
  },

  // Mode grid
  modeGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 7,
    width: '100%',
    marginBottom: 12,
  },
  modeCard: {
    padding: 12,
    borderRadius: 11,
    borderWidth: 1,
  },
  modeName: {
    fontSize: 12.5,
    fontFamily: 'Nunito_400Regular',
    marginBottom: 3,
  },
  modeDesc: {
    fontSize: 10,
    fontFamily: 'Nunito_400Regular',
    fontWeight: '300',
    lineHeight: 14,
  },

  // Orb area
  orbSection: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 20,
  },
  orbWrap: {
    width: ORB + 20,
    height: ORB + 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
  orb: {
    width: ORB,
    height: ORB,
    borderRadius: ORB / 2,
    alignItems: 'center',
    justifyContent: 'center',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.2,
    shadowRadius: 20,
    elevation: 10,
  },
  // Below orb
  belowOrb: { minHeight: 28, alignItems: 'center', justifyContent: 'center' },
  timer: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 22,
    letterSpacing: 3,
    fontVariant: ['tabular-nums'],
  },
  uploadLabel: { fontSize: 14, fontFamily: 'Nunito_400Regular' },
  hint: { fontSize: 12, fontFamily: 'Nunito_400Regular', textAlign: 'center' },
  errorText: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    marginTop: 12,
    textAlign: 'center',
  },
  bottomHint: {
    position: 'absolute',
    bottom: 48,
    alignSelf: 'center',
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
  },
});
