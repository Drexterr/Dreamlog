/**
 * Offline queue: persists pending uploads in AsyncStorage so they survive
 * app restarts and network outages.
 *
 * Usage:
 *   - Call enqueue() when a recording finishes but network is unavailable.
 *   - Call flush() when connectivity is restored.
 *   - Call removeItem() after a successful upload.
 *
 * A recording is never dropped silently: items that keep failing (or fail
 * permanently, e.g. the free-plan quota) are marked `failed` and kept until the
 * user retries or discards them (getFailedItems / retryItem / discardItem).
 * Local audio files are deleted once uploaded or discarded.
 */

import AsyncStorage from '@react-native-async-storage/async-storage';
import * as FileSystem from 'expo-file-system/legacy';
import { OfflineQueueItem } from '../types';
import { uploadRecording } from './upload';

const QUEUE_KEY = 'dreamlog:offline_queue';
const MAX_ITEM_ATTEMPTS = 5;
const PENDING_DIR = 'pending-recordings/';

// ── Persistence ─────────────────────────────────────────────────────────────
async function readQueue(): Promise<OfflineQueueItem[]> {
  try {
    const raw = await AsyncStorage.getItem(QUEUE_KEY);
    return raw ? (JSON.parse(raw) as OfflineQueueItem[]) : [];
  } catch {
    return [];
  }
}

async function writeQueue(items: OfflineQueueItem[]): Promise<void> {
  await AsyncStorage.setItem(QUEUE_KEY, JSON.stringify(items));
}

// Serialize read-modify-write operations so concurrent saves cannot overwrite.
let mutationTail: Promise<void> = Promise.resolve();
function mutateQueue(update: (items: OfflineQueueItem[]) => OfflineQueueItem[]): Promise<void> {
  const mutation = mutationTail.then(async () => writeQueue(update(await readQueue())));
  mutationTail = mutation.catch(() => {});
  return mutation;
}

// Recordings land in the cache directory, which the OS may purge while the
// device is offline. Move queued ones somewhere durable. Fail-soft: if the
// move isn't possible the original path is kept.
async function persistRecording(id: string, uri: string): Promise<string> {
  const docDir = FileSystem.documentDirectory;
  if (!docDir || uri.startsWith(docDir)) return uri;
  try {
    await FileSystem.makeDirectoryAsync(docDir + PENDING_DIR, { intermediates: true });
    const ext = uri.includes('.') ? uri.slice(uri.lastIndexOf('.')) : '.m4a';
    const dest = `${docDir}${PENDING_DIR}${id}${ext}`;
    await FileSystem.moveAsync({ from: uri, to: dest });
    return dest;
  } catch {
    return uri;
  }
}

async function deleteRecording(uri: string): Promise<void> {
  try {
    await FileSystem.deleteAsync(uri, { idempotent: true });
  } catch {
    // Best effort - a leftover file is harmless compared to a lost upload.
  }
}

// Errors no amount of retrying will fix: a 4xx from the API (quota reached,
// validation) other than timeouts / rate limits, or a missing local file.
function isPermanentFailure(err: unknown): boolean {
  const status = (err as { response?: { status?: number } } | null)?.response?.status;
  if (typeof status === 'number') {
    return status >= 400 && status < 500 && status !== 408 && status !== 429;
  }
  return err instanceof Error && err.message.startsWith('Recording file not found');
}

// ── Public API ───────────────────────────────────────────────────────────────
export async function enqueue(item: Omit<OfflineQueueItem, 'attempts'>): Promise<void> {
  const localUri = await persistRecording(item.id, item.localUri);
  await mutateQueue(queue => [...queue, { ...item, localUri, attempts: 0 }]);
}

export async function removeItem(id: string): Promise<void> {
  await mutateQueue(queue => queue.filter((i) => i.id !== id));
}

export async function getQueue(): Promise<OfflineQueueItem[]> {
  return readQueue();
}

/** Recordings that stopped auto-retrying and need the user to decide. */
export async function getFailedItems(): Promise<OfflineQueueItem[]> {
  return (await readQueue()).filter(i => i.failed);
}

/** Puts a failed recording back into automatic retry. */
export async function retryItem(id: string): Promise<void> {
  await mutateQueue(queue => queue.map(i => (i.id === id ? { ...i, failed: false, attempts: 0, lastError: undefined } : i)));
}

/** Permanently discards a recording the user chose not to keep. */
export async function discardItem(id: string): Promise<void> {
  const item = (await readQueue()).find(i => i.id === id);
  await removeItem(id);
  if (item) await deleteRecording(item.localUri);
}

/**
 * Attempt to upload all queued (non-failed) items.
 * Items that succeed are removed from the queue and their local file deleted.
 * Items that fail increment their attempt counter; permanent failures and
 * items reaching MAX_ITEM_ATTEMPTS are marked `failed` and kept for the user.
 */
let flushInFlight = false;

export async function flush(
  onItemUploaded?: (item: OfflineQueueItem) => void,
  onItemFailed?: (item: OfflineQueueItem, err: Error) => void,
): Promise<void> {
  // Guard against overlapping flushes (e.g. rapid reconnect events) which
  // would double-upload the same items.
  if (flushInFlight) return;
  flushInFlight = true;

  try {
    const queue = await readQueue();
    const pending = queue.filter(i => !i.failed);
    if (pending.length === 0) return;

    const remaining: OfflineQueueItem[] = [];
    const uploaded: OfflineQueueItem[] = [];

    for (const item of pending) {
      let uploadedKey = item.uploadedKey;
      try {
        await uploadRecording(item.localUri, item.durationSec, undefined, item.mode ?? 'processing', {
          uploadedKey,
          onUploaded: (key) => { uploadedKey = key; },
        });
        uploaded.push(item);
        onItemUploaded?.(item);
        // Remove from queue by NOT adding to remaining[].
      } catch (err) {
        const error = err instanceof Error ? err : new Error(String(err));
        const attempts = item.attempts + 1;
        const updated: OfflineQueueItem = {
          ...item,
          attempts,
          failed: isPermanentFailure(err) || attempts >= MAX_ITEM_ATTEMPTS,
          lastError: error.message,
          // Keep the key of audio already in storage: the next flush registers
          // it instead of uploading (and orphaning) another copy (N7).
          uploadedKey,
        };
        onItemFailed?.(updated, error);
        remaining.push(updated);
      }
    }

    // Preserve any items enqueued while this flush was running, and failed
    // items that were not part of this flush.
    const processedIds = new Set(pending.map((i) => i.id));
    await mutateQueue(current => {
      const currentIds = new Set(current.map(i => i.id));
      const untouched = current.filter(i => !processedIds.has(i.id));
      return [...remaining.filter(i => currentIds.has(i.id)), ...untouched];
    });

    for (const item of uploaded) await deleteRecording(item.localUri);
  } finally {
    flushInFlight = false;
  }
}
