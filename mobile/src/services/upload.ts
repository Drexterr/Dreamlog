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

/** Lets a retried upload continue from an earlier attempt (offline queue). */
export interface UploadResume {
  /** Key of audio an earlier attempt already uploaded. */
  uploadedKey?: string;
  /** Called once the audio is in storage, so the caller can persist the key. */
  onUploaded?: (key: string) => void;
}

/** Thrown upload error; uploadedKey is set when the audio did reach storage. */
export type UploadError = Error & { uploadedKey?: string };

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
  resume?: UploadResume,
): Promise<UploadResult> {
  const file = new File(localUri);
  if (!file.exists) {
    throw new Error('Recording file not found: ' + localUri);
  }
  const sizeBytes = file.size ?? 0;

  let lastError: Error | null = null;
  // Set once the audio is safely in storage. Retries then only re-run the step
  // that failed, with the SAME key: re-presigning would orphan the first object
  // (never deleted - ADR-005) and could create a duplicate entry. The backend
  // treats POST /entries as idempotent per audio_key.
  let uploadedKey: string | null = resume?.uploadedKey ?? null;
  // A resumed key may have expired from storage (orphan lifecycle rules) -
  // then upload once more instead of failing for good.
  let resumedKey = uploadedKey !== null;

  for (let attempt = 0; attempt < MAX_UPLOAD_ATTEMPTS; attempt++) {
    if (attempt > 0) {
      const backoffMs = Math.pow(2, attempt - 1) * 1000;
      await sleep(backoffMs);
    }

    try {
      if (!uploadedKey) {
        // Step 1: Get upload URL from backend.
        onProgress?.({ phase: 'presigning' });
        const { upload_url, audio_key } = await api.presign();

        // Step 2: Upload via native HTTP (bypasses whatwg-fetch polyfill which
        // doesn't handle binary bodies correctly in React Native).
        onProgress?.({ phase: 'uploading', uploadedBytes: 0, totalBytes: sizeBytes });

        const uploadResult = await FileSystem.uploadAsync(upload_url, localUri, {
          httpMethod: 'PUT',
          uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
          headers: { 'Content-Type': 'audio/mp4' },
        });

        if (uploadResult.status < 200 || uploadResult.status >= 300) {
          throw new Error(`Storage upload failed with status ${uploadResult.status}: ${uploadResult.body}`);
        }

        onProgress?.({ phase: 'uploading', uploadedBytes: sizeBytes, totalBytes: sizeBytes });
        uploadedKey = audio_key;
        resume?.onUploaded?.(audio_key);
      }

      // Step 3: Notify backend - creates DB row + queues transcription job.
      onProgress?.({ phase: 'registering' });
      const entry = await api.createEntry({
        audio_key: uploadedKey,
        audio_size_bytes: sizeBytes,
        duration_sec: durationSec,
        mode,
      });

      onProgress?.({ phase: 'done' });
      return { entry };
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err));

      if (resumedKey && isMissingAudio(err)) {
        resumedKey = false;
        uploadedKey = null;
        continue;
      }

      // Do not retry on known non-recoverable errors (e.g., 4xx from backend).
      if (isApiError(err) && err.response && err.response.status < 500) {
        throw withUploadedKey(lastError, uploadedKey);
      }
    }
  }

  throw withUploadedKey(lastError ?? new Error('Upload failed after max retries'), uploadedKey);
}

function withUploadedKey(err: Error, key: string | null): UploadError {
  if (key) (err as UploadError).uploadedKey = key;
  return err;
}

// POST /entries reports 400 "audio file not found in storage" when the
// object behind a resumed key is gone.
function isMissingAudio(err: unknown): boolean {
  if (!isApiError(err) || err.response?.status !== 400) return false;
  const data = err.response.data as { message?: unknown; error?: unknown } | undefined;
  const msg = String(data?.message ?? data?.error ?? '');
  return msg.includes('not found in storage');
}

/**
 * PUTs a local file to a pre-signed storage URL with native binary upload (the
 * same mechanism as recordings - whatwg-fetch mishandles binary bodies in
 * React Native). Throws on a non-2xx response. Used for therapist note photos.
 */
export async function putFileToPresignedUrl(uploadUrl: string, localUri: string, contentType: string): Promise<void> {
  const result = await FileSystem.uploadAsync(uploadUrl, localUri, {
    httpMethod: 'PUT',
    uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
    headers: { 'Content-Type': contentType },
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error(`Storage upload failed with status ${result.status}`);
  }
}

const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));
