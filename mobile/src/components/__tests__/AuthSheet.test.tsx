import React from 'react';
import { act, fireEvent, render, waitFor } from '@testing-library/react-native';
import AuthSheet from '../AuthSheet';
import { supabase, handleDeepLink } from '../../lib/supabase';
import * as WebBrowser from 'expo-web-browser';
import { T } from '../../testIDs';

jest.mock('../../lib/supabase', () => ({ supabase: { auth: { signInWithPassword: jest.fn(), signUp: jest.fn(), signInWithOAuth: jest.fn(), signInWithIdToken: jest.fn() } }, handleDeepLink: jest.fn() }));
jest.mock('../../context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../../theme').THEMES.curious }) }));
jest.mock('expo-web-browser', () => ({ openAuthSessionAsync: jest.fn() }));
jest.mock('expo-linking', () => ({ createURL: () => 'ode://auth/callback' }));
jest.mock('expo-apple-authentication', () => ({ AppleAuthenticationScope: {}, AppleAuthenticationButtonType: {}, AppleAuthenticationButtonStyle: {}, signInAsync: jest.fn(), AppleAuthenticationButton: () => null }));

const login = supabase.auth.signInWithPassword as jest.Mock;
const signup = supabase.auth.signUp as jest.Mock;
const oauth = supabase.auth.signInWithOAuth as jest.Mock;
const browser = WebBrowser.openAuthSessionAsync as jest.Mock;
beforeEach(() => {
  jest.resetAllMocks();
  login.mockResolvedValue({ error: null }); signup.mockResolvedValue({ data: { session: null }, error: null });
  oauth.mockResolvedValue({ data: { url: 'https://fixture.invalid/auth' }, error: null });
  browser.mockResolvedValue({ type: 'cancel' });
});

function emailForm(register = false) {
  const close = jest.fn();
  const view = render(<AuthSheet visible onClose={close} />);
  fireEvent.press(view.getByTestId(T.auth.useEmail));
  if (register) fireEvent.press(view.getByTestId(T.auth.tabRegister));
  return { ...view, close };
}
function fill(view: ReturnType<typeof emailForm>, password = 'secret123') {
  fireEvent.changeText(view.getByTestId(T.auth.emailInput), ' qa@example.test ');
  fireEvent.changeText(view.getByTestId(T.auth.passwordInput), password);
}

