import React from 'react';
import { Alert } from 'react-native';
import { act, fireEvent, render } from '@testing-library/react-native';
import TextJournalComposer from '../TextJournalComposer';
import { api } from '../../api/client';
import NetInfo from '@react-native-community/netinfo';
const saved = jest.fn(), cancel = jest.fn(), voice = jest.fn();
jest.mock('../../context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../../theme').THEMES.curious }) }));
jest.mock('../../api/client', () => ({ api: { createTextEntry: jest.fn() } }));
const props = { mode: 'processing' as const, modes: [{ key: 'processing' as const, label: 'Process' }], onSelectMode: jest.fn(), onVoice: voice, onCancel: cancel, onSaved: saved, enabled: true };
beforeEach(() => {
  jest.clearAllMocks(); (api.createTextEntry as jest.Mock).mockReset().mockResolvedValue({ id: 'typed-entry' });
  (NetInfo.fetch as jest.Mock).mockResolvedValue({ isConnected: true });
  jest.spyOn(Alert, 'alert').mockImplementation(() => {});
});
afterEach(() => jest.restoreAllMocks());

it('explains the complete-journal message and three follow-up exchanges, and rejects empty/missing-mode input', () => {
  const view = render(<TextJournalComposer {...props} mode={null} />);
  expect(view.getByText(/Write your whole journal in one message/)).toBeTruthy();
  expect(view.getByTestId('text-journal-send')).toBeDisabled();
  fireEvent.changeText(view.getByTestId('text-journal-input'), '  ');
  fireEvent.press(view.getByTestId('text-journal-send'));
  expect(api.createTextEntry).not.toHaveBeenCalled();
  expect(view.getByTestId('text-journal-input').props.maxLength).toBe(20000);
});

it('sends a trimmed complete journal once and opens its reflection processing flow', async () => {
  const view = render(<TextJournalComposer {...props} />);
  fireEvent.changeText(view.getByTestId('text-journal-input'), ' My whole journal. ');
  await act(async () => { fireEvent.press(view.getByTestId('text-journal-send')); fireEvent.press(view.getByTestId('text-journal-send')); });
  expect(api.createTextEntry).toHaveBeenCalledTimes(1);
  expect(api.createTextEntry).toHaveBeenCalledWith({ content: 'My whole journal.', request_id: expect.any(String), mode: 'processing' });
  expect(saved).toHaveBeenCalledWith('typed-entry');
});

it('keeps failed text and reuses the retry token without creating a second journal', async () => {
  (api.createTextEntry as jest.Mock).mockRejectedValueOnce(new Error('lost response'));
  const view = render(<TextJournalComposer {...props} />);
  fireEvent.changeText(view.getByTestId('text-journal-input'), 'Keep this journal');
  await act(async () => { fireEvent.press(view.getByTestId('text-journal-send')); });
  expect(view.getByTestId('text-journal-input').props.value).toBe('Keep this journal');
  expect(view.getByTestId('text-journal-error')).toBeTruthy();
  await act(async () => { fireEvent.press(view.getByTestId('text-journal-send')); });
  const calls = (api.createTextEntry as jest.Mock).mock.calls;
  expect(calls[0][0].request_id).toBe(calls[1][0].request_id);
});

it('preserves the draft offline and never calls the entry API', async () => {
  (NetInfo.fetch as jest.Mock).mockResolvedValue({ isConnected: false });
  const view = render(<TextJournalComposer {...props} />);
  fireEvent.changeText(view.getByTestId('text-journal-input'), 'Offline writing');
  await act(async () => { fireEvent.press(view.getByTestId('text-journal-send')); });
  expect(api.createTextEntry).not.toHaveBeenCalled();
  expect(view.getByText(/You’re offline/)).toBeTruthy();
  expect(view.getByTestId('text-journal-input').props.value).toBe('Offline writing');
});

it('asks before discarding a draft or switching to voice', () => {
  const view = render(<TextJournalComposer {...props} />);
  fireEvent.changeText(view.getByTestId('text-journal-input'), 'Unsent');
  fireEvent.press(view.getByTestId('record-voice-mode'));
  expect(voice).not.toHaveBeenCalled();
  const actions = (Alert.alert as jest.Mock).mock.calls.at(-1)[2];
  expect(actions[0].text).toBe('Keep writing'); actions[1].onPress();
  expect(voice).toHaveBeenCalledTimes(1);
});

it('ignores a late save after unmount or sign-out', async () => {
  let resolve!: (value: unknown) => void;
  (api.createTextEntry as jest.Mock).mockReturnValue(new Promise(done => { resolve = done; }));
  const view = render(<TextJournalComposer {...props} />);
  fireEvent.changeText(view.getByTestId('text-journal-input'), 'Private');
  await act(async () => { fireEvent.press(view.getByTestId('text-journal-send')); });
  view.rerender(<TextJournalComposer {...props} enabled={false} />);
  expect(view.getByTestId('text-journal-input').props.value).toBe('');
  view.rerender(<TextJournalComposer {...props} enabled />);
  await act(async () => { resolve({ id: 'private-entry' }); });
  expect(saved).not.toHaveBeenCalled();
});
