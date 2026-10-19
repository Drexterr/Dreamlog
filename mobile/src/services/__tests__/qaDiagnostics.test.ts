import { logQaScreen, logQaOnboardingStep } from '../qaDiagnostics';

describe('QA diagnostics privacy', () => {
  const original = process.env.EXPO_PUBLIC_QA_DIAGNOSTICS;
  let spy: jest.SpyInstance;
  beforeEach(() => { spy = jest.spyOn(console, 'info').mockImplementation(() => {}); });
  afterEach(() => { spy.mockRestore(); process.env.EXPO_PUBLIC_QA_DIAGNOSTICS = original; });
  it('is disabled by default', () => {
    delete process.env.EXPO_PUBLIC_QA_DIAGNOSTICS;
    logQaScreen(['record']); expect(spy).not.toHaveBeenCalled();
  });
  it('logs feature categories without dynamic identifiers', () => {
    process.env.EXPO_PUBLIC_QA_DIAGNOSTICS = '1';
    logQaScreen(['reflection', 'private-entry-id']);
    expect(spy).toHaveBeenCalledWith('[ODE_QA] {"event":"screen_entered","stage":"reflection"}');
  });
  it('does not log unknown routes or personal strings', () => {
    process.env.EXPO_PUBLIC_QA_DIAGNOSTICS = '1';
    logQaScreen(['private@example.com']);
    expect(spy).toHaveBeenCalledWith('[ODE_QA] {"event":"screen_entered","stage":"other"}');
  });
  it('labels onboarding steps without values entered by the user', () => {
    process.env.EXPO_PUBLIC_QA_DIAGNOSTICS = '1';
    logQaOnboardingStep(3); logQaOnboardingStep('intro');
    expect(spy).toHaveBeenNthCalledWith(1, '[ODE_QA] {"event":"onboarding_step_entered","stage":"name"}');
    expect(spy).toHaveBeenNthCalledWith(2, '[ODE_QA] {"event":"onboarding_step_entered","stage":"intro"}');
  });
});
