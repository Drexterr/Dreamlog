import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  SafeAreaView,
  StatusBar,
  ActivityIndicator,
  Animated,
} from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { useRecorder } from '../../src/hooks/useRecorder';
import { uploadRecording } from '../../src/services/upload';
import type { JourneySession, JourneyStep } from '../../src/types';

// ── Step row ──────────────────────────────────────────────────────────────────
function StepRow({
  step,
  isCurrent,
  onViewEntry,
}: {
  step: JourneyStep;
  isCurrent: boolean;
  onViewEntry: (entryId: string) => void;
}) {
  const { colors } = useTheme();

  const dotColor = step.completed
    ? colors.moodGreen
    : isCurrent
    ? colors.purple400
    : colors.textFaint;

  return (
    <View style={styles.stepRow}>
      <View style={styles.stepTimeline}>
        <View style={[styles.stepDot, { backgroundColor: dotColor, borderColor: dotColor }]} />
      </View>

      <View style={[styles.stepContent, { borderColor: colors.borderFaint }]}>
        <Text style={[styles.stepIndex, { color: colors.textMuted }]}>Step {step.step_index + 1}</Text>
        <Text
          style={[
            styles.stepPrompt,
            { color: step.completed ? colors.textMuted : isCurrent ? colors.textPrimary : colors.textSecondary },
          ]}
        >
          {step.prompt}
        </Text>
        {step.completed && step.entry_id && (
          <TouchableOpacity onPress={() => onViewEntry(step.entry_id!)} style={styles.viewEntryBtn}>
            <Text style={[styles.viewEntryText, { color: colors.purple300 }]}>View reflection →</Text>
          </TouchableOpacity>
        )}
        {step.completed && !step.entry_id && (
          <Text style={[styles.completedLabel, { color: colors.moodGreen }]}>Completed</Text>
        )}
      </View>
    </View>
  );
}

// ── Recording UI for the current step ─────────────────────────────────────────
function RecordStep({
  sessionId,
  stepPrompt,
  onAdvanced,
}: {
  sessionId: string;
  stepPrompt: string;
  onAdvanced: (session: JourneySession) => void;
}) {
  const { colors } = useTheme();
  const recorder = useRecorder();
  const isRecording = recorder.state === 'recording';
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const pulseAnim = useRef(new Animated.Value(1)).current;

  useEffect(() => {
    if (isRecording) {
      const loop = Animated.loop(
        Animated.sequence([
          Animated.timing(pulseAnim, { toValue: 1.12, duration: 700, useNativeDriver: true }),
          Animated.timing(pulseAnim, { toValue: 1, duration: 700, useNativeDriver: true }),
        ]),
      );
      loop.start();
      return () => loop.stop();
    } else {
      pulseAnim.setValue(1);
    }
  }, [isRecording]);

  const handleStop = useCallback(async () => {
    const result = await recorder.stopRecording();
    if (!result) return;

    setUploadError(null);
    setUploading(true);
    try {
      const { entry } = await uploadRecording(result.uri, result.durationSec);
      const updated = await api.advanceJourneySession(sessionId, entry.id);
      onAdvanced(updated);
    } catch {
      setUploadError('Upload failed. Tap to try again.');
    } finally {
      setUploading(false);
    }
  }, [recorder, sessionId, onAdvanced]);

  const formatDuration = (ms: number) => {
    const s = Math.floor(ms / 1000);
    return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
  };

  return (
    <View style={[styles.recordPanel, { backgroundColor: colors.card, borderColor: colors.border }]}>
      <Text style={[styles.recordLabel, { color: colors.textMuted }]}>YOUR TURN</Text>
      <Text style={[styles.recordPrompt, { color: colors.textPrimary }]}>{stepPrompt}</Text>

      {uploading ? (
        <View style={styles.recordCenter}>
          <ActivityIndicator color={colors.purple400} />
          <Text style={[styles.recordHint, { color: colors.textMuted }]}>Uploading…</Text>
        </View>
      ) : (
        <View style={styles.recordCenter}>
          <Animated.View style={{ transform: [{ scale: pulseAnim }] }}>
            <TouchableOpacity
              style={[
                styles.recordBtn,
                {
                  backgroundColor: isRecording ? colors.danger + '22' : colors.purple700,
                  borderColor: isRecording ? colors.danger : colors.purple500,
                },
              ]}
              onPress={isRecording ? handleStop : recorder.startRecording}
              activeOpacity={0.8}
            >
              {isRecording ? (
                <View style={[styles.stopSquare, { backgroundColor: colors.danger }]} />
              ) : (
                <View style={[styles.micDot, { backgroundColor: colors.purple300 }]} />
              )}
            </TouchableOpacity>
          </Animated.View>

          {isRecording && recorder.durationMs > 0 && (
            <Text style={[styles.recordTimer, { color: colors.purple300 }]}>
              {formatDuration(recorder.durationMs)}
            </Text>
          )}
          {!isRecording && !uploadError && (
            <Text style={[styles.recordHint, { color: colors.textMuted }]}>
              Tap to record your response
            </Text>
          )}
          {uploadError && (
            <Text style={[styles.errorHint, { color: colors.danger }]}>{uploadError}</Text>
          )}
        </View>
      )}
    </View>
  );
}

