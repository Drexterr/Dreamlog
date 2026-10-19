import React from 'react';
import { Alert, Share } from 'react-native';
import { act, fireEvent, render, waitFor } from '@testing-library/react-native';
import ShareScreen from '../app/share/[id]';
import { api } from '../src/api/client';
import { T } from '../src/testIDs';

const mockBack = jest.fn();
jest.mock('expo-router', () => ({ useRouter: () => ({ back: mockBack }) }));
jest.mock('../src/context/ThemeContext', () => ({ useTheme: () => ({ colors: require('../src/theme').THEMES.curious }) }));
jest.mock('../src/api/client', () => ({ api: { listShareLinks: jest.fn(), createShareLink: jest.fn(), revokeShareLink: jest.fn() } }));
const list = api.listShareLinks as jest.Mock, create = api.createShareLink as jest.Mock, revoke = api.revokeShareLink as jest.Mock;
const fixture = { token: 'a'.repeat(64), passcode: '1357', url: 'https://fixture.invalid/share/token', expires_at: '2099-10-15T12:00:00Z' };
const stored = { id: 'a334a788-168b-43ab-ac76-b2d8536dd951', token: fixture.token, url: fixture.url, expires_at: fixture.expires_at };
let alert: jest.SpyInstance, nativeShare: jest.SpyInstance;
beforeEach(() => {
  jest.clearAllMocks(); list.mockReset().mockResolvedValue({ links: [] });
  create.mockReset().mockResolvedValue(fixture); revoke.mockReset().mockResolvedValue(undefined);
  alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
  nativeShare = jest.spyOn(Share, 'share').mockResolvedValue({ action: Share.sharedAction });
});
afterEach(() => { jest.restoreAllMocks(); });
async function screen() {
  const view = render(<ShareScreen />);
  await act(async () => {});
  return view;
}
async function generated() {
  list.mockResolvedValueOnce({ links: [] }).mockResolvedValue({ links: [stored] });
  const view = await screen();
  await act(async () => { fireEvent.press(view.getByTestId(T.share.createButton)); });
  expect(view.getByTestId(T.share.sendButton)).toBeTruthy();
  return view;
}
function confirmRevoke() {
  const call = alert.mock.calls.filter(args => args[0] === 'Revoke link').at(-1);
  return call?.[2]?.find((button: { text: string }) => button.text === 'Revoke')?.onPress?.();
}

