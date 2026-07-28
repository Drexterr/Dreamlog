import { Platform } from 'react-native';
import type { Purchase } from 'expo-iap';
import type { BillingPeriod } from '../types';

// Store SKUs. Must match the products configured in App Store Connect and the
// Play Console, and the catalogue in backend/internal/services/iap.go - keep
// all three in sync. Passes are consumable products (30-day / 365-day passes),
// not auto-renewing subscriptions.
export const IAP_PRODUCT_IDS: Record<'plus' | 'pro', Record<BillingPeriod, string>> = {
  plus: {
    monthly: 'com.ode.app.plus.monthly',
    annual: 'com.ode.app.plus.annual',
  },
  pro: {
    monthly: 'com.ode.app.pro.monthly',
    annual: 'com.ode.app.pro.annual',
  },
};

// What the backend needs to verify the purchase server-side, plus the raw
// store purchase so it can be finished after the backend grants the plan.
export interface StorePurchase {
  platform: 'ios' | 'android';
  productId: string;
  // iOS: base64 app receipt (verified via Apple verifyReceipt).
  // Android: Play Billing purchase token (verified via the Play Developer API).
  purchaseToken: string;
  raw: Purchase;
}

type IapModule = typeof import('expo-iap');

let iapModule: IapModule | null | undefined;

// Lazy require: the native module is absent in Expo Go, where an import-time
// require would crash the whole app. Purchases then report unavailable and the
// upgrade screen falls back to the backend's dev-stub grant.
function loadIap(): IapModule | null {
  if (iapModule !== undefined) return iapModule;
  try {
    iapModule = require('expo-iap') as IapModule;
  } catch {
    iapModule = null;
  }
  return iapModule;
}

export function iapAvailable(): boolean {
  return loadIap() !== null;
}

let connected = false;

async function ensureConnected(iap: IapModule): Promise<void> {
  if (connected) return;
  await iap.initConnection();
  connected = true;
}

export function isUserCancelled(err: unknown): boolean {
  return (
    typeof err === 'object' &&
    err !== null &&
    (err as { code?: string }).code === 'user-cancelled'
  );
}

// Launches the native store purchase sheet for the given plan pass and returns
// what the backend needs for verification. Returns null when IAP is
// unavailable (Expo Go / simulator without store access). Throws on store
// errors, including user cancellation - check with isUserCancelled().
export async function purchasePlanPass(
  plan: 'plus' | 'pro',
  period: BillingPeriod,
): Promise<StorePurchase | null> {
  const iap = loadIap();
  if (!iap) return null;

  await ensureConnected(iap);
  const sku = IAP_PRODUCT_IDS[plan][period];

  const result = await iap.requestPurchase({
    type: 'in-app',
    request: {
      apple: { sku },
      google: { skus: [sku] },
    },
  });
  const purchase = Array.isArray(result) ? result[0] : result;
  if (!purchase) {
    throw new Error('The store did not return a purchase.');
  }

  let purchaseToken: string | null | undefined;
  if (Platform.OS === 'ios') {
    // verifyReceipt takes the base64 app receipt; right after a first purchase
    // the receipt file may not be on disk yet, so refresh as a fallback.
    purchaseToken = await iap.getReceiptDataIOS();
    if (!purchaseToken) {
      purchaseToken = await iap.requestReceiptRefreshIOS();
    }
  } else {
    purchaseToken = purchase.purchaseToken;
  }
  if (!purchaseToken) {
    throw new Error('The store did not return a purchase token.');
  }

  return {
    platform: Platform.OS === 'ios' ? 'ios' : 'android',
    productId: sku,
    purchaseToken,
    raw: purchase,
  };
}

// Marks the transaction consumed with the store. Call ONLY after the backend
// has verified the purchase and granted the plan: an unfinished transaction is
// re-delivered by the store on next launch (recoverable), a finished but
// unrecorded one is money lost.
export async function finishPurchase(purchase: StorePurchase): Promise<void> {
  const iap = loadIap();
  if (!iap) return;
  try {
    await iap.finishTransaction({ purchase: purchase.raw, isConsumable: true });
  } catch {
    // Non-fatal: the store re-delivers unfinished transactions.
  }
}