// ── Session screen ─────────────────────────────────────────────────────────────
export default function JourneySessionScreen() {
  const { sessionId } = useLocalSearchParams<{ sessionId: string }>();
  const router = useRouter();
  const { colors } = useTheme();

  const [session, setSession] = useState<JourneySession | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    if (!sessionId) return;
    setLoading(true);
    api.getJourneySession(sessionId)
      .then(setSession)
      .catch(() => setError('Could not load session.'))
      .finally(() => setLoading(false));
  }, [sessionId]);

  useEffect(() => { load(); }, [load]);

  const handleAdvanced = useCallback((updated: JourneySession) => {
    setSession(updated);
  }, []);

  const handleViewEntry = useCallback((entryId: string) => {
    router.push({ pathname: '/reflection/[id]', params: { id: entryId } });
  }, [router]);

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor: colors.bg }]}>
        <SafeAreaView style={styles.center}>
          <ActivityIndicator color={colors.purple400} />
        </SafeAreaView>
      </View>
    );
  }

  if (error || !session) {
    return (
      <View style={[styles.container, { backgroundColor: colors.bg }]}>
        <SafeAreaView style={styles.center}>
          <Text style={[styles.errorText, { color: colors.textMuted }]}>{error ?? 'Session not found.'}</Text>
          <TouchableOpacity onPress={() => router.back()} style={[styles.retryBtn, { borderColor: colors.border }]}>
            <Text style={[styles.retryText, { color: colors.purple300 }]}>Go back</Text>
          </TouchableOpacity>
        </SafeAreaView>
      </View>
    );
  }

  const isComplete = session.status === 'completed';
  const pct = session.total_steps > 0 ? session.current_step / session.total_steps : 0;
  const currentStep: JourneyStep | undefined = session.steps[session.current_step];

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={styles.safe}>
        {/* Header */}
        <View style={styles.header}>
          <TouchableOpacity onPress={() => router.back()} style={styles.backBtn} hitSlop={{ top: 8, bottom: 8, left: 8, right: 8 }}>
            <Text style={[styles.backArrow, { color: colors.textSecondary }]}>←</Text>
          </TouchableOpacity>
          <Text style={[styles.headerTitle, { color: colors.textPrimary }]} numberOfLines={1}>
            {session.journey_title}
          </Text>
          <Text style={[styles.headerMeta, { color: colors.textMuted }]}>
            {isComplete ? '✓' : `${session.current_step}/${session.total_steps}`}
          </Text>
        </View>

        {/* Progress bar */}
        <View style={[styles.progressTrack, { backgroundColor: colors.border }]}>
          <View
            style={[
              styles.progressFill,
              {
                width: `${(isComplete ? 1 : pct) * 100}%`,
                backgroundColor: isComplete ? colors.moodGreen : colors.purple500,
              },
            ]}
          />
        </View>

        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
          {/* Completion banner */}
          {isComplete && (
            <View style={[styles.completeBanner, { backgroundColor: colors.moodGreen + '18', borderColor: colors.moodGreen + '40' }]}>
              <Text style={[styles.completeBannerTitle, { color: colors.moodGreen }]}>Journey complete</Text>
              <Text style={[styles.completeBannerSub, { color: colors.textSecondary }]}>
                You finished all {session.total_steps} steps. Tap any step to revisit your reflection.
              </Text>
            </View>
          )}

          {/* Steps */}
          <View style={styles.steps}>
            {session.steps.map((step, i) => (
              <StepRow
                key={step.step_index}
                step={step}
                isCurrent={!isComplete && i === session.current_step}
                onViewEntry={handleViewEntry}
              />
            ))}
          </View>

          {/* Record panel for current step */}
          {!isComplete && currentStep && (
            <RecordStep
              sessionId={session.id}
              stepPrompt={currentStep.prompt}
              onAdvanced={handleAdvanced}
            />
          )}
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  safe: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 16 },

  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingTop: 12,
    paddingBottom: 8,
  },
  backBtn: { width: 32, alignItems: 'flex-start' },
  backArrow: { fontSize: 22 },
  headerTitle: {
    flex: 1,
    textAlign: 'center',
    fontSize: 17,
    fontFamily: 'CormorantGaramond_500Medium',
    letterSpacing: 0.3,
    marginHorizontal: 8,
  },
  headerMeta: {
    width: 32,
    textAlign: 'right',
    fontSize: 12,
    fontFamily: 'Nunito_600SemiBold',
  },

  progressTrack: {
    height: 2,
    marginHorizontal: 20,
    borderRadius: 1,
    overflow: 'hidden',
    marginBottom: 4,
  },
  progressFill: { height: '100%', borderRadius: 1 },

  scroll: { paddingHorizontal: 20, paddingBottom: 48, paddingTop: 12 },

  completeBanner: {
    borderRadius: 14,
    borderWidth: 1,
    padding: 18,
    marginBottom: 24,
    gap: 6,
  },
  completeBannerTitle: {
    fontSize: 16,
    fontFamily: 'CormorantGaramond_500Medium',
    letterSpacing: 0.2,
  },
  completeBannerSub: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 19,
  },

  steps: { marginBottom: 24 },
  stepRow: {
    flexDirection: 'row',
    gap: 14,
    marginBottom: 16,
  },
  stepTimeline: { alignItems: 'center', width: 14, paddingTop: 4 },
  stepDot: {
    width: 12,
    height: 12,
    borderRadius: 6,
    borderWidth: 2,
  },
  stepContent: {
    flex: 1,
    paddingBottom: 16,
    borderBottomWidth: 1,
    gap: 4,
  },
  stepIndex: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1,
    textTransform: 'uppercase',
  },
  stepPrompt: {
    fontSize: 15,
    fontFamily: 'CormorantGaramond_500Medium',
    lineHeight: 22,
    letterSpacing: 0.1,
  },
  viewEntryBtn: { marginTop: 4 },
  viewEntryText: { fontSize: 12, fontFamily: 'Nunito_400Regular' },
  completedLabel: { fontSize: 12, fontFamily: 'Nunito_600SemiBold', marginTop: 4 },

  recordPanel: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 20,
    gap: 14,
    marginTop: 8,
  },
  recordLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.5,
  },
  recordPrompt: {
    fontSize: 16,
    fontFamily: 'CormorantGaramond_500Medium',
    lineHeight: 24,
    letterSpacing: 0.1,
  },
  recordCenter: { alignItems: 'center', gap: 14, paddingVertical: 8 },
  recordBtn: {
    width: 72,
    height: 72,
    borderRadius: 36,
    borderWidth: 2,
    alignItems: 'center',
    justifyContent: 'center',
  },
  stopSquare: { width: 22, height: 22, borderRadius: 4 },
  micDot: { width: 24, height: 24, borderRadius: 12 },
  recordTimer: {
    fontSize: 22,
    fontFamily: 'Nunito_300Light',
    letterSpacing: 2,
  },
  recordHint: { fontSize: 13, fontFamily: 'Nunito_400Regular', textAlign: 'center' },
  errorHint: { fontSize: 12, fontFamily: 'Nunito_400Regular', textAlign: 'center' },

  errorText: { fontSize: 14, fontFamily: 'Nunito_400Regular', textAlign: 'center' },
  retryBtn: { paddingHorizontal: 20, paddingVertical: 10, borderRadius: 20, borderWidth: 1 },
  retryText: { fontSize: 14, fontFamily: 'Nunito_600SemiBold' },
});
