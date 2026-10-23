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

// Therapy session credits (pay-per-use): one consumable SKU per price point,
// each granting one prepaid session. Must match IAPTherapyCatalog in
// backend/internal/services/iap.go. `member` is the Pro member price and is
// only redeemable while the account is on Pro - offer it to Pro users only.
export type TherapyCreditKind = 'single' | 'member';
export const THERAPY_CREDIT_PRODUCT_IDS: Record<TherapyCreditKind, string> = {
  single: 'com.talktoode.app.therapy.single',
  member: 'com.talktoode.app.therapy.member',
};

// Maps a store SKU back to the therapy credit it buys, or null.
export function therapyCreditForSku(sku: string): TherapyCreditKind | null {
  for (const kind of ['single', 'member'] as const) {
    if (THERAPY_CREDIT_PRODUCT_IDS[kind] === sku) return kind;
  }
  return null;
}

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
  accountToken?: string,
): Promise<StorePurchase | null> {
  const iap = loadIap();
  if (!iap) return null;

  await ensureConnected(iap);
  return buySku(iap, IAP_PRODUCT_IDS[plan][period], accountToken);
}

// Launches the store purchase sheet for one prepaid therapy session. Same
// contract as purchasePlanPass: null when IAP is unavailable (the dev backend
// then grants without verification), throws on store errors / cancellation.
export async function purchaseTherapyCredit(
  kind: TherapyCreditKind,
  accountToken?: string,
): Promise<StorePurchase | null> {
  const iap = loadIap();
  if (!iap) return null;

  await ensureConnected(iap);
  return buySku(iap, THERAPY_CREDIT_PRODUCT_IDS[kind], accountToken);
}

async function buySku(iap: IapModule, sku: string, accountToken?: string): Promise<StorePurchase> {
  const result = await iap.requestPurchase({
    type: 'in-app',
    // The account token binds the purchase to this account (Apple
    // appAccountToken / Play obfuscatedAccountId): the backend refuses it for
    // anyone else, and recovery leaves it for its owner after an account switch.
    request: {
      apple: { sku, ...(accountToken ? { appAccountToken: accountToken } : {}) },
      google: { skus: [sku], ...(accountToken ? { obfuscatedAccountId: accountToken } : {}) },
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

// The account token the purchase was tagged with, if any.
function purchaseAccountToken(purchase: Purchase): string | null {
  const p = purchase as Purchase & { appAccountToken?: string | null; obfuscatedAccountIdAndroid?: string | null };
  return p.appAccountToken ?? p.obfuscatedAccountIdAndroid ?? null;
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
// granting the plan. Call on launch once the user is signed in. Plan passes
// are redeemed through `redeem` (POST /billing/upgrade) and therapy credits
// through `redeemTherapyCredit` (POST /billing/therapy-credits) - both
// idempotent for the same user - and each purchase is finished only after its
// redeem succeeds. Fail-silent: anything that fails stays unfinished and is
// retried on the next launch. Without `redeemTherapyCredit`, therapy purchases
// are left unfinished for a caller that can redeem them.
export async function recoverUnfinishedPurchases(
  redeem: (purchase: StorePurchase, plan: 'plus' | 'pro', period: BillingPeriod) => Promise<void>,
  accountToken?: string,
  redeemTherapyCredit?: (purchase: StorePurchase, kind: TherapyCreditKind) => Promise<void>,
): Promise<void> {
  const iap = loadIap();
  if (!iap || recovering) return;
  recovering = true;
  try {
    await ensureConnected(iap);
    const pending = await iap.getAvailablePurchases();
    for (const purchase of pending ?? []) {
      const target = planForSku(purchase.productId);
      const creditKind = therapyCreditForSku(purchase.productId);
      if (!target && !(creditKind && redeemTherapyCredit)) continue;
      // A purchase made by another account on this device (a sign-in switch)
      // stays unfinished for its owner instead of being granted to whoever
      // is signed in now (N17). Untagged purchases predate tagging.
      const owner = purchaseAccountToken(purchase);
      if (owner && accountToken && owner.toLowerCase() !== accountToken.toLowerCase()) continue;
      try {
        const storePurchase = await toStorePurchase(iap, purchase, purchase.productId);
        if (target) {
          await redeem(storePurchase, target.plan, target.period);
        } else if (creditKind && redeemTherapyCredit) {
          await redeemTherapyCredit(storePurchase, creditKind);
        }
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
