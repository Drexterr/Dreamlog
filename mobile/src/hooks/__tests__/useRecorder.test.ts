import { act, renderHook } from '@testing-library/react-native';
import { Audio } from 'expo-av';
import { useRecorder } from '../useRecorder';

jest.mock('expo-av', () => ({ Audio: {
  requestPermissionsAsync: jest.fn(), setAudioModeAsync: jest.fn(),
  Recording: { createAsync: jest.fn() },
  AndroidOutputFormat: { MPEG_4: 1 }, AndroidAudioEncoder: { AAC: 1 },
  IOSOutputFormat: { MPEG4AAC: 1 }, IOSAudioQuality: { MEDIUM: 1 },
} }));

const permission = Audio.requestPermissionsAsync as jest.Mock;
const create = Audio.Recording.createAsync as jest.Mock;
const audioMode = Audio.setAudioModeAsync as jest.Mock;
let recording: { stopAndUnloadAsync: jest.Mock; getStatusAsync: jest.Mock; getURI: jest.Mock };
beforeEach(() => {
  jest.useFakeTimers(); jest.clearAllMocks();
  recording = {
    stopAndUnloadAsync: jest.fn().mockResolvedValue(undefined),
    getStatusAsync: jest.fn().mockResolvedValue({ durationMillis: 3250 }),
    getURI: jest.fn().mockReturnValue('clip.m4a'),
  };
  permission.mockResolvedValue({ status: 'granted' });
  create.mockResolvedValue({ recording }); audioMode.mockResolvedValue(undefined);
});
afterEach(() => { jest.useRealTimers(); });

describe('recorder lifecycle', () => {
  it('starts idle', () => { const { result } = renderHook(useRecorder); expect(result.current.state).toBe('idle'); });
  it('does not create recording when permission is denied', async () => {
    permission.mockResolvedValue({ status: 'denied' });
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    expect(result.current.state).toBe('error'); expect(create).not.toHaveBeenCalled();
    expect(result.current.error).toBe('Microphone permission denied');
  });
  it('creates AAC mono audio and updates the duration', async () => {
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    expect(create).toHaveBeenCalledWith(expect.objectContaining({ android: expect.objectContaining({ sampleRate: 44100, numberOfChannels: 1, extension: '.m4a' }) }));
    act(() => jest.advanceTimersByTime(2000)); expect(result.current.durationMs).toBe(2000);
  });
  it('reports native duration before unloading audio', async () => {
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    let clip;
    await act(async () => { clip = await result.current.stopRecording(); });
    expect(clip).toEqual({ uri: 'clip.m4a', durationSec: 3.25, sizeBytes: 0 });
    expect(recording.getStatusAsync.mock.invocationCallOrder[0]).toBeLessThan(recording.stopAndUnloadAsync.mock.invocationCallOrder[0]);
    expect(audioMode).toHaveBeenLastCalledWith({ allowsRecordingIOS: false });
  });
  it('falls back to timer duration when native duration is zero', async () => {
    recording.getStatusAsync.mockResolvedValue({ durationMillis: 0 });
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    act(() => jest.advanceTimersByTime(2000));
    let clip;
    await act(async () => { clip = await result.current.stopRecording(); });
    expect(clip).toEqual(expect.objectContaining({ durationSec: 2 }));
  });
  it('returns null when stopped before starting', async () => {
    const { result } = renderHook(useRecorder); expect(await result.current.stopRecording()).toBeNull();
  });
  it('handles missing recorded URI', async () => {
    recording.getURI.mockReturnValue(null);
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); await result.current.stopRecording(); });
    expect(result.current.state).toBe('error'); expect(result.current.error).toBe('Recording URI not available');
  });
  it('handles native recording creation failure', async () => {
    create.mockRejectedValue(new Error('native capture unavailable'));
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    expect(result.current.error).toBe('native capture unavailable');
  });
  it('discards and clears duration', async () => {
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    act(() => jest.advanceTimersByTime(2000));
    await act(async () => { await result.current.discardRecording(); });
    expect(result.current.state).toBe('idle'); expect(result.current.durationMs).toBe(0);
    act(() => jest.advanceTimersByTime(2000));
    expect(result.current.durationMs).toBe(0);
  });
  it('unloads audio and clears timers on unmount', async () => {
    const clearTick = jest.spyOn(global, 'clearInterval');
    const clearMax = jest.spyOn(global, 'clearTimeout');
    const { result, unmount } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    unmount(); expect(recording.stopAndUnloadAsync).toHaveBeenCalledTimes(1);
    expect(clearTick).toHaveBeenCalled(); expect(clearMax).toHaveBeenCalled();
    clearTick.mockRestore(); clearMax.mockRestore();
  });
  it('auto-stops at 30 minutes', async () => {
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    await act(async () => { await jest.advanceTimersByTimeAsync(30 * 60 * 1000); });
    expect(result.current.state).toBe('stopped'); expect(recording.stopAndUnloadAsync).toHaveBeenCalledTimes(1);
  });
  it('can submit the recording after automatic stop without unloading it twice', async () => {
    recording.getStatusAsync.mockImplementation(async () => recording.stopAndUnloadAsync.mock.calls.length ? { durationMillis: 0 } : { durationMillis: 1800000 });
    recording.stopAndUnloadAsync.mockImplementation(async () => {
      if (recording.stopAndUnloadAsync.mock.calls.length > 1) throw new Error('already unloaded');
    });
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    await act(async () => { await jest.advanceTimersByTimeAsync(30 * 60 * 1000); });
    let clip;
    await act(async () => { clip = await result.current.stopRecording(); });
    expect(clip).toEqual(expect.objectContaining({ uri: 'clip.m4a', durationSec: 1800 }));
    expect(recording.stopAndUnloadAsync).toHaveBeenCalledTimes(1);
  });
  it('restores playback audio mode after discarding a recording', async () => {
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    await act(async () => { await result.current.discardRecording(); });
    expect(audioMode).toHaveBeenLastCalledWith({ allowsRecordingIOS: false });
  });
  it('restores playback mode even when native discard fails', async () => {
    recording.stopAndUnloadAsync.mockRejectedValue(new Error('already unloaded'));
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    await act(async () => { await result.current.discardRecording(); });
    expect(result.current.state).toBe('idle');
    expect(audioMode).toHaveBeenLastCalledWith({ allowsRecordingIOS: false });
  });
  it('restores playback mode when recording creation fails', async () => {
    create.mockRejectedValue(new Error('capture unavailable'));
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    expect(result.current.error).toBe('capture unavailable');
    expect(audioMode).toHaveBeenLastCalledWith({ allowsRecordingIOS: false });
  });
  it('restores playback mode when the stopped recording has no URI', async () => {
    recording.getURI.mockReturnValue(null);
    const { result } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    await act(async () => { await result.current.stopRecording(); });
    expect(result.current.error).toBe('Recording URI not available');
    expect(audioMode).toHaveBeenLastCalledWith({ allowsRecordingIOS: false });
  });
  it('restores playback mode when an active recorder unmounts', async () => {
    const { result, unmount } = renderHook(useRecorder);
    await act(async () => { await result.current.startRecording(); });
    await act(async () => { unmount(); });
    expect(audioMode).toHaveBeenLastCalledWith({ allowsRecordingIOS: false });
  });
});
