import AsyncStorage from '@react-native-async-storage/async-storage';
import { discardItem, enqueue, flush, getFailedItems, getQueue, removeItem, retryItem } from '../offlineQueue';
import { uploadRecording } from '../upload';

jest.mock('../upload', () => ({ uploadRecording: jest.fn() }));
const upload = uploadRecording as jest.Mock;
const item = (id = 'a') => ({ id, localUri: `${id}.m4a`, durationSec: 3, sizeBytes: 100, createdAt: '2026-10-12T00:00:00Z', mode: 'dream' as const });

beforeEach(async () => {
  await AsyncStorage.clear();
  jest.clearAllMocks();
  upload.mockResolvedValue({ entry: { id: 'entry' } });
});

describe('offline recordings', () => {
  it('starts empty', async () => { expect(await getQueue()).toEqual([]); });
  it('persists recordings with zero attempts', async () => {
    await enqueue(item());
    expect(await getQueue()).toEqual([{ ...item(), attempts: 0 }]);
  });
  it('removes only the selected recording', async () => {
    await enqueue(item('a')); await enqueue(item('b')); await removeItem('a');
    expect((await getQueue()).map(i => i.id)).toEqual(['b']);
  });
  it('uploads the original mode and removes successful items', async () => {
    await enqueue(item()); await flush();
    expect(upload).toHaveBeenCalledWith('a.m4a', 3, undefined, 'dream');
    expect(await getQueue()).toEqual([]);
  });
  it('does not upload an empty queue', async () => {
    await flush(); expect(upload).not.toHaveBeenCalled();
  });
  it('keeps failed items and increments attempts', async () => {
    await enqueue(item()); upload.mockRejectedValue(new Error('offline'));
    const failed = jest.fn(); await flush(undefined, failed);
    expect((await getQueue())[0].attempts).toBe(1);
    expect(failed).toHaveBeenCalledWith(expect.objectContaining({ id: 'a', attempts: 1 }), expect.any(Error));
  });
  it('continues to upload later items after one failure', async () => {
    await enqueue(item('a')); await enqueue(item('b'));
    upload.mockRejectedValueOnce(new Error('offline'));
    await flush(); expect(upload).toHaveBeenCalledTimes(2);
    expect((await getQueue()).map(i => i.id)).toEqual(['a']);
  });
  it('does not double upload during overlapping flushes', async () => {
    await enqueue(item());
    let resolve!: (value: unknown) => void;
    let started!: () => void;
    const uploadStarted = new Promise<void>(r => { started = r; });
    upload.mockImplementation(() => new Promise(r => { resolve = r; started(); }));
    const running = flush();
    await uploadStarted;
    await flush(); resolve({ entry: { id: 'entry' } }); await running;
    expect(upload).toHaveBeenCalledTimes(1);
  });
  it('preserves recordings added during an upload', async () => {
    await enqueue(item('a'));
    upload.mockImplementationOnce(async () => { await enqueue(item('b')); return { entry: { id: 'entry' } }; });
    await flush(); expect((await getQueue()).map(i => i.id)).toEqual(['b']);
  });
  it('recovers after a failed flush on reconnect', async () => {
    await enqueue(item()); upload.mockRejectedValueOnce(new Error('offline'));
    await flush(); await flush(); expect(await getQueue()).toEqual([]);
  });
  it('tolerates corrupt persisted JSON without crashing', async () => {
    await AsyncStorage.setItem('dreamlog:offline_queue', '{bad');
    expect(await getQueue()).toEqual([]);
  });
  it('does not lose simultaneous recordings', async () => {
    await Promise.all([enqueue(item('a')), enqueue(item('b'))]);
    expect((await getQueue()).map(i => i.id).sort()).toEqual(['a', 'b']);
  });
  it('does not resurrect an item removed during a failed upload', async () => {
    await enqueue(item());
    upload.mockImplementationOnce(async () => { await removeItem('a'); throw new Error('offline'); });
    await flush(); expect(await getQueue()).toEqual([]);
  });
  // M7: recordings are never dropped silently.
  it('keeps an item as failed after max attempts instead of dropping it', async () => {
    await enqueue(item()); upload.mockRejectedValue(new Error('offline'));
    for (let i = 0; i < 5; i++) await flush();
    const [q] = await getQueue();
    expect(q).toEqual(expect.objectContaining({ id: 'a', failed: true, attempts: 5 }));
    expect(await getFailedItems()).toHaveLength(1);
    upload.mockClear(); await flush();
    expect(upload).not.toHaveBeenCalled();
  });
  it('marks a permanent 4xx failure (quota) as failed immediately', async () => {
    await enqueue(item());
    upload.mockRejectedValue(Object.assign(new Error('quota'), { response: { status: 409 } }));
    await flush();
    expect((await getQueue())[0]).toEqual(expect.objectContaining({ failed: true, attempts: 1, lastError: 'quota' }));
  });
  it('retries a failed item when the user asks, and discards on request', async () => {
    await enqueue(item('a')); await enqueue(item('b'));
    upload.mockRejectedValue(Object.assign(new Error('quota'), { response: { status: 409 } }));
    await flush();
    upload.mockResolvedValue({ entry: { id: 'entry' } });
    await retryItem('a'); await discardItem('b'); await flush();
    expect(await getQueue()).toEqual([]);
  });
});
