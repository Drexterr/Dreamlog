import { uploadRecording } from '../upload';
import { api } from '../../api/client';
import { File } from 'expo-file-system/next';
import * as FileSystem from 'expo-file-system/legacy';
import { uploadBreadcrumb } from '../telemetry';

jest.mock('../../api/client', () => ({ api: { presign: jest.fn(), createEntry: jest.fn() }, isApiError: (e: { response?: unknown }) => Boolean(e.response) }));
jest.mock('expo-file-system/next', () => ({ File: jest.fn() }));
jest.mock('expo-file-system/legacy', () => ({ uploadAsync: jest.fn(), FileSystemUploadType: { BINARY_CONTENT: 0 } }));
jest.mock('../telemetry', () => ({ ...jest.requireActual('../telemetry'), uploadBreadcrumb: jest.fn() }));
const presign = api.presign as jest.Mock;
const register = api.createEntry as jest.Mock;
const put = FileSystem.uploadAsync as jest.Mock;
beforeEach(() => {
  jest.useFakeTimers(); jest.resetAllMocks();
  (File as unknown as jest.Mock).mockImplementation(() => ({ exists: true, size: 123 }));
  presign.mockResolvedValue({ upload_url: 'https://fixture.invalid/upload', audio_key: 'owned/clip' });
  put.mockResolvedValue({ status: 200 }); register.mockResolvedValue({ id: 'entry' });
});
afterEach(() => jest.useRealTimers());

describe('recording upload', () => {
  it('uploads native binary then registers size, duration and mode', async () => {
    const progress = jest.fn();
    expect(await uploadRecording('clip.m4a', 3, progress, 'dream')).toEqual({ entry: { id: 'entry' } });
    expect(put).toHaveBeenCalledWith('https://fixture.invalid/upload', 'clip.m4a', expect.objectContaining({ httpMethod: 'PUT', uploadType: 0 }));
    expect(register).toHaveBeenCalledWith({ audio_key: 'owned/clip', audio_size_bytes: 123, duration_sec: 3, mode: 'dream' });
    expect(progress.mock.calls.map(([p]) => p.phase)).toEqual(['presigning', 'uploading', 'uploading', 'registering', 'done']);
  });
  it('defaults to processing mode', async () => {
    await uploadRecording('clip.m4a', 3); expect(register).toHaveBeenCalledWith(expect.objectContaining({ mode: 'processing' }));
  });
  it('rejects a missing local file before contacting the API', async () => {
    (File as unknown as jest.Mock).mockImplementation(() => ({ exists: false }));
    await expect(uploadRecording('missing', 3)).rejects.toThrow('Recording file not found'); expect(presign).not.toHaveBeenCalled();
  });
  it('retries transient storage failure and completes', async () => {
    put.mockResolvedValueOnce({ status: 503, body: 'unavailable' });
    const result = uploadRecording('clip.m4a', 3);
    await jest.runAllTimersAsync(); await expect(result).resolves.toEqual({ entry: { id: 'entry' } });
    expect(put).toHaveBeenCalledTimes(2); expect(register).toHaveBeenCalledTimes(1);
  });
  it('retries only registration, with the same key, once the audio is uploaded', async () => {
    register.mockRejectedValueOnce(Object.assign(new Error('timeout'), { response: { status: 503 } }));
    const result = uploadRecording('clip.m4a', 3);
    await jest.runAllTimersAsync(); await expect(result).resolves.toEqual({ entry: { id: 'entry' } });
    expect(presign).toHaveBeenCalledTimes(1); expect(put).toHaveBeenCalledTimes(1);
    expect(register).toHaveBeenCalledTimes(2);
    expect(register.mock.calls.every(([body]) => body.audio_key === 'owned/clip')).toBe(true);
  });
  it('does not retry API validation failure', async () => {
    presign.mockRejectedValue(Object.assign(new Error('invalid request'), { response: { status: 400 } }));
    await expect(uploadRecording('clip.m4a', 3)).rejects.toThrow('invalid request'); expect(presign).toHaveBeenCalledTimes(1);
  });
  it('stops after three failed uploads and creates no entry', async () => {
    put.mockResolvedValue({ status: 503, body: 'unavailable' });
    const outcome = uploadRecording('clip.m4a', 3).catch(error => error);
    await jest.runAllTimersAsync(); expect(await outcome).toEqual(expect.objectContaining({ message: expect.stringContaining('503') }));
    expect(put).toHaveBeenCalledTimes(3); expect(register).not.toHaveBeenCalled();
  });
});

