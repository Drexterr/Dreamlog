import React from 'react';
import { Alert } from 'react-native';
import { act, fireEvent, render, waitFor } from '@testing-library/react-native';
import TherapyPricingScreen from '../app/therapy/pricing';
import { api } from '../src/api/client';
import * as iap from '../src/services/iap';
import { T, therapyPricingCtaID } from '../src/testIDs';

const mockPush = jest.fn();
const mockBack = jest.fn();
jest.mock('expo-router', () => ({ useRouter: () => ({ push: mockPush, back: mockBack }) }));
jest.mock('../src/context/ThemeContext', () => ({
  useTheme: () => ({ colors: require('../src/theme').THEMES.curious }),
}));
jest.mock('../src/api/client', () => ({
  api: { getBillingPlan: jest.fn(), buyTherapyCredits: jest.fn() },
}));
jest.mock('../src/services/region', () => ({
  resetAndDetectRegion: jest.fn(async () => 'inr'),
  THERAPY_SESSION_PRICE: { inr: '₹499', usd: '$7.99', eur: '€7.99' },
  THERAPY_MEMBER_SESSION_PRICE: { inr: '₹299', usd: '$4.99', eur: '€4.99' },
  PLAN_PRICE_SHORT: { plus: { inr: '₹249', usd: '$5.99', eur: '€5.99' }, pro: { inr: '₹499', usd: '$9.99', eur: '€9.99' } },
}));
jest.mock('../src/services/iap', () => ({
  purchaseTherapyCredit: jest.fn(),
  finishPurchase: jest.fn(async () => undefined),
  isUserCancelled: (err: unknown) => (err as { code?: string })?.code === 'user-cancelled',
  THERAPY_CREDIT_PRODUCT_IDS: {
    single: 'com.talktoode.app.therapy.single',
    member: 'com.talktoode.app.therapy.member',
  },
}));

const getPlan = api.getBillingPlan as jest.Mock;
const buyCredits = api.buyTherapyCredits as jest.Mock;
const purchase = iap.purchaseTherapyCredit as jest.Mock;
const finish = iap.finishPurchase as jest.Mock;

const storePurchase = {
  platform: 'android' as const,
  productId: 'com.talktoode.app.therapy.single',
  purchaseToken: 'play-token',
  transactionId: undefined,
  raw: { id: 'GPA.1' },
};

function plan(p: string, credits = 0) {
  return { plan: p, plan_expires_at: null, limits: {}, all_plans: {}, purchase_account_token: 'acct-1', therapy_credits: credits };
}

let alert: jest.SpyInstance;
beforeEach(() => {
  jest.clearAllMocks();
  getPlan.mockResolvedValue(plan('free'));
  buyCredits.mockResolvedValue({ therapy_credits: 1, credits_granted: 1 });
  purchase.mockResolvedValue(storePurchase);
  alert = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
});
afterEach(() => jest.restoreAllMocks());

async function renderScreen() {
  const view = render(<TherapyPricingScreen />);
  await waitFor(() => expect(view.getByTestId(T.therapyPricing.screen)).toBeTruthy());
  await waitFor(() => expect(getPlan).toHaveBeenCalled());
  return view;
}

async function buy(view: ReturnType<typeof render>) {
  await act(async () => { fireEvent.press(view.getByTestId(therapyPricingCtaID('single'))); });
}

