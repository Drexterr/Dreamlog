/**
 * JournalScreen — main screen for recording and viewing entries.
 *
 * Flow:
 *   1. User taps record → permissions requested → recording starts
 *   2. User taps stop → recording stops → upload begins
 *   3. Upload: presign → PUT to storage → POST /entries
 *   4. Entry appears in list with status "pending"
 *   5. User can poll or refresh to see "completed" + transcript
 */

import React, { useCallback, useEffect, useState } from 'react';
import {
  View,
  FlatList,
  Text,
  StyleSheet,
  Alert,
  RefreshControl,
  SafeAreaView,
} from 'react-native';
import NetInfo from '@react-native-community/netinfo';
import { useRecorder } from '../hooks/useRecorder';
import { RecordButton } from '../components/RecordButton';
import { uploadRecording, UploadProgress } from '../services/upload';
import { flush as flushOfflineQueue, enqueue as queueOffline } from '../services/offlineQueue';
import { api } from '../api/client';
import { Entry, OfflineQueueItem } from '../types';
import { uuid } from '../utils/uuid';

export default function JournalScreen() {
  const recorder = useRecorder();
  const [entries, setEntries] = useState<Entry[]>([]);
  const [uploading, setUploading] = useState(false);
  const [uploadPhase, setUploadPhase] = useState<string>('');
  const [refreshing, setRefreshing] = useState(false);

  // Load entries on mount.
  useEffect(() => {
    loadEntries();
  }, []);

  // Flush offline queue when connectivity is restored.
  useEffect(() => {
    const unsub = NetInfo.addEventListener((state) => {
      if (state.isConnected) {
        flushOfflineQueue(
          () => loadEntries(),
          (item, err) => console.warn('offline flush failed', item.id, err.message),
        );
      }
    });
    return unsub;
  }, []);

  const loadEntries = useCallback(async () => {
    try {
      const resp = await api.listEntries(1, 30);
      setEntries(resp.entries);
    } catch (err) {
      console.error('Failed to load entries', err);
    }
  }, []);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    await loadEntries();
    setRefreshing(false);
  }, [loadEntries]);

  const handleRecordPress = useCallback(async () => {
    if (recorder.state === 'idle') {
      await recorder.startRecording();
      return;
    }

    if (recorder.state === 'recording') {
      const result = await recorder.stopRecording();
      if (!result) return;

      setUploading(true);

      // Check connectivity.
      const net = await NetInfo.fetch();
      if (!net.isConnected) {
        // Queue for later upload.
        const item: Omit<OfflineQueueItem, 'attempts'> = {
          id: uuid(),
          localUri: result.uri,
          durationSec: result.durationSec,
          sizeBytes: result.sizeBytes,
          createdAt: new Date().toISOString(),
        };
        await queueOffline(item);
        Alert.alert('Saved Offline', 'Recording saved. It will upload when you\'re back online.');
        setUploading(false);
        recorder.discardRecording();
        return;
      }

      try {
        await uploadRecording(result.uri, result.durationSec, (progress: UploadProgress) => {
          const labels: Record<UploadProgress['phase'], string> = {
            presigning: 'Preparing…',
            uploading: 'Uploading…',
            registering: 'Processing…',
            done: 'Done',
          };
          setUploadPhase(labels[progress.phase]);
        });
        await loadEntries();
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Upload failed';
        Alert.alert('Upload Failed', msg);
      } finally {
        setUploading(false);
        setUploadPhase('');
        recorder.discardRecording();
      }
    }
  }, [recorder, loadEntries]);

  const buttonState = uploading ? 'stopped' : recorder.state;

  return (
    <SafeAreaView style={styles.container}>
      <Text style={styles.title}>DreamLog</Text>

      <View style={styles.recorderSection}>
        <RecordButton
          state={buttonState}
          durationMs={recorder.durationMs}
          onPress={handleRecordPress}
          disabled={uploading}
        />
        {uploading && <Text style={styles.uploadPhase}>{uploadPhase}</Text>}
        {recorder.error && <Text style={styles.error}>{recorder.error}</Text>}
      </View>

      <FlatList
        data={entries}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} />}
        ListEmptyComponent={
          <Text style={styles.emptyText}>No entries yet. Record your first journal.</Text>
        }
        renderItem={({ item }) => <EntryCard entry={item} />}
      />
    </SafeAreaView>
  );
}

function EntryCard({ entry }: { entry: Entry }) {
  const statusColor: Record<Entry['status'], string> = {
    pending: '#f59e0b',
    processing: '#3b82f6',
    completed: '#10b981',
    failed: '#ef4444',
  };

  return (
    <View style={styles.card}>
      <View style={styles.cardHeader}>
        <Text style={styles.cardDate}>
          {new Date(entry.created_at).toLocaleDateString('en-US', {
            month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
          })}
        </Text>
        <View style={[styles.statusBadge, { backgroundColor: statusColor[entry.status] }]}>
          <Text style={styles.statusText}>{entry.status}</Text>
        </View>
      </View>

      {entry.transcript ? (
        <Text style={styles.transcript} numberOfLines={4}>{entry.transcript}</Text>
      ) : (
        <Text style={styles.noTranscript}>
          {entry.status === 'failed' ? entry.error_msg ?? 'Transcription failed' : 'Transcribing…'}
        </Text>
      )}

      <Text style={styles.meta}>
        {Math.floor(entry.duration_sec / 60)}m {Math.floor(entry.duration_sec % 60)}s
        {entry.language ? ` · ${entry.language}` : ''}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#f9fafb' },
  title: { fontSize: 28, fontWeight: '700', color: '#111827', textAlign: 'center', paddingTop: 24, paddingBottom: 8 },
  recorderSection: { alignItems: 'center', paddingVertical: 32, gap: 12 },
  uploadPhase: { fontSize: 14, color: '#6b7280' },
  error: { fontSize: 13, color: '#ef4444' },
  list: { paddingHorizontal: 16, paddingBottom: 32 },
  emptyText: { textAlign: 'center', color: '#9ca3af', marginTop: 48 },
  card: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.08,
    shadowRadius: 4,
    elevation: 2,
  },
  cardHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 },
  cardDate: { fontSize: 13, color: '#6b7280' },
  statusBadge: { borderRadius: 12, paddingHorizontal: 8, paddingVertical: 2 },
  statusText: { fontSize: 11, color: '#fff', fontWeight: '600', textTransform: 'uppercase' },
  transcript: { fontSize: 15, color: '#111827', lineHeight: 22 },
  noTranscript: { fontSize: 14, color: '#9ca3af', fontStyle: 'italic' },
  meta: { fontSize: 12, color: '#9ca3af', marginTop: 8 },
});
