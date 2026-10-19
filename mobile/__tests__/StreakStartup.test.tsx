import React from 'react';
import { Text, TouchableOpacity } from 'react-native';
import { act, fireEvent, render } from '@testing-library/react-native';
import { StreakProvider, useStreak } from '../src/context/StreakContext';
import { api } from '../src/api/client';
jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../src/theme').THEMES.curious }) }));
jest.mock('../src/api/client', () => ({ api: { streak: jest.fn(), useStreakFreeze: jest.fn() } }));
const broken = { current_streak: 0, longest_streak: 12, total_days: 20, next_milestone: 7, freeze_count: 2 };
const active = { ...broken, current_streak: 3, freeze_count: 1 };
function Controls({ tab = 'Home' }: { tab?: string }) {
  const { streak, openRenewal, refreshStreak } = useStreak();
  return <><Text>{tab}</Text><Text testID="current-streak">{streak?.current_streak ?? 'loading'}</Text><TouchableOpacity testID="manual-renew" onPress={openRenewal}><Text>Renew streak</Text></TouchableOpacity><TouchableOpacity testID="refresh" onPress={refreshStreak}><Text>Refresh</Text></TouchableOpacity></>;
}
function App({ ready = true, enabled = true, tab = 'Home' }: { ready?: boolean; enabled?: boolean; tab?: string }) {
  return <StreakProvider enabled={enabled} startupReady={ready}><Controls tab={tab} /></StreakProvider>;
}
async function screen(props = {}) { const view = render(<App {...props} />); await act(async () => {}); return view; }
beforeEach(() => {
  jest.clearAllMocks();
  (api.streak as jest.Mock).mockReset().mockResolvedValue(broken);
  (api.useStreakFreeze as jest.Mock).mockReset().mockResolvedValue({ freeze_count: 1, freeze_date: '2026-10-18' });
});

it('shows once at startup on Home, waits for overlays, and does not repeat after tab visits or resume', async () => {
  const view = await screen({ ready: false });
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
  view.rerender(<App ready />); await act(async () => {});
  expect(view.getByTestId('streak-renewal-dialog')).toBeTruthy();
  expect(view.getByText('2 freezes available')).toBeTruthy();
  fireEvent.press(view.getByTestId('renew-streak-dismiss'));
  view.rerender(<App ready tab="Mood" />); await act(async () => {});
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
  view.rerender(<App ready={false} tab="Explore" />);
  view.rerender(<App ready tab="Home" />); await act(async () => {});
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
  fireEvent.press(view.getByTestId('manual-renew'));
  expect(view.getByTestId('streak-renewal-dialog')).toBeTruthy();
  view.unmount();
  const next = await screen();
  expect(next.getByTestId('streak-renewal-dialog')).toBeTruthy();
});

it.each([['active', active], ['new', { ...broken, total_days: 0 }]])('does not prompt a %s user', async (_, streak) => {
  (api.streak as jest.Mock).mockResolvedValue(streak);
  const view = await screen(); expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
});

it('does not fetch as a guest and clears private state and dialogs on sign-out', async () => {
  const view = await screen({ enabled: false }); expect(api.streak).not.toHaveBeenCalled();
  view.rerender(<App enabled />); await act(async () => {});
  expect(view.getByTestId('streak-renewal-dialog')).toBeTruthy();
  view.rerender(<App enabled={false} />); await act(async () => {});
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
  expect(view.getByTestId('current-streak').props.children).toBe('loading');
  view.rerender(<App enabled />); await act(async () => {});
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
  fireEvent.press(view.getByTestId('manual-renew'));
  expect(view.getByTestId('streak-renewal-dialog')).toBeTruthy();
});

it('disables renewal with no freezes or an active streak', async () => {
  (api.streak as jest.Mock).mockResolvedValue({ ...broken, freeze_count: 0 });
  const view = await screen();
  expect(view.getByTestId('renew-streak-confirm')).toBeDisabled();
  fireEvent.press(view.getByTestId('renew-streak-confirm'));
  expect(api.useStreakFreeze).not.toHaveBeenCalled();
  view.unmount(); (api.streak as jest.Mock).mockResolvedValue(active);
  const next = await screen(); fireEvent.press(next.getByTestId('manual-renew'));
  expect(next.getByTestId('renew-streak-confirm')).toBeDisabled();
});

