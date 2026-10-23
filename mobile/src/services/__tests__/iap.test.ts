import type { StorePurchase } from '../iap';

// expo-iap is a native module; the service requires it lazily.
const mockIap = {
  initConnection: jest.fn(async () => true),
  getAvailablePurchases: jest.fn(),
  getReceiptDataIOS: jest.fn(async () => 'base64-receipt'),
  requestReceiptRefreshIOS: jest.fn(async () => 'refreshed-receipt'),
  finishTransaction: jest.fn(async () => undefined),
  requestPurchase: jest.fn(),
};
jest.mock('expo-iap', () => mockIap, { virtual: true });

function loadIap(): typeof import('../iap') {
  let mod!: typeof import('../iap');
  jest.isolateModules(() => {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    mod = require('../iap');
  });
  return mod;
}

const purchase = (productId: string, id = 'tx-' + productId) => ({
  id,
  productId,
  purchaseToken: 'play-token-' + productId,
  platform: 'ios',
});

beforeEach(() => jest.clearAllMocks());

describe('plan SKUs', () => {
  it('maps every store SKU back to its plan and period', () => {
    const { planForSku, IAP_PRODUCT_IDS } = loadIap();
    expect(planForSku(IAP_PRODUCT_IDS.plus.monthly)).toEqual({ plan: 'plus', period: 'monthly' });
    expect(planForSku(IAP_PRODUCT_IDS.pro.annual)).toEqual({ plan: 'pro', period: 'annual' });
    expect(planForSku('com.someone.else.sku')).toBeNull();
  });
});

// H11: purchases the store still holds as unfinished are redeemed on launch and
// finished only after the backend grants the plan.
describe('recoverUnfinishedPurchases', () => {
  it('redeems known SKUs, finishes only successful ones, skips unknown products', async () => {
    const { recoverUnfinishedPurchases, IAP_PRODUCT_IDS } = loadIap();
    mockIap.getAvailablePurchases.mockResolvedValue([
      purchase(IAP_PRODUCT_IDS.plus.monthly),
      purchase('com.unrelated.product'),
      purchase(IAP_PRODUCT_IDS.pro.annual),
    ]);
    const redeemed: Array<[StorePurchase, string, string]> = [];
    await recoverUnfinishedPurchases(async (p, plan, period) => {
      redeemed.push([p, plan, period]);
      if (plan === 'pro') throw new Error('backend unavailable');
    });

    expect(redeemed.map(([, plan, period]) => `${plan}/${period}`)).toEqual(['plus/monthly', 'pro/annual']);
    const [plus] = redeemed[0];
    expect(plus.productId).toBe(IAP_PRODUCT_IDS.plus.monthly);
    expect(plus.purchaseToken).toBe('base64-receipt');           // iOS: app receipt
    expect(plus.transactionId).toBe('tx-' + IAP_PRODUCT_IDS.plus.monthly);

    // Only the purchase the backend accepted is finished; the failed one stays
    // unfinished so the store re-delivers it next launch.
    expect(mockIap.finishTransaction).toHaveBeenCalledTimes(1);
    expect(mockIap.finishTransaction).toHaveBeenCalledWith(expect.objectContaining({ isConsumable: true }));
  });

  it('is fail-silent when the store is unavailable', async () => {
    const { recoverUnfinishedPurchases } = loadIap();
    mockIap.getAvailablePurchases.mockRejectedValue(new Error('store offline'));
    const redeem = jest.fn();
    await expect(recoverUnfinishedPurchases(redeem)).resolves.toBeUndefined();
    expect(redeem).not.toHaveBeenCalled();
  });
});

// N17: purchases carry the buyer's account token, and recovery never grants
// another account's unfinished purchase to whoever is signed in now.
describe('purchase account binding', () => {
  it('tags a new purchase with the account token on both stores', async () => {
    const { purchasePlanPass, IAP_PRODUCT_IDS } = loadIap();
    mockIap.requestPurchase.mockResolvedValue(purchase(IAP_PRODUCT_IDS.plus.monthly));
    await purchasePlanPass('plus', 'monthly', 'acct-1');
    expect(mockIap.requestPurchase).toHaveBeenCalledWith(expect.objectContaining({
      request: {
        apple: { sku: IAP_PRODUCT_IDS.plus.monthly, appAccountToken: 'acct-1' },
        google: { skus: [IAP_PRODUCT_IDS.plus.monthly], obfuscatedAccountId: 'acct-1' },
      },
    }));
  });

  it('leaves another account\'s purchase unfinished and redeems own and untagged ones', async () => {
    const { recoverUnfinishedPurchases, IAP_PRODUCT_IDS } = loadIap();
    const sku = IAP_PRODUCT_IDS.plus.monthly;
    mockIap.getAvailablePurchases.mockResolvedValue([
      { ...purchase(sku, 'tx-other'), appAccountToken: 'acct-OTHER' },
      { ...purchase(sku, 'tx-mine'), appAccountToken: 'ACCT-ME' },
      { ...purchase(sku, 'tx-android-mine'), obfuscatedAccountIdAndroid: 'acct-me' },
      purchase(sku, 'tx-untagged'),
    ]);
    const redeemed: string[] = [];
    await recoverUnfinishedPurchases(async (p) => { redeemed.push(p.raw.id); }, 'acct-me');
    expect(redeemed).toEqual(['tx-mine', 'tx-android-mine', 'tx-untagged']);
    expect(mockIap.finishTransaction).toHaveBeenCalledTimes(3);
  });
});

