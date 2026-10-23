import React from 'react';
import { AccessibilityInfo, Animated, Text, TouchableOpacity } from 'react-native';
import { act, fireEvent, render, waitFor } from '@testing-library/react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { captureRef, releaseCapture } from 'react-native-view-shot';
import { ThemeProvider, useTheme, THEME_TRANSITION_MS } from '../ThemeContext';
import { api } from '../../api/client';

jest.mock('react-native-view-shot', () => ({ captureRef: jest.fn(), releaseCapture: jest.fn() }));
jest.mock('../../api/client', () => ({ api: { me: jest.fn(), updateMe: jest.fn() } }));
let completions: Array<(result: { finished: boolean }) => void>;
let motionListener: (value: boolean) => void;

function Controls() {
  const { theme, setTheme, setThemeWithBubble, setThemeWithTransition } = useTheme();
  return <>
    <Text testID="theme">{theme}</Text>
    <TouchableOpacity testID="anxiety" onPress={() => setThemeWithBubble('anxiety', 0, 0)} />
    <TouchableOpacity testID="stress" onPress={() => setThemeWithBubble('stress', 0, 0)} />
    <TouchableOpacity testID="grief" onPress={() => setThemeWithBubble('grief', 0, 0)} />
    <TouchableOpacity testID="direct" onPress={() => setTheme('career')} />
    <TouchableOpacity testID="onboarding" onPress={() => setThemeWithTransition('stress', { syncAccount: false, onApply: applied, onComplete: completed })} />
    <TouchableOpacity testID="same-goal" onPress={() => setThemeWithTransition('curious', { syncAccount: false, onApply: applied, onComplete: completed })} />
  </>;
}
const applied = jest.fn();
const completed = jest.fn();
async function setup() {
  const view = render(<ThemeProvider><Controls /></ThemeProvider>, { createNodeMock: () => ({}) });
  await act(async () => {});
  return view;
}
beforeEach(async () => {
  jest.restoreAllMocks();
  jest.clearAllMocks();
  await AsyncStorage.clear();
  completions = [];
  (captureRef as jest.Mock).mockReset().mockResolvedValue('file:///theme-snapshot.png');
  (api.me as jest.Mock).mockResolvedValue({ goal: 'curious' });
  (api.updateMe as jest.Mock).mockResolvedValue({});
  jest.spyOn(AccessibilityInfo, 'isReduceMotionEnabled').mockResolvedValue(false);
  jest.spyOn(AccessibilityInfo, 'addEventListener').mockImplementation(((_event: string, listener: (value: boolean) => void) => {
    motionListener = listener;
    return { remove: jest.fn() };
  }) as any);
  jest.spyOn(Animated, 'timing').mockImplementation(() => ({
    start: callback => { if (callback) completions.push(callback); },
    stop: jest.fn(), reset: jest.fn(),
  }));
  jest.spyOn(global, 'requestAnimationFrame').mockImplementation(callback => { callback(0); return 1; });
});

it('waits for image decode, then crossfades all UI with a native 280ms animation and saves once', async () => {
  const view = await setup();
  fireEvent.press(view.getByTestId('anxiety'));
  await waitFor(() => expect(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true })).toBeTruthy());
  expect(view.getByTestId('theme')).toHaveTextContent('curious');
  expect(api.updateMe).not.toHaveBeenCalled();
  fireEvent(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true }), 'load');
  expect(view.getByTestId('theme')).toHaveTextContent('anxiety');
  expect(Animated.timing).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ duration: THEME_TRANSITION_MS, useNativeDriver: true, toValue: 0 }));
  await act(async () => { completions.shift()?.({ finished: true }); });
  expect(view.queryByTestId('theme-transition-snapshot', { includeHiddenElements: true })).toBeNull();
  expect(releaseCapture).toHaveBeenCalledWith('file:///theme-snapshot.png');
  expect(api.updateMe).toHaveBeenCalledTimes(1);
  expect(api.updateMe).toHaveBeenCalledWith({ goal: 'anxiety' });
});

it('keeps the latest rapid selection rather than dropping it', async () => {
  const view = await setup();
  fireEvent.press(view.getByTestId('anxiety'));
  await waitFor(() => expect(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true })).toBeTruthy());
  fireEvent(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true }), 'load');
  fireEvent.press(view.getByTestId('stress'));
  fireEvent.press(view.getByTestId('grief'));
  await act(async () => { completions.shift()?.({ finished: true }); });
  await waitFor(() => expect(captureRef).toHaveBeenCalledTimes(2));
  fireEvent(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true }), 'load');
  await act(async () => { completions.shift()?.({ finished: true }); });
  expect(view.getByTestId('theme')).toHaveTextContent('grief');
  expect(api.updateMe).toHaveBeenNthCalledWith(2, { goal: 'grief' });
  expect(api.updateMe).not.toHaveBeenCalledWith({ goal: 'stress' });
});

