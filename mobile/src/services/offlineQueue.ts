/**
 * Offline queue: persists pending uploads in AsyncStorage so they survive
 * app restarts and network outages.
 *
 * Usage:
 *   - Call enqueue() when a recording finishes but network is unavailable.
 *   - Call flush() when connectivity is restored.
 *   - Call removeItem() after a successful upload.
 */

import AsyncStorage from '@react-native-async-storage/async-storage';
import { OfflineQueueItem } from '../types';
import { uploadRecording } from './upload';

const QUEUE_KEY = 'dreamlog:offline_queue';
const MAX_ITEM_ATTEMPTS = 5;

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

// ── Public API ───────────────────────────────────────────────────────────────
export async function enqueue(item: Omit<OfflineQueueItem, 'attempts'>): Promise<void> {
  const queue = await readQueue();
  queue.push({ ...item, attempts: 0 });
  await writeQueue(queue);
}

export async function removeItem(id: string): Promise<void> {
  const queue = await readQueue();
  await writeQueue(queue.filter((i) => i.id !== id));
}

export async function getQueue(): Promise<OfflineQueueItem[]> {
  return readQueue();
}

/**
 * Attempt to upload all queued items.
 * Items that succeed are removed from the queue.
 * Items that fail increment their attempt counter.
 * Items that exceed MAX_ITEM_ATTEMPTS are discarded (with logging).
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
    if (queue.length === 0) return;

    const remaining: OfflineQueueItem[] = [];

    for (const item of queue) {
      try {
        await uploadRecording(item.localUri, item.durationSec, undefined, item.mode ?? 'processing');
        onItemUploaded?.(item);
        // Remove from queue by NOT adding to remaining[].
      } catch (err) {
        const updated = { ...item, attempts: item.attempts + 1 };
        const error = err instanceof Error ? err : new Error(String(err));
        onItemFailed?.(updated, error);

        if (updated.attempts < MAX_ITEM_ATTEMPTS) {
          remaining.push(updated);
        }
        // Else: silently drop - too many failures, avoid infinite accumulation.
      }
    }

    // Preserve any items enqueued while this flush was running.
    const current = await readQueue();
    const processedIds = new Set(queue.map((i) => i.id));
    const newlyEnqueued = current.filter((i) => !processedIds.has(i.id));
    await writeQueue([...remaining, ...newlyEnqueued]);
  } finally {
    flushInFlight = false;
  }
}