describe('authentication sheet', () => {
  it('renders social controls and supplied prompt', () => {
    const view = render(<AuthSheet visible prompt="Save your journal" onClose={jest.fn()} />);
    expect(view.getByText('Save your journal')).toBeTruthy(); expect(view.getByTestId(T.auth.google)).toBeTruthy();
  });
  it('blocks empty credentials before contacting auth', () => {
    const view = emailForm(); fireEvent.press(view.getByTestId(T.auth.submit));
    expect(view.getByText('Email and password are required.')).toBeTruthy(); expect(login).not.toHaveBeenCalled();
  });
  it('requires a registration name', () => {
    const view = emailForm(true); fill(view); fireEvent.press(view.getByTestId(T.auth.submit));
    expect(view.getByText('Name is required.')).toBeTruthy(); expect(signup).not.toHaveBeenCalled();
  });
  it('rejects a short registration password', () => {
    const view = emailForm(true); fill(view, '12345'); fireEvent.changeText(view.getByTestId(T.auth.nameInput), 'QA');
    fireEvent.press(view.getByTestId(T.auth.submit)); expect(view.getByText('Password must be at least 6 characters.')).toBeTruthy();
  });
  it('trims email and passes credentials to Supabase', async () => {
    const view = emailForm(); fill(view);
    await act(async () => { fireEvent.press(view.getByTestId(T.auth.submit)); });
    expect(login).toHaveBeenCalledWith({ email: 'qa@example.test', password: 'secret123' });
  });
  it('preserves meaningful password whitespace', async () => {
    const view = emailForm(); fill(view, ' secret123 '); fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(() => expect(login).toHaveBeenCalledWith({ email: 'qa@example.test', password: ' secret123 ' }));
  });
  it('shows provider error and allows retry', async () => {
    login.mockResolvedValueOnce({ error: new Error('Wrong credentials') });
    const view = emailForm(); fill(view); fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(() => expect(view.getByText('Wrong credentials')).toBeTruthy());
    fireEvent.changeText(view.getByTestId(T.auth.passwordInput), 'corrected');
    expect(view.queryByText('Wrong credentials')).toBeNull();
    fireEvent.press(view.getByTestId(T.auth.submit)); await waitFor(() => expect(login).toHaveBeenCalledTimes(2));
  });
  it('registers with name and closes when email verification is required', async () => {
    const view = emailForm(true); fill(view); fireEvent.changeText(view.getByTestId(T.auth.nameInput), ' QA ');
    fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(() => expect(view.close).toHaveBeenCalledTimes(1));
    expect(signup).toHaveBeenCalledWith(expect.objectContaining({ email: 'qa@example.test', options: expect.objectContaining({ data: { full_name: 'QA' } }) }));
  });
  it('shows registration provider errors', async () => {
    signup.mockResolvedValueOnce({ error: new Error('Email already exists') });
    const view = emailForm(true); fill(view); fireEvent.changeText(view.getByTestId(T.auth.nameInput), 'QA');
    fireEvent.press(view.getByTestId(T.auth.submit)); await waitFor(() => expect(view.getByText('Email already exists')).toBeTruthy());
  });
  it('resets credentials and navigation on dismissal', () => {
    const view = emailForm(); fill(view); fireEvent.press(view.getByTestId(T.auth.later));
    expect(view.close).toHaveBeenCalled(); fireEvent.press(view.getByTestId(T.auth.useEmail));
    expect(view.getByTestId(T.auth.emailInput).props.value).toBe(''); expect(view.getByTestId(T.auth.passwordInput).props.value).toBe('');
  });
  it('handles cancelled Google auth and enables controls again', async () => {
    const view = render(<AuthSheet visible onClose={jest.fn()} />); fireEvent.press(view.getByTestId(T.auth.google));
    await waitFor(() => expect(browser).toHaveBeenCalled());
    await waitFor(() => expect(view.getByTestId(T.auth.google)).toBeEnabled()); expect(handleDeepLink).not.toHaveBeenCalled();
  });
  it('processes successful Google auth callback', async () => {
    browser.mockResolvedValueOnce({ type: 'success', url: 'ode://auth/callback?code=fixture' });
    const view = render(<AuthSheet visible onClose={jest.fn()} />); fireEvent.press(view.getByTestId(T.auth.google));
    await waitFor(() => expect(handleDeepLink).toHaveBeenCalledWith('ode://auth/callback?code=fixture'));
  });
  it('handles missing Google provider URL without opening browser', async () => {
    oauth.mockResolvedValueOnce({ data: {}, error: null });
    const view = emailForm(); fireEvent.press(view.getByTestId(T.auth.google));
    await waitFor(() => expect(view.getByText('Could not start Google sign-in')).toBeTruthy()); expect(browser).not.toHaveBeenCalled();
  });
  it('prevents keyboard submission from duplicating an in-flight request', async () => {
    let resolve!: (value: unknown) => void;
    login.mockImplementationOnce(() => new Promise(r => { resolve = r; }));
    const view = emailForm(); fill(view); fireEvent.press(view.getByTestId(T.auth.submit));
    fireEvent(view.getByTestId(T.auth.passwordInput), 'submitEditing');
    expect(login).toHaveBeenCalledTimes(1);
    await act(async () => { resolve({ error: null }); });
  });
  it('can sign in again after a successful sheet is hidden and reopened', async () => {
    const view = emailForm(); fill(view); fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(() => expect(login).toHaveBeenCalledTimes(1));
    view.rerender(<AuthSheet visible={false} onClose={view.close} />);
    view.rerender(<AuthSheet visible onClose={view.close} />);
    fireEvent.press(view.getByTestId(T.auth.useEmail)); fill(view); fireEvent.press(view.getByTestId(T.auth.submit));
    await waitFor(() => expect(login).toHaveBeenCalledTimes(2));
  });
});
