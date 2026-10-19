import React from 'react';
import { ScrollView } from 'react-native';
import { act, fireEvent, render } from '@testing-library/react-native';
import MoodScreen from '../app/(tabs)/mood';
import { api } from '../src/api/client';
const mockNavigate = jest.fn();
const mockOpenRenewal = jest.fn();
let mockStreak = { current_streak: 3, longest_streak: 8, total_days: 12, freeze_count: 2, next_milestone: 7 };
jest.mock('expo-router', () => ({
  useRouter: () => ({ navigate: mockNavigate }),
  useFocusEffect: (callback: () => void) => require('react').useEffect(callback, [callback]),
}));
jest.mock('../src/context/StreakContext', () => ({ useStreak: () => ({
  streak: mockStreak,
  renewed: false, openRenewal: mockOpenRenewal, refreshStreak: jest.fn(),
}) }));

jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../src/theme').THEMES.curious, moodToColor: () => '#008000' }) }));
jest.mock('../src/components/PatternRadar', () => () => null);
jest.mock('../src/components/ShareInsightModal', () => () => null);
jest.mock('../src/api/client', () => ({ api: {
  weeklyMood: jest.fn(), streak: jest.fn(), getLatestWeeklyReview: jest.fn(),
  moodHistory: jest.fn(), getLatestAnnualReview: jest.fn(), useStreakFreeze: jest.fn(),
} }));
const history = api.moodHistory as jest.Mock;
const sample = (entries: number) => ({ days: [], avg_mood: entries ? 60 : null, mood_delta: null, entry_count: entries, top_emotions: [] });
function deferred() {
  let resolve!: (value: unknown) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
beforeEach(() => {
  mockStreak = { current_streak: 3, longest_streak: 8, total_days: 12, freeze_count: 2, next_milestone: 7 };
  jest.clearAllMocks(); history.mockReset().mockResolvedValue(sample(11));
  (api.weeklyMood as jest.Mock).mockResolvedValue({ days: [] });
  (api.streak as jest.Mock).mockResolvedValue({ current_streak: 1, longest_streak: 1, total_days: 1, freeze_count: 0, next_milestone: 7 });
  (api.getLatestWeeklyReview as jest.Mock).mockResolvedValue(null);
  (api.getLatestAnnualReview as jest.Mock).mockResolvedValue(null);
});
async function screen() { const view = render(<MoodScreen />); await act(async () => {}); return view; }

it('loads one initial history request and renders a genuine empty graph', async () => {
  history.mockResolvedValue(sample(0));
  const view = await screen(); expect(history).toHaveBeenCalledTimes(1);
  expect(history).toHaveBeenCalledWith('30d'); expect(view.getByText('Keep journaling to see your life graph.')).toBeTruthy();
});
it('keeps the mounted scroll view during a range request and avoids full-screen reloads', async () => {
  const view = await screen(), scroll = view.UNSAFE_getByType(ScrollView), pending = deferred();
  history.mockReturnValue(pending.promise);
  await act(async () => { fireEvent.press(view.getByTestId('mood-range-90d')); });
  expect(view.getByText('Mood Map')).toBeTruthy(); expect(view.UNSAFE_getByType(ScrollView)).toBe(scroll);
  expect(api.weeklyMood).toHaveBeenCalledTimes(1); expect(history).toHaveBeenCalledTimes(2);
  await act(async () => { pending.resolve(sample(90)); }); expect(view.getByText('90 entries')).toBeTruthy();
});
it('does not request the already selected range again', async () => {
  const view = await screen(); fireEvent.press(view.getByTestId('mood-range-30d'));
  expect(history).toHaveBeenCalledTimes(1);
});
it('ignores an older range response after a newer selection completes', async () => {
  const view = await screen(), old = deferred(); history.mockImplementation(range => range === '90d' ? old.promise : Promise.resolve(sample(365)));
  await act(async () => { fireEvent.press(view.getByTestId('mood-range-90d')); });
  await act(async () => { fireEvent.press(view.getByTestId('mood-range-365d')); });
  await act(async () => { old.resolve(sample(90)); });
  expect(view.getByText('365 entries')).toBeTruthy(); expect(view.queryByText('90 entries')).toBeNull();
});
it('ignores an older rejection after the current range has loaded', async () => {
  const view = await screen(), old = deferred(); history.mockImplementation(range => range === '90d' ? old.promise : Promise.resolve(sample(365)));
  await act(async () => { fireEvent.press(view.getByTestId('mood-range-90d')); });
  await act(async () => { fireEvent.press(view.getByTestId('mood-range-365d')); });
  await act(async () => { old.reject(new Error('old outage')); });
  expect(view.getByText('365 entries')).toBeTruthy(); expect(view.queryByText('Could not load mood history.')).toBeNull();
});
it('shows a history failure without claiming empty data and retries the current range', async () => {
  const view = await screen(); history.mockRejectedValueOnce(new Error('offline')).mockResolvedValue(sample(90));
  await act(async () => { fireEvent.press(view.getByTestId('mood-range-90d')); });
  expect(view.getByText('Could not load mood history.')).toBeTruthy();
  expect(view.queryByText('Keep journaling to see your life graph.')).toBeNull();
  await act(async () => { fireEvent.press(view.getByText('Retry')); });
  expect(history).toHaveBeenLastCalledWith('90d'); expect(view.getByText('90 entries')).toBeTruthy();
});

it('always renders seven calendar slots, keeping missing entries as gaps and excluding old/future data', async () => {
  const today = new Date();
  const day = (offset: number) => new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate() + offset)).toISOString().slice(0, 10);
  (api.weeklyMood as jest.Mock).mockResolvedValue({ days: [
    { day: day(-8), avg_mood: 91 }, { day: day(-2), avg_mood: 40 },
    { day: day(0), avg_mood: 60 }, { day: day(1), avg_mood: 99 },
  ] });
  const view = await screen();
  expect(view.getAllByTestId(/^mood-trend-day-/)).toHaveLength(7);
  expect(view.getAllByTestId(/^mood-trend-bar-/)).toHaveLength(2);
  expect(view.queryByText('mood 91')).toBeNull();
  expect(view.queryByText('mood 99')).toBeNull();
  expect(view.getByTestId('mood-trend-bar-' + day(0))).toHaveStyle({ maxWidth: 5 });
});