// N7: resuming from an earlier attempt's key registers it without uploading.
describe('resumed uploads', () => {
  it('registers a resumed key without presigning or uploading', async () => {
    await uploadRecording('clip.m4a', 3, undefined, 'processing', { uploadedKey: 'audio/u/prev.aac' });
    expect(presign).not.toHaveBeenCalled(); expect(put).not.toHaveBeenCalled();
    expect(register).toHaveBeenCalledWith(expect.objectContaining({ audio_key: 'audio/u/prev.aac' }));
  });
  it('reports the key once uploaded and on the error when registration fails', async () => {
    register.mockRejectedValue(Object.assign(new Error('quota'), { response: { status: 409, data: {} } }));
    const onUploaded = jest.fn();
    const err = await uploadRecording('clip.m4a', 3, undefined, 'processing', { onUploaded }).catch(e => e);
    expect(onUploaded).toHaveBeenCalledWith('owned/clip');
    expect(err.uploadedKey).toBe('owned/clip');
  });
  it('re-uploads once when the resumed object expired from storage', async () => {
    register.mockRejectedValueOnce(Object.assign(new Error('missing'), { response: { status: 400, data: { message: 'audio file not found in storage - upload it before creating the entry' } } }));
    const onUploaded = jest.fn();
    const result = uploadRecording('clip.m4a', 3, undefined, 'processing', { uploadedKey: 'audio/u/old.aac', onUploaded });
    await jest.runAllTimersAsync(); await expect(result).resolves.toEqual({ entry: { id: 'entry' } });
    expect(put).toHaveBeenCalledTimes(1); expect(onUploaded).toHaveBeenCalledWith('owned/clip');
    expect(register).toHaveBeenLastCalledWith(expect.objectContaining({ audio_key: 'owned/clip' }));
  });
});

// N17: note photos go through the same native binary PUT as recordings.
describe('putFileToPresignedUrl', () => {
  it('PUTs the file as binary with its content type', async () => {
    const { putFileToPresignedUrl } = jest.requireActual('../upload') as typeof import('../upload');
    put.mockResolvedValue({ status: 200 });
    await putFileToPresignedUrl('https://fixture.invalid/put', 'file:///photo.jpg', 'image/jpeg');
    expect(put).toHaveBeenCalledWith('https://fixture.invalid/put', 'file:///photo.jpg', expect.objectContaining({ httpMethod: 'PUT', uploadType: 0, headers: { 'Content-Type': 'image/jpeg' } }));
  });
  it('throws on a storage error', async () => {
    const { putFileToPresignedUrl } = jest.requireActual('../upload') as typeof import('../upload');
    put.mockResolvedValue({ status: 403 });
    await expect(putFileToPresignedUrl('https://fixture.invalid/put', 'file:///photo.jpg', 'image/png')).rejects.toThrow('403');
  });
});

