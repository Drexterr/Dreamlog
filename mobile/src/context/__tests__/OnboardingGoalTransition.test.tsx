import React from 'react';
import { Animated } from 'react-native';
import { act, fireEvent, render } from '@testing-library/react-native';
import OnboardingScreen from '../../../app/onboarding';
import { T, onboardingGoalID } from '../../testIDs';

const mockTransition = jest.fn();
jest.mock('../ThemeContext', () => ({
  useTheme: () => ({ colors: require('../../theme').THEMES.curious, setThemeWithTransition: mockTransition }),
}));
jest.mock('expo-router', () => ({ useRouter: () => ({ replace: jest.fn() }) }));
jest.mock('../../services/region', () => ({ setRegionFromCountry: jest.fn() }));
jest.mock('../../services/guestStorage', () => ({
  saveGuestPreferences: jest.fn(), markOnboardingDone: jest.fn(), markTourPending: jest.fn(),
}));

beforeEach(() => {
  jest.clearAllMocks();
  jest.spyOn(Animated, 'timing').mockImplementation(() => ({ start: jest.fn(), stop: jest.fn(), reset: jest.fn() }));
  jest.spyOn(Animated, 'loop').mockImplementation(() => ({ start: jest.fn(), stop: jest.fn(), reset: jest.fn() }));
});
afterEach(() => jest.restoreAllMocks());

it.each(['anxiety', 'stress', 'relationships', 'grief', 'depression', 'career', 'trauma', 'curious'])(
  'uses the shared local-only crossfade for the %s onboarding goal', goal => {
    const view = render(<OnboardingScreen />);
    fireEvent.press(view.getByTestId(T.onboarding.begin));
    fireEvent.press(view.getByTestId(T.onboarding.introSkip));
    jest.mocked(Animated.timing).mockClear();
    fireEvent.press(view.getByTestId(onboardingGoalID(goal)));
    expect(mockTransition).toHaveBeenCalledWith(goal, expect.objectContaining({ syncAccount: false }));
    expect(view.queryByTestId(T.onboarding.revealContinue)).toBeNull();
    fireEvent.press(view.getByTestId(onboardingGoalID(goal)));
    expect(mockTransition).toHaveBeenCalledTimes(1);
    const options = mockTransition.mock.calls[0][1];
    act(() => options.onApply());
    expect(view.getByTestId(T.onboarding.revealContinue)).toBeTruthy();
    expect(Animated.timing).not.toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ duration: 1250 }));
    act(() => options.onComplete());
    fireEvent.press(view.getByTestId(T.onboarding.revealContinue));
    expect(view.getByTestId(T.onboarding.nameInput)).toBeTruthy();
  },
);
