import React from 'react';
import { Alert, Animated, Linking } from 'react-native';
import { act, fireEvent, render, waitFor } from '@testing-library/react-native';
import ReflectionScreen from '../app/reflection/[id]';
import { api } from '../src/api/client';
import { detectUserCountry } from '../src/services/region';
import { T } from '../src/testIDs';

const mockPush = jest.fn(), mockReplace = jest.fn();
let mockParams: { id?: string } = { id: 'entry-fixture' };
let mockResourceOverrides: Record<string, unknown> = {};
jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockPush, replace: mockReplace }),
  useLocalSearchParams: () => mockParams,
}));
jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({
  colors: require('../src/theme').THEMES.curious, moodToColor: () => '#008080',
}) }));
jest.mock('../src/api/client', () => ({
  api: { getEntry: jest.fn(), getAnalysis: jest.fn(), listEntries: jest.fn(), scheduleCheckin: jest.fn() },
  isApiError: (error: any) => error?.fixtureApiError === true,
}));
jest.mock('../src/services/region', () => ({ detectUserCountry: jest.fn() }));
// No real phone numbers or external resource handlers are used by this suite.
jest.mock('../src/services/helplines', () => ({
  helplinesForCountry: (country: string | null) => ({
    code: country === 'IN' ? 'IN' : 'INTL', countryName: 'Fixture country',
    lines: [{ name: country === 'IN' ? 'Local support fixture' : 'Global support fixture', detail: 'Test resource', action: { kind: 'link', url: 'https://fixture.invalid/resource' } }],
    ...mockResourceOverrides,
  }),
  helplineHref: (action: { url: string }) => action.url,
}));

const getEntry = api.getEntry as jest.Mock, getAnalysis = api.getAnalysis as jest.Mock;
const list = api.listEntries as jest.Mock, schedule = api.scheduleCheckin as jest.Mock;
const detect = detectUserCountry as jest.Mock;
const analysis = (extra = {}) => ({
  reflection: 'First reflected paragraph.\n\nSecond reflected paragraph.', summary: 'Fixture summary',
  mood_score: 65, is_crisis: false, topics: [], emotional_tone: [], key_quotes: [], ...extra,
});
const entry = (status = 'completed') => ({ id: 'entry-fixture', status, created_at: '2026-10-12T12:00:00Z' });
const apiError = (status: number, error: string) => ({ fixtureApiError: true, response: { status, data: { error } } });
let alert: jest.SpyInstance, canOpen: jest.SpyInstance, open: jest.SpyInstance;
beforeEach(() => {
  jest.clearAllMocks(); mockParams = { id: 'entry-fixture' };
  mockResourceOverrides = {};
  getEntry.mockReset().mockResolvedValue(entry()); getAnalysis.mockReset().mockResolvedValue(analysis());
  list.mockReset().mockResolvedValue({ total: 3 }); schedule.mockReset().mockResolvedValue({});
  detect.mockReset().mockResolvedValue('IN');
  alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
  canOpen = jest.spyOn(Linking, 'canOpenURL').mockResolvedValue(true);
  open = jest.spyOn(Linking, 'openURL').mockResolvedValue(undefined);
  jest.spyOn(Animated, 'parallel').mockReturnValue({ start: jest.fn(), stop: jest.fn(), reset: jest.fn() });
});
afterEach(() => { jest.restoreAllMocks(); });
async function normal() {
  const view = render(<ReflectionScreen />);
  await waitFor(() => expect(view.getByTestId(T.reflection.doneButton)).toBeTruthy());
  return view;
}

