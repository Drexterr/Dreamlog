/**
 * useRecorder - manages the full audio recording lifecycle.
 *
 * Audio config:
 *   - Format:    AAC (m4a container)
 *   - Channels:  mono (1)
 *   - Sample rate: 44100 Hz
 *   - Bit rate:  64 kbps
 *
 * Enforces a 30-minute max recording duration by auto-stopping.
 */

import { useRef, useState, useCallback, useEffect } from 'react';
import { Audio } from 'expo-av';

const MAX_DURATION_MS = 30 * 60 * 1000; // 30 minutes

export type RecorderState = 'idle' | 'recording' | 'stopped' | 'error';

export interface RecordingResult {
  uri: string;
  durationSec: number;
  sizeBytes: number;
}

export interface UseRecorderReturn {
  state: RecorderState;
  durationMs: number;
  error: string | null;
  startRecording: () => Promise<void>;
  stopRecording: () => Promise<RecordingResult | null>;
  discardRecording: () => Promise<void>;
}

const RECORDING_OPTIONS: Audio.RecordingOptions = {
  android: {
    extension: '.m4a',
    outputFormat: Audio.AndroidOutputFormat.MPEG_4,
    audioEncoder: Audio.AndroidAudioEncoder.AAC,
    sampleRate: 44100,
    numberOfChannels: 1,
    bitRate: 64_000,
  },
  ios: {
    extension: '.m4a',
    outputFormat: Audio.IOSOutputFormat.MPEG4AAC,
    audioQuality: Audio.IOSAudioQuality.MEDIUM,
    sampleRate: 44100,
    numberOfChannels: 1,
    bitRate: 64_000,
    linearPCMBitDepth: 16,
    linearPCMIsBigEndian: false,
    linearPCMIsFloat: false,
  },
  web: {},
};

export function useRecorder(): UseRecorderReturn {
  const recordingRef = useRef<Audio.Recording | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const maxTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [state, setState] = useState<RecorderState>('idle');
  const [durationMs, setDurationMs] = useState(0);
  const [error, setError] = useState<string | null>(null);

  // Cleanup on unmount.
  useEffect(() => {
    return () => {
      clearInterval(timerRef.current ?? undefined);
      clearTimeout(maxTimerRef.current ?? undefined);
      if (recordingRef.current) {
        recordingRef.current.stopAndUnloadAsync().catch(() => {});
        recordingRef.current = null;
      }
    };
  }, []);

  const startRecording = useCallback(async () => {
    try {
      setError(null);
      setDurationMs(0);

      // Ensure any existing recording is stopped and unloaded first
      if (recordingRef.current) {
        try {
          await recordingRef.current.stopAndUnloadAsync();
        } catch {}
        recordingRef.current = null;
      }

      const { status } = await Audio.requestPermissionsAsync();
      if (status !== 'granted') {
        setError('Microphone permission denied');
        setState('error');
        return;
      }

      await Audio.setAudioModeAsync({
        allowsRecordingIOS: true,
        playsInSilentModeIOS: true,
      });

      const { recording } = await Audio.Recording.createAsync(RECORDING_OPTIONS);
      recordingRef.current = recording;
      setState('recording');

      // Tick timer every second.
      const startTime = Date.now();
      timerRef.current = setInterval(() => {
        setDurationMs(Date.now() - startTime);
      }, 1000);

      // Auto-stop at max duration.
      maxTimerRef.current = setTimeout(async () => {
        if (recordingRef.current) {
          clearInterval(timerRef.current ?? undefined);
          await recordingRef.current.stopAndUnloadAsync();
          setState('stopped');
        }
      }, MAX_DURATION_MS);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to start recording';
      setError(msg);
      setState('error');
    }
  }, []);

  const stopRecording = useCallback(async (): Promise<RecordingResult | null> => {
    if (!recordingRef.current) return null;

    clearInterval(timerRef.current ?? undefined);
    clearTimeout(maxTimerRef.current ?? undefined);

    try {
      await recordingRef.current.stopAndUnloadAsync();
      const uri = recordingRef.current.getURI();
      const status = await recordingRef.current.getStatusAsync();

      recordingRef.current = null;
      setState('stopped');

      if (!uri) {
        setError('Recording URI not available');
        setState('error');
        return null;
      }

      // Reset audio mode.
      await Audio.setAudioModeAsync({ allowsRecordingIOS: false });

      const durationSec = (status.durationMillis ?? durationMs) / 1000;
      return { uri, durationSec, sizeBytes: 0 }; // sizeBytes filled by upload service
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to stop recording';
      setError(msg);
      setState('error');
      return null;
    }
  }, [durationMs]);

  const discardRecording = useCallback(async () => {
    clearInterval(timerRef.current ?? undefined);
    clearTimeout(maxTimerRef.current ?? undefined);

    if (recordingRef.current) {
      try {
        await recordingRef.current.stopAndUnloadAsync();
      } catch {
        // Ignore errors during discard.
      }
      recordingRef.current = null;
    }

    setDurationMs(0);
    setState('idle');
    setError(null);
  }, []);

  return { state, durationMs, error, startRecording, stopRecording, discardRecording };
}