it('consumes one freeze with duplicate taps, applies the UTC date, and refreshes the shared streak', async () => {
  const view = await screen();
  let resolve!: (value: unknown) => void;
  (api.useStreakFreeze as jest.Mock).mockReturnValue(new Promise(done => { resolve = done; }));
  (api.streak as jest.Mock).mockResolvedValue(active);
  const yesterday = new Date(); yesterday.setUTCDate(yesterday.getUTCDate() - 1);
  act(() => { fireEvent.press(view.getByTestId('renew-streak-confirm')); fireEvent.press(view.getByTestId('renew-streak-confirm')); });
  expect(api.useStreakFreeze).toHaveBeenCalledTimes(1);
  expect(api.useStreakFreeze).toHaveBeenCalledWith(yesterday.toISOString().slice(0, 10));
  expect(view.getByTestId('renew-streak-dismiss')).toBeDisabled();
  await act(async () => { resolve({ freeze_count: 1 }); });
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
  expect(view.getByTestId('current-streak').props.children).toBe(3);
});

it('retains an acknowledged freeze after refresh failure and after dismissal without consuming twice', async () => {
  const view = await screen();
  (api.streak as jest.Mock).mockRejectedValueOnce(new Error('offline')).mockResolvedValue(active);
  await act(async () => { fireEvent.press(view.getByTestId('renew-streak-confirm')); });
  expect(view.getByText(/another freeze will not be used/)).toBeTruthy();
  expect(view.getByText('1 freeze available')).toBeTruthy();
  fireEvent.press(view.getByTestId('renew-streak-dismiss'));
  fireEvent.press(view.getByTestId('manual-renew'));
  expect(view.getByText('Refresh streak')).toBeTruthy();
  await act(async () => { fireEvent.press(view.getByTestId('renew-streak-confirm')); });
  expect(api.useStreakFreeze).toHaveBeenCalledTimes(1);
  expect(view.getByTestId('current-streak').props.children).toBe(3);
});

it('allows retry when applying the freeze fails and keeps the available count', async () => {
  const view = await screen();
  (api.useStreakFreeze as jest.Mock).mockRejectedValueOnce(new Error('offline'));
  await act(async () => { fireEvent.press(view.getByTestId('renew-streak-confirm')); });
  expect(view.getByText('Could not renew your streak. Try again.')).toBeTruthy();
  expect(view.getByText('2 freezes available')).toBeTruthy();
  (api.streak as jest.Mock).mockResolvedValue(active);
  await act(async () => { fireEvent.press(view.getByTestId('renew-streak-confirm')); });
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
});

it('ignores stale reads after sign-out', async () => {
  let resolve!: (value: unknown) => void;
  (api.streak as jest.Mock).mockReturnValue(new Promise(done => { resolve = done; }));
  const view = await screen(); view.rerender(<App enabled={false} />);
  await act(async () => { resolve(broken); });
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
  expect(view.getByTestId('current-streak').props.children).toBe('loading');
});

it('ignores a mutation acknowledgment after sign-out without reading another account', async () => {
  const view = await screen();
  let resolve!: (value: unknown) => void;
  (api.useStreakFreeze as jest.Mock).mockReturnValue(new Promise(done => { resolve = done; }));
  fireEvent.press(view.getByTestId('renew-streak-confirm'));
  view.rerender(<App enabled={false} />);
  await act(async () => { resolve({ freeze_count: 1 }); });
  expect(api.streak).toHaveBeenCalledTimes(1);
  expect(view.getByTestId('current-streak').props.children).toBe('loading');
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
});

it('prevents a stale background read from overwriting a successful renewal', async () => {
  const view = await screen();
  let resolve!: (value: unknown) => void;
  (api.streak as jest.Mock).mockReturnValueOnce(new Promise(done => { resolve = done; })).mockResolvedValue(active);
  fireEvent.press(view.getByTestId('refresh'));
  await act(async () => { fireEvent.press(view.getByTestId('renew-streak-confirm')); });
  expect(view.getByTestId('current-streak').props.children).toBe(3);
  await act(async () => { resolve(broken); });
  expect(view.getByTestId('current-streak').props.children).toBe(3);
});

it('does not prompt mid-session when a previously active streak is lost', async () => {
  (api.streak as jest.Mock).mockResolvedValueOnce(active).mockResolvedValue(broken);
  const view = await screen();
  await act(async () => { fireEvent.press(view.getByTestId('refresh')); });
  expect(view.getByTestId('current-streak').props.children).toBe(0);
  expect(view.queryByTestId('streak-renewal-dialog')).toBeNull();
  fireEvent.press(view.getByTestId('manual-renew'));
  expect(view.getByTestId('streak-renewal-dialog')).toBeTruthy();
});
