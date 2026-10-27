import fs from 'fs';
import os from 'os';
import path from 'path';
// eslint-disable-next-line @typescript-eslint/no-require-imports
const { copyGoogleServices } = require('../scripts/eas-google-services.js') as {
  copyGoogleServices: (env: Record<string, string | undefined>, dest: string, log: () => void) => { status: string };
};

// Bare Android project: Gradle reads android/app/google-services.json, which is
// gitignored, so EAS cloud builds must copy it from the GOOGLE_SERVICES_JSON file variable.
describe('eas-build-post-install google-services hook', () => {
  let dir: string;
  let dest: string;
  let source: string;
  const log = () => undefined;
  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), 'eas-gs-'));
    dest = path.join(dir, 'android', 'app', 'google-services.json');
    source = path.join(dir, 'secret-copy.json');
    fs.writeFileSync(source, JSON.stringify({ project_info: { project_id: 'dreamlog-48f94' } }));
  });
  afterEach(() => fs.rmSync(dir, { recursive: true, force: true }));

  it('copies the file variable into the Android app folder on an EAS Android build', () => {
    const r = copyGoogleServices({ EAS_BUILD: 'true', EAS_BUILD_PLATFORM: 'android', GOOGLE_SERVICES_JSON: source }, dest, log);
    expect(r.status).toBe('copied');
    expect(JSON.parse(fs.readFileSync(dest, 'utf8')).project_info.project_id).toBe('dreamlog-48f94');
  });

  it('does nothing outside EAS (local installs) and for iOS builds', () => {
    expect(copyGoogleServices({ GOOGLE_SERVICES_JSON: source }, dest, log).status).toBe('skipped');
    expect(copyGoogleServices({ EAS_BUILD: 'true', EAS_BUILD_PLATFORM: 'ios', GOOGLE_SERVICES_JSON: source }, dest, log).status).toBe('skipped');
    expect(fs.existsSync(dest)).toBe(false);
  });

  it('fails early with instructions when an Android cloud build has no file at all', () => {
    expect(() => copyGoogleServices({ EAS_BUILD: 'true', EAS_BUILD_PLATFORM: 'android' }, dest, log)).toThrow(/GOOGLE_SERVICES_JSON/);
  });

  it('accepts a file already in place when the variable is absent', () => {
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    fs.writeFileSync(dest, '{}');
    expect(copyGoogleServices({ EAS_BUILD: 'true', EAS_BUILD_PLATFORM: 'android' }, dest, log).status).toBe('present');
  });

  it('rejects a variable pointing at a missing file or at invalid JSON', () => {
    const env = { EAS_BUILD: 'true', EAS_BUILD_PLATFORM: 'android' };
    expect(() => copyGoogleServices({ ...env, GOOGLE_SERVICES_JSON: path.join(dir, 'nope.json') }, dest, log)).toThrow(/does not exist/);
    fs.writeFileSync(source, 'not json');
    expect(() => copyGoogleServices({ ...env, GOOGLE_SERVICES_JSON: source }, dest, log)).toThrow();
    expect(fs.existsSync(dest)).toBe(false);
  });
});
