import React from 'react';
import { act, fireEvent, render } from '@testing-library/react-native';
import EntriesScreen from '../app/entries';
import { api } from '../src/api/client';

const mockPush = jest.fn(), mockBack = jest.fn();
jest.mock('expo-router', () => ({ useRouter: () => ({ push: mockPush, back: mockBack }) }));
jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../src/theme').THEMES.curious, moodToColor: () => '#008000' }) }));
jest.mock('../src/api/client', () => ({ api: { getTimeline: jest.fn(), searchEntries: jest.fn() } }));
const timeline = api.getTimeline as jest.Mock, search = api.searchEntries as jest.Mock;
const entry = (id: string, duration = 65, status = 'completed') => ({ id, duration_sec: duration, status, created_at: '2026-10-14T08:00:00Z' });
const initial = { entry: entry('initial'), analysis: { summary: 'QA entry summary', mood_score: 60, topics: ['work'] } };
beforeEach(() => {
  jest.useFakeTimers(); jest.clearAllMocks();
  timeline.mockReset().mockResolvedValue({ entries: [initial] }); search.mockReset().mockResolvedValue({ entries: [] });
});
afterEach(() => jest.useRealTimers());
async function screen() { const view = render(<EntriesScreen />); await act(async () => {}); return view; }
async function query(view: ReturnType<typeof render>, text: string) {
  fireEvent.changeText(view.getByTestId('entries-search-input'), text);
  await act(async () => { jest.advanceTimersByTime(400); });
}

it('loads timeline and opens an analyzed journal entry', async () => {
  const view = await screen(); expect(timeline).toHaveBeenCalledWith(1, 30);
  fireEvent.press(view.getByText('QA entry summary')); expect(mockPush).toHaveBeenCalledWith('/reflection/initial');
});
it('debounces trimmed Unicode search', async () => {
  const view = await screen(); fireEvent.changeText(view.getByTestId('entries-search-input'), ' old ');
  fireEvent.changeText(view.getByTestId('entries-search-input'), '  नमस्ते  ');
  await act(async () => { jest.advanceTimersByTime(400); });
  expect(search).toHaveBeenCalledTimes(1); expect(search).toHaveBeenCalledWith('नमस्ते', 20);
});
it('opens a completed search result even without a joined analysis object', async () => {
  search.mockResolvedValue({ entries: [entry('search-result', 12)] }); const view = await screen(); await query(view, 'work');
  fireEvent.press(view.getByText('0:12')); expect(mockPush).toHaveBeenCalledWith('/reflection/search-result');
});
it('opens pending entries in processing rather than doing nothing', async () => {
  timeline.mockResolvedValue({ entries: [{ entry: entry('pending', 12, 'pending') }] });
  const view = await screen(); fireEvent.press(view.getByText('0:12')); expect(mockPush).toHaveBeenCalledWith('/processing/pending');
});
it('keeps new search results when an older response arrives late', async () => {
  let finish!: (value: unknown) => void;
  search.mockReturnValueOnce(new Promise(resolve => { finish = resolve; })).mockResolvedValue({ entries: [entry('new', 34)] });
  const view = await screen(); await query(view, 'old'); await query(view, 'new');
  await act(async () => { finish({ entries: [entry('old', 13)] }); });
  expect(view.getByText('0:34')).toBeTruthy(); expect(view.queryByText('0:13')).toBeNull();
});
it('ignores old search responses after clearing the query', async () => {
  let finish!: (value: unknown) => void;
  search.mockReturnValue(new Promise(resolve => { finish = resolve; }));
  const view = await screen(); await query(view, 'old'); await query(view, '');
  await act(async () => { finish({ entries: [entry('old', 13)] }); });
  expect(view.getByText('QA entry summary')).toBeTruthy(); expect(view.queryByText('0:13')).toBeNull();
});
it('cancels scheduled searches on unmount', async () => {
  const view = await screen(); fireEvent.changeText(view.getByTestId('entries-search-input'), 'pending'); view.unmount();
  await act(async () => { jest.advanceTimersByTime(400); }); expect(search).not.toHaveBeenCalled();
});
it('shows initial load failure instead of claiming there are no entries and retries', async () => {
  timeline.mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ entries: [initial] });
  const view = await screen(); expect(view.getByText('Could not load entries.')).toBeTruthy();
  expect(view.queryByText('No entries yet.\nRecord your first journal.')).toBeNull();
  await act(async () => { fireEvent.press(view.getByText('Retry')); }); expect(view.getByText('QA entry summary')).toBeTruthy();
});
it('shows search failure, preserves the previous list, and retries the same query', async () => {
  search.mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ entries: [entry('recovered', 22)] });
  const view = await screen(); await query(view, 'work'); expect(view.getByText('Search failed. Try again.')).toBeTruthy();
  expect(view.getByText('QA entry summary')).toBeTruthy();
  await act(async () => { fireEvent.press(view.getByText('Retry')); jest.advanceTimersByTime(400); });
  expect(view.getByText('0:22')).toBeTruthy(); expect(search).toHaveBeenLastCalledWith('work', 20);
});
it('shows a valid empty search state', async () => {
  const view = await screen(); await query(view, 'not-found'); expect(view.getByText('No matching entries.')).toBeTruthy();
});
it('returns to the previous screen', async () => {
  const view = await screen(); fireEvent.press(view.getByTestId('entries-back')); expect(mockBack).toHaveBeenCalledTimes(1);
});
