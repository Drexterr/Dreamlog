/**
 * Direct-to-storage upload using pre-signed PUT URLs.
 *
 * Flow:
 *   1. GET pre-signed URL from backend
 *   2. PUT audio file directly to storage (no backend bandwidth used)
 *   3. POST /entries to create DB record + queue transcription job
 *
 * Retry policy: up to 3 attempts with exponential backoff (1s, 2s, 4s).
 */

import { File } from 'expo-file-system/next';
import * as FileSystem from 'expo-file-system/legacy';
import { api, isApiError } from '../api/client';
import { Entry, EntryMode } from '../types';

const MAX_UPLOAD_ATTEMPTS = 3;

export interface UploadResult {
  entry: Entry;
}

export interface UploadProgress {
  phase: 'presigning' | 'uploading' | 'registering' | 'done';
  uploadedBytes?: number;
  totalBytes?: number;
}

export async function uploadRecording(
  localUri: string,
  durationSec: number,
  onProgress?: (p: UploadProgress) => void,
  mode: EntryMode = 'processing',
): Promise<UploadResult> {
  const file = new File(localUri);
  if (!file.exists) {
    throw new Error('Recording file not found: ' + localUri);
  }
  const sizeBytes = file.size ?? 0;

  let lastError: Error | null = null;

  for (let attempt = 0; attempt < MAX_UPLOAD_ATTEMPTS; attempt++) {
    if (attempt > 0) {
      const backoffMs = Math.pow(2, attempt - 1) * 1000;
      await sleep(backoffMs);
    }

    try {
      // Step 1: Get upload URL from backend.
      onProgress?.({ phase: 'presigning' });
      const { upload_url, audio_key } = await api.presign();

      // Step 2: Upload via native HTTP (bypasses whatwg-fetch polyfill which
      // doesn't handle binary bodies correctly in React Native).
      onProgress?.({ phase: 'uploading', uploadedBytes: 0, totalBytes: sizeBytes });

      const uploadResult = await FileSystem.uploadAsync(upload_url, localUri, {
        httpMethod: 'PUT',
        uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
        headers: { 'Content-Type': 'audio/aac' },
      });

      if (uploadResult.status < 200 || uploadResult.status >= 300) {
        throw new Error(`Storage upload failed with status ${uploadResult.status}: ${uploadResult.body}`);
      }

      onProgress?.({ phase: 'uploading', uploadedBytes: sizeBytes, totalBytes: sizeBytes });

      // Step 3: Notify backend - creates DB row + queues transcription job.
      onProgress?.({ phase: 'registering' });
      const entry = await api.createEntry({
        audio_key,
        audio_size_bytes: sizeBytes,
        duration_sec: durationSec,
        mode,
      });

      onProgress?.({ phase: 'done' });
      return { entry };
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err));

      // Do not retry on known non-recoverable errors (e.g., 4xx from backend).
      if (isApiError(err) && err.response && err.response.status < 500) {
        throw lastError;
      }
    }
  }

  throw lastError ?? new Error('Upload failed after max retries');
}

const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));
