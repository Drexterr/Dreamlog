import { getLocales } from 'expo-localization';
import AsyncStorage from '@react-native-async-storage/async-storage';

export type RegionCurrency = 'inr' | 'usd';

const STORAGE_KEY = 'dreamlog_region_currency';

// Detects whether the user's device is set to India's region.
// Result is cached so subsequent reads are instant.
export async function detectAndCacheRegion(): Promise<RegionCurrency> {
  const cached = await AsyncStorage.getItem(STORAGE_KEY);
  if (cached === 'inr' || cached === 'usd') return cached;

  try {
    const locales = getLocales();
    // getLocales() returns locales in preference order; first entry is the primary locale
    const regionCode = locales[0]?.regionCode ?? '';
    const currency: RegionCurrency = regionCode === 'IN' ? 'inr' : 'usd';
    await AsyncStorage.setItem(STORAGE_KEY, currency);
    return currency;
  } catch {
    return 'usd';
  }
}

export async function getCachedRegion(): Promise<RegionCurrency | null> {
  const v = await AsyncStorage.getItem(STORAGE_KEY);
  if (v === 'inr' || v === 'usd') return v;
  return null;
}

// Pricing helpers — use these everywhere instead of hardcoding ₹ strings.
export const THERAPY_SESSION_PRICE: Record<RegionCurrency, string> = {
  inr: '₹499',
  usd: '$4.99',
};

export const PLAN_PRICE: Record<'plus' | 'pro', Record<RegionCurrency, string>> = {
  plus: { inr: '₹199 / month', usd: '$7.99 / month' },
  pro:  { inr: '₹499 / month', usd: '$14.99 / month' },
};