describe('sharing list and creation', () => {
  it('loads an empty list without displaying active-link rows', async () => {
    const view = await screen(); expect(list).toHaveBeenCalledTimes(1);
    expect(view.queryByText('ACTIVE LINKS')).toBeNull(); expect(view.getByTestId(T.share.createButton)).toBeTruthy();
  });
  it('handles a response with no links array', async () => {
    list.mockResolvedValue({}); const view = await screen(); expect(view.queryByTestId(T.share.revokeButton)).toBeNull();
  });
  it('shows existing links without recovering any plaintext passcodes', async () => {
    list.mockResolvedValue({ links: [stored, { ...stored, id: 'other', token: 'b'.repeat(64) }] });
    const view = await screen(); expect(view.getAllByTestId(T.share.revokeButton)).toHaveLength(2);
    expect(view.queryByText(fixture.passcode)).toBeNull(); expect(view.queryByTestId(T.share.sendButton)).toBeNull();
  });
  it('returns to the previous screen', async () => {
    const view = await screen(); fireEvent.press(view.getByTestId(T.share.back)); expect(mockBack).toHaveBeenCalledTimes(1);
  });
  it('creates a link and displays the once-returned passcode', async () => {
    const view = await generated(); expect(create).toHaveBeenCalledTimes(1);
    expect(view.getByText(fixture.passcode)).toBeTruthy(); expect(view.queryByTestId(T.share.createButton)).toBeNull();
  });
  it('prevents repeated button presses while creation is pending', async () => {
    create.mockReturnValue(new Promise(() => {})); const view = await screen();
    fireEvent.press(view.getByTestId(T.share.createButton)); fireEvent.press(view.getByTestId(T.share.createButton));
    expect(create).toHaveBeenCalledTimes(1); expect(view.queryByTestId(T.share.sendButton)).toBeNull();
  });
  it('does not claim success on creation failure and supports retry', async () => {
    create.mockRejectedValueOnce(new Error('quota exceeded')); const view = await screen();
    fireEvent.press(view.getByTestId(T.share.createButton));
    await waitFor(() => expect(alert).toHaveBeenCalledWith('Error', 'Failed to create share link. Please try again.'));
    expect(view.queryByTestId(T.share.sendButton)).toBeNull();
    fireEvent.press(view.getByTestId(T.share.createButton)); await waitFor(() => expect(view.getByTestId(T.share.sendButton)).toBeTruthy());
  });
  it('makes a list failure visible instead of implying there are no active links', async () => {
    list.mockRejectedValue(new Error('offline')); const view = await screen();
    expect(view.getByText('Could not load active links.')).toBeTruthy();
  });
  it('does not lose a created link when the initial list arrives late', async () => {
    let resolve!: (value: { links: typeof stored[] }) => void;
    list.mockReturnValueOnce(new Promise(done => { resolve = done; })).mockResolvedValue({ links: [stored] });
    const view = await screen(); fireEvent.press(view.getByTestId(T.share.createButton));
    await waitFor(() => expect(view.getByTestId(T.share.sendButton)).toBeTruthy());
    await act(async () => { resolve({ links: [] }); });
    expect(view.getByTestId(T.share.revokeButton)).toBeTruthy();
  });
  it('retries a failed list and clears the error only after success', async () => {
    list.mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ links: [stored] });
    const view = await screen(); expect(view.getByText('Could not load active links.')).toBeTruthy();
    fireEvent.press(view.getByTestId(T.share.retryButton));
    await waitFor(() => expect(view.getByTestId(T.share.revokeButton)).toBeTruthy());
    expect(view.queryByText('Could not load active links.')).toBeNull();
  });
  it('preserves a successful creation if reloading its UUID fails, then retries the list', async () => {
    list.mockResolvedValueOnce({ links: [] }).mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ links: [stored] });
    const view = await screen(); fireEvent.press(view.getByTestId(T.share.createButton));
    await waitFor(() => expect(view.getByText('Could not load active links.')).toBeTruthy());
    expect(view.getByText(fixture.passcode)).toBeTruthy(); expect(alert).not.toHaveBeenCalled();
    fireEvent.press(view.getByTestId(T.share.retryButton));
    await waitFor(() => expect(view.getByTestId(T.share.revokeButton)).toBeTruthy());
    expect(create).toHaveBeenCalledTimes(1);
  });
  it('discloses the display name actually present in the backend share payload', async () => {
    const view = await screen(); expect(view.getByText(/Your display name/)).toBeTruthy();
    expect(view.queryByText(/Entry count/)).toBeNull(); expect(view.queryByText(/reflections, personal details/)).toBeNull();
  });
  it('does not fetch or display a late creation result after unmount', async () => {
    let resolve!: (value: typeof fixture) => void;
    create.mockReturnValue(new Promise(done => { resolve = done; })); const view = await screen();
    fireEvent.press(view.getByTestId(T.share.createButton)); view.unmount();
    await act(async () => { resolve(fixture); }); expect(list).toHaveBeenCalledTimes(1); expect(alert).not.toHaveBeenCalled();
  });
});

