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