// Therapy pay-per-use: session credits are their own consumable SKUs.
describe('therapy session credits', () => {
  it('maps credit SKUs, and keeps them apart from plan SKUs', () => {
    const { therapyCreditForSku, planForSku, THERAPY_CREDIT_PRODUCT_IDS } = loadIap();
    expect(therapyCreditForSku(THERAPY_CREDIT_PRODUCT_IDS.single)).toBe('single');
    expect(therapyCreditForSku(THERAPY_CREDIT_PRODUCT_IDS.member)).toBe('member');
    expect(therapyCreditForSku('com.talktoode.app.plus.monthly')).toBeNull();
    expect(planForSku(THERAPY_CREDIT_PRODUCT_IDS.single)).toBeNull();
  });

  it('buys a credit tagged with the account token', async () => {
    const { purchaseTherapyCredit, THERAPY_CREDIT_PRODUCT_IDS } = loadIap();
    const sku = THERAPY_CREDIT_PRODUCT_IDS.member;
    mockIap.requestPurchase.mockResolvedValue([purchase(sku)]);
    const p = await purchaseTherapyCredit('member', 'acct-1');
    expect(mockIap.requestPurchase).toHaveBeenCalledWith(expect.objectContaining({
      type: 'in-app',
      request: {
        apple: { sku, appAccountToken: 'acct-1' },
        google: { skus: [sku], obfuscatedAccountId: 'acct-1' },
      },
    }));
    expect(p?.productId).toBe(sku);
    // The purchase is NOT finished here - only after the backend grants it.
    expect(mockIap.finishTransaction).not.toHaveBeenCalled();
  });

  it('throws when the store returns no purchase', async () => {
    const { purchaseTherapyCredit } = loadIap();
    mockIap.requestPurchase.mockResolvedValue([]);
    await expect(purchaseTherapyCredit('single')).rejects.toThrow('did not return a purchase');
  });

  it('recovery routes credit SKUs to the credit redeemer and finishes only successes', async () => {
    const { recoverUnfinishedPurchases, IAP_PRODUCT_IDS, THERAPY_CREDIT_PRODUCT_IDS } = loadIap();
    mockIap.getAvailablePurchases.mockResolvedValue([
      purchase(THERAPY_CREDIT_PRODUCT_IDS.single, 'tx-credit-ok'),
      purchase(IAP_PRODUCT_IDS.plus.monthly, 'tx-plan'),
      purchase(THERAPY_CREDIT_PRODUCT_IDS.member, 'tx-credit-fail'),
    ]);
    const plans: string[] = [];
    const creditsRedeemed: string[] = [];
    await recoverUnfinishedPurchases(
      async (p) => { plans.push(p.raw.id); },
      undefined,
      async (p, kind) => {
        creditsRedeemed.push(`${p.raw.id}:${kind}`);
        if (kind === 'member') throw new Error('403 not on Pro');
      },
    );
    expect(plans).toEqual(['tx-plan']);
    expect(creditsRedeemed).toEqual(['tx-credit-ok:single', 'tx-credit-fail:member']);
    expect(mockIap.finishTransaction).toHaveBeenCalledTimes(2);
  });

  it('leaves credit purchases unfinished when no credit redeemer is given', async () => {
    const { recoverUnfinishedPurchases, THERAPY_CREDIT_PRODUCT_IDS } = loadIap();
    mockIap.getAvailablePurchases.mockResolvedValue([purchase(THERAPY_CREDIT_PRODUCT_IDS.single)]);
    const redeem = jest.fn();
    await recoverUnfinishedPurchases(redeem);
    expect(redeem).not.toHaveBeenCalled();
    expect(mockIap.finishTransaction).not.toHaveBeenCalled();
  });

  it('never redeems another account\'s credit purchase', async () => {
    const { recoverUnfinishedPurchases, THERAPY_CREDIT_PRODUCT_IDS } = loadIap();
    mockIap.getAvailablePurchases.mockResolvedValue([
      { ...purchase(THERAPY_CREDIT_PRODUCT_IDS.single, 'tx-other'), appAccountToken: 'acct-other' },
    ]);
    const redeemCredit = jest.fn();
    await recoverUnfinishedPurchases(jest.fn(), 'acct-me', redeemCredit);
    expect(redeemCredit).not.toHaveBeenCalled();
  });
});
