import AsyncStorage from '@react-native-async-storage/async-storage';
import { api } from '../../api/client';
import { getAppRole, setAppRole, resolvePostAuthRoute } from '../postAuthRoute';
jest.mock('../../api/client', () => ({ api: { getTerms: jest.fn(), therapistMe: jest.fn(), me: jest.fn() } }));
const terms = api.getTerms as jest.Mock;
const therapist = api.therapistMe as jest.Mock;
const me = api.me as jest.Mock;
beforeEach(async () => {
  jest.clearAllMocks(); terms.mockReset(); therapist.mockReset(); me.mockReset(); await AsyncStorage.clear();
  terms.mockResolvedValue({ tos_accepted_at: '2026-10-12', tos_version: '1', current_version: '1' });
  me.mockResolvedValue({ goal: 'stress' }); therapist.mockResolvedValue({ id: 'therapist' });
});
describe('post-authentication routing', () => {
  it('defaults to journal role', async () => { expect(await getAppRole()).toBe('user'); });
  it('persists therapist role', async () => { await setAppRole('therapist'); expect(await getAppRole()).toBe('therapist'); });
  it('treats unknown stored role as user', async () => { await AsyncStorage.setItem('dreamlog_role', 'admin'); expect(await getAppRole()).toBe('user'); });
  it('gates unsigned terms before reading user data', async () => {
    terms.mockResolvedValue({ tos_accepted_at: null }); expect(await resolvePostAuthRoute()).toBe('/accept-terms'); expect(me).not.toHaveBeenCalled();
  });
  it('gates outdated terms for therapists too', async () => {
    await setAppRole('therapist'); terms.mockResolvedValue({ tos_accepted_at: 'date', tos_version: '1', current_version: '2' });
    expect(await resolvePostAuthRoute()).toBe('/accept-terms'); expect(therapist).not.toHaveBeenCalled();
  });
  it('routes returning journal user to tabs', async () => { expect(await resolvePostAuthRoute()).toBe('/(tabs)'); });
  it('routes user without a goal to onboarding', async () => {
    me.mockResolvedValue({ goal: null }); expect(await resolvePostAuthRoute()).toBe('/onboarding');
  });
  it('routes registered therapist to workspace', async () => {
    await setAppRole('therapist'); expect(await resolvePostAuthRoute()).toBe('/therapist'); expect(me).not.toHaveBeenCalled();
  });
  it('routes unregistered therapist to registration', async () => {
    await setAppRole('therapist'); therapist.mockRejectedValue({ response: { status: 404 } }); expect(await resolvePostAuthRoute()).toBe('/therapist/register');
  });
  it('skips only the terms lookup when explicitly requested', async () => {
    expect(await resolvePostAuthRoute({ skipTermsGate: true })).toBe('/(tabs)'); expect(terms).not.toHaveBeenCalled(); expect(me).toHaveBeenCalled();
  });
  it('keeps sign-in usable when terms service is temporarily offline', async () => {
    terms.mockRejectedValue(new Error('offline')); expect(await resolvePostAuthRoute()).toBe('/(tabs)');
  });
  it('keeps a journal destination when profile fetch fails', async () => {
    me.mockRejectedValue(new Error('offline')); expect(await resolvePostAuthRoute()).toBe('/(tabs)');
  });
});
