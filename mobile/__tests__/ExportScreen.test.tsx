import React from 'react';
import { ActivityIndicator, Alert, StyleSheet } from 'react-native';
import { act, fireEvent, render } from '@testing-library/react-native';
import * as FileSystem from 'expo-file-system/legacy';
import * as Sharing from 'expo-sharing';
import ExportScreen from '../app/export';
import { api } from '../src/api/client';
import { T, exportPeriodID } from '../src/testIDs';
import { THEMES } from '../src/theme';

const mockBack = jest.fn();
jest.mock('expo-router', () => ({ useRouter: () => ({ back: mockBack }) }));
jest.mock('../src/context/ThemeContext', () => ({
  useTheme: () => ({ colors: require('../src/theme').THEMES.curious }),
}));
jest.mock('../src/api/client', () => ({ api: { exportPDFParams: jest.fn() } }));
jest.mock('expo-file-system/legacy', () => ({
  cacheDirectory: 'file:///mock-cache/', downloadAsync: jest.fn(),
}));
jest.mock('expo-sharing', () => ({ isAvailableAsync: jest.fn(), shareAsync: jest.fn() }));

const params = api.exportPDFParams as jest.Mock;
const download = FileSystem.downloadAsync as jest.Mock;
const available = Sharing.isAvailableAsync as jest.Mock;
const share = Sharing.shareAsync as jest.Mock;
const request = {
  url: 'https://fixture.invalid/export/pdf?period=monthly',
  headers: { Authorization: 'Bearer unit-only-token' },
};
const result = { status: 200, uri: 'file:///mock-cache/download-result.pdf' };
const shareOptions = {
  mimeType: 'application/pdf', dialogTitle: 'Share your Ode journal', UTI: 'com.adobe.pdf',
};
let alert: jest.SpyInstance;

beforeEach(() => {
  jest.clearAllMocks();
  params.mockReset().mockResolvedValue(request);
  download.mockReset().mockResolvedValue(result);
  available.mockReset().mockResolvedValue(true);
  share.mockReset().mockResolvedValue(undefined);
  Object.defineProperty(FileSystem, 'cacheDirectory', { value: 'file:///mock-cache/', configurable: true });
  jest.spyOn(Date.prototype, 'toISOString').mockReturnValue('2026-10-14T12:00:00.000Z');
  alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
});
afterEach(() => jest.restoreAllMocks());

type Screen = ReturnType<typeof render>;
async function exportPDF(view: Screen) {
  await act(async () => { fireEvent.press(view.getByTestId(T.exportScreen.exportButton)); });
}
function expectReady(view: Screen) {
  expect(view.getByTestId(T.exportScreen.exportButton)).toBeEnabled();
  expect(view.getByText('Generate & Download PDF')).toBeTruthy();
  expect(view.UNSAFE_queryByType(ActivityIndicator)).toBeNull();
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}

describe('export selection and mocked authorized handoff', () => {
  it('defaults to monthly and passes authorization unchanged to the legacy downloader', async () => {
    const view = render(<ExportScreen />);
    expect(StyleSheet.flatten(view.getByTestId(exportPeriodID('monthly')).props.style).borderColor).toBe(THEMES.curious.brand);
    expectReady(view);
    expect(params).not.toHaveBeenCalled();
    await exportPDF(view);
    expect(params).toHaveBeenCalledTimes(1);
    expect(params).toHaveBeenCalledWith('monthly');
    expect(download).toHaveBeenCalledTimes(1);
    expect(download).toHaveBeenCalledWith(request.url, 'file:///mock-cache/ode-monthly-2026-10.pdf', { headers: request.headers });
    expect(download.mock.calls[0][2].headers).toBe(request.headers);
    expect(available).toHaveBeenCalledTimes(1);
    expect(share).toHaveBeenCalledTimes(1);
    expect(share).toHaveBeenCalledWith(result.uri, shareOptions);
    expect(alert).not.toHaveBeenCalled();
    expectReady(view);
  }, 15000);

  it('selects yearly, then allows returning to monthly for another export', async () => {
    const view = render(<ExportScreen />);
    const yearlyRequest = { ...request, url: 'https://fixture.invalid/export/pdf?period=yearly' };
    params.mockResolvedValueOnce(yearlyRequest);
    fireEvent.press(view.getByTestId(exportPeriodID('yearly')));
    expect(StyleSheet.flatten(view.getByTestId(exportPeriodID('yearly')).props.style).borderColor).toBe(THEMES.curious.brand);
    expect(StyleSheet.flatten(view.getByTestId(exportPeriodID('monthly')).props.style).borderColor).toBe(THEMES.curious.borderFaint);
    await exportPDF(view);
    expect(params).toHaveBeenNthCalledWith(1, 'yearly');
    expect(download).toHaveBeenNthCalledWith(1, yearlyRequest.url, 'file:///mock-cache/ode-yearly-2026-10.pdf', { headers: yearlyRequest.headers });
    fireEvent.press(view.getByTestId(exportPeriodID('monthly')));
    await exportPDF(view);
    expect(params).toHaveBeenNthCalledWith(2, 'monthly');
    expectReady(view);
  });

  it('uses the filename fallback when cacheDirectory is unavailable', async () => {
    Object.defineProperty(FileSystem, 'cacheDirectory', { value: null, configurable: true });
    available.mockResolvedValue(false);
    const view = render(<ExportScreen />);
    await exportPDF(view);
    expect(download).toHaveBeenCalledWith(request.url, 'ode-monthly-2026-10.pdf', { headers: request.headers });
    expect(alert).toHaveBeenCalledWith('Saved', 'PDF saved to ode-monthly-2026-10.pdf');
    expect(share).not.toHaveBeenCalled();
    expectReady(view);
  });

  it('reports Saved without opening a share sheet when sharing is unavailable', async () => {
    available.mockResolvedValueOnce(false);
    const view = render(<ExportScreen />);
    await exportPDF(view);
    expect(alert).toHaveBeenCalledTimes(1);
    expect(alert).toHaveBeenCalledWith('Saved', 'PDF saved to file:///mock-cache/ode-monthly-2026-10.pdf');
    expect(share).not.toHaveBeenCalled();
    expectReady(view);
    await exportPDF(view);
    expect(share).toHaveBeenCalledTimes(1);
  });

  it('navigates back without exporting', () => {
    const view = render(<ExportScreen />);
    fireEvent.press(view.getByTestId(T.exportScreen.back));
    expect(mockBack).toHaveBeenCalledTimes(1);
    expect(params).not.toHaveBeenCalled();
    expect(download).not.toHaveBeenCalled();
  });
});