it('keeps long reviews out of Mood and links to Explore without requesting review endpoints', async () => {
  const view = await screen();
  expect(api.getLatestWeeklyReview).not.toHaveBeenCalled();
  expect(api.getLatestAnnualReview).not.toHaveBeenCalled();
  expect(view.queryByText('WEEKLY REVIEW')).toBeNull();
  expect(view.queryByText('YEAR IN REVIEW')).toBeNull();
  fireEvent.press(view.getByTestId('mood-open-reflections'));
  expect(mockNavigate).toHaveBeenCalledWith({ pathname: '/(tabs)/timeline', params: { section: 'reflections' } });
  expect(view.queryByText('Share streak')).toBeNull();
  fireEvent.press(view.getByTestId('mood-renew-streak'));
  expect(mockOpenRenewal).toHaveBeenCalledTimes(1);
});

it('keeps milestone celebration and dismissal without a streak sharing action', async () => {
  mockStreak = { ...mockStreak, current_streak: 7 };
  const view = await screen();
  expect(view.getByText('One week. Every single day.')).toBeTruthy();
  expect(view.queryByTestId('mood-milestone-share')).toBeNull();
  fireEvent.press(view.getByTestId('mood-milestone-continue'));
  expect(view.queryByText('One week. Every single day.')).toBeNull();
  view.rerender(<MoodScreen />);
  expect(view.queryByText('One week. Every single day.')).toBeNull();
});
