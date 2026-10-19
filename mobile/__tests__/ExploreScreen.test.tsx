import React from 'react';
import { act, fireEvent, render } from '@testing-library/react-native';
import ExploreScreen from '../app/(tabs)/timeline';
import ShareInsightModal from '../src/components/ShareInsightModal';
import { api } from '../src/api/client';
const mockPush = jest.fn(), mockRequestAuth = jest.fn();
let mockAuthenticated = true;
jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useLocalSearchParams: () => ({ section: 'reflections' }),
  useFocusEffect: (callback: () => void) => require('react').useEffect(callback, [callback]),
}));
jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../src/theme').THEMES.curious }) }));
jest.mock('../src/context/AuthContext', () => ({ useAuth: () => ({ isAuthenticated: mockAuthenticated, requestAuth: mockRequestAuth }) }));
jest.mock('../src/context/StreakContext', () => ({ useStreak: () => ({ streak: { current_streak: 3 } }) }));
jest.mock('../src/components/ShareInsightModal', () => jest.fn(() => null));
jest.mock('../src/api/client', () => ({ api: { getLatestWeeklyReview: jest.fn(), getLatestAnnualReview: jest.fn() } }));
const weekly = { id: 'w', status: 'completed', week_start: '2026-10-11', narrative: 'A little steadier this week.', entry_count: 5, top_emotions: ['calm'], mood_arc: [{ date: '2026-10-12', avg_mood: 60 }] };
const annual = { id: 'y', status: 'completed', year: 2025, narrative: 'A year of noticing.', entry_count: 100, avg_mood: 62, top_emotions: [], top_topics: [], mood_arc: [{ month: '2025-01', avg_mood: 60, entry_count: 9 }] };
beforeEach(() => {
  jest.clearAllMocks(); mockAuthenticated = true;
  (api.getLatestWeeklyReview as jest.Mock).mockReset().mockResolvedValue(weekly);
  (api.getLatestAnnualReview as jest.Mock).mockReset().mockResolvedValue(annual);
});
async function screen() { const view = render(<ExploreScreen />); await act(async () => {}); return view; }

it('shows both reviews in Explore while keeping all existing journal and tool links', async () => {
  const view = await screen();
  expect(view.getByTestId('explore-weekly-review')).toBeTruthy();
  expect(view.getByText('2025 in Review')).toBeTruthy();
  expect(view.getByText(weekly.narrative)).toBeTruthy();
  expect(view.getByText(annual.narrative)).toBeTruthy();
  for (const [id, route] of [['explore-entries', '/entries'], ['explore-therapy', '/therapy'], ['explore-journeys', '/journeys'], ['explore-relationships', '/relationships']]) {
    fireEvent.press(view.getByTestId(id)); expect(mockPush).toHaveBeenLastCalledWith(route);
  }
  fireEvent.press(view.getByTestId('explore-dream'));
  expect(mockPush).toHaveBeenLastCalledWith({ pathname: '/record', params: { mode: 'dream' } });
  fireEvent.press(view.getByTestId('explore-share-review'));
  const props = (ShareInsightModal as jest.Mock).mock.calls.at(-1)[0];
  expect(props).toMatchObject({ visible: true, weekStart: weekly.week_start, moodArc: weekly.mood_arc, topEmotions: weekly.top_emotions, entryCount: 5, streak: 3 });
  act(() => props.onClose());
  expect((ShareInsightModal as jest.Mock).mock.calls.at(-1)[0].visible).toBe(false);
});

it('distinguishes no generated reviews from a load failure and retries without hiding a successful review', async () => {
  (api.getLatestWeeklyReview as jest.Mock).mockRejectedValueOnce(new Error('offline'));
  const view = await screen();
  expect(view.getByText('Some reflections could not be loaded.')).toBeTruthy();
  expect(view.getByText('2025 in Review')).toBeTruthy();
  expect(view.queryByText('Your reflections will appear here as you keep journaling.')).toBeNull();
  await act(async () => { fireEvent.press(view.getByTestId('explore-retry-reviews')); });
  expect(view.getByTestId('explore-weekly-review')).toBeTruthy();
  expect(view.queryByText('Some reflections could not be loaded.')).toBeNull();
});

it('treats 404 and pending reviews as empty, and never fetches private reviews for a guest', async () => {
  (api.getLatestWeeklyReview as jest.Mock).mockRejectedValue({ response: { status: 404 } });
  (api.getLatestAnnualReview as jest.Mock).mockResolvedValue({ ...annual, status: 'pending' });
  const view = await screen();
  expect(view.getByText('Your reflections will appear here as you keep journaling.')).toBeTruthy();
  expect(view.queryByText('Some reflections could not be loaded.')).toBeNull();
  jest.clearAllMocks(); mockAuthenticated = false;
  view.rerender(<ExploreScreen />); await act(async () => {});
  expect(api.getLatestWeeklyReview).not.toHaveBeenCalled();
  expect(view.queryByText(weekly.narrative)).toBeNull();
  fireEvent.press(view.getByText('Sign in to see your reflections'));
  expect(mockRequestAuth).toHaveBeenCalledTimes(1);
  fireEvent.press(view.getByTestId('explore-dream'));
  expect(mockPush).not.toHaveBeenCalled();
  expect(mockRequestAuth).toHaveBeenCalledTimes(2);
});

it('ignores a private response after sign-out', async () => {
  let resolve!: (value: unknown) => void;
  (api.getLatestWeeklyReview as jest.Mock).mockReturnValue(new Promise(done => { resolve = done; }));
  const view = await screen(); mockAuthenticated = false;
  view.rerender(<ExploreScreen />);
  await act(async () => { resolve(weekly); });
  expect(view.queryByText(weekly.narrative)).toBeNull();
  expect(view.queryByTestId('explore-share-review')).toBeNull();
});