describe('reflection loading and recovery', () => {
  it('shows loading while analysis is pending', async () => {
    getAnalysis.mockReturnValue(new Promise(() => {}));
    const view = render(<ReflectionScreen />);
    await act(async () => {});
    expect(view.getByText(/reflecting on what you said/)).toBeTruthy();
    expect(view.queryByTestId(T.reflection.doneButton)).toBeNull();
  });
  it('loads entry and analysis for the requested ID and renders paragraphs', async () => {
    const view = await normal();
    expect(getEntry).toHaveBeenCalledWith('entry-fixture'); expect(getAnalysis).toHaveBeenCalledWith('entry-fixture');
    expect(view.getByText('First reflected paragraph.')).toBeTruthy();
    expect(view.getByText('Second reflected paragraph.')).toBeTruthy();
  });
  it.each(['pending', 'processing'])('shows a retryable state for %s entries', async (status) => {
    getEntry.mockResolvedValue(entry(status)); getAnalysis.mockRejectedValue(new Error('not ready'));
    const view = render(<ReflectionScreen />);
    await waitFor(() => expect(view.getByText(/Still reflecting on this one/)).toBeTruthy());
    expect(view.getByText('Try again')).toBeTruthy();
  });
  it('explains processing failure without suggesting a reload can regenerate it', async () => {
    getEntry.mockResolvedValue(entry('failed')); getAnalysis.mockRejectedValue(new Error('missing'));
    const view = render(<ReflectionScreen />);
    await waitFor(() => expect(view.getByText("This entry couldn't be processed")).toBeTruthy());
    expect(view.queryByText('Try again')).toBeNull();
    fireEvent.press(view.getByText('Go back')); expect(mockReplace).toHaveBeenCalledWith('/(tabs)');
  });
  it('recovers from both fetches failing on retry', async () => {
    getEntry.mockRejectedValueOnce(new Error('offline')); getAnalysis.mockRejectedValueOnce(new Error('offline'));
    const view = render(<ReflectionScreen />);
    await waitFor(() => expect(view.getByText('Reflection unavailable')).toBeTruthy());
    fireEvent.press(view.getByText('Try again'));
    await waitFor(() => expect(view.getByTestId(T.reflection.doneButton)).toBeTruthy());
    expect(getEntry).toHaveBeenCalledTimes(2); expect(getAnalysis).toHaveBeenCalledTimes(2);
  });
  it('can display available analysis even when entry metadata is unavailable', async () => {
    getEntry.mockRejectedValue(new Error('metadata unavailable'));
    const view = await normal(); expect(view.getByText('First reflected paragraph.')).toBeTruthy();
  });
  it('does not stay in loading forever when the route ID is absent', async () => {
    mockParams = {}; const view = render(<ReflectionScreen />);
    await waitFor(() => expect(view.getByText('Reflection unavailable')).toBeTruthy());
    expect(getEntry).not.toHaveBeenCalled(); expect(getAnalysis).not.toHaveBeenCalled();
  });
  it('does not let a slow previous entry overwrite the current route', async () => {
    let resolveOld!: (value: ReturnType<typeof analysis>) => void;
    const old = new Promise(resolve => { resolveOld = resolve; });
    mockParams = { id: 'entry-old' };
    getAnalysis.mockImplementation((id: string) => id === 'entry-old' ? old : Promise.resolve(analysis({ reflection: 'Current entry reflection' })));
    const view = render(<ReflectionScreen />);
    mockParams = { id: 'entry-new' }; view.rerender(<ReflectionScreen />);
    await waitFor(() => expect(view.getByText('Current entry reflection')).toBeTruthy());
    await act(async () => { resolveOld(analysis({ reflection: 'Stale entry reflection' })); });
    expect(view.getByText('Current entry reflection')).toBeTruthy();
    expect(view.queryByText('Stale entry reflection')).toBeNull();
  });
  it('does not animate a late reflection response after unmount', async () => {
    let resolve!: (value: ReturnType<typeof analysis>) => void;
    getAnalysis.mockReturnValue(new Promise(done => { resolve = done; }));
    const view = render(<ReflectionScreen />);
    await act(async () => {});
    view.unmount();
    await act(async () => { resolve(analysis()); });
    expect(Animated.parallel).not.toHaveBeenCalled();
  });
});

