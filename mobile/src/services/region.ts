import { getLocales } from 'expo-localization';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { api } from '../api/client';

export type RegionCurrency = 'inr' | 'usd' | 'eur';

const STORAGE_KEY = 'dreamlog_region_currency';

// European countries (EU + EEA + UK + others) - all shown EUR pricing.
const EUROPE_COUNTRY_CODES = new Set([
  'AD', 'AL', 'AT', 'BA', 'BE', 'BG', 'CH', 'CY', 'CZ', 'DE', 'DK', 'EE',
  'ES', 'FI', 'FR', 'GB', 'GR', 'HR', 'HU', 'IE', 'IS', 'IT', 'LI', 'LT',
  'LU', 'LV', 'MC', 'MD', 'ME', 'MK', 'MT', 'NL', 'NO', 'PL', 'PT', 'RO',
  'RS', 'SE', 'SI', 'SK', 'SM', 'UA', 'VA',
]);

function isCurrency(v: string | null): v is RegionCurrency {
  return v === 'inr' || v === 'usd' || v === 'eur';
}

// Maps an ISO 3166-1 alpha-2 country code to the display currency:
// India → INR, Europe → EUR, everywhere else → USD.
export function currencyForCountry(code: string | undefined | null): RegionCurrency {
  const c = (code ?? '').toUpperCase();
  if (c === 'IN') return 'inr';
  if (EUROPE_COUNTRY_CODES.has(c)) return 'eur';
  return 'usd';
}

// Caches the currency derived from the country the user picked at account
// creation (onboarding "Where are you based?" step). Call after PUT /me.
export async function setRegionFromCountry(code: string | undefined | null): Promise<RegionCurrency> {
  const currency = currencyForCountry(code);
  await AsyncStorage.setItem(STORAGE_KEY, currency);
  return currency;
}

// Resolves the user's display currency. Precedence:
//   1. cached value (instant)
//   2. the country on the user's profile (asked at account creation)
//   3. device locale - many Indian Android devices use "English (US)" as their
//      language (regionCode: 'US') but still report currencyCode: 'INR', so
//      checking currency is more reliable than region alone.
export async function detectAndCacheRegion(): Promise<RegionCurrency> {
  const cached = await AsyncStorage.getItem(STORAGE_KEY);
  if (isCurrency(cached)) return cached;

  // Profile country is the authoritative source.
  try {
    const user = await api.me();
    if (user.country) {
      return await setRegionFromCountry(user.country);
    }
  } catch { /* not signed in yet, or network error - fall back to locale */ }

  try {
    const locales = getLocales();
    const primary = locales[0];
    let currency: RegionCurrency = 'usd';
    if (
      primary?.regionCode === 'IN' ||
      primary?.currencyCode === 'INR' ||
      locales.some((l) => l.regionCode === 'IN' || l.currencyCode === 'INR')
    ) {
      currency = 'inr';
    } else if (
      primary?.currencyCode === 'EUR' ||
      EUROPE_COUNTRY_CODES.has(primary?.regionCode ?? '') ||
      locales.some((l) => l.currencyCode === 'EUR' || EUROPE_COUNTRY_CODES.has(l.regionCode ?? ''))
    ) {
      currency = 'eur';
    }
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
  return isCurrency(v) ? v : null;
}

// Pricing helpers - use these everywhere instead of hardcoding ₹ strings.
// Canonical prices live in docs/PRICING.md - keep in sync.
export const THERAPY_SESSION_PRICE: Record<RegionCurrency, string> = {
  inr: '₹499',
  usd: '$7.99',
  eur: '€7.99',
};

// Discounted extra-session price for Pro members (beyond the included session).
export const THERAPY_MEMBER_SESSION_PRICE: Record<RegionCurrency, string> = {
  inr: '₹299',
  usd: '$4.99',
  eur: '€4.99',
};

// Bare plan prices (no "/ month") for payment button labels.
export const PLAN_PRICE_SHORT: Record<'plus' | 'pro', Record<RegionCurrency, string>> = {
  plus: { inr: '₹249', usd: '$5.99', eur: '€5.99' },
  pro:  { inr: '₹499', usd: '$9.99', eur: '€9.99' },
};

export const PLAN_PRICE: Record<'plus' | 'pro', Record<RegionCurrency, string>> = {
  plus: { inr: '₹249 / month', usd: '$5.99 / month', eur: '€5.99 / month' },
  pro:  { inr: '₹499 / month', usd: '$9.99 / month', eur: '€9.99 / month' },
};
