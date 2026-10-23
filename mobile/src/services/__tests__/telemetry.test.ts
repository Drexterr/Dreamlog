// Sentry breadcrumbs for the upload path: allowlisted, PII-free data only, and
// a silent no-op whenever Sentry is unavailable.

const mockAddBreadcrumb = jest.fn();
jest.mock('@sentry/react-native', () => ({ addBreadcrumb: (...args: unknown[]) => mockAddBreadcrumb(...args) }));

function loadTelemetry(): typeof import('../telemetry') {
  let mod!: typeof import('../telemetry');
  jest.isolateModules(() => {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    mod = require('../telemetry');
  });
  return mod;
}

const ORIGINAL_DSN = process.env.EXPO_PUBLIC_SENTRY_DSN;
beforeEach(() => {
  mockAddBreadcrumb.mockReset();
  process.env.EXPO_PUBLIC_SENTRY_DSN = 'https://public@sentry.invalid/1';
});
afterAll(() => {
  if (ORIGINAL_DSN === undefined) delete process.env.EXPO_PUBLIC_SENTRY_DSN;
  else process.env.EXPO_PUBLIC_SENTRY_DSN = ORIGINAL_DSN;
});

describe('uploadBreadcrumb', () => {
  it('records an upload-category breadcrumb with the given level', () => {
    const { uploadBreadcrumb } = loadTelemetry();
    uploadBreadcrumb('upload: storage put', { phase: 'put', attempt: 2, size: '1-5MB', mode: 'dream', resumed: false }, 'warning');
    expect(mockAddBreadcrumb).toHaveBeenCalledWith({
      category: 'upload',
      message: 'upload: storage put',
      level: 'warning',
      data: { phase: 'put', attempt: 2, size: '1-5MB', mode: 'dream', resumed: false },
    });
  });

  it('defaults to info and omits empty data', () => {
    const { uploadBreadcrumb } = loadTelemetry();
    uploadBreadcrumb('upload: started');
    expect(mockAddBreadcrumb).toHaveBeenCalledWith({ category: 'upload', message: 'upload: started', level: 'info', data: undefined });
  });

  it('drops anything that could carry PII: unknown keys, keys/URLs/paths, non-finite numbers', () => {
    const { uploadBreadcrumb } = loadTelemetry();
    const sneaky = {
      phase: 'put',
      mode: 'audio/0b6c1d2e-user/clip.m4a',           // a storage key smuggled as a mode
      size: 'https://bucket.invalid/x?X-Amz-Signature=abc',
      status: Number.NaN,
      audio_key: 'audio/user/clip.m4a',
      url: 'https://bucket.invalid/put',
      transcript: 'my private words',
      attempt: 1,
    } as unknown as import('../telemetry').UploadCrumbData;
    uploadBreadcrumb('upload: attempt failed', sneaky, 'warning');
    const crumb = mockAddBreadcrumb.mock.calls[0][0];
    expect(crumb.data).toEqual({ phase: 'put', attempt: 1 });
    expect(JSON.stringify(crumb)).not.toMatch(/audio\/|https?:|private|Signature/);
  });

  it('is a no-op when Sentry is not configured', () => {
    delete process.env.EXPO_PUBLIC_SENTRY_DSN;
    const { uploadBreadcrumb } = loadTelemetry();
    uploadBreadcrumb('upload: started', { attempt: 1 });
    expect(mockAddBreadcrumb).not.toHaveBeenCalled();
  });

  it('never throws when Sentry throws', () => {
    mockAddBreadcrumb.mockImplementation(() => { throw new Error('native bridge gone'); });
    const { uploadBreadcrumb } = loadTelemetry();
    expect(() => uploadBreadcrumb('upload: started', { attempt: 1 })).not.toThrow();
  });

  it('never throws when the native module is missing (Expo Go)', () => {
    jest.resetModules();
    jest.doMock('@sentry/react-native', () => { throw new Error('Cannot find native module'); });
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const mod = require('../telemetry') as typeof import('../telemetry');
    expect(() => mod.uploadBreadcrumb('upload: started')).not.toThrow();
    expect(mockAddBreadcrumb).not.toHaveBeenCalled();
    jest.resetModules();
    jest.doMock('@sentry/react-native', () => ({ addBreadcrumb: (...args: unknown[]) => mockAddBreadcrumb(...args) }));
  });

  it('is a no-op when the module has no addBreadcrumb', () => {
    jest.resetModules();
    jest.doMock('@sentry/react-native', () => ({}));
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const mod = require('../telemetry') as typeof import('../telemetry');
    expect(() => mod.uploadBreadcrumb('upload: started')).not.toThrow();
    expect(mockAddBreadcrumb).not.toHaveBeenCalled();
    jest.resetModules();
    jest.doMock('@sentry/react-native', () => ({ addBreadcrumb: (...args: unknown[]) => mockAddBreadcrumb(...args) }));
  });

  it('caches the module lookup and can be reset for tests', () => {
    const t = loadTelemetry();
    t.uploadBreadcrumb('a');
    t.uploadBreadcrumb('b');
    expect(mockAddBreadcrumb).toHaveBeenCalledTimes(2);
    delete process.env.EXPO_PUBLIC_SENTRY_DSN;
    t.uploadBreadcrumb('still cached');
    expect(mockAddBreadcrumb).toHaveBeenCalledTimes(3);
    t.__resetTelemetryForTests();
    t.uploadBreadcrumb('now disabled');
    expect(mockAddBreadcrumb).toHaveBeenCalledTimes(3);
  });
});

describe('sizeBucket', () => {
  it.each([
    [0, '<100KB'],
    [100 * 1024, '<100KB'],
    [200 * 1024, '100KB-1MB'],
    [1024 * 1024, '1-5MB'],
    [6 * 1024 * 1024, '5-20MB'],
    [25 * 1024 * 1024, '20-50MB'],
    [80 * 1024 * 1024, '>=50MB'],
    [-1, 'unknown'],
    [Number.NaN, 'unknown'],
    [undefined, 'unknown'],
  ])('%p bytes -> %s', (bytes, bucket) => {
    expect(loadTelemetry().sizeBucket(bytes as number | undefined)).toBe(bucket);
  });
});

describe('errorStatus', () => {
  it('reads an Axios-style status and ignores everything else', () => {
    const { errorStatus } = loadTelemetry();
    expect(errorStatus({ response: { status: 503 } })).toBe(503);
    expect(errorStatus(new Error('offline'))).toBeUndefined();
    expect(errorStatus(null)).toBeUndefined();
    expect(errorStatus({ response: { status: '500' } })).toBeUndefined();
  });
});
