'use client';

import { useState } from 'react';
import { SiteFrame, SiteHeader, SiteFooter } from '../../components/SiteFrame';
import { useCurrency, type Currency } from '../../lib/currency';
const prices = {
  INR: {
    plus: [249, 1999],
    pro: [499, 4499],
    session: 499,
    extra: 299
  },
  USD: {
    plus: [5.99, 39.99],
    pro: [9.99, 79.99],
    session: 7.99,
    extra: 4.99
  },
  EUR: {
    plus: [5.99, 39.99],
    pro: [9.99, 79.99],
    session: 7.99,
    extra: 4.99
  }
};
export default function PricingPage() {
  const detected = useCurrency();
  const [selected, setSelected] = useState<Currency | null>(null),
    [annual, setAnnual] = useState(true);
  const currency = selected || detected.currency,
    price = prices[currency];
  const money = (value: number) => new Intl.NumberFormat(currency === 'INR' ? 'en-IN' : 'en-US', {
    style: 'currency',
    currency,
    maximumFractionDigits: currency === 'INR' ? 0 : 2
  }).format(value);
  const plans = [{
    name: 'Free',
    title: 'A place to begin.',
    amount: 0,
    caption: 'No paid pass needed.',
    features: ['10 journal entries each month', 'AI reflection', 'Three follow-up exchanges', 'Seven-day mood chart', 'Crisis screening']
  }, {
    name: 'Ode Plus',
    title: 'The complete journal.',
    amount: price.plus[annual ? 1 : 0],
    caption: annual ? `Save ${currency === 'INR' ? '33' : '44'}% compared with twelve 30-day passes.` : 'One purchase. Thirty days of access.',
    features: ['Unlimited entries, all five modes', 'Hindi and Hinglish support', 'Dream Decoder and Life Chapters', 'Life Graph and mood history', 'Weekly and annual reviews', 'Streak freeze', 'Five therapist shares per month', 'PDF export and Apple Health sync']
  }, {
    name: 'Ode Pro',
    title: 'A longer conversation.',
    amount: price.pro[annual ? 1 : 0],
    caption: annual ? `Save ${currency === 'INR' ? '25' : '33'}% compared with twelve 30-day passes.` : 'One purchase. Thirty days of access.',
    features: ['Everything in Plus', 'One AI session each month', `Extra sessions: ${money(price.extra)} each`, 'Unlimited therapist shares', 'Priority processing']
  }];
  return <SiteFrame><SiteHeader /><main id="main"><section className="section"><div className="wrap"><div className="pricing-head"><div><p className="kicker">One-time passes</p><h1>Room to begin.<br />Space to go further.</h1><p className="quiet">Start Free. Choose more when it helps.</p></div></div><div className="pricing-controls"><div className="tabs" aria-label="Pass duration"><button aria-pressed={!annual} onClick={() => setAnnual(false)}>30 days</button><button aria-pressed={annual} onClick={() => setAnnual(true)}>365 days</button></div><select aria-label="Pricing currency" value={currency} onChange={e => setSelected(e.target.value as Currency)}>{['INR', 'USD', 'EUR'].map(c => <option key={c}>{c}</option>)}</select><span className="small-note quiet">Passes do not renew automatically.</span></div><div className="plans">{plans.map((p, i) => <div className="plan" key={p.name}><p className="plan-name">{p.name}</p><h2>{p.title}</h2><p className="price">{money(p.amount)} {i > 0 && <small>/ {annual ? 365 : 30} days</small>}</p><p className="caption">{p.caption}</p><ul>{p.features.map(f => <li key={f}>{f}</li>)}</ul><a className="text-btn" href="/#availability">{i === 0 ? 'Get a launch update ↗' : 'Available through the app ↗'}</a></div>)}</div><div className="price-note">Standalone AI session: {money(price.session)}. AI sessions are for reflection and do not replace clinical therapy. Annual Pro includes one session per month while active. Final local store prices are shown before purchase. Purchases will be available in the iOS and Android apps when stores launch.</div><div className="rule-item"><h3>Looking for your team?</h3><p>Corporate plans are sales-led. <a href="/teams">Talk about team wellness →</a></p></div></div></section></main><SiteFooter /></SiteFrame>;
}
