'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useCurrency } from '../../lib/currency';
import type { Currency } from '../../lib/currency';

type Period = 'monthly' | 'annual';

// Canonical prices live in docs/PRICING.md - keep in sync.
const PRICES: Record<string, Record<Currency, string>> = {
  free:    { INR: '₹0',   EUR: '€0',    USD: '$0'    },
  therapy: { INR: '₹499', EUR: '€7.99', USD: '$7.99' },
};

const PLAN_PRICES: Record<'plus' | 'pro', Record<Period, Record<Currency, string>>> = {
  plus: {
    monthly: { INR: '₹249',   EUR: '€5.99',  USD: '$5.99'  },
    annual:  { INR: '₹1,999', EUR: '€39.99', USD: '$39.99' },
  },
  pro: {
    monthly: { INR: '₹499',   EUR: '€9.99',  USD: '$9.99'  },
    annual:  { INR: '₹4,499', EUR: '€79.99', USD: '$79.99' },
  },
};

// Per-month equivalent of the annual pass + savings vs 12 monthly passes.
const ANNUAL_MONTHLY_EQUIV: Record<'plus' | 'pro', Record<Currency, string>> = {
  plus: { INR: '₹167', EUR: '€3.33', USD: '$3.33' },
  pro:  { INR: '₹375', EUR: '€6.67', USD: '$6.67' },
};

const ANNUAL_SAVINGS: Record<'plus' | 'pro', Record<Currency, string>> = {
  plus: { INR: 'Save 33%', EUR: 'Save 44%', USD: 'Save 44%' },
  pro:  { INR: 'Save 25%', EUR: 'Save 33%', USD: 'Save 33%' },
};

const MAX_SAVINGS: Record<Currency, string> = {
  INR: 'Save up to 33%', EUR: 'Save up to 44%', USD: 'Save up to 44%',
};

function Check() {
  return <span style={{ color: '#c8955a', flexShrink: 0, marginTop: 2 }}>✓</span>;
}

