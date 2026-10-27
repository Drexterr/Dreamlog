// The EAS cloud build has no google-services.json (it is gitignored): app.config.js
// must point Android at the GOOGLE_SERVICES_JSON file variable when EAS provides it.
// eslint-disable-next-line @typescript-eslint/no-require-imports
const makeConfig = require('../app.config.js') as (ctx: { config: Record<string, any> }) => Record<string, any>;

const base = { name: 'Ode', android: { package: 'com.talktoode.app', googleServicesFile: './google-services.json', permissions: ['RECORD_AUDIO'] } };

describe('app.config.js', () => {
  const saved = process.env.GOOGLE_SERVICES_JSON;
  afterEach(() => {
    if (saved === undefined) delete process.env.GOOGLE_SERVICES_JSON;
    else process.env.GOOGLE_SERVICES_JSON = saved;
  });

  it('uses the EAS file variable when it is set', () => {
    process.env.GOOGLE_SERVICES_JSON = '/tmp/eas-build/google-services.json';
    expect(makeConfig({ config: base }).android.googleServicesFile).toBe('/tmp/eas-build/google-services.json');
  });

  it('keeps the local path from app.json when the variable is absent or empty', () => {
    delete process.env.GOOGLE_SERVICES_JSON;
    expect(makeConfig({ config: base }).android.googleServicesFile).toBe('./google-services.json');
    process.env.GOOGLE_SERVICES_JSON = '';
    expect(makeConfig({ config: base }).android.googleServicesFile).toBe('./google-services.json');
  });

  it('leaves every other setting untouched', () => {
    process.env.GOOGLE_SERVICES_JSON = '/x/google-services.json';
    const out = makeConfig({ config: base });
    expect(out.name).toBe('Ode');
    expect(out.android.package).toBe('com.talktoode.app');
    expect(out.android.permissions).toEqual(['RECORD_AUDIO']);
  });

  it('still works when app.json has no android block', () => {
    delete process.env.GOOGLE_SERVICES_JSON;
    expect(() => makeConfig({ config: { name: 'Ode' } })).not.toThrow();
  });
});
