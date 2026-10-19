import React from 'react';
import { act, fireEvent, render, waitFor } from '@testing-library/react-native';
import AuthScreen from '../app/auth';
import { supabase } from '../src/lib/supabase';
import { resolvePostAuthRoute, setAppRole } from '../src/services/postAuthRoute';
import { T } from '../src/testIDs';

const mockReplace = jest.fn();
jest.mock('expo-router', () => ({ useRouter: () => ({ replace: mockReplace }) }));
jest.mock('../src/lib/supabase', () => ({ supabase: { auth: { signInWithPassword: jest.fn(), signUp: jest.fn() } }, handleDeepLink: jest.fn() }));
jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../src/theme').THEMES.curious }) }));
jest.mock('../src/services/postAuthRoute', () => ({ resolvePostAuthRoute: jest.fn(), setAppRole: jest.fn() }));
jest.mock('../src/api/client', () => ({ api: { acceptTerms: jest.fn().mockResolvedValue({}) } }));
jest.mock('expo-web-browser', () => ({ openAuthSessionAsync: jest.fn() }));
jest.mock('expo-linking', () => ({ createURL: () => 'ode://auth/callback' }));
jest.mock('expo-apple-authentication', () => ({ AppleAuthenticationScope: {}, AppleAuthenticationButtonType: {}, AppleAuthenticationButtonStyle: {}, AppleAuthenticationButton: () => null }));
const login = supabase.auth.signInWithPassword as jest.Mock;
beforeEach(() => {
  jest.clearAllMocks(); login.mockReset(); login.mockResolvedValue({ error: null });
  (resolvePostAuthRoute as jest.Mock).mockResolvedValue('/(tabs)'); (setAppRole as jest.Mock).mockResolvedValue(undefined);
});
function form() {
  const view = render(<AuthScreen />);
  fireEvent.changeText(view.getByTestId(T.auth.emailInput), ' qa@example.test ');
  fireEvent.changeText(view.getByTestId(T.auth.passwordInput), 'secret123');
  return view;
}
describe('full authentication screen', () => {
  it('requires credentials before contacting auth', () => {
    const view=render(<AuthScreen />); fireEvent.press(view.getByTestId(T.auth.submit));
    expect(view.getByText('Email and password are required.')).toBeTruthy(); expect(login).not.toHaveBeenCalled();
  });
  it('routes after successful email login', async () => {
    const view=form(); fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(()=>expect(mockReplace).toHaveBeenCalledWith('/(tabs)'));
    expect(login).toHaveBeenCalledWith({ email:'qa@example.test', password:'secret123' });
  });
  it('preserves the password exactly as entered', async () => {
    const view=form(); fireEvent.changeText(view.getByTestId(T.auth.passwordInput),' secret123 '); fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(()=>expect(login).toHaveBeenCalledWith({ email:'qa@example.test',password:' secret123 ' }));
  });
  it('blocks duplicate keyboard submission while login is pending', async () => {
    let finish!: (value: unknown)=>void; login.mockImplementationOnce(()=>new Promise(r=>{finish=r;}));
    const view=form(); fireEvent.press(view.getByTestId(T.auth.submit)); fireEvent(view.getByTestId(T.auth.passwordInput),'submitEditing');
    expect(login).toHaveBeenCalledTimes(1); await act(async()=>{finish({error:null});});
  });
  it('shows auth failure and does not navigate', async () => {
    login.mockResolvedValueOnce({error:new Error('Wrong credentials')}); const view=form(); fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(()=>expect(view.getByText('Wrong credentials')).toBeTruthy()); expect(mockReplace).not.toHaveBeenCalled();
  });
  it('uses the terms-gated destination returned by the routing service', async () => {
    (resolvePostAuthRoute as jest.Mock).mockResolvedValueOnce('/accept-terms'); const view=form(); fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(()=>expect(mockReplace).toHaveBeenCalledWith('/accept-terms'));
  });
  it('saves role selection and can return to journal role', () => {
    const view=form(); fireEvent.press(view.getByTestId(T.authRole.therapist)); expect(setAppRole).toHaveBeenCalledWith('therapist');
    expect(view.getByTestId(T.authRole.therapistHint)).toBeTruthy(); fireEvent.press(view.getByTestId(T.authRole.me)); expect(setAppRole).toHaveBeenLastCalledWith('user');
  });
  it('clears credentials when switching registration mode', () => {
    const view=form(); fireEvent.press(view.getByTestId(T.auth.tabRegister));
    expect(view.getByTestId(T.auth.emailInput).props.value).toBe(''); expect(view.getByTestId(T.auth.passwordInput).props.value).toBe('');
  });
  it('requires consent before registration', () => {
    const view=form(); fireEvent.press(view.getByTestId(T.auth.tabRegister));
    fireEvent.changeText(view.getByTestId(T.auth.emailInput),'qa@example.test'); fireEvent.changeText(view.getByTestId(T.auth.passwordInput),'secret123');
    fireEvent.changeText(view.getByPlaceholderText('Your name'),'QA'); fireEvent.press(view.getByTestId(T.auth.submit));
    expect(view.getByText('Please accept the Terms of Service and Privacy Policy to continue.')).toBeTruthy();
  });
});