describe('reflection content and navigation', () => {
  it.each([0, 2, 3])('only shows grounding at the threshold: total %s', async (total) => {
    list.mockResolvedValue({ total }); const view = await normal();
    expect(Boolean(view.queryByText(`Grounded in your ${total} entries`))).toBe(total >= 3);
  });
  it('does not lose the reflection if entry count fails', async () => {
    list.mockRejectedValue(new Error('offline')); const view = await normal();
    expect(view.getByText('First reflected paragraph.')).toBeTruthy(); expect(view.queryByText(/Grounded in your/)).toBeNull();
  });
  it('limits topics to four and emotional labels to three', async () => {
    getAnalysis.mockResolvedValue(analysis({ topics: ['t1', 't2', 't3', 't4', 't5'], emotional_tone: ['e1', 'e2', 'e3', 'e4'].map(emotion => ({ emotion, intensity: 0.6 })) }));
    const view = await normal(); expect(view.getByText('t4')).toBeTruthy(); expect(view.queryByText('t5')).toBeNull();
    expect(view.getByText('e3')).toBeTruthy(); expect(view.queryByText('e4')).toBeNull();
  });
  it('renders dream lenses, type, six symbols, insight and a key quote', async () => {
    getAnalysis.mockResolvedValue(analysis({ psychological_lens: 'Psychological fixture', vedic_lens: 'Vedic fixture', dream_type: 'lucid', dream_symbols: ['s1', 's2', 's3', 's4', 's5', 's6', 's7'], connection_insight: 'Pattern fixture', key_quotes: ['Quoted fixture'] }));
    const view = await normal();
    for (const text of ['Psychological fixture', 'Vedic fixture', 'lucid dream', 's6', 'Pattern fixture']) expect(view.getByText(text)).toBeTruthy();
    expect(view.queryByText('s7')).toBeNull(); expect(view.getByText('"Quoted fixture"')).toBeTruthy();
  });
  it.each([undefined, '', '  \n\n \t'])('renders a fallback when reflection is %s', async (reflection) => {
    getAnalysis.mockResolvedValue(analysis({ reflection })); const view = await normal();
    expect(view.getByText('No reflection available yet.')).toBeTruthy();
  });
  it('returns home and routes sharing to the correct entry', async () => {
    const view = await normal(); fireEvent.press(view.getByTestId(T.reflection.shareTherapistButton));
    expect(mockPush).toHaveBeenCalledWith('/share/entry-fixture');
    fireEvent.press(view.getByTestId(T.reflection.doneButton)); expect(mockReplace).toHaveBeenCalledWith('/(tabs)');
  });
  it('builds the follow-up seed with summary, emotions, two topics and one quote', async () => {
    getAnalysis.mockResolvedValue(analysis({ topics: ['work', 'family', 'extra'], emotional_tone: ['calm', 'hope', 'worry', 'extra'].map(emotion => ({ emotion, intensity: 0.5 })), key_quotes: ['My own words'] }));
    const view = await normal(); fireEvent.press(view.getByTestId(T.reflection.tellMoreButton));
    const route = mockPush.mock.calls[0][0]; expect(route.pathname).toBe('/followup/[id]'); expect(route.params.id).toBe('entry-fixture');
    for (const text of ['Fixture summary', 'calm, hope, worry', 'around work', 'about family', 'My own words']) expect(route.params.seedMessage).toContain(text);
    expect(route.params.seedMessage).not.toContain('extra');
  });
  it('uses a general follow-up prompt if analysis has no seed context', async () => {
    getAnalysis.mockResolvedValue(analysis({ summary: '' })); const view = await normal();
    fireEvent.press(view.getByTestId(T.reflection.tellMoreButton));
    expect(mockPush.mock.calls[0][0].params.seedMessage).toContain('What feels most present');
  });
  it('handles missing optional metadata and a default mood safely', async () => {
    getAnalysis.mockResolvedValue(analysis({ mood_score: undefined, summary: '', topics: undefined, emotional_tone: undefined, key_quotes: undefined }));
    const view = await normal(); fireEvent.press(view.getByTestId(T.reflection.tellMoreButton));
    expect(mockPush.mock.calls[0][0].params.seedMessage).toContain('What feels most present');
    expect(view.getByText('First reflected paragraph.')).toBeTruthy();
  });
  it.each([
    [{ psychological_lens: 'Single psychological lens' }, 'Single psychological lens'],
    [{ vedic_lens: 'Single Vedic lens' }, 'Single Vedic lens'],
    [{ dream_symbols: ['Only symbol'] }, 'Only symbol'],
  ])('renders a partial dream analysis without requiring both lenses: %s', async (extra, text) => {
    getAnalysis.mockResolvedValue(analysis(extra)); const view = await normal();
    expect(view.getByText('DREAM DECODER')).toBeTruthy(); expect(view.getByText(text)).toBeTruthy();
  });
  it('creates a useful seed with only one topic', async () => {
    getAnalysis.mockResolvedValue(analysis({ topics: ['work'], summary: '' })); const view = await normal();
    fireEvent.press(view.getByTestId(T.reflection.tellMoreButton));
    expect(mockPush.mock.calls[0][0].params.seedMessage).toContain('around work');
  });
});