// Sentry breadcrumbs (category 'upload'): every phase, retry and failure is
// recorded with phase / attempt / status / size bucket / mode only.
describe('upload breadcrumbs', () => {
  const crumbs = () => (uploadBreadcrumb as jest.Mock).mock.calls.map(([message, data, level]) => ({ message, data, level }));
  const messages = () => crumbs().map(c => c.message);
  const expectNoPII = () => {
    const all = JSON.stringify(crumbs());
    for (const secret of ['owned/clip', 'fixture.invalid', 'clip.m4a', 'audio/u/', 'photo.jpg', 'unavailable']) {
      expect(all).not.toContain(secret);
    }
  };

  it('records each phase of a successful upload', async () => {
    await uploadRecording('clip.m4a', 3, undefined, 'dream');
    expect(messages()).toEqual([
      'upload: started', 'upload: presign', 'upload: storage put', 'upload: register entry', 'upload: done',
    ]);
    expect(crumbs()[0].data).toEqual({ mode: 'dream', size: '<100KB', resumed: false });
    expect(crumbs()[2].data).toEqual({ phase: 'put', attempt: 1, size: '<100KB' });
    expectNoPII();
  });

  it('records the storage status and the retry', async () => {
    put.mockResolvedValueOnce({ status: 503, body: 'unavailable' });
    const result = uploadRecording('clip.m4a', 3);
    await jest.runAllTimersAsync(); await result;
    const failed = crumbs().find(c => c.message === 'upload: attempt failed');
    expect(failed).toEqual({ message: 'upload: attempt failed', data: { phase: 'put', attempt: 1, status: 503 }, level: 'warning' });
    expect(crumbs()).toContainEqual({ message: 'upload: retrying', data: { attempt: 2, mode: 'processing' }, level: undefined });
    expect(messages()[messages().length - 1]).toBe('upload: done');
    expectNoPII();
  });

  it('records a non-retryable API failure with its status and phase', async () => {
    register.mockRejectedValue(Object.assign(new Error('quota'), { response: { status: 409, data: {} } }));
    await uploadRecording('clip.m4a', 3).catch(() => undefined);
    expect(crumbs()).toContainEqual({ message: 'upload: not retryable', data: { phase: 'register', attempt: 1, status: 409 }, level: 'error' });
    expectNoPII();
  });

  it('records exhausting all retries', async () => {
    put.mockResolvedValue({ status: 500, body: 'unavailable' });
    const outcome = uploadRecording('clip.m4a', 3).catch(() => undefined);
    await jest.runAllTimersAsync(); await outcome;
    expect(crumbs().filter(c => c.message === 'upload: attempt failed')).toHaveLength(3);
    expect(crumbs()[crumbs().length - 1]).toEqual({ message: 'upload: failed after retries', data: { attempt: 3, resumed: false }, level: 'error' });
  });

  it('records a resumed key, and its re-upload when the object expired', async () => {
    register.mockRejectedValueOnce(Object.assign(new Error('missing'), { response: { status: 400, data: { message: 'audio file not found in storage' } } }));
    const result = uploadRecording('clip.m4a', 3, undefined, 'processing', { uploadedKey: 'audio/u/old.aac' });
    await jest.runAllTimersAsync(); await result;
    expect(crumbs()[0].data).toEqual(expect.objectContaining({ resumed: true }));
    expect(messages()).toContain('upload: reusing audio already in storage');
    expect(messages()).toContain('upload: resumed audio missing from storage, uploading again');
    expectNoPII();
  });

  it('records a missing recording file', async () => {
    (File as unknown as jest.Mock).mockImplementation(() => ({ exists: false }));
    await uploadRecording('clip.m4a', 3).catch(() => undefined);
    expect(crumbs()).toEqual([{ message: 'upload: recording file missing', data: { mode: 'processing' }, level: 'error' }]);
  });

  it('records note photo PUTs, including the failure status', async () => {
    const { putFileToPresignedUrl } = jest.requireActual('../upload') as typeof import('../upload');
    put.mockResolvedValueOnce({ status: 200 });
    await putFileToPresignedUrl('https://fixture.invalid/put', 'file:///photo.jpg', 'image/jpeg');
    put.mockResolvedValueOnce({ status: 403 });
    await putFileToPresignedUrl('https://fixture.invalid/put', 'file:///photo.jpg', 'image/jpeg').catch(() => undefined);
    put.mockRejectedValueOnce(new Error('network'));
    await putFileToPresignedUrl('https://fixture.invalid/put', 'file:///photo.jpg', 'image/jpeg').catch(() => undefined);
    expect(crumbs()).toContainEqual({ message: 'upload: note photo put done', data: { phase: 'note_put', status: 200 }, level: undefined });
    expect(crumbs()).toContainEqual({ message: 'upload: note photo put failed', data: { phase: 'note_put', status: 403 }, level: 'error' });
    expect(crumbs()).toContainEqual({ message: 'upload: note photo put failed', data: { phase: 'note_put' }, level: 'error' });
    expectNoPII();
  });

  it('uploads normally with the real (Sentry-disabled) telemetry', async () => {
    (uploadBreadcrumb as jest.Mock).mockImplementation((...args: Parameters<typeof uploadBreadcrumb>) =>
      (jest.requireActual('../telemetry') as typeof import('../telemetry')).uploadBreadcrumb(...args));
    await expect(uploadRecording('clip.m4a', 3)).resolves.toEqual({ entry: { id: 'entry' } });
  });
});
