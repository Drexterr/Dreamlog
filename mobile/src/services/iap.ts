import { Platform } from 'react-native';
import type { Purchase } from 'expo-iap';
import type { BillingPeriod } from '../types';

// Store SKUs. Must match the products configured in App Store Connect and the
// Play Console, and the catalogue in backend/internal/services/iap.go - keep
// all three in sync. Passes are consumable products (30-day / 365-day passes),
// not auto-renewing subscriptions.
export const IAP_PRODUCT_IDS: Record<'plus' | 'pro', Record<BillingPeriod, string>> = {
  plus: {
    monthly: 'com.talktoode.app.plus.monthly',
    annual: 'com.talktoode.app.plus.annual',
  },
  pro: {
    monthly: 'com.talktoode.app.pro.monthly',
    annual: 'com.talktoode.app.pro.annual',
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
  // Store transaction ID (iOS): pins the exact receipt entry being redeemed so
  // the backend never matches an older, already-used transaction.
  transactionId?: string;
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
  return toStorePurchase(iap, purchase, sku);
}

// Builds the verification payload for a store purchase.
async function toStorePurchase(iap: IapModule, purchase: Purchase, sku: string): Promise<StorePurchase> {
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
    transactionId: Platform.OS === 'ios' ? purchase.id : undefined,
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

// Maps a store SKU back to the plan pass it grants.
export function planForSku(sku: string): { plan: 'plus' | 'pro'; period: BillingPeriod } | null {
  for (const plan of ['plus', 'pro'] as const) {
    for (const period of ['monthly', 'annual'] as const) {
      if (IAP_PRODUCT_IDS[plan][period] === sku) return { plan, period };
    }
  }
  return null;
}

let recovering = false;

// Re-redeems purchases the store still holds as unfinished - the app was killed
// or the network dropped between the store charging the user and the backend
// granting the plan. Call on launch once the user is signed in. Each purchase
// is redeemed through `redeem` (POST /billing/upgrade, idempotent for the same
// user) and finished only after it succeeds. Fail-silent: anything that fails
// stays unfinished and is retried on the next launch.
export async function recoverUnfinishedPurchases(
  redeem: (purchase: StorePurchase, plan: 'plus' | 'pro', period: BillingPeriod) => Promise<void>,
): Promise<void> {
  const iap = loadIap();
  if (!iap || recovering) return;
  recovering = true;
  try {
    await ensureConnected(iap);
    const pending = await iap.getAvailablePurchases();
    for (const purchase of pending ?? []) {
      const target = planForSku(purchase.productId);
      if (!target) continue;
      try {
        const storePurchase = await toStorePurchase(iap, purchase, purchase.productId);
        await redeem(storePurchase, target.plan, target.period);
        await finishPurchase(storePurchase);
      } catch {
        // Leave it unfinished - retried next launch, and on Android an
        // unacknowledged purchase is refunded by Play if never granted.
      }
    }
  } catch {
    // Store unavailable - try again next launch.
  } finally {
    recovering = false;
  }
}
