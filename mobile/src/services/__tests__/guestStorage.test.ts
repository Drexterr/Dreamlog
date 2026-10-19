import AsyncStorage from '@react-native-async-storage/async-storage';
import { clearGuestPreferences, hasCompletedOnboarding, loadGuestPreferences, markOnboardingDone, markTourPending, saveGuestPreferences } from '../guestStorage';
beforeEach(async () => { jest.clearAllMocks(); await AsyncStorage.clear(); });
describe('guest onboarding persistence', () => {
  it('starts without onboarding completion or guest preferences', async () => {
    expect(await hasCompletedOnboarding()).toBe(false);
    expect(await loadGuestPreferences()).toEqual({ goal: null, name: null, ageRange: null, country: null });
  });
  it('persists completion across later reads', async () => { await markOnboardingDone(); expect(await hasCompletedOnboarding()).toBe(true); });
  it('does not treat an arbitrary stored value as completed', async () => {
    await AsyncStorage.setItem('@dreamlog/onboarding_done', 'false'); expect(await hasCompletedOnboarding()).toBe(false);
  });
  it('persists the pending tour flag', async () => { await markTourPending(); expect(await AsyncStorage.getItem('@dreamlog/tour_pending')).toBe('1'); });
  it('round trips all guest preferences', async () => {
    await saveGuestPreferences({ goal: 'stress', name: 'QA', ageRange: '25_34', country: 'IN' });
    expect(await loadGuestPreferences()).toEqual({ goal: 'stress', name: 'QA', ageRange: '25_34', country: 'IN' });
  });
  it('partial changes preserve other preferences', async () => {
    await saveGuestPreferences({ goal: 'stress', name: 'QA' }); await saveGuestPreferences({ country: 'IN' });
    expect(await loadGuestPreferences()).toEqual({ goal: 'stress', name: 'QA', ageRange: null, country: 'IN' });
  });
  it('empty preference update performs no storage write', async () => {
    await saveGuestPreferences({}); expect(AsyncStorage.multiSet).not.toHaveBeenCalled();
  });
  it('clears guest profile without clearing onboarding or auth storage', async () => {
    await markOnboardingDone(); await AsyncStorage.setItem('dreamlog_role', 'therapist'); await saveGuestPreferences({ name: 'QA' });
    await clearGuestPreferences(); expect((await loadGuestPreferences()).name).toBeNull();
    expect(await hasCompletedOnboarding()).toBe(true); expect(await AsyncStorage.getItem('dreamlog_role')).toBe('therapist');
  });
  it('preserves Unicode names', async () => {
    await saveGuestPreferences({ name: '\u092d\u093e\u0930\u0924' }); expect((await loadGuestPreferences()).name).toBe('\u092d\u093e\u0930\u0924');
  });
});