describe('native summary sharing with mocked handlers', () => {
  it('shares the URL and passcode without raw transcript or audio data', async () => {
    const view = await generated(); fireEvent.press(view.getByTestId(T.share.sendButton));
    await waitFor(() => expect(nativeShare).toHaveBeenCalledTimes(1));
    const payload = nativeShare.mock.calls[0][0];
    expect(payload.title).toBe('My Ode Summary'); expect(payload.message).toContain(fixture.url);
    expect(payload.message).toContain('Passcode: 1357'); expect(payload.message).toContain('72 hours');
    expect(payload).not.toHaveProperty('url'); expect(payload).not.toHaveProperty('transcript');
  });
  it('keeps the link usable if the native share sheet fails', async () => {
    nativeShare.mockRejectedValue(new Error('no share handler')); const view = await generated();
    await act(async () => { fireEvent.press(view.getByTestId(T.share.sendButton)); });
    expect(view.getByText(fixture.passcode)).toBeTruthy();
    expect(view.getByTestId(T.share.sendButton)).toBeTruthy();
  });
  it('does not revoke a link when sharing is dismissed', async () => {
    nativeShare.mockResolvedValue({ action: Share.dismissedAction }); const view = await generated();
    await act(async () => { fireEvent.press(view.getByTestId(T.share.sendButton)); }); expect(revoke).not.toHaveBeenCalled();
  });
});

describe('share revocation', () => {
  it('requires confirmation and cancelling makes no API call', async () => {
    list.mockResolvedValue({ links: [stored] }); const view = await screen();
    fireEvent.press(view.getByTestId(T.share.revokeButton));
    const buttons = alert.mock.calls.at(-1)?.[2]; expect(buttons.find((button: { text: string }) => button.text === 'Cancel').style).toBe('cancel');
    expect(revoke).not.toHaveBeenCalled(); expect(view.getByTestId(T.share.revokeButton)).toBeTruthy();
  });
  it('revokes an existing link using its stored UUID', async () => {
    list.mockResolvedValue({ links: [stored] }); const view = await screen();
    fireEvent.press(view.getByTestId(T.share.revokeButton)); await act(async () => { await confirmRevoke(); });
    expect(revoke).toHaveBeenCalledWith(stored.id); expect(view.queryByTestId(T.share.revokeButton)).toBeNull();
  });
  it('resolves the UUID before revoking a newly created token', async () => {
    const view = await generated(); await waitFor(() => expect(view.getByTestId(T.share.revokeButton)).toBeTruthy());
    fireEvent.press(view.getByTestId(T.share.revokeButton)); await act(async () => { await confirmRevoke(); });
    expect(revoke).toHaveBeenCalledWith(stored.id); expect(revoke).not.toHaveBeenCalledWith(fixture.token);
  });
  it('removes the passcode and send controls after the new link is revoked', async () => {
    const view = await generated(); fireEvent.press(view.getByTestId(T.share.revokeButton));
    await act(async () => { await confirmRevoke(); });
    expect(view.queryByTestId(T.share.sendButton)).toBeNull(); expect(view.queryByText(fixture.passcode)).toBeNull();
    expect(view.getByTestId(T.share.createButton)).toBeTruthy();
  });
  it('does not remove active rows when revocation fails', async () => {
    revoke.mockRejectedValue(new Error('offline')); list.mockResolvedValue({ links: [stored] }); const view = await screen();
    fireEvent.press(view.getByTestId(T.share.revokeButton)); await act(async () => { await confirmRevoke(); });
    expect(alert).toHaveBeenLastCalledWith('Error', 'Failed to revoke link.');
    expect(view.getByTestId(T.share.revokeButton)).toBeTruthy();
  });
  it('does not submit duplicate revokes while the confirmed request is pending', async () => {
    let resolve!: () => void;
    revoke.mockReturnValue(new Promise<void>(done => { resolve = done; }));
    list.mockResolvedValue({ links: [stored] }); const view = await screen();
    fireEvent.press(view.getByTestId(T.share.revokeButton));
    await act(async () => { void confirmRevoke(); void confirmRevoke(); });
    expect(revoke).toHaveBeenCalledTimes(1); expect(view.getByText('Revoking...')).toBeTruthy();
    await act(async () => { resolve(); }); expect(view.queryByTestId(T.share.revokeButton)).toBeNull();
  });
});