describe('reflection check-in scheduling', () => {
  it('schedules once and marks the button completed', async () => {
    const view = await normal(); fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    await waitFor(() => expect(view.getByText(/check in with you tomorrow/)).toBeTruthy());
    fireEvent.press(view.getByTestId(T.reflection.checkinButton)); expect(schedule).toHaveBeenCalledTimes(1);
    expect(schedule).toHaveBeenCalledWith('entry-fixture');
  });
  it('disables scheduling while a request is pending', async () => {
    schedule.mockReturnValue(new Promise(() => {})); const view = await normal();
    fireEvent.press(view.getByTestId(T.reflection.checkinButton)); fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    expect(view.getByText(/Scheduling/)).toBeTruthy(); expect(schedule).toHaveBeenCalledTimes(1);
  });
  it('treats an existing schedule conflict as success', async () => {
    schedule.mockRejectedValue(apiError(409, 'already scheduled')); const view = await normal();
    fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    await waitFor(() => expect(view.getByText(/check in with you tomorrow/)).toBeTruthy()); expect(alert).not.toHaveBeenCalled();
  });
  it('explains disabled notifications and permits retry', async () => {
    schedule.mockRejectedValueOnce(apiError(409, 'nudges disabled')); const view = await normal();
    fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    await waitFor(() => expect(alert).toHaveBeenCalledWith('Notifications are off', expect.any(String)));
    fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    await waitFor(() => expect(schedule).toHaveBeenCalledTimes(2));
  });
  it('explains server failure without marking a schedule successful', async () => {
    schedule.mockRejectedValue(apiError(500, 'unavailable')); const view = await normal();
    fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    await waitFor(() => expect(alert).toHaveBeenCalledWith('Could not schedule', expect.any(String)));
    expect(view.queryByText(/check in with you tomorrow/)).toBeNull();
  });
  it('recovers the scheduling button after a network exception', async () => {
    schedule.mockRejectedValueOnce(new Error('offline')); const view = await normal();
    fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    await waitFor(() => expect(alert).toHaveBeenCalledWith('Could not schedule', expect.any(String)));
    fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    await waitFor(() => expect(view.getByText(/check in with you tomorrow/)).toBeTruthy());
  });
  it('handles a schedule conflict with no provider error detail', async () => {
    schedule.mockRejectedValue({ fixtureApiError: true, response: { status: 409, data: {} } });
    const view = await normal(); fireEvent.press(view.getByTestId(T.reflection.checkinButton));
    await waitFor(() => expect(view.getByText(/check in with you tomorrow/)).toBeTruthy());
  });
});

describe('reflection crisis UI with mock link handlers', () => {
  async function crisis() {
    getAnalysis.mockResolvedValue(analysis({ is_crisis: true })); const view = render(<ReflectionScreen />);
    await waitFor(() => expect(view.getByText("You're not alone.")).toBeTruthy()); return view;
  }
  it('shows localized resources instead of normal sharing/follow-up content', async () => {
    const view = await crisis(); expect(view.getByText('Local support fixture')).toBeTruthy();
    expect(view.queryByTestId(T.reflection.tellMoreButton)).toBeNull(); expect(view.queryByTestId(T.reflection.shareTherapistButton)).toBeNull();
    expect(view.queryByText('First reflected paragraph.')).toBeNull();
  });
  it('uses international resources if country detection fails', async () => {
    detect.mockRejectedValue(new Error('offline')); const view = await crisis();
    expect(view.getByText('Global support fixture')).toBeTruthy(); expect(view.queryByText('via Practo')).toBeNull();
  });
  it('opens only the mocked resource intent when a handler exists', async () => {
    const view = await crisis(); fireEvent.press(view.getByText('Local support fixture'));
    await waitFor(() => expect(open).toHaveBeenCalledWith('https://fixture.invalid/resource'));
    expect(canOpen).toHaveBeenCalledWith('https://fixture.invalid/resource');
  });
  it('does not invoke an unavailable resource handler', async () => {
    canOpen.mockResolvedValue(false); const view = await crisis();
    await act(async () => { fireEvent.press(view.getByText('Local support fixture')); }); expect(open).not.toHaveBeenCalled();
  });
  it.each(['call', 'text'])('renders %s resource labels and hours with fixture-only details', async (kind) => {
    mockResourceOverrides = { emergency: 'fixture-emergency', lines: [{ name: 'Action fixture', detail: 'Test resource', hours: 'Fixture hours', action: { kind, url: 'https://fixture.invalid/resource' } }] };
    const view = await crisis(); expect(view.getByText(/Test resource.*Fixture hours/)).toBeTruthy();
    expect(view.getByText(/fixture-emergency/)).toBeTruthy();
    expect(view.getAllByText(kind === 'text' ? /Text/ : /Call/).length).toBeGreaterThan(0);
    fireEvent.press(view.getByText('Action fixture'));
    await waitFor(() => expect(open).toHaveBeenCalledWith('https://fixture.invalid/resource'));
  });
  it.each([
    ['Find a therapist near you', 'https://www.practo.com/therapist'],
    ['Talk online - YourDOST', 'https://yourdost.com/'],
    ['Find more resources near you', 'https://findahelpline.com/'],
  ])('routes %s only through the mocked URL handler', async (label, prefix) => {
    const view = await crisis(); fireEvent.press(view.getByText(label));
    await waitFor(() => expect(open).toHaveBeenCalledWith(expect.any(String)));
    expect(open.mock.calls[0][0].startsWith(prefix)).toBe(true);
    expect(open).toHaveBeenCalledTimes(1);
  });
  it('allows leaving the crisis screen without launching any resource', async () => {
    const view = await crisis(); fireEvent.press(view.getByText("I'm okay for now"));
    expect(mockReplace).toHaveBeenCalledWith('/(tabs)'); expect(open).not.toHaveBeenCalled();
  });
});
