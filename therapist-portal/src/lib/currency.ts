'use client';

import { useEffect, useState } from 'react';

export type Currency = 'INR' | 'EUR' | 'USD';

// Mirrors mobile/src/services/region.ts EUROPE_COUNTRY_CODES - keep in sync.
// European countries (EU + EEA + UK + others) - all shown EUR pricing.
export const EUROPE_COUNTRY_CODES = new Set([
  'AD', 'AL', 'AT', 'BA', 'BE', 'BG', 'CH', 'CY', 'CZ', 'DE', 'DK', 'EE',
  'ES', 'FI', 'FR', 'GB', 'GR', 'HR', 'HU', 'IE', 'IS', 'IT', 'LI', 'LT',
  'LU', 'LV', 'MC', 'MD', 'ME', 'MK', 'MT', 'NL', 'NO', 'PL', 'PT', 'RO',
  'RS', 'SE', 'SI', 'SK', 'SM', 'UA', 'VA',
]);

// India -> INR, Europe -> EUR, everywhere else -> USD.
export function currencyForCountry(code: string | undefined | null): Currency {
  const c = (code ?? '').toUpperCase();
  if (c === 'IN') return 'INR';
  if (EUROPE_COUNTRY_CODES.has(c)) return 'EUR';
  return 'USD';
}

// Detects the visitor's currency from location. Precedence:
//   1. India/Europe timezone shortcut (instant, no network round trip)
//   2. IP geolocation (ipapi.co) - covers every other country code
//   3. USD fallback if geolocation fails or times out
export function useCurrency(): { currency: Currency; ready: boolean } {
  const [currency, setCurrency] = useState<Currency>('USD');
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (tz === 'Asia/Calcutta' || tz === 'Asia/Kolkata') {
      setCurrency('INR');
      setReady(true);
      return;
    }
    // Common European timezones resolve instantly without a network call;
    // ipapi.co below is still the source of truth for everything else.
    if (tz.startsWith('Europe/')) {
      setCurrency('EUR');
      setReady(true);
      return;
    }

    fetch('https://ipapi.co/json/')
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => setCurrency(currencyForCountry(d?.country_code)))
      .catch(() => {})
      .finally(() => setReady(true));
  }, []);

  return { currency, ready };
}