describe('negative paths and finally retry', () => {
  it.each([201, 204, 301, 401, 403, 500])('refuses to share HTTP %s and allows a successful retry', async status => {
    download.mockResolvedValueOnce({ ...result, status });
    const view = render(<ExportScreen />);
    await exportPDF(view);
    expect(alert).toHaveBeenCalledTimes(1);
    expect(alert).toHaveBeenCalledWith('Export failed', 'Could not generate your PDF. Please try again.');
    expect(available).not.toHaveBeenCalled();
    expect(share).not.toHaveBeenCalled();
    expectReady(view);
    await exportPDF(view);
    expect(params).toHaveBeenCalledTimes(2);
    expect(download).toHaveBeenCalledTimes(2);
    expect(share).toHaveBeenCalledTimes(1);
    expectReady(view);
  });

  it.each(['params', 'download', 'availability', 'share'] as const)('handles rejected %s operations and retries', async stage => {
    const operation = { params, download, availability: available, share }[stage];
    operation.mockRejectedValueOnce(new Error('unit-only failure'));
    const view = render(<ExportScreen />);
    await exportPDF(view);
    expect(alert).toHaveBeenCalledTimes(1);
    expect(alert).toHaveBeenCalledWith('Error', 'Something went wrong. Please try again.');
    expect(download).toHaveBeenCalledTimes(stage === 'params' ? 0 : 1);
    expect(available).toHaveBeenCalledTimes(['params', 'download'].includes(stage) ? 0 : 1);
    expect(share).toHaveBeenCalledTimes(stage === 'share' ? 1 : 0);
    expectReady(view);
    await exportPDF(view);
    expect(params).toHaveBeenCalledTimes(2);
    expect(share).toHaveBeenLastCalledWith(result.uri, shareOptions);
    expectReady(view);
  });

  it('catches a synchronous export-parameter exception and reenables retry', async () => {
    params.mockImplementationOnce(() => { throw new Error('sync unit-only failure'); });
    const view = render(<ExportScreen />);
    await exportPDF(view);
    expect(alert).toHaveBeenCalledWith('Error', 'Something went wrong. Please try again.');
    expect(download).not.toHaveBeenCalled();
    expectReady(view);
    await exportPDF(view);
    expect(share).toHaveBeenCalledTimes(1);
  });
});

describe('pending operation duplicate-tap protection', () => {
  it.each(['params', 'download', 'availability', 'share'] as const)('stays disabled while %s is pending', async stage => {
    const pending = deferred<unknown>();
    const operation = { params, download, availability: available, share }[stage];
    operation.mockReturnValueOnce(pending.promise);
    const view = render(<ExportScreen />);
    await exportPDF(view);
    expect(view.getByTestId(T.exportScreen.exportButton)).toBeDisabled();
    expect(view.queryByText('Generate & Download PDF')).toBeNull();
    expect(view.UNSAFE_getByType(ActivityIndicator)).toBeTruthy();
    fireEvent.press(view.getByTestId(T.exportScreen.exportButton));
    fireEvent.press(view.getByTestId(T.exportScreen.exportButton));
    expect(params).toHaveBeenCalledTimes(1);
    expect(operation).toHaveBeenCalledTimes(1);
    expect(alert).not.toHaveBeenCalled();
    await act(async () => {
      pending.resolve({ params: request, download: result, availability: true, share: undefined }[stage]);
    });
    expect(share).toHaveBeenCalledTimes(1);
    expectReady(view);
    await exportPDF(view);
    expect(params).toHaveBeenCalledTimes(2);
    expect(share).toHaveBeenCalledTimes(2);
    expectReady(view);
  });
});
