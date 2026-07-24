import { getLocales } from 'expo-localization';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { api } from '../api/client';

export type RegionCurrency = 'inr' | 'usd' | 'eur';

const STORAGE_KEY = 'dreamlog_region_currency';
const COUNTRY_KEY = 'dreamlog_region_country';

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
// Also caches the raw country code so crisis helplines can be localised (see
// ./helplines).
export async function setRegionFromCountry(code: string | undefined | null): Promise<RegionCurrency> {
  const currency = currencyForCountry(code);
  await AsyncStorage.setItem(STORAGE_KEY, currency);
  await cacheUserCountry(code);
  return currency;
}

// ── Country (for crisis helplines) ────────────────────────────────────────────
// Independent of currency: helplines are localised per-country, not per-currency.

// Stores the user's chosen ISO 3166-1 alpha-2 country. Pass null/"OTHER" to
// clear it (falls back to international helplines).
export async function cacheUserCountry(code: string | undefined | null): Promise<void> {
  const c = (code ?? '').toUpperCase();
  if (c && c !== 'OTHER') {
    await AsyncStorage.setItem(COUNTRY_KEY, c);
  } else {
    await AsyncStorage.removeItem(COUNTRY_KEY);
  }
}

export async function getCachedCountry(): Promise<string | null> {
  return AsyncStorage.getItem(COUNTRY_KEY);
}

// Resolves the user's country (ISO alpha-2) for helpline localisation.
// Precedence: cached (chosen at onboarding) → profile → device locale.
// Returns null when nothing is known, so callers show international resources.
export async function detectUserCountry(): Promise<string | null> {
  const cached = await AsyncStorage.getItem(COUNTRY_KEY);
  if (cached) return cached;

  // Profile country is authoritative (asked at account creation).
  try {
    const user = await api.me();
    if (user.country) {
      await cacheUserCountry(user.country);
      return user.country.toUpperCase();
    }
  } catch { /* not signed in yet, or network error - fall back to locale */ }

  // Device locale as a last resort; not cached (it's only a guess).
  try {
    const region = getLocales()[0]?.regionCode;
    if (region) return region.toUpperCase();
  } catch { /* ignore */ }

  return null;
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

// Annual passes (one-time 365-day pass, mirrors the monthly 30-day pass model).
// India discount is deliberately shallower than global: INR margins are FX-exposed
// (see docs/PRICING.md §"Annual passes" for the floor math).
export const PLAN_PRICE_ANNUAL_SHORT: Record<'plus' | 'pro', Record<RegionCurrency, string>> = {
  plus: { inr: '₹1,999', usd: '$39.99', eur: '€39.99' },
  pro:  { inr: '₹4,499', usd: '$79.99', eur: '€79.99' },
};

export const PLAN_PRICE_ANNUAL: Record<'plus' | 'pro', Record<RegionCurrency, string>> = {
  plus: { inr: '₹1,999 / year', usd: '$39.99 / year', eur: '€39.99 / year' },
  pro:  { inr: '₹4,499 / year', usd: '$79.99 / year', eur: '€79.99 / year' },
};

// Per-month equivalent of the annual pass, shown under the annual price.
export const PLAN_PRICE_ANNUAL_MONTHLY_EQUIV: Record<'plus' | 'pro', Record<RegionCurrency, string>> = {
  plus: { inr: '₹167 / mo', usd: '$3.33 / mo', eur: '€3.33 / mo' },
  pro:  { inr: '₹375 / mo', usd: '$6.67 / mo', eur: '€6.67 / mo' },
};

// Savings vs 12 monthly passes, rounded down to a whole percent.
export const PLAN_ANNUAL_SAVINGS: Record<'plus' | 'pro', Record<RegionCurrency, string>> = {
  plus: { inr: 'Save 33%', usd: 'Save 44%', eur: 'Save 44%' },
  pro:  { inr: 'Save 25%', usd: 'Save 33%', eur: 'Save 33%' },
};

// Largest savings per currency - shown on the Annual toggle segment.
export const MAX_ANNUAL_SAVINGS: Record<RegionCurrency, string> = {
  inr: 'Save up to 33%',
  usd: 'Save up to 44%',
  eur: 'Save up to 44%',
};
