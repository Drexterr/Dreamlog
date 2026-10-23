import React from 'react';
import { Alert } from 'react-native';
import { act, fireEvent, render } from '@testing-library/react-native';
import TherapySessionScreen from '../app/therapy/session';
import { api } from '../src/api/client';
import { T } from '../src/testIDs';
const mockRecorder = { state: 'idle', durationMs: 0, error: null, startRecording: jest.fn(), stopRecording: jest.fn(), discardRecording: jest.fn() };
jest.mock('expo-router', () => ({ useRouter: () => ({ replace: jest.fn(), back: jest.fn() }), useLocalSearchParams: () => ({ id: 'session' }) }));
jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../src/theme').THEMES.curious }) }));
jest.mock('../src/services/region', () => ({ detectUserCountry: jest.fn().mockResolvedValue('IN') }));
jest.mock('../src/hooks/useRecorder', () => ({ useRecorder: () => mockRecorder }));
jest.mock('../src/components/BrandOrbGlyph', () => () => null);
jest.mock('../src/api/client', () => ({ api: { getTherapySession: jest.fn(), sendTherapyMessage: jest.fn(), endTherapySession: jest.fn() } }));
const reply = { user_message: { id: 'u', role: 'user', input_mode: 'text', content: 'How I feel' }, assistant_message: { id: 'a', role: 'assistant', input_mode: 'text', content: 'A thoughtful reply' }, session_state: { status: 'active', time_remaining_sec: 500, crisis_warnings: 0 } };
beforeEach(() => {
  jest.useFakeTimers(); jest.clearAllMocks(); mockRecorder.state = 'idle';
  (api.getTherapySession as jest.Mock).mockResolvedValue({ status: 'active', time_remaining_sec: 600, messages: [] });
  (api.sendTherapyMessage as jest.Mock).mockReset().mockResolvedValue(reply);
  jest.spyOn(Alert, 'alert').mockImplementation(() => {});
});
afterEach(() => { jest.clearAllTimers(); jest.useRealTimers(); jest.restoreAllMocks(); });
async function screen() { const view = render(<TherapySessionScreen />); await act(async () => {}); return view; }

it('keeps therapy in chat after sending and allows an explicit return to voice', async () => {
  const view = await screen(); fireEvent.press(view.getByTestId(T.therapy.chatButton));
  fireEvent.changeText(view.getByTestId(T.therapy.textInput), ' How I feel ');
  await act(async () => { fireEvent.press(view.getByTestId(T.therapy.sendButton)); fireEvent.press(view.getByTestId(T.therapy.sendButton)); });
  expect(api.sendTherapyMessage).toHaveBeenCalledTimes(1);
  expect(api.sendTherapyMessage).toHaveBeenCalledWith('session', { content: 'How I feel', input_mode: 'text', request_id: expect.stringMatching(/^[0-9a-f-]{36}$/) });
  expect(view.getByText('A thoughtful reply')).toBeTruthy();
  expect(view.getByTestId(T.therapy.textInput)).toBeTruthy();
  fireEvent.press(view.getByTestId(T.therapy.voiceButton));
  expect(view.getByTestId(T.therapy.chatButton)).toBeTruthy();
});

it('restores a failed message for retry', async () => {
  (api.sendTherapyMessage as jest.Mock).mockRejectedValueOnce(new Error('offline'));
  const view = await screen(); fireEvent.press(view.getByTestId(T.therapy.chatButton));
  fireEvent.changeText(view.getByTestId(T.therapy.textInput), 'Keep my message');
  await act(async () => { fireEvent.press(view.getByTestId(T.therapy.sendButton)); });
  expect(view.getByTestId(T.therapy.textInput).props.value).toBe('Keep my message');
});

it('does not hide an active microphone behind the chat screen', async () => {
  mockRecorder.state = 'recording'; const view = await screen();
  expect(view.getByTestId(T.therapy.chatButton)).toBeDisabled();
  fireEvent.press(view.getByTestId(T.therapy.chatButton));
  expect(view.queryByTestId(T.therapy.textInput)).toBeNull();
});

// N4: resending the same text reuses its request_id so the server can return
// the turn it already finished instead of running it twice.
it('reuses the request id when the same message is resent', async () => {
  (api.sendTherapyMessage as jest.Mock).mockRejectedValueOnce(new Error('timeout'));
  const view = await screen(); fireEvent.press(view.getByTestId(T.therapy.chatButton));
  fireEvent.changeText(view.getByTestId(T.therapy.textInput), 'Same words');
  await act(async () => { fireEvent.press(view.getByTestId(T.therapy.sendButton)); });
  await act(async () => { fireEvent.press(view.getByTestId(T.therapy.sendButton)); });
  const calls = (api.sendTherapyMessage as jest.Mock).mock.calls;
  expect(calls).toHaveLength(2);
  expect(calls[1][1].request_id).toBe(calls[0][1].request_id);
});

it('uses a new request id for a different message', async () => {
  const view = await screen(); fireEvent.press(view.getByTestId(T.therapy.chatButton));
  fireEvent.changeText(view.getByTestId(T.therapy.textInput), 'First');
  await act(async () => { fireEvent.press(view.getByTestId(T.therapy.sendButton)); });
  fireEvent.changeText(view.getByTestId(T.therapy.textInput), 'First');
  await act(async () => { fireEvent.press(view.getByTestId(T.therapy.sendButton)); });
  const calls = (api.sendTherapyMessage as jest.Mock).mock.calls;
  expect(calls[1][1].request_id).not.toBe(calls[0][1].request_id);
});

// N4: after a timeout the session is re-fetched; a turn the server stored is
// shown and the draft cleared instead of inviting a duplicate resend.
it('re-fetches the session after a timeout and clears a stored draft', async () => {
  (api.sendTherapyMessage as jest.Mock).mockRejectedValueOnce(new Error('timeout'));
  const view = await screen();
  (api.getTherapySession as jest.Mock).mockResolvedValue({ status: 'active', time_remaining_sec: 500, messages: [reply.user_message, reply.assistant_message] });
  fireEvent.press(view.getByTestId(T.therapy.chatButton));
  fireEvent.changeText(view.getByTestId(T.therapy.textInput), 'How I feel');
  await act(async () => { fireEvent.press(view.getByTestId(T.therapy.sendButton)); });
  await act(async () => {});
  expect(api.getTherapySession).toHaveBeenCalledTimes(2);
  expect(view.getByText('A thoughtful reply')).toBeTruthy();
  expect(view.getByTestId(T.therapy.textInput).props.value).toBe('');
});
