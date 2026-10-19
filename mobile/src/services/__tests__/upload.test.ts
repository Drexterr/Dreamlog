import { uploadRecording } from '../upload';
import { api } from '../../api/client';
import { File } from 'expo-file-system/next';
import * as FileSystem from 'expo-file-system/legacy';

jest.mock('../../api/client', () => ({ api: { presign: jest.fn(), createEntry: jest.fn() }, isApiError: (e: { response?: unknown }) => Boolean(e.response) }));
jest.mock('expo-file-system/next', () => ({ File: jest.fn() }));
jest.mock('expo-file-system/legacy', () => ({ uploadAsync: jest.fn(), FileSystemUploadType: { BINARY_CONTENT: 0 } }));
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