it('respects reduced motion without capturing or animating the screen', async () => {
  jest.spyOn(AccessibilityInfo, 'isReduceMotionEnabled').mockResolvedValue(true);
  const view = await setup();
  fireEvent.press(view.getByTestId('stress'));
  expect(view.getByTestId('theme')).toHaveTextContent('stress');
  expect(captureRef).not.toHaveBeenCalled();
  expect(Animated.timing).not.toHaveBeenCalled();
});

it('reveals onboarding underneath the crossfade and saves its goal locally only', async () => {
  const view = await setup();
  fireEvent.press(view.getByTestId('onboarding'));
  await waitFor(() => expect(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true })).toBeTruthy());
  expect(applied).not.toHaveBeenCalled();
  fireEvent(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true }), 'load');
  expect(applied).toHaveBeenCalledTimes(1);
  expect(completed).not.toHaveBeenCalled();
  await act(async () => { completions.shift()?.({ finished: true }); });
  expect(completed).toHaveBeenCalledTimes(1);
  expect(AsyncStorage.setItem).toHaveBeenCalledWith('dreamlog:theme', 'stress');
  expect(api.updateMe).not.toHaveBeenCalled();
});

it.each([false, true])('advances onboarding for an unchanged goal with reduced motion %s', async reduced => {
  jest.spyOn(AccessibilityInfo, 'isReduceMotionEnabled').mockResolvedValue(reduced);
  const view = await setup();
  fireEvent.press(view.getByTestId('same-goal'));
  expect(applied).toHaveBeenCalledTimes(1);
  expect(completed).toHaveBeenCalledTimes(1);
  expect(captureRef).not.toHaveBeenCalled();
  expect(api.updateMe).not.toHaveBeenCalled();
});

it('advances onboarding immediately without motion or an account request when reduced motion is enabled', async () => {
  jest.spyOn(AccessibilityInfo, 'isReduceMotionEnabled').mockResolvedValue(true);
  const view = await setup();
  fireEvent.press(view.getByTestId('onboarding'));
  expect(view.getByTestId('theme')).toHaveTextContent('stress');
  expect(applied).toHaveBeenCalledTimes(1);
  expect(completed).toHaveBeenCalledTimes(1);
  expect(captureRef).not.toHaveBeenCalled();
  expect(Animated.timing).not.toHaveBeenCalled();
  expect(api.updateMe).not.toHaveBeenCalled();
});

it('completes safely if capture or image decode fails', async () => {
  (captureRef as jest.Mock).mockRejectedValueOnce(new Error('capture unavailable'));
  const view = await setup();
  fireEvent.press(view.getByTestId('anxiety'));
  await waitFor(() => expect(view.getByTestId('theme')).toHaveTextContent('anxiety'));
  fireEvent.press(view.getByTestId('grief'));
  await waitFor(() => expect(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true })).toBeTruthy());
  fireEvent(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true }), 'error');
  expect(view.getByTestId('theme')).toHaveTextContent('grief');
  expect(view.queryByTestId('theme-transition-snapshot', { includeHiddenElements: true })).toBeNull();
});

it('releases late captures after direct changes or unmount without reverting the selected theme', async () => {
  let resolveCapture!: (uri: string) => void;
  (captureRef as jest.Mock).mockImplementation(() => new Promise(resolve => { resolveCapture = resolve; }));
  const view = await setup();
  fireEvent.press(view.getByTestId('anxiety'));
  fireEvent.press(view.getByTestId('direct'));
  await act(async () => { resolveCapture('file:///late.png'); });
  expect(view.getByTestId('theme')).toHaveTextContent('career');
  expect(releaseCapture).toHaveBeenCalledWith('file:///late.png');
  fireEvent.press(view.getByTestId('stress'));
  view.unmount();
  await act(async () => { resolveCapture('file:///unmounted.png'); });
  expect(releaseCapture).toHaveBeenCalledWith('file:///unmounted.png');
});

it('finishes an active transition when reduced motion is enabled', async () => {
  const view = await setup();
  fireEvent.press(view.getByTestId('anxiety'));
  await waitFor(() => expect(view.getByTestId('theme-transition-snapshot', { includeHiddenElements: true })).toBeTruthy());
  act(() => motionListener(true));
  expect(view.getByTestId('theme')).toHaveTextContent('anxiety');
  expect(view.queryByTestId('theme-transition-snapshot', { includeHiddenElements: true })).toBeNull();
});

it('does not let late account hydration overwrite a manual choice', async () => {
  let resolveUser!: (value: { goal: string }) => void;
  (api.me as jest.Mock).mockReturnValue(new Promise(resolve => { resolveUser = resolve; }));
  const view = await setup();
  fireEvent.press(view.getByTestId('direct'));
  await act(async () => { resolveUser({ goal: 'stress' }); });
  expect(view.getByTestId('theme')).toHaveTextContent('career');
});