describe('therapy pricing - pay-per-use sessions', () => {
  it('offers the standalone single session and Pro to non-Pro users', async () => {
    const view = await renderScreen();
    expect(view.getByText('Single session')).toBeTruthy();
    expect(view.getByText('Buy a session · ₹499')).toBeTruthy();
    expect(view.getByTestId(therapyPricingCtaID('pro'))).toBeTruthy();
    expect(view.queryByTestId(T.therapyPricing.credits)).toBeNull();
  });

  it('offers Pro members the member price and hides the Pro upsell', async () => {
    getPlan.mockResolvedValue(plan('pro'));
    const view = await renderScreen();
    await waitFor(() => expect(view.getByText('Buy a session · ₹299')).toBeTruthy());
    expect(view.queryByTestId(therapyPricingCtaID('pro'))).toBeNull();
    await buy(view);
    expect(purchase).toHaveBeenCalledWith('member', 'acct-1');
  });

  it('buys, lets the backend verify, finishes the transaction only after the grant, then opens the persona picker', async () => {
    const order: string[] = [];
    buyCredits.mockImplementation(async () => { order.push('grant'); return { therapy_credits: 1, credits_granted: 1 }; });
    finish.mockImplementation(async () => { order.push('finish'); });
    const view = await renderScreen();
    await buy(view);

    expect(purchase).toHaveBeenCalledWith('single', 'acct-1');
    expect(buyCredits).toHaveBeenCalledWith({
      platform: 'android', product_id: 'com.talktoode.app.therapy.single', purchase_token: 'play-token', transaction_id: undefined,
    });
    expect(order).toEqual(['grant', 'finish']);
    expect(mockPush).toHaveBeenCalledWith('/therapy/persona-picker');
    expect(alert).not.toHaveBeenCalled();
  });

  it('sends only the product id when IAP is unavailable (dev backend grants)', async () => {
    purchase.mockResolvedValue(null);
    const view = await renderScreen();
    await buy(view);
    expect(buyCredits).toHaveBeenCalledWith({ product_id: 'com.talktoode.app.therapy.single' });
    expect(finish).not.toHaveBeenCalled();
    expect(mockPush).toHaveBeenCalledWith('/therapy/persona-picker');
  });

  it('does nothing visible when the user cancels the store sheet', async () => {
    purchase.mockRejectedValue({ code: 'user-cancelled' });
    const view = await renderScreen();
    await buy(view);
    expect(buyCredits).not.toHaveBeenCalled();
    expect(alert).not.toHaveBeenCalled();
    expect(mockPush).not.toHaveBeenCalled();
  });

  it('tells the user they were not charged when the store fails', async () => {
    purchase.mockRejectedValue(new Error('billing unavailable'));
    const view = await renderScreen();
    await buy(view);
    expect(buyCredits).not.toHaveBeenCalled();
    expect(alert).toHaveBeenCalledWith('Purchase failed', expect.stringContaining('not been charged'));
  });

  it('leaves the transaction unfinished when the backend cannot grant, so it is retried', async () => {
    buyCredits.mockRejectedValue(new Error('500'));
    const view = await renderScreen();
    await buy(view);
    expect(finish).not.toHaveBeenCalled();
    expect(mockPush).not.toHaveBeenCalled();
    expect(alert).toHaveBeenCalledWith('Something went wrong', expect.stringContaining('reopen the app'));
    // The button is usable again afterwards.
    expect(view.getByTestId(therapyPricingCtaID('single'))).toBeEnabled();
  });

  it('shows the prepaid balance with a shortcut to start a session', async () => {
    getPlan.mockResolvedValue(plan('plus', 2));
    const view = await renderScreen();
    await waitFor(() => expect(view.getByText('You have 2 prepaid sessions')).toBeTruthy());
    fireEvent.press(view.getByTestId(T.therapyPricing.useCredit));
    expect(mockPush).toHaveBeenCalledWith('/therapy/persona-picker');
  });

  it('still offers the standalone session when the plan cannot be loaded', async () => {
    getPlan.mockRejectedValue(new Error('offline'));
    const view = await renderScreen();
    await buy(view);
    expect(purchase).toHaveBeenCalledWith('single', undefined);
  });

  it('routes the Pro card to the upgrade screen', async () => {
    const view = await renderScreen();
    fireEvent.press(view.getByTestId(therapyPricingCtaID('pro')));
    expect(mockPush).toHaveBeenCalledWith('/upgrade');
    expect(purchase).not.toHaveBeenCalled();
  });
});