export default function PricingPage() {
  const { currency, ready } = useCurrency();
  // Annual is the default: it's the better deal and the savings badge sells it.
  const [period, setPeriod] = useState<Period>('annual');
  const p = (key: string) => PRICES[key][currency];
  const planPrice = (plan: 'plus' | 'pro') => PLAN_PRICES[plan][period][currency];
  const perLine = (plan: 'plus' | 'pro') =>
    period === 'annual' ? `/ year · that's ${ANNUAL_MONTHLY_EQUIV[plan][currency]} / mo` : '/ month';
  const memberExtra = currency === 'INR' ? '₹299' : currency === 'EUR' ? '€4.99' : '$4.99';

  const saveBadge = (plan: 'plus' | 'pro') =>
    period === 'annual' ? (
      <span style={{ background: 'rgba(200,149,90,0.15)', border: '1px solid rgba(200,149,90,0.45)', color: '#c8955a', fontSize: '0.66rem', fontWeight: 800, letterSpacing: '0.5px', padding: '3px 9px', borderRadius: 100, whiteSpace: 'nowrap' }}>
        {ANNUAL_SAVINGS[plan][currency]}
      </span>
    ) : null;

  return (
    <div style={{ background: '#18150f', color: '#e8ddd0', minHeight: '100vh', fontFamily: "'Hanken Grotesk', sans-serif" }}>
      {/* Nav */}
      <nav style={{ maxWidth: 1100, margin: '0 auto', padding: '24px 40px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Link href="/" style={{ display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none', color: '#e8ddd0' }}>
          <div style={{ width: 28, height: 28, borderRadius: 7, background: '#c8955a', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Erode', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1rem', color: '#18150f' }}>O</div>
          <span style={{ fontFamily: "'Erode', serif", fontSize: '1.1rem', fontWeight: 600 }}>Ode</span>
        </Link>
        <Link href="/#download" style={{ background: '#c8955a', color: '#18150f', borderRadius: 100, padding: '9px 20px', fontSize: '0.84rem', fontWeight: 700, textDecoration: 'none' }}>
          Download Free
        </Link>
      </nav>

      <main style={{ maxWidth: 1100, margin: '0 auto', padding: '60px 40px 100px' }}>

        {/* Header */}
        <div style={{ textAlign: 'center', marginBottom: 64 }}>
          <h1 style={{ fontFamily: "'Erode', serif", fontSize: 'clamp(2.8rem, 6vw, 5rem)', fontWeight: 300, margin: '0 0 16px', lineHeight: 1.05, letterSpacing: '-0.02em' }}>
            Honest pricing.
          </h1>
          <p style={{ fontSize: '1rem', color: 'rgba(232,221,208,0.55)', margin: 0, lineHeight: 1.7 }}>
            Most people start Free and stay there. You never need to upgrade — it&apos;s allowed.
          </p>

          {/* Monthly / Annual toggle */}
          <div style={{ display: 'inline-flex', marginTop: 32, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.09)', borderRadius: 100, padding: 4, gap: 4 }}>
            {(['monthly', 'annual'] as const).map(pd => {
              const selected = period === pd;
              return (
                <button
                  key={pd}
                  onClick={() => setPeriod(pd)}
                  style={{
                    display: 'inline-flex', alignItems: 'center', gap: 8, cursor: 'pointer',
                    background: selected ? '#c8955a' : 'transparent',
                    color: selected ? '#18150f' : 'rgba(232,221,208,0.55)',
                    border: 'none', borderRadius: 100, padding: '9px 20px',
                    fontSize: '0.84rem', fontWeight: 700, fontFamily: 'inherit',
                    transition: 'background 0.2s, color 0.2s',
                  }}
                >
                  {pd === 'monthly' ? 'Monthly' : 'Annual'}
                  {pd === 'annual' && (
                    <span style={{
                      background: selected ? 'rgba(24,21,15,0.18)' : 'rgba(200,149,90,0.15)',
                      color: selected ? '#18150f' : '#c8955a',
                      fontSize: '0.64rem', fontWeight: 800, letterSpacing: '0.4px',
                      padding: '2px 8px', borderRadius: 100, whiteSpace: 'nowrap',
                      opacity: ready ? 1 : 0, transition: 'opacity 0.3s',
                    }}>
                      {MAX_SAVINGS[currency]}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>

        {/* Free row */}
        <div style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 16, padding: '20px 28px', marginBottom: 14, display: 'flex', alignItems: 'center', gap: 20, flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, flexShrink: 0 }}>
            <span style={{ fontSize: '0.8rem', fontWeight: 600, color: 'rgba(232,221,208,0.45)' }}>Free</span>
            <span style={{ fontFamily: "'Erode', serif", fontSize: '1.8rem', fontWeight: 300, opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{p('free')}</span>
            <span style={{ fontSize: '0.75rem', color: 'rgba(232,221,208,0.35)' }}>forever</span>
          </div>
          <div style={{ flex: 1, display: 'flex', gap: 18, flexWrap: 'wrap' }}>
            {['10 entries / month', 'AI reflection', '7-day mood chart', '3-turn follow-up', 'Crisis detection'].map(f => (
              <span key={f} style={{ fontSize: '0.8rem', color: 'rgba(232,221,208,0.5)', display: 'inline-flex', gap: 7, alignItems: 'center' }}>
                <Check />{f}
              </span>
            ))}
          </div>
          <Link href="/#download" style={{ padding: '9px 22px', borderRadius: 100, border: '1px solid rgba(255,255,255,0.15)', color: '#e8ddd0', fontSize: '0.84rem', textDecoration: 'none', whiteSpace: 'nowrap', flexShrink: 0 }}>
            Download Free
          </Link>
        </div>

        {/* Plus + Pro cards */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14, marginBottom: 14 }}>
          {/* Plus — featured */}
          <div style={{ background: 'rgba(200,149,90,0.07)', border: '1px solid rgba(212,165,106,0.35)', borderRadius: 20, padding: '32px 28px', display: 'flex', flexDirection: 'column', position: 'relative' }}>
            <div style={{ position: 'absolute', top: -12, left: '50%', transform: 'translateX(-50%)', background: '#c8955a', color: '#18150f', fontSize: '0.65rem', fontWeight: 800, letterSpacing: '1.5px', textTransform: 'uppercase', padding: '5px 14px', borderRadius: 100, whiteSpace: 'nowrap' }}>Most popular</div>
            <div style={{ marginBottom: 6, fontSize: '0.82rem', fontWeight: 600, color: 'rgba(232,221,208,0.45)' }}>Ode+</div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 4 }}>
              <span style={{ fontFamily: "'Erode', serif", fontSize: '2.4rem', fontWeight: 300, lineHeight: 1, opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{planPrice('plus')}</span>
              {saveBadge('plus')}
            </div>
            <div style={{ fontSize: '0.78rem', color: 'rgba(232,221,208,0.35)', marginBottom: 16 }}>{perLine('plus')}</div>
            <p style={{ fontSize: '0.84rem', color: 'rgba(232,221,208,0.5)', margin: '0 0 20px', lineHeight: 1.6 }}>Unlimited entries. All modes. The complete journaling product.</p>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 10, marginBottom: 24 }}>
              {['Unlimited entries', 'All 5 journaling modes', 'Dream Decoder (Jungian + Vedic)', 'Life Graph & Mood History', 'Weekly + Annual Reviews', 'Life Chapters', 'PDF export', 'Therapist share (5/month)', 'Hindi + Hinglish support'].map(f => (
                <div key={f} style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
                  <Check />
                  <span style={{ fontSize: '0.84rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.4 }}>{f}</span>
                </div>
              ))}
            </div>
            <Link href="/#download" style={{ background: '#c8955a', color: '#18150f', borderRadius: 12, padding: '13px', textAlign: 'center', fontWeight: 700, fontSize: '0.88rem', textDecoration: 'none' }}>
              Get Ode+
            </Link>
          </div>

          {/* Pro */}
          <div style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 20, padding: '32px 28px', display: 'flex', flexDirection: 'column' }}>
            <div style={{ marginBottom: 6, fontSize: '0.82rem', fontWeight: 600, color: 'rgba(232,221,208,0.45)' }}>Ode Pro</div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 4 }}>
              <span style={{ fontFamily: "'Erode', serif", fontSize: '2.4rem', fontWeight: 300, lineHeight: 1, opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{planPrice('pro')}</span>
              {saveBadge('pro')}
            </div>
            <div style={{ fontSize: '0.78rem', color: 'rgba(232,221,208,0.35)', marginBottom: 16 }}>{perLine('pro')}</div>
            <p style={{ fontSize: '0.84rem', color: 'rgba(232,221,208,0.5)', margin: '0 0 20px', lineHeight: 1.6 }}>Everything in Plus, plus one therapy session every month.</p>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 10, marginBottom: 24 }}>
              {[`Everything in Ode+`, '1 Therapy Session / month', `Extra sessions at ${memberExtra} (member price)`, 'Unlimited therapist share', 'Priority processing'].map(f => (
                <div key={f} style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
                  <Check />
                  <span style={{ fontSize: '0.84rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.4 }}>{f}</span>
                </div>
              ))}
            </div>
            <Link href="/#download" style={{ border: '1px solid rgba(255,255,255,0.15)', color: '#e8ddd0', borderRadius: 12, padding: '13px', textAlign: 'center', fontWeight: 600, fontSize: '0.88rem', textDecoration: 'none' }}>
              Get Ode Pro
            </Link>
          </div>
        </div>

        {/* Therapy session standalone */}
        <div style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.07)', borderRadius: 16, padding: '24px 28px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 24, flexWrap: 'wrap', marginBottom: 14 }}>
          <div>
            <div style={{ fontSize: '0.82rem', fontWeight: 600, color: 'rgba(232,221,208,0.45)', marginBottom: 6 }}>Therapy Session</div>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
              <span style={{ fontFamily: "'Erode', serif", fontSize: '2rem', fontWeight: 300, opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{p('therapy')}</span>
              <span style={{ fontSize: '0.78rem', color: 'rgba(232,221,208,0.35)' }}>per session</span>
            </div>
            <p style={{ fontSize: '0.84rem', color: 'rgba(232,221,208,0.45)', margin: '8px 0 0', lineHeight: 1.6, maxWidth: 480 }}>
              Pay-per-use on any plan. Journal-context-aware AI conversation up to 1 hour. Voice or text. Post-session summary. Crisis detection active. First session always free.
            </p>
          </div>
          <Link href="/#download" style={{ border: '1px solid rgba(255,255,255,0.12)', color: '#e8ddd0', borderRadius: 12, padding: '11px 22px', fontSize: '0.84rem', textDecoration: 'none', whiteSpace: 'nowrap', flexShrink: 0 }}>
            Download to start
          </Link>
        </div>

        <p style={{ textAlign: 'right', fontSize: '0.76rem', color: 'rgba(232,221,208,0.3)', marginBottom: 72 }}>
          one-time passes (30 or 365 days) · managed in-app · no auto-renew
        </p>

        {/* FAQ */}
        <div style={{ borderTop: '1px solid rgba(255,255,255,0.07)', paddingTop: 56 }}>
          <h2 style={{ fontFamily: "'Erode', serif", fontSize: 'clamp(1.6rem, 3vw, 2.2rem)', fontWeight: 300, margin: '0 0 40px' }}>Common questions</h2>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 60px' }}>
            {[
              { q: 'Does it auto-renew?', a: 'No. Every plan is a one-time pass — 30 days for monthly, 365 days for annual. When it expires you return to Free automatically. Buy again if you want to continue.' },
              { q: 'Can I switch plans?', a: 'Yes, any time. Buy a new plan in-app and it activates immediately. Unused days on your current pass are not refunded.' },
              { q: 'Is the free plan actually free?', a: 'Yes, genuinely. 10 entries a month, full AI reflection, 7-day mood chart, crisis detection. No card required.' },
              { q: 'What currencies do you accept?', a: 'Prices are shown in INR for India, USD for most countries. All purchases are in-app purchases handled by the App Store or Google Play.' },
              { q: 'Is Therapy Mode safe?', a: 'Crisis detection runs on every message. If distress is flagged, you get hotline resources immediately and the session ends safely.' },
              { q: 'Can a therapist access my data?', a: 'Only if you share it. You generate a passcode-protected link. It expires in 72 hours. Your therapist sees mood trends and AI summaries — never raw recordings.' },
            ].map(item => (
              <div key={item.q} style={{ padding: '20px 0', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
                <div style={{ fontSize: '0.92rem', fontWeight: 600, color: '#e8ddd0', marginBottom: 8 }}>{item.q}</div>
                <p style={{ fontSize: '0.85rem', color: 'rgba(232,221,208,0.5)', margin: 0, lineHeight: 1.7 }}>{item.a}</p>
              </div>
            ))}
          </div>
        </div>
      </main>

      <footer style={{ borderTop: '1px solid rgba(255,255,255,0.07)', padding: '28px 40px', maxWidth: 1100, margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 16 }}>
        <span style={{ fontSize: '0.74rem', color: 'rgba(232,221,208,0.3)' }}>© 2026 Ode</span>
        <div style={{ display: 'flex', gap: 24 }}>
          {[['Home', '/'], ['Privacy', '/privacy'], ['Terms', '/terms'], ['Support', 'mailto:support@dreamlog.app']].map(([label, href]) => (
            <a key={label} href={href} style={{ fontSize: '0.8rem', color: 'rgba(232,221,208,0.4)', textDecoration: 'none' }}>{label}</a>
          ))}
        </div>
      </footer>

      <style>{`
        @media (max-width: 640px) {
          main { padding: 40px 20px 80px !important; }
          nav { padding: 20px !important; }
          div[style*="grid-template-columns: 1fr 1fr"] { grid-template-columns: 1fr !important; }
        }
      `}</style>
    </div>
  );
}
