import React from 'react';
import {
  TouchableOpacity,
  View,
  Text,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { RecorderState } from '../hooks/useRecorder';

interface Props {
  state: RecorderState;
  durationMs: number;
  onPress: () => void;
  disabled?: boolean;
}

export function RecordButton({ state, durationMs, onPress, disabled }: Props) {
  const label = {
    idle: 'Hold to Record',
    recording: formatDuration(durationMs),
    stopped: 'Processing…',
    error: 'Error – Tap to Retry',
  }[state];

  const isRecording = state === 'recording';
  const isProcessing = state === 'stopped';

  return (
    <TouchableOpacity
      onPress={onPress}
      disabled={disabled || isProcessing}
      activeOpacity={0.8}
      style={styles.wrapper}
    >
      <View style={[styles.button, isRecording && styles.recording, isProcessing && styles.processing]}>
        {isProcessing ? (
          <ActivityIndicator color="#fff" size="large" />
        ) : (
          <View style={[styles.inner, isRecording && styles.innerRecording]} />
        )}
      </View>
      <Text style={[styles.label, isRecording && styles.labelRecording]}>{label}</Text>
    </TouchableOpacity>
  );
}

function formatDuration(ms: number): string {
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60).toString().padStart(2, '0');
  const s = (totalSec % 60).toString().padStart(2, '0');
  return `${m}:${s}`;
}

const styles = StyleSheet.create({
  wrapper: { alignItems: 'center', gap: 12 },
  button: {
    width: 88,
    height: 88,
    borderRadius: 44,
    backgroundColor: '#4f46e5',
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.3,
    shadowRadius: 8,
    elevation: 8,
  },
  recording: { backgroundColor: '#dc2626', transform: [{ scale: 1.1 }] },
  processing: { backgroundColor: '#6b7280' },
  inner: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: 'rgba(255,255,255,0.9)',
  },
  innerRecording: {
    width: 20,
    height: 20,
    borderRadius: 4,
  },
  label: { fontSize: 14, color: '#374151', fontWeight: '600' },
  labelRecording: { color: '#dc2626' },
});
