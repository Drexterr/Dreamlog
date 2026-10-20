import React from 'react';
import { act, fireEvent, render } from '@testing-library/react-native';
import { Alert, Animated } from 'react-native';
import RecordScreen from '../app/record';
import { uploadRecording } from '../src/services/upload';
import { enqueue } from '../src/services/offlineQueue';
import NetInfo from '@react-native-community/netinfo';
import { api } from '../src/api/client';

const mockBack = jest.fn();
const mockReplace = jest.fn();
const mockRecorder = { state: 'idle', durationMs: 0, error: null as string | null,
  startRecording: jest.fn(), stopRecording: jest.fn(), discardRecording: jest.fn() };
jest.mock('expo-router', () => ({ useRouter: () => ({ back: mockBack, replace: mockReplace }), useLocalSearchParams: () => ({}) }));
jest.mock('../src/hooks/useRecorder', () => ({ useRecorder: () => mockRecorder }));
jest.mock('../src/context/AuthContext', () => ({ useAuth: () => ({ isAuthenticated: true, requestAuth: jest.fn() }) }));
jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../src/theme').THEMES.curious }) }));
jest.mock('../src/components/BrandOrbGlyph', () => () => null);
jest.mock('../src/api/client', () => ({ api: { createTextEntry: jest.fn() } }));
jest.mock('../src/services/upload', () => ({ uploadRecording: jest.fn() }));
jest.mock('../src/services/offlineQueue', () => ({ enqueue: jest.fn() }));
jest.mock('@react-native-community/netinfo', () => ({ fetch: jest.fn() }));

beforeEach(() => {
  jest.clearAllMocks(); mockRecorder.error = null;
  mockRecorder.startRecording.mockResolvedValue(true);
  mockRecorder.stopRecording.mockResolvedValue({ uri: 'qa.m4a', durationSec: 3, sizeBytes: 10 });
  mockRecorder.discardRecording.mockResolvedValue(undefined);
  (NetInfo.fetch as jest.Mock).mockResolvedValue({ isConnected: true });
  (uploadRecording as jest.Mock).mockResolvedValue({ entry: { id: 'qa-entry' } });
  (enqueue as jest.Mock).mockResolvedValue(undefined);
  jest.spyOn(Animated, 'loop').mockReturnValue({ start: jest.fn(), stop: jest.fn(), reset: jest.fn() });
  jest.spyOn(Alert, 'alert').mockImplementation(() => {});
});
afterEach(() => jest.restoreAllMocks());

async function start(view: ReturnType<typeof render>, mode = 'processing') {
  fireEvent.press(view.getByTestId('record-mode-' + mode));
  await act(async () => { fireEvent.press(view.getByTestId('record-orb')); });
}

it('does not start without a mode', async () => {
  const view = render(<RecordScreen />);
  await act(async () => { fireEvent.press(view.getByTestId('record-orb')); });
  expect(mockRecorder.startRecording).not.toHaveBeenCalled();
});
it('does not claim listening after microphone denial', async () => {
  mockRecorder.error = 'Microphone permission denied'; mockRecorder.startRecording.mockResolvedValue(false);
  const view = render(<RecordScreen />); await start(view);
  expect(view.queryByText('listening')).toBeNull(); expect(view.getByText('tap to begin')).toBeTruthy();
  expect(view.getByText('Microphone permission denied')).toBeTruthy();
});
it('does not claim listening after native capture fails', async () => {
  mockRecorder.startRecording.mockResolvedValue(false);
  const view = render(<RecordScreen />); await start(view);
  expect(view.queryByText('listening')).toBeNull();
});
it('guards repeated start presses while native capture is pending', async () => {
  let finish!: (started: boolean) => void;
  mockRecorder.startRecording.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
  const view = render(<RecordScreen />); fireEvent.press(view.getByTestId('record-mode-processing'));
  await act(async () => { fireEvent.press(view.getByTestId('record-orb')); fireEvent.press(view.getByTestId('record-orb')); });
  expect(mockRecorder.startRecording).toHaveBeenCalledTimes(1);
  await act(async () => { finish(true); }); expect(view.getByText('listening')).toBeTruthy();
});
it.each(['processing', 'rant', 'gratitude', 'decision', 'dream'])('uploads the selected %s mode', async mode => {
  const view = render(<RecordScreen />); await start(view, mode);
  await act(async () => { fireEvent.press(view.getByTestId('record-orb')); });
  expect(uploadRecording).toHaveBeenCalledWith('qa.m4a', 3, expect.any(Function), mode);
  expect(mockReplace).toHaveBeenCalledWith('/processing/qa-entry');
});
it('queues offline audio and returns only after acknowledgment', async () => {
  (NetInfo.fetch as jest.Mock).mockResolvedValue({ isConnected: false });
  const view = render(<RecordScreen />); await start(view);
  await act(async () => { fireEvent.press(view.getByTestId('record-orb')); });
  expect(enqueue).toHaveBeenCalledWith(expect.objectContaining({ localUri: 'qa.m4a', mode: 'processing' }));
  expect(uploadRecording).not.toHaveBeenCalled(); expect(mockBack).not.toHaveBeenCalled();
  const buttons = (Alert.alert as jest.Mock).mock.calls[0][2]; buttons[0].onPress(); expect(mockBack).toHaveBeenCalled();
});
it('cancels without upload', async () => {
  const view = render(<RecordScreen />); await start(view);
  await act(async () => { fireEvent.press(view.getByTestId('record-cancel')); });
  expect(mockRecorder.discardRecording).toHaveBeenCalled(); expect(mockBack).toHaveBeenCalled(); expect(uploadRecording).not.toHaveBeenCalled();
});

it('opens typed journaling, preserves the chosen mode, and processes a saved entry without recording', async () => {
  (api.createTextEntry as jest.Mock).mockResolvedValue({ id: 'typed' });
  const view = render(<RecordScreen />);
  fireEvent.press(view.getByTestId('record-mode-gratitude'));
  fireEvent.press(view.getByTestId('record-write-mode'));
  fireEvent.changeText(view.getByTestId('text-journal-input'), 'My gratitude journal');
  await act(async () => { fireEvent.press(view.getByTestId('text-journal-send')); });
  expect(api.createTextEntry).toHaveBeenCalledWith(expect.objectContaining({ mode: 'gratitude', content: 'My gratitude journal' }));
  expect(mockReplace).toHaveBeenCalledWith({ pathname: '/processing/[id]', params: { id: 'typed' } });
  expect(mockRecorder.startRecording).not.toHaveBeenCalled();
});
