import { getLocales } from 'expo-localization';
import AsyncStorage from '@react-native-async-storage/async-storage';

export type RegionCurrency = 'inr' | 'usd';

const STORAGE_KEY = 'dreamlog_region_currency';

// Detects whether the user's device is set to India's region.
// Result is cached so subsequent reads are instant.
// Detection checks both regionCode and currencyCode - many Indian Android devices
// use "English (US)" as their language (regionCode: 'US') but still report
// currencyCode: 'INR', so checking currency is more reliable.
export async function detectAndCacheRegion(): Promise<RegionCurrency> {
  const cached = await AsyncStorage.getItem(STORAGE_KEY);
  if (cached === 'inr' || cached === 'usd') return cached;

  try {
    const locales = getLocales();
    const primary = locales[0];
    const isIndia =
      primary?.regionCode === 'IN' ||
      primary?.currencyCode === 'INR' ||
      locales.some((l) => l.regionCode === 'IN' || l.currencyCode === 'INR');
    const currency: RegionCurrency = isIndia ? 'inr' : 'usd';
    await AsyncStorage.setItem(STORAGE_KEY, currency);
    return currency;
  } catch {
    return 'usd';
  }
}

// Force re-detection and overwrite the cached value.
// Call this if the cached value is known to be stale.
export async function resetAndDetectRegion(): Promise<RegionCurrency> {
  await AsyncStorage.removeItem(STORAGE_KEY);
  return detectAndCacheRegion();
}

export async function getCachedRegion(): Promise<RegionCurrency | null> {
  const v = await AsyncStorage.getItem(STORAGE_KEY);
  if (v === 'inr' || v === 'usd') return v;
  return null;
}

// Pricing helpers - use these everywhere instead of hardcoding ₹ strings.
// Canonical prices live in docs/PRICING.md - keep in sync.
export const THERAPY_SESSION_PRICE: Record<RegionCurrency, string> = {
  inr: '₹499',
  usd: '$7.99',
};

// Discounted extra-session price for Pro members (beyond the included session).
export const THERAPY_MEMBER_SESSION_PRICE: Record<RegionCurrency, string> = {
  inr: '₹299',
  usd: '$4.99',
};

export const PLAN_PRICE: Record<'plus' | 'pro', Record<RegionCurrency, string>> = {
  plus: { inr: '₹249 / month', usd: '$5.99 / month' },
  pro:  { inr: '₹499 / month', usd: '$9.99 / month' },
};
