import AsyncStorage from '@react-native-async-storage/async-storage';
import type { UserGoal, AgeRange } from '../types';

const ONBOARDING_DONE = '@dreamlog/onboarding_done';
const GUEST_GOAL      = '@dreamlog/guest_goal';
const GUEST_NAME      = '@dreamlog/guest_name';
const GUEST_AGE       = '@dreamlog/guest_age';
const GUEST_COUNTRY   = '@dreamlog/guest_country';

export async function markOnboardingDone(): Promise<void> {
  await AsyncStorage.setItem(ONBOARDING_DONE, '1');
}

export async function hasCompletedOnboarding(): Promise<boolean> {
  const val = await AsyncStorage.getItem(ONBOARDING_DONE);
  return val === '1';
}

export async function saveGuestPreferences(prefs: {
  goal?: UserGoal;
  name?: string;
  ageRange?: AgeRange;
  country?: string;
}): Promise<void> {
  const pairs: [string, string][] = [];
  if (prefs.goal)     pairs.push([GUEST_GOAL, prefs.goal]);
  if (prefs.name)     pairs.push([GUEST_NAME, prefs.name]);
  if (prefs.ageRange) pairs.push([GUEST_AGE, prefs.ageRange]);
  if (prefs.country)  pairs.push([GUEST_COUNTRY, prefs.country]);
  if (pairs.length > 0) await AsyncStorage.multiSet(pairs);
}

export async function loadGuestPreferences(): Promise<{
  goal: UserGoal | null;
  name: string | null;
  ageRange: AgeRange | null;
  country: string | null;
}> {
  const [[, goal], [, name], [, ageRange], [, country]] = await AsyncStorage.multiGet([
    GUEST_GOAL, GUEST_NAME, GUEST_AGE, GUEST_COUNTRY,
  ]);
  return {
    goal: (goal ?? null) as UserGoal | null,
    name: name ?? null,
    ageRange: (ageRange ?? null) as AgeRange | null,
    country: country ?? null,
  };
}

export async function clearGuestPreferences(): Promise<void> {
  await AsyncStorage.multiRemove([GUEST_GOAL, GUEST_NAME, GUEST_AGE, GUEST_COUNTRY]);
}
