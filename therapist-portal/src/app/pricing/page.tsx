'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';

type Currency = 'INR' | 'EUR' | 'USD';

const PRICES: Record<string, Record<Currency, string>> = {
  free:    { INR: '₹0',   EUR: '€0',    USD: '$0'    },
  plus:    { INR: '₹249', EUR: '€4.99', USD: '$5.99' },
  pro:     { INR: '₹499', EUR: '€8.99', USD: '$9.99' },
  therapy: { INR: '₹499', EUR: '€5.99', USD: '$7.99' },
};

function useCurrency(): { currency: Currency; ready: boolean } {
  const [currency, setCurrency] = useState<Currency>('USD');
  const [ready, setReady] = useState(false);
  useEffect(() => {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (tz === 'Asia/Calcutta' || tz === 'Asia/Kolkata') {
      setCurrency('INR'); setReady(true); return;
    }
    fetch('https://ipapi.co/json/')
      .then(r => r.ok ? r.json() : null)
      .then(d => {
        if (d?.country_code === 'IN') setCurrency('INR');
        else if (d?.country_code === 'FR') setCurrency('EUR');
        else setCurrency('USD');
      })
      .catch(() => {})
      .finally(() => setReady(true));
  }, []);
  return { currency, ready };
}

function Check() {
  return <span style={{ color: '#c8955a', flexShrink: 0, marginTop: 2 }}>✓</span>;
}

export default function PricingPage() {
  const { currency, ready } = useCurrency();
  const p = (key: string) => PRICES[key][currency];
  const per = '/ month';
  const memberExtra = currency === 'INR' ? '₹299' : currency === 'EUR' ? '€3.99' : '$4.99';

  return (
    <div style={{ background: '#18150f', color: '#e8ddd0', minHeight: '100vh', fontFamily: "'Nunito', sans-serif" }}>
      {/* Nav */}
      <nav style={{ maxWidth: 1100, margin: '0 auto', padding: '24px 40px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Link href="/" style={{ display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none', color: '#e8ddd0' }}>
          <div style={{ width: 28, height: 28, borderRadius: 7, background: '#c8955a', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1rem', color: '#18150f' }}>D</div>
          <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.1rem', fontWeight: 600 }}>DreamLog</span>
        </Link>
        <Link href="/#download" style={{ background: '#c8955a', color: '#18150f', borderRadius: 100, padding: '9px 20px', fontSize: '0.84rem', fontWeight: 700, textDecoration: 'none' }}>
          Download Free
        </Link>
      </nav>

      <main style={{ maxWidth: 1100, margin: '0 auto', padding: '60px 40px 100px' }}>

        {/* Header */}
        <div style={{ textAlign: 'center', marginBottom: 64 }}>
          <h1 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(2.8rem, 6vw, 5rem)', fontWeight: 300, margin: '0 0 16px', lineHeight: 1.05, letterSpacing: '-0.02em' }}>
            Honest pricing.
          </h1>
          <p style={{ fontSize: '1rem', color: 'rgba(232,221,208,0.55)', margin: 0, lineHeight: 1.7 }}>
            Most people start Free and stay there. You never need to upgrade — it&apos;s allowed.
          </p>
        </div>

        {/* Free row */}
        <div style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 16, padding: '20px 28px', marginBottom: 14, display: 'flex', alignItems: 'center', gap: 20, flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, flexShrink: 0 }}>
            <span style={{ fontSize: '0.8rem', fontWeight: 600, color: 'rgba(232,221,208,0.45)' }}>Free</span>
            <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.8rem', fontWeight: 300, opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{p('free')}</span>
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
            <div style={{ marginBottom: 6, fontSize: '0.82rem', fontWeight: 600, color: 'rgba(232,221,208,0.45)' }}>DreamLog+</div>
            <div style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '2.4rem', fontWeight: 300, lineHeight: 1, marginBottom: 4, opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{p('plus')}</div>
            <div style={{ fontSize: '0.78rem', color: 'rgba(232,221,208,0.35)', marginBottom: 16 }}>{per}</div>
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
              Get DreamLog+
            </Link>
          </div>

          {/* Pro */}
          <div style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 20, padding: '32px 28px', display: 'flex', flexDirection: 'column' }}>
            <div style={{ marginBottom: 6, fontSize: '0.82rem', fontWeight: 600, color: 'rgba(232,221,208,0.45)' }}>DreamLog Pro</div>
            <div style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '2.4rem', fontWeight: 300, lineHeight: 1, marginBottom: 4, opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{p('pro')}</div>
            <div style={{ fontSize: '0.78rem', color: 'rgba(232,221,208,0.35)', marginBottom: 16 }}>{per}</div>
            <p style={{ fontSize: '0.84rem', color: 'rgba(232,221,208,0.5)', margin: '0 0 20px', lineHeight: 1.6 }}>Everything in Plus, plus one therapy session every month.</p>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 10, marginBottom: 24 }}>
              {[`Everything in DreamLog+`, '1 Therapy Session / month', `Extra sessions at ${memberExtra} (member price)`, 'Unlimited therapist share', 'Priority processing'].map(f => (
                <div key={f} style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
                  <Check />
                  <span style={{ fontSize: '0.84rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.4 }}>{f}</span>
                </div>
              ))}
            </div>
            <Link href="/#download" style={{ border: '1px solid rgba(255,255,255,0.15)', color: '#e8ddd0', borderRadius: 12, padding: '13px', textAlign: 'center', fontWeight: 600, fontSize: '0.88rem', textDecoration: 'none' }}>
              Get DreamLog Pro
            </Link>
          </div>
        </div>

        {/* Therapy session standalone */}
        <div style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.07)', borderRadius: 16, padding: '24px 28px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 24, flexWrap: 'wrap', marginBottom: 14 }}>
          <div>
            <div style={{ fontSize: '0.82rem', fontWeight: 600, color: 'rgba(232,221,208,0.45)', marginBottom: 6 }}>Therapy Session</div>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
              <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '2rem', fontWeight: 300, opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{p('therapy')}</span>
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
          30-day passes · managed in-app · no auto-renew
        </p>

        {/* FAQ */}
        <div style={{ borderTop: '1px solid rgba(255,255,255,0.07)', paddingTop: 56 }}>
          <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(1.6rem, 3vw, 2.2rem)', fontWeight: 300, margin: '0 0 40px' }}>Common questions</h2>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 60px' }}>
            {[
              { q: 'Does it auto-renew?', a: 'No. Every plan is a 30-day pass. When it expires you return to Free automatically. Buy again if you want to continue.' },
              { q: 'Can I switch plans?', a: 'Yes, any time. Buy a new plan in-app and it activates immediately. Unused days on your current pass are not refunded.' },
              { q: 'Is the free plan actually free?', a: 'Yes, genuinely. 10 entries a month, full AI reflection, 7-day mood chart, crisis detection. No card required.' },
              { q: 'What currencies do you accept?', a: 'Prices are shown in INR for India, USD for most countries. Payment is processed by Stripe or in-app purchase depending on your platform.' },
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
        <span style={{ fontSize: '0.74rem', color: 'rgba(232,221,208,0.3)' }}>© 2026 DreamLog</span>
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
