'use client';

import { useEffect, useState } from 'react';

interface VersionInfo { android_store_url: string; ios_store_url: string; }

/* ── Currency detection ───────────────────────────────────────────────────── */
type Currency = 'INR' | 'EUR' | 'USD';

const PRICES: Record<string, Record<Currency, string>> = {
  free:    { INR: '₹0',   EUR: '€0',    USD: '$0'    },
  plus:    { INR: '₹249', EUR: '€4.99', USD: '$5.99' },
  pro:     { INR: '₹499', EUR: '€8.99', USD: '$9.99' },
  therapy: { INR: '₹499', EUR: '€5.99', USD: '$7.99' },
};

const PERIOD: Record<Currency, string> = {
  INR: '/ month',
  EUR: '/ month',
  USD: '/ month',
};

function useCurrency(): { currency: Currency; ready: boolean } {
  const [currency, setCurrency] = useState<Currency>('USD');
  const [ready, setReady] = useState(false);
  useEffect(() => {
    // India timezone is unambiguous — resolve instantly without network
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (tz === 'Asia/Calcutta' || tz === 'Asia/Kolkata') {
      setCurrency('INR'); setReady(true); return;
    }
    // France vs rest-of-world needs a country code — use IP API
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

function useVersionInfo(): VersionInfo {
  const [info, setInfo] = useState<VersionInfo>({ android_store_url: '#', ios_store_url: '#' });
  useEffect(() => {
    const base = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
    fetch(`${base}/version`)
      .then(r => r.ok ? r.json() : null)
      .then(d => { if (d?.android_store_url || d?.ios_store_url) setInfo({ android_store_url: d.android_store_url || '#', ios_store_url: d.ios_store_url || '#' }); })
      .catch(() => {});
  }, []);
  return info;
}

function useScrolled(threshold = 10) {
  const [scrolled, setScrolled] = useState(false);
  useEffect(() => {
    const fn = () => setScrolled(window.scrollY > threshold);
    window.addEventListener('scroll', fn, { passive: true });
    return () => window.removeEventListener('scroll', fn);
  }, [threshold]);
  return scrolled;
}

function useReveal() {
  useEffect(() => {
    const targets = document.querySelectorAll('.reveal, .stagger, .reveal-left, .reveal-right');
    if (!targets.length) return;
    const io = new IntersectionObserver(
      entries => entries.forEach(e => { if (e.isIntersecting) e.target.classList.add('visible'); }),
      { threshold: 0.08 }
    );
    targets.forEach(t => io.observe(t));
    return () => io.disconnect();
  }, []);
}

/* ── SVG Icons ────────────────────────────────────────────────────────────── */
function IconHeart() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/>
    </svg>
  );
}
function IconCompass() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10"/>
      <polygon points="16.24 7.76 14.12 14.12 7.76 16.24 9.88 9.88 16.24 7.76"/>
    </svg>
  );
}
function IconLens() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="11" cy="11" r="8"/>
      <line x1="21" y1="21" x2="16.65" y2="16.65"/>
      <line x1="8" y1="11" x2="14" y2="11"/>
      <line x1="11" y1="8" x2="11" y2="14"/>
    </svg>
  );
}
function IconBreath() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
      <circle cx="12" cy="12" r="2.5"/>
      <circle cx="12" cy="12" r="6" strokeOpacity="0.55"/>
      <circle cx="12" cy="12" r="10" strokeOpacity="0.25"/>
    </svg>
  );
}
function IconMic() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
      <path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
      <line x1="12" y1="19" x2="12" y2="23"/>
      <line x1="8" y1="23" x2="16" y2="23"/>
    </svg>
  );
}
function IconSparkle() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z"/>
    </svg>
  );
}
function IconChat() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
    </svg>
  );
}
function IconMoon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
    </svg>
  );
}
function IconBook() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
      <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
    </svg>
  );
}
function IconChart() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
    </svg>
  );
}
function IconPeople() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
      <circle cx="9" cy="7" r="4"/>
      <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
      <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
    </svg>
  );
}
function IconPath() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="5" r="2"/>
      <path d="M12 7c0 0-6 4-6 9a6 6 0 0 0 12 0c0-5-6-9-6-9z"/>
      <circle cx="12" cy="16" r="1" fill="currentColor"/>
    </svg>
  );
}
function IconCalendar() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
      <line x1="16" y1="2" x2="16" y2="6"/>
      <line x1="8" y1="2" x2="8" y2="6"/>
      <line x1="3" y1="10" x2="21" y2="10"/>
    </svg>
  );
}

/* ── Feature icon wrapper ─────────────────────────────────────────────────── */
function FeatureIcon({ children }: { children: React.ReactNode }) {
  return (
    <div style={{
      width: 40, height: 40, borderRadius: 10,
      background: 'rgba(200,149,90,0.08)', border: '1px solid rgba(200,149,90,0.15)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      color: 'var(--gold)', marginBottom: 14, flexShrink: 0,
    }}>
      {children}
    </div>
  );
}

/* ── App mockup ───────────────────────────────────────────────────────────── */
function AppMockup() {
  const bars = [0.3, 0.65, 0.5, 1, 0.7, 0.9, 0.45, 0.85, 0.55, 0.35, 0.75, 0.6, 0.95, 0.5, 0.4, 0.7];
  const emotions = [
    { label: 'Reflective', color: '#c8955a', pct: 72 },
    { label: 'Hopeful',    color: '#5b8db8', pct: 55 },
    { label: 'Tired',      color: '#8b7aab', pct: 34 },
  ];
  return (
    <div style={{ position: 'relative', width: 340, flexShrink: 0 }}>

      {/* Phone frame */}
      <div style={{
        background: '#1a1710', border: '1.5px solid rgba(255,255,255,0.1)',
        borderRadius: 44, padding: '14px 18px 28px',
        boxShadow: '0 48px 96px rgba(0,0,0,0.7), 0 0 0 1px rgba(255,255,255,0.04)',
        animation: 'floatPhone 7s ease-in-out infinite',
      }}>
        {/* Pill notch */}
        <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 14 }}>
          <div style={{ width: 80, height: 8, background: '#0c0a07', borderRadius: 100 }} />
        </div>

        {/* Status bar */}
        <div style={{ display: 'flex', justifyContent: 'space-between', color: 'rgba(232,221,208,0.35)', fontSize: 11, marginBottom: 22, padding: '0 4px' }}>
          <span>9:41</span><span>TUESDAY</span>
        </div>

        {/* Screen content */}
        <div style={{ padding: '0 4px' }}>
          <div style={{ fontSize: 9, letterSpacing: 2, color: 'rgba(232,221,208,0.3)', textTransform: 'uppercase', marginBottom: 8 }}>
            A QUIET EVENING
          </div>
          <div className="serif" style={{ fontSize: 24, fontWeight: 300, color: '#e8ddd0', marginBottom: 26, lineHeight: 1.3 }}>
            What&apos;s on your<br />mind tonight?
          </div>

          {/* Waveform */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 2.5, height: 44, marginBottom: 22, justifyContent: 'center' }}>
            {bars.map((h, i) => (
              <div key={i} style={{
                width: 3, background: '#c8955a', borderRadius: 3,
                height: `${h * 100}%`, opacity: 0.8,
                animation: `wave ${1.1 + i * 0.09}s ease-in-out ${i * 0.07}s infinite`,
              }} />
            ))}
          </div>

          {/* Mic button */}
          <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 18 }}>
            <div style={{
              width: 64, height: 64, borderRadius: '50%', background: '#c8955a',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              boxShadow: '0 0 0 10px rgba(200,149,90,0.1)',
            }}>
              <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="#18150f" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
                <path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
                <line x1="12" y1="19" x2="12" y2="23"/>
                <line x1="8" y1="23" x2="16" y2="23"/>
              </svg>
            </div>
          </div>

          {/* Transcript snippet */}
          <div style={{
            background: 'rgba(200,149,90,0.06)', border: '1px solid rgba(200,149,90,0.1)',
            borderRadius: 12, padding: '10px 12px',
          }}>
            <p className="serif" style={{ fontStyle: 'italic', fontSize: 12, color: 'rgba(232,221,208,0.6)', margin: 0, lineHeight: 1.55 }}>
              &ldquo;I think I was more tired than I realised this week...&rdquo;
            </p>
            <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginTop: 6 }}>
              <div style={{ width: 5, height: 5, borderRadius: '50%', background: '#c8955a' }} />
              <span style={{ fontSize: 10, color: 'rgba(232,221,208,0.3)', letterSpacing: 0.5 }}>LISTENING · 1:24</span>
            </div>
          </div>
        </div>
      </div>

      {/* Mood float card — top right */}
      <div style={{
        position: 'absolute', top: 40, right: -180,
        background: '#26221a', border: '1px solid rgba(200,149,90,0.15)',
        borderRadius: 18, padding: '16px 20px', width: 200,
        boxShadow: '0 24px 48px rgba(0,0,0,0.6)',
        animation: 'floatCardA 8s ease-in-out 1s infinite',
      }}>
        <div style={{ fontSize: 9, letterSpacing: 2, color: 'rgba(232,221,208,0.3)', textTransform: 'uppercase', marginBottom: 8 }}>
          MOOD · THIS WEEK
        </div>
        <div className="serif" style={{ fontSize: 52, fontWeight: 300, color: '#e8ddd0', lineHeight: 1, marginBottom: 2 }}>6.4</div>
        <div style={{ fontSize: 11, color: '#5a9367', marginBottom: 14, fontWeight: 500 }}>+0.8 vs last week</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}>
          {emotions.map(e => (
            <div key={e.label} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div style={{ flex: 1, height: 3, background: 'rgba(255,255,255,0.06)', borderRadius: 2, overflow: 'hidden' }}>
                <div style={{ height: '100%', width: `${e.pct}%`, background: e.color, borderRadius: 2 }} />
              </div>
              <span style={{ fontSize: 10, color: 'rgba(232,221,208,0.5)', width: 58, flexShrink: 0 }}>{e.label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Pattern float card — bottom left */}
      <div style={{
        position: 'absolute', bottom: 60, left: -130,
        background: '#26221a', border: '1px solid rgba(200,149,90,0.12)',
        borderRadius: 16, padding: '14px 18px', width: 210,
        boxShadow: '0 16px 40px rgba(0,0,0,0.6)',
        animation: 'floatCardB 9s ease-in-out 2s infinite',
      }}>
        <div style={{ fontSize: 9, letterSpacing: 2, color: 'rgba(232,221,208,0.3)', textTransform: 'uppercase', marginBottom: 8 }}>
          A PATTERN
        </div>
        <p className="serif" style={{ fontStyle: 'italic', fontSize: 13.5, color: '#e8ddd0', lineHeight: 1.6, margin: 0 }}>
          &ldquo;You sound lighter on weeks you walk in the morning.&rdquo;
        </p>
      </div>
    </div>
  );
}

/* ── Store badges ─────────────────────────────────────────────────────────── */
function AppStoreBadge({ href, disabled }: { href: string; disabled: boolean }) {
  return (
    <a href={disabled ? undefined : href} target={disabled ? undefined : '_blank'} rel="noreferrer"
      title={disabled ? 'App Store listing coming soon' : 'Download on the App Store'}
      style={{ display: 'inline-flex', alignItems: 'center', gap: 10, background: disabled ? 'rgba(255,255,255,0.05)' : '#000', border: `1px solid ${disabled ? 'rgba(255,255,255,0.1)' : 'rgba(255,255,255,0.22)'}`, borderRadius: 12, padding: '11px 20px', cursor: disabled ? 'default' : 'pointer', opacity: disabled ? 0.5 : 1, textDecoration: 'none' }}>
      <svg width="20" height="24" viewBox="0 0 814 1000" fill="white">
        <path d="M788.1 340.9c-5.8 4.5-108.2 62.2-108.2 190.5 0 148.4 130.3 200.9 134.2 202.2-.6 3.2-20.7 71.9-68.7 141.9-42.8 61.6-87.5 123.1-155.5 123.1s-85.5-39.5-164-39.5c-76 0-103.7 40.8-165.9 40.8s-105-37.5-166.5-123.1C46.5 713.1 0 592.5 0 478.4c0-209.5 136.1-320.3 270.5-320.3 36.8 0 105.1 12.5 149.8 12.5 42.3 0 121.3-16.4 177.5-16.4 37.1 0 153.2 3.2 234.4 86.5zm-97.4-188.3c27.7-33.2 47.5-79.4 47.5-125.6 0-6.4-.5-12.9-1.6-18.2-45 1.7-98.4 30-130.4 66.3-25.1 28.4-47.5 74.6-47.5 121.4 0 7.1 1.1 14.2 1.6 16.4 2.7.5 7.1 1.1 11.4 1.1 40.6 0 91.9-27.2 119-61.4z"/>
      </svg>
      <div style={{ textAlign: 'left' }}>
        <div style={{ fontSize: '0.6rem', color: 'rgba(255,255,255,0.65)', lineHeight: 1, marginBottom: 2 }}>Download on the</div>
        <div style={{ fontSize: '1rem', fontWeight: 600, color: '#fff', lineHeight: 1 }}>App Store</div>
      </div>
    </a>
  );
}

function PlayStoreBadge({ href, disabled }: { href: string; disabled: boolean }) {
  return (
    <a href={disabled ? undefined : href} target={disabled ? undefined : '_blank'} rel="noreferrer"
      title={disabled ? 'Play Store listing coming soon' : 'Get it on Google Play'}
      style={{ display: 'inline-flex', alignItems: 'center', gap: 10, background: disabled ? 'rgba(255,255,255,0.05)' : '#000', border: `1px solid ${disabled ? 'rgba(255,255,255,0.1)' : 'rgba(255,255,255,0.22)'}`, borderRadius: 12, padding: '11px 20px', cursor: disabled ? 'default' : 'pointer', opacity: disabled ? 0.5 : 1, textDecoration: 'none' }}>
      <svg width="20" height="22" viewBox="0 0 22 24">
        <path d="M1.2 0.5C0.5 0.9 0 1.7 0 2.6v18.8c0 .9.5 1.7 1.2 2.1l.1.1 10.5-10.5v-.2L1.3.4l-.1.1z" fill="#4FC3F7"/>
        <path d="M15.3 15.1l-3.5-3.5v-.2l3.5-3.5.1.1 4.1 2.4c1.2.7 1.2 1.8 0 2.5l-4.1 2.3-.1-.1z" fill="#FFCA28"/>
        <path d="M15.4 15l-3.6-3.6L1.2 22c.4.4 1 .4 1.7.1l12.5-7.1" fill="#F06292"/>
        <path d="M15.4 8.9L2.9 1.9C2.2 1.5 1.6 1.6 1.2 2l10.6 10.5 3.6-3.6z" fill="#66BB6A"/>
      </svg>
      <div style={{ textAlign: 'left' }}>
        <div style={{ fontSize: '0.6rem', color: 'rgba(255,255,255,0.65)', lineHeight: 1, marginBottom: 2 }}>Get it on</div>
        <div style={{ fontSize: '1rem', fontWeight: 600, color: '#fff', lineHeight: 1 }}>Google Play</div>
      </div>
    </a>
  );
}

function DownloadButtons({ androidUrl, iosUrl }: { androidUrl: string; iosUrl: string }) {
  return (
    <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', justifyContent: 'center' }}>
      <AppStoreBadge href={iosUrl} disabled={iosUrl === '#'} />
      <PlayStoreBadge href={androidUrl} disabled={androidUrl === '#'} />
    </div>
  );
}

/* ── Pricing card ─────────────────────────────────────────────────────────── */
function PricingCard({ name, price, period, sub, features, cta, featured, downloadHref, ready }: {
  name: string; price: string; period: string;
  sub: string; features: string[]; cta: string; featured?: boolean; downloadHref: string; ready: boolean;
}) {
  return (
    <div className={featured ? 'pricing-featured' : 'pricing-card'} style={{ background: featured ? 'rgba(200,149,90,0.06)' : 'var(--bg-card)', border: `1px solid ${featured ? 'rgba(200,149,90,0.28)' : 'var(--border)'}`, borderRadius: 20, padding: '28px 24px', display: 'flex', flexDirection: 'column', position: 'relative' }}>
      {featured && (
        <div style={{ position: 'absolute', top: -12, left: '50%', transform: 'translateX(-50%)', background: 'var(--gold)', color: '#18150f', fontSize: '0.65rem', fontWeight: 700, letterSpacing: '1.5px', textTransform: 'uppercase', padding: '4px 12px', borderRadius: 100, whiteSpace: 'nowrap' }}>Most popular</div>
      )}
      <div style={{ marginBottom: 20 }}>
        <div style={{ fontSize: '0.82rem', fontWeight: 600, color: 'var(--muted)', marginBottom: 8 }}>{name}</div>
        <div className="serif" style={{ fontSize: '2.2rem', fontWeight: 300, color: 'var(--text)', lineHeight: 1, opacity: ready ? 1 : 0, transition: 'opacity 0.3s', whiteSpace: 'nowrap' }}>{price}</div>
        <div style={{ fontSize: '0.78rem', color: 'var(--muted-2)', marginTop: 4, minHeight: '1.2em' }}>{period}</div>
      </div>
      <p style={{ fontSize: '0.84rem', color: 'var(--muted)', margin: '0 0 20px', lineHeight: 1.55 }}>{sub}</p>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 9, marginBottom: 24 }}>
        {features.map(f => (
          <div key={f} style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
            <span style={{ color: 'var(--gold)', flexShrink: 0, marginTop: 1 }}>✓</span>
            <span style={{ fontSize: '0.84rem', color: 'var(--muted)', lineHeight: 1.4 }}>{f}</span>
          </div>
        ))}
      </div>
      <a href={downloadHref} className={featured ? 'btn-primary' : 'btn-ghost'} style={{ textAlign: 'center', borderRadius: 12, padding: '12px' }}>{cta}</a>
    </div>
  );
}

/* ── FAQ ──────────────────────────────────────────────────────────────────── */
function FAQ({ q, a }: { q: string; a: string }) {
  const [open, setOpen] = useState(false);
  return (
    <div style={{ borderBottom: '1px solid var(--border)', padding: '20px 0', cursor: 'pointer' }} onClick={() => setOpen(o => !o)}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 16 }}>
        <span style={{ fontSize: '0.95rem', fontWeight: 500, color: 'var(--text)', lineHeight: 1.4 }}>{q}</span>
        <span style={{ color: 'var(--muted)', fontSize: '1.2rem', flexShrink: 0, transform: open ? 'rotate(45deg)' : 'none', transition: 'transform 0.2s ease', display: 'inline-block' }}>+</span>
      </div>
      {open && <p style={{ margin: '12px 0 0', fontSize: '0.88rem', color: 'var(--muted)', lineHeight: 1.7 }}>{a}</p>}
    </div>
  );
}

/* ── Breathing orb ────────────────────────────────────────────────────────── */
function BreathingOrb() {
  const [phase, setPhase] = useState<'idle' | 'inhale' | 'hold' | 'exhale'>('idle');
  const [label, setLabel] = useState('Tap to breathe');
  const [running, setRunning] = useState(false);

  useEffect(() => {
    if (!running) return;
    let cancelled = false;
    async function cycle() {
      while (!cancelled) {
        setPhase('inhale'); setLabel('Inhale…');
        await new Promise(r => setTimeout(r, 4000));
        if (cancelled) break;
        setPhase('hold'); setLabel('Hold…');
        await new Promise(r => setTimeout(r, 4000));
        if (cancelled) break;
        setPhase('exhale'); setLabel('Exhale…');
        await new Promise(r => setTimeout(r, 6000));
        if (cancelled) break;
      }
    }
    cycle();
    return () => { cancelled = true; };
  }, [running]);

  const dur = phase === 'inhale' ? 4 : phase === 'hold' ? 0.3 : 6;
  const scale = phase === 'idle' ? 1 : phase === 'inhale' ? 1.35 : phase === 'hold' ? 1.35 : 0.85;
  const glow = phase === 'exhale' ? 'rgba(91,141,184,0.22)' : 'rgba(200,149,90,0.28)';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 24 }}
      onClick={() => { if (!running) { setRunning(true); } }}>
      <div style={{
        position: 'relative', width: 160, height: 160, cursor: running ? 'default' : 'pointer',
      }}>
        {/* Outer ring — always animated in idle, transition-driven when guided */}
        <div style={{
          position: 'absolute', inset: 0, borderRadius: '50%',
          border: '1px solid rgba(200,149,90,0.18)',
          opacity: 0.5,
          ...(phase === 'idle'
            ? { animation: 'breathIdle 5s ease-in-out infinite' }
            : { transform: `scale(${scale * 1.3})`, transition: `transform ${dur}s ease-in-out` }),
        }} />
        {/* Mid ring */}
        <div style={{
          position: 'absolute', inset: 16, borderRadius: '50%',
          border: '1px solid rgba(200,149,90,0.28)',
          opacity: 0.65,
          ...(phase === 'idle'
            ? { animation: 'breathIdle 5s ease-in-out 0.4s infinite' }
            : { transform: `scale(${scale * 1.15})`, transition: `transform ${dur}s ease-in-out` }),
        }} />
        {/* Core */}
        <div style={{
          position: 'absolute', inset: 32, borderRadius: '50%',
          background: `radial-gradient(circle, ${glow} 0%, transparent 55%)`,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          transition: 'background 1.2s ease',
          ...(phase === 'idle'
            ? { animation: 'breathIdle 5s ease-in-out 0.8s infinite' }
            : { transform: `scale(${scale})`, transition: `transform ${dur}s ease-in-out, background 1s ease` }),
        }}>
          <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--gold)" strokeWidth="1.5" strokeLinecap="round">
            <circle cx="12" cy="12" r="2.5"/>
            <circle cx="12" cy="12" r="6" strokeOpacity="0.55"/>
          </svg>
        </div>
      </div>
      <div style={{ textAlign: 'center' }}>
        <div style={{
          fontFamily: "'Cormorant Garamond', serif",
          fontSize: running ? '1.35rem' : '1rem',
          fontStyle: 'italic',
          fontWeight: 300,
          color: running ? 'var(--text)' : 'var(--muted)',
          letterSpacing: 0.5,
          transition: 'color 0.3s, font-size 0.4s ease',
        }}>{label}</div>
        {running && (
          <button onClick={e => { e.stopPropagation(); setRunning(false); setPhase('idle'); setLabel('Tap to breathe'); }}
            style={{ marginTop: 12, background: 'none', border: 'none', color: 'var(--muted-2)', fontSize: '0.72rem', cursor: 'pointer', fontFamily: 'inherit', letterSpacing: '0.5px' }}>
            stop
          </button>
        )}
      </div>
    </div>
  );
}

/* ── Main ─────────────────────────────────────────────────────────────────── */
export default function LandingPage() {
  const scrolled = useScrolled();
  const version = useVersionInfo();
  const { currency, ready } = useCurrency();
  useReveal();
  const dlHref = '#download';
  const p = (key: string) => PRICES[key][currency];
  const per = PERIOD[currency];

  return (
    <div style={{ background: 'var(--bg)', color: 'var(--text)', overflowX: 'hidden' }}>

      {/* Paper grain — subtle noise layer reinforcing the journal/editorial feel */}
      <div aria-hidden="true" style={{
        position: 'fixed', inset: 0, pointerEvents: 'none', zIndex: 9999,
        backgroundImage: "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='250' height='250'%3E%3Cfilter id='g'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.8' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='250' height='250' filter='url(%23g)'/%3E%3C/svg%3E\")",
        opacity: 0.028,
      }} />

      {/* Nav */}
      <nav className={`landing-nav${scrolled ? ' scrolled' : ''}`}>
        <a href="/" style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{ width: 28, height: 28, borderRadius: 7, background: 'var(--gold)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1rem', color: '#18150f' }}>D</div>
          <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.1rem', fontWeight: 600 }}>DreamLog</span>
        </a>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <a href="#features" className="nav-link" style={{ padding: '8px 14px', fontSize: '0.86rem', color: 'var(--muted)', fontWeight: 500 }}>Features</a>
          <a href="#pricing" className="nav-link" style={{ padding: '8px 14px', fontSize: '0.86rem', color: 'var(--muted)', fontWeight: 500 }}>Pricing</a>
          <a href="/login" className="nav-link" style={{ padding: '8px 14px', fontSize: '0.86rem', color: 'var(--muted)', fontWeight: 500 }}>For Therapists</a>
          <a href={dlHref} className="btn-primary" style={{ padding: '9px 18px', fontSize: '0.84rem', borderRadius: 10 }}>Download App</a>
        </div>
      </nav>

      {/* ── Hero ──────────────────────────────────────────────────────── */}
      <section className="landing-section" style={{ paddingTop: 150, paddingBottom: 120, position: 'relative', overflow: 'hidden' }}>
        {/* Asymmetric warm wash — top-left corner bleed, not a centered orb */}
        <div style={{
          position: 'absolute', top: '-20%', left: '-10%',
          width: '65%', height: '90%',
          background: 'radial-gradient(ellipse 55% 65% at 28% 38%, rgba(200,149,90,0.07) 0%, rgba(180,130,70,0.03) 45%, transparent 70%)',
          pointerEvents: 'none', transform: 'rotate(-8deg)',
        }} />
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 80, position: 'relative' }}>
          <div style={{ flex: 1, maxWidth: 500 }}>
            <div className="eyebrow" style={{ marginBottom: 20, animation: 'heroIn 0.6s ease both', animationDelay: '0.05s' }}>A PRIVATE EMOTIONAL JOURNAL</div>
            <h1 className="serif" style={{ fontSize: 'clamp(2.6rem, 5vw, 4.2rem)', fontWeight: 300, lineHeight: 1.12, margin: '0 0 24px', color: 'var(--text)', animation: 'heroIn 0.7s ease both', animationDelay: '0.15s' }}>
              Your thoughts,{' '}
              <em style={{ color: 'var(--gold)', fontStyle: 'italic' }}>out loud.</em>
            </h1>
            <p style={{ fontSize: '1.05rem', color: 'var(--muted)', lineHeight: 1.75, margin: '0 0 36px', maxWidth: 420, animation: 'heroIn 0.7s ease both', animationDelay: '0.28s' }}>
              Talk for two minutes or twenty. DreamLog listens, transcribes, and reflects back what your mind has been quietly trying to process, grounded in everything you&apos;ve shared before.
            </p>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', animation: 'heroIn 0.7s ease both', animationDelay: '0.4s' }}>
              <a href={dlHref} className="btn-primary" style={{ padding: '13px 26px', fontSize: '0.92rem', borderRadius: 12 }}>Start Journaling →</a>
              <a href="/login" className="btn-ghost" style={{ padding: '13px 26px', fontSize: '0.92rem', borderRadius: 12 }}>For Therapists</a>
            </div>
            <p className="serif" style={{ margin: '28px 0 0', fontSize: '0.92rem', color: 'var(--muted)', fontStyle: 'italic', animation: 'heroIn 0.7s ease both', animationDelay: '0.52s' }}>
              &ldquo;I finally have a place where I can understand myself.&rdquo;
            </p>
          </div>
          <div style={{ flexShrink: 0, paddingRight: 200 }} className="hero-mock">
            <AppMockup />
          </div>
        </div>
      </section>


      {/* ── How it works ──────────────────────────────────────────────── */}
      <section className="landing-section reveal" id="how" style={{ paddingTop: 80, paddingBottom: 80 }}>
        <h2 className="serif" style={{ fontSize: 'clamp(1.8rem, 3.5vw, 2.6rem)', fontWeight: 300, margin: '0 0 72px', maxWidth: 420 }}>
          Three quiet steps.<br />Then it remembers.
        </h2>
        <div className="how-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 0, position: 'relative' }}>
          {/* connecting line */}
          <div style={{ position: 'absolute', top: 18, left: '16.66%', right: '16.66%', height: 1, background: 'linear-gradient(90deg, transparent, var(--border-mid), var(--border-mid), transparent)', pointerEvents: 'none' }} />
          {[
            { n: '1', title: 'Speak freely', body: 'Open DreamLog and talk. No typing, no prompts. Just your voice and as much time as you need.' },
            { n: '2', title: 'AI reflects', body: "Your thoughts become an emotional reflection, grounded in what you've shared in past entries, not just today's." },
            { n: '3', title: 'Understand yourself', body: 'See recurring emotions, the people who keep appearing, and the slow shape of your own growth.' },
          ].map((step, i) => (
            <div key={step.n} style={{ paddingRight: i < 2 ? 48 : 0 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 28 }}>
                <div className="serif" style={{ fontSize: '0.72rem', fontWeight: 500, color: 'var(--gold)', opacity: 0.6, letterSpacing: 1 }}>{step.n}</div>
                <div style={{ flex: 1, height: 1, background: 'var(--border)' }} />
              </div>
              <h3 style={{ fontSize: '1.15rem', fontWeight: 600, color: 'var(--text)', margin: '0 0 14px', letterSpacing: '-0.01em' }}>{step.title}</h3>
              <p style={{ fontSize: '0.88rem', color: 'var(--muted)', lineHeight: 1.8, margin: 0 }}>{step.body}</p>
            </div>
          ))}
        </div>
      </section>


      {/* ── Journal as mirror ─────────────────────────────────────────── */}
      <section className="landing-section reveal" style={{ paddingTop: 0 }}>
        <div className="journal-grid" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 64, alignItems: 'center' }}>
          <div>
            <h2 className="serif" style={{ fontSize: 'clamp(1.8rem, 3vw, 2.5rem)', fontWeight: 300, margin: '0 0 20px', lineHeight: 1.2 }}>
              It feels like a journal,<br />not a dashboard.
            </h2>
            <p style={{ fontSize: '0.95rem', color: 'var(--muted)', lineHeight: 1.8, margin: 0 }}>
              No metrics. No streaks. No guilt. DreamLog summarises what your week was actually made of, and lets you begin to notice what keeps coming back.
            </p>
            <p style={{ fontSize: '0.95rem', color: 'var(--muted)', lineHeight: 1.8, margin: '16px 0 0' }}>
              There&apos;s a pull pattern around <span style={{ color: 'var(--text)' }}>family</span>. You speak about your <span style={{ color: 'var(--text)' }}>father</span> gently, without realising it. Two of your last three entries began with that name.
            </p>
          </div>

          {/* Reflection card — bigger */}
          <div style={{ background: 'var(--bg-card)', border: '1px solid var(--border-mid)', borderRadius: 20, padding: '36px 40px' }}>
            <div style={{ fontSize: 9, letterSpacing: 2, color: 'var(--muted-2)', textTransform: 'uppercase', marginBottom: 16 }}>
              TUESDAY · REFLECTION
            </div>
            <div className="serif" style={{ fontSize: '1.15rem', fontStyle: 'italic', color: 'var(--text)', lineHeight: 1.75, marginBottom: 28 }}>
              &ldquo;You sounded lighter on the morning you walked. The evenings, you mentioned a worry about your father, gently, without realising it.&rdquo;
            </div>
            <div style={{ fontSize: 9, letterSpacing: 2, color: 'var(--muted-2)', textTransform: 'uppercase', marginBottom: 12 }}>
              THIS WEEK, YOUR REFLECTIONS CIRCLED BACK TO
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 7 }}>
              {['Family', 'Rest', 'Morning Walks', 'Old Friends'].map(t => (
                <span key={t} className="chip" style={{ fontSize: '0.78rem', padding: '5px 13px' }}>{t}</span>
              ))}
            </div>
            <div style={{ marginTop: 24, paddingTop: 24, borderTop: '1px solid var(--border)' }}>
              <div style={{ fontSize: 9, letterSpacing: 2, color: 'var(--muted-2)', textTransform: 'uppercase', marginBottom: 10 }}>
                A SOFT QUESTION TO CARRY
              </div>
              <p className="serif" style={{ fontSize: '0.98rem', fontStyle: 'italic', color: 'var(--muted)', margin: 0, lineHeight: 1.7 }}>
                What would it feel like to say what you meant, without the pause before it?
              </p>
            </div>
          </div>
        </div>
      </section>

      <hr className="section-divider" />

      {/* ── Therapy mode ──────────────────────────────────────────────── */}
      <section className="landing-section reveal" id="therapy">
        <div className="therapy-intro-grid" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 64, alignItems: 'center', marginBottom: 48 }}>
          {/* Left: copy */}
          <div>
            <h2 className="serif" style={{ fontSize: 'clamp(1.8rem, 3vw, 2.5rem)', fontWeight: 300, margin: '0 0 20px', lineHeight: 1.2 }}>
              A conversation that already<br />knows your story.
            </h2>
            <p style={{ fontSize: '0.95rem', color: 'var(--muted)', lineHeight: 1.8, margin: 0 }}>
              Not a chatbot. A companion voice that has read everything you&apos;ve ever said to it, and comes prepared. Up to an hour. Voice or text. Four distinct tones to match where you are.
            </p>
            <p style={{ fontSize: '0.82rem', color: 'var(--muted-2)', margin: '20px 0 0', lineHeight: 1.7 }}>
              {p('therapy')} per session · included monthly with Pro · not a replacement for therapy
            </p>
          </div>
          {/* Right: breathing orb */}
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '40px 0' }}>
            <BreathingOrb />
          </div>
        </div>
        <div className="therapy-personas-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 14 }}>
          {[
            { icon: <IconHeart />, name: 'Comforting', sub: 'Warm, validating, feelings-first. Holds space before it asks anything.' },
            { icon: <IconCompass />, name: 'Rational', sub: 'Structured and Socratic. Helps you think clearly without judgment.' },
            { icon: <IconLens />, name: 'CBT-Informed', sub: 'Names thought patterns gently. Asks what the evidence actually says.' },
            { icon: <IconBreath />, name: 'Mindful', sub: 'Grounding and present. Works with breath, not despite it.' },
          ].map(p => (
            <div key={p.name} className="card" style={{ padding: '22px 20px' }}>
              <div style={{ color: 'var(--gold)', marginBottom: 14 }}>{p.icon}</div>
              <div style={{ fontWeight: 600, fontSize: '0.92rem', color: 'var(--text)', marginBottom: 9 }}>{p.name}</div>
              <p style={{ fontSize: '0.82rem', color: 'var(--muted)', margin: 0, lineHeight: 1.65 }}>{p.sub}</p>
            </div>
          ))}
        </div>
      </section>

      {/* ── Dream Decoder ─────────────────────────────────────────────── */}
      <section className="landing-section reveal" id="dream" style={{ paddingTop: 0 }}>
        <h2 className="serif" style={{ fontSize: 'clamp(1.8rem, 3vw, 2.5rem)', fontWeight: 300, margin: '0 0 12px' }}>
          Two lenses for the dreams<br />you can&apos;t shake.
        </h2>
        <p style={{ fontSize: '0.95rem', color: 'var(--muted)', margin: '0 0 40px', lineHeight: 1.75, maxWidth: 520 }}>
          Mysterious, considered, modern. Not mystical. Not fortune-telling. Just two thoughtful ways to look at the images your mind made last night.
        </p>
        <div className="dream-grid" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          {[
            {
              icon: 'Ψ', label: 'PSYCHOLOGICAL REFLECTION', name: 'Jungian lens',
              body: "Read your dream as an inner conversation. The shadows, the figures, the unresolved themes, surfaced gently, without prescription.",
              quote: "The man you couldn't find in the dream might be the part of you that hasn't been allowed to speak.",
            },
            {
              icon: 'ॐ', label: 'INDIAN SYMBOLIC REFLECTION', name: 'Vedic lens',
              body: "Interpret your dream through traditional Indian symbolism: rivers, doors, animals, time of night. Cultural, careful, contextual.",
              quote: "The river that wouldn't carry you may signify a passage you have not yet asked permission to cross.",
            },
          ].map(lens => (
            <div key={lens.name} className="card dream-card" style={{ padding: '32px 36px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 20 }}>
                <div className="eyebrow">{lens.label}</div>
                <span className="dream-symbol" style={{ fontSize: '1.5rem', color: 'var(--gold)', lineHeight: 1, opacity: 0.75 }}>{lens.icon}</span>
              </div>
              <h3 style={{ fontSize: '1.3rem', fontWeight: 600, color: 'var(--text)', margin: '0 0 14px' }}>{lens.name}</h3>
              <p style={{ fontSize: '0.9rem', color: 'var(--text)', margin: '0 0 22px', lineHeight: 1.75, opacity: 0.8 }}>{lens.body}</p>
              <p className="serif" style={{ fontStyle: 'italic', fontSize: '0.95rem', color: 'var(--muted)', margin: 0, lineHeight: 1.7, borderLeft: '2px solid var(--border-mid)', paddingLeft: 14 }}>
                &ldquo;{lens.quote}&rdquo;
              </p>
            </div>
          ))}
        </div>
      </section>

      <hr className="section-divider" />

      {/* ── Features ──────────────────────────────────────────────────── */}
      <section className="landing-section reveal" id="features">
        <h2 className="serif" style={{ fontSize: 'clamp(1.8rem, 3vw, 2.5rem)', fontWeight: 300, margin: '0 0 48px' }}>
          Quiet tools for<br />understanding yourself.
        </h2>

        {/* 3 hero features — staggered slide-up */}
        <div className="features-hero-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2, marginBottom: 2 }}>
          {[
            { icon: <IconMic />, name: 'Voice Journaling', body: "Speak naturally. No typing, no prompts. DreamLog captures your thoughts the way they actually arrive, unfiltered, in your own voice." },
            { icon: <IconSparkle />, name: 'AI Emotional Reflection', body: "Your entries become a private mirror. Mood, themes, recurring phrases, the quiet things you didn't say out loud, surfaced gently after every recording." },
            { icon: <IconChat />, name: 'Therapy Mode', body: "A companion conversation that already understands your emotional history. Up to an hour, voice or text, four distinct tones." },
          ].map((f, i) => (
            <div key={f.name} className="feat-hero reveal" style={{
              background: i === 1 ? 'rgba(200,149,90,0.04)' : 'var(--bg-card)',
              border: `1px solid ${i === 1 ? 'rgba(200,149,90,0.18)' : 'var(--border)'}`,
              borderRadius: i === 0 ? '16px 0 0 16px' : i === 2 ? '0 16px 16px 0' : 0,
              padding: '32px 28px 36px',
              transitionDelay: `${i * 0.1}s`,
            }}>
              <div className="feat-icon" style={{ color: 'var(--gold)', marginBottom: 20, opacity: 0.75 }}>{f.icon}</div>
              <div style={{ fontWeight: 700, fontSize: '1rem', color: 'var(--text)', marginBottom: 12, letterSpacing: '-0.01em' }}>{f.name}</div>
              <p style={{ fontSize: '0.88rem', color: 'var(--muted)', margin: 0, lineHeight: 1.8 }}>{f.body}</p>
            </div>
          ))}
        </div>

        {/* 6 secondary features — hover accent rows */}
        <div className="features-sub-grid" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 0, background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: '0 0 16px 16px', overflow: 'hidden' }}>
          {[
            { icon: <IconMoon />, name: 'Dream Decoder', body: "Jungian + Vedic lenses for the dreams you can't shake." },
            { icon: <IconBook />, name: 'Life Chapters', body: "Your years, organised into the stories you've actually lived." },
            { icon: <IconChart />, name: 'Mood Tracking', body: "No streaks, no guilt. Just patterns you can finally see." },
            { icon: <IconPeople />, name: 'Relationship Map', body: "See how the people you mention shape your emotional weather." },
            { icon: <IconPath />, name: 'Guided Journeys', body: "Short paths through grief, change, anxiety, and self-trust." },
            { icon: <IconCalendar />, name: 'Weekly & Annual Reviews', body: "A letter from your past self, every Sunday and one each December." },
          ].map((f, i) => (
            <div key={f.name} className="feat-row" style={{
              display: 'flex', gap: 14, alignItems: 'flex-start',
              padding: '20px 24px',
              borderTop: i >= 2 ? '1px solid var(--border)' : 'none',
              borderRight: i % 2 === 0 ? '1px solid var(--border)' : 'none',
            }}>
              <div className="feat-row-icon" style={{ color: 'var(--gold)', opacity: 0.45, flexShrink: 0, marginTop: 1 }}>{f.icon}</div>
              <div>
                <div style={{ fontSize: '0.84rem', fontWeight: 600, color: 'var(--text)', marginBottom: 3 }}>{f.name}</div>
                <div style={{ fontSize: '0.78rem', color: 'var(--muted)', lineHeight: 1.6 }}>{f.body}</div>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* ── Testimonials ──────────────────────────────────────────────── */}
      <section className="landing-section reveal">
        <div className="testimonials-grid" style={{ display: 'grid', gridTemplateColumns: '55fr 45fr', gap: 16, alignItems: 'stretch' }}>
          {/* Large pull quote */}
          <div className="testimonial-large reveal-left" style={{ background: 'rgba(200,149,90,0.04)', border: '1px solid rgba(200,149,90,0.14)', borderRadius: 20, padding: '44px 48px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
            <div className="serif" style={{ fontSize: '1.4rem', fontWeight: 300, fontStyle: 'italic', lineHeight: 1.7, color: 'var(--text)', marginBottom: 32 }}>
              &ldquo;It noticed I kept mentioning my father without realising it. Three entries in a row. Nobody had ever pointed that out before.&rdquo;
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <div style={{ width: 32, height: 32, borderRadius: '50%', background: 'rgba(200,149,90,0.12)', border: '1px solid rgba(200,149,90,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '0.7rem', color: 'var(--gold)', fontFamily: "'Cormorant Garamond', serif", fontWeight: 600 }}>A</div>
              <span style={{ fontSize: '0.8rem', color: 'var(--muted)' }}>A user, 41, after the weekly review</span>
            </div>
          </div>
          {/* Two smaller quotes stacked */}
          <div className="reveal-right" style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div className="testimonial-small" style={{ flex: 1, background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 20, padding: '28px 32px' }}>
              <p className="serif" style={{ fontStyle: 'italic', fontSize: '1rem', lineHeight: 1.75, color: 'var(--text)', margin: '0 0 20px' }}>
                &ldquo;I&apos;ve kept journals for years but always gave up after a week. Talking feels different. Something about hearing my own voice makes the reflection land.&rdquo;
              </p>
              <p style={{ fontSize: '0.78rem', color: 'var(--muted)', margin: 0 }}>Priya, 28, 3-month user</p>
            </div>
            <div className="testimonial-small" style={{ flex: 1, background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 20, padding: '28px 32px' }}>
              <p className="serif" style={{ fontStyle: 'italic', fontSize: '1rem', lineHeight: 1.75, color: 'var(--text)', margin: '0 0 20px' }}>
                &ldquo;The AI doesn&apos;t say what I want to hear. It says what I actually said. That&apos;s a harder and better thing.&rdquo;
              </p>
              <p style={{ fontSize: '0.78rem', color: 'var(--muted)', margin: 0 }}>A user, 34, six weeks in</p>
            </div>
          </div>
        </div>
      </section>

      {/* ── Pricing ───────────────────────────────────────────────────── */}
      <section className="landing-section reveal" id="pricing">
        <h2 className="serif" style={{ fontSize: 'clamp(1.8rem, 3vw, 2.5rem)', fontWeight: 300, margin: '0 0 10px' }}>Honest pricing.</h2>
        <p style={{ fontSize: '0.9rem', color: 'var(--muted)', margin: '0 0 44px' }}>Most people start Free. You never need to upgrade. It&apos;s allowed.</p>
        <div className="pricing-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16 }}>
          <PricingCard name="Free" price={p('free')} period="forever" sub="10 entries a month. Reflections, mood tracking, 3-turn follow-ups." features={['10 entries / month', 'AI emotional reflection', '7-day mood chart', '3-turn follow-up conversation', 'Crisis detection always on']} cta="Download Free" downloadHref={dlHref} ready={ready} />
          <PricingCard name="DreamLog+" price={p('plus')} period={per} sub="Unlimited entries. All modes. The complete journaling product." features={['Unlimited entries', 'All 5 journaling modes', 'Dream Decoder (Jungian + Vedic)', 'Life Graph & Mood History', 'Weekly + Annual Reviews', 'Life Chapters', 'PDF export', 'Therapist share (5/month)']} cta="Get DreamLog+" downloadHref={dlHref} featured ready={ready} />
          <PricingCard name="DreamLog Pro" price={p('pro')} period={per} sub="Everything in Plus, + 1 therapy." features={['Everything in Plus', '1 Therapy Session / month', `Extra sessions at ${currency === 'INR' ? '₹299' : currency === 'EUR' ? '€3.99' : '$4.99'}`, 'Unlimited therapist share', 'Priority processing']} cta="Get DreamLog Pro" downloadHref={dlHref} ready={ready} />
        </div>
        <p style={{ marginTop: 16, fontSize: '0.78rem', color: 'var(--muted-2)', textAlign: 'center' }}>
          All subscriptions are managed in-app after download. One-time 30-day passes, no auto-renew.
        </p>
      </section>

      {/* ── Corporate Wellness ────────────────────────────────────────── */}
      <section style={{ background: 'var(--bg-card-2)', borderTop: '1px solid var(--border)', borderBottom: '1px solid var(--border)' }}>
        <div className="b2b-inner" style={{ maxWidth: 1320, margin: '0 auto', padding: '64px 60px', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 60, alignItems: 'start' }}>
          <div>
            <div style={{ fontSize: '0.6rem', letterSpacing: '2.5px', textTransform: 'uppercase', color: 'var(--gold)', fontWeight: 600, marginBottom: 18, opacity: 0.7 }}>FOR ORGANISATIONS</div>
            <h2 className="serif" style={{ fontSize: 'clamp(1.6rem, 3vw, 2.4rem)', fontWeight: 300, margin: '0 0 16px', lineHeight: 1.2, color: 'var(--text)' }}>
              DreamLog for teams<br />who actually care.
            </h2>
            <p style={{ fontSize: '0.9rem', color: 'var(--muted)', lineHeight: 1.8, margin: '0 0 28px', maxWidth: 420 }}>
              Aggregate, anonymized emotional wellbeing insights across your team. Individual data never leaves the user&apos;s device without explicit consent.
            </p>
            <a href="mailto:support@dreamlog.app" className="btn-ghost" style={{ padding: '12px 26px', borderRadius: 100, whiteSpace: 'nowrap', fontSize: '0.86rem', display: 'inline-flex' }}>
              Talk to us →
            </a>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 0, paddingTop: 4 }}>
            {[
              { num: '100%', label: 'Anonymous to admins', body: 'No individual entry, transcript, or recording is ever visible to HR or management.' },
              { num: 'Weekly', label: 'Aggregate mood report', body: 'Team-level emotional patterns surfaced without identifying anyone.' },
              { num: 'Yours', label: 'Data stays with employees', body: 'Individual data never leaves the user without their explicit opt-in.' },
            ].map((item, i) => (
              <div key={item.label} style={{ padding: '20px 0', borderTop: i === 0 ? 'none' : '1px solid var(--border)', display: 'flex', gap: 20, alignItems: 'flex-start' }}>
                <div className="serif" style={{ fontSize: '1.5rem', fontWeight: 300, color: 'var(--gold)', lineHeight: 1, flexShrink: 0, minWidth: 64 }}>{item.num}</div>
                <div>
                  <div style={{ fontSize: '0.84rem', fontWeight: 600, color: 'var(--text)', marginBottom: 4 }}>{item.label}</div>
                  <div style={{ fontSize: '0.8rem', color: 'var(--muted)', lineHeight: 1.65 }}>{item.body}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ── For clinicians ────────────────────────────────────────────── */}
      <section className="landing-section-sm reveal">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 48, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 280 }}>
            <div style={{ fontSize: '0.6rem', letterSpacing: '2.5px', textTransform: 'uppercase', color: 'var(--gold)', fontWeight: 600, marginBottom: 16, opacity: 0.7 }}>FOR CLINICIANS</div>
            <h2 className="serif" style={{ fontSize: 'clamp(1.5rem, 2.5vw, 2rem)', fontWeight: 300, margin: '0 0 14px' }}>
              For the therapists who send their clients here.
            </h2>
            <p style={{ fontSize: '0.9rem', color: 'var(--muted)', lineHeight: 1.75, margin: '0 0 24px' }}>
              Your clients own the data. You only see what they choose to share, and only while the link is live.
            </p>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 11 }}>
              {[
                'AI-generated pre-session brief drawn from the last 5 entries',
                'Mood trends and top emotions, no raw transcripts or recordings',
                'Passcode-protected share links that auto-expire in 72 hours',
                'Client-controlled: they generate the link, not you',
              ].map(item => (
                <div key={item} style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
                  <span style={{ color: 'var(--gold)', flexShrink: 0, fontSize: '0.7rem', marginTop: 3, opacity: 0.7 }}>✦</span>
                  <span style={{ fontSize: '0.86rem', color: 'var(--muted)', lineHeight: 1.55 }}>{item}</span>
                </div>
              ))}
            </div>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 12, flexShrink: 0, paddingTop: 4 }}>
            <a href="/login" className="btn-ghost" style={{ padding: '13px 26px', borderRadius: 12, whiteSpace: 'nowrap' }}>
              Open Therapist Portal →
            </a>
            <span style={{ fontSize: '0.76rem', color: 'var(--muted-2)', textAlign: 'right' }}>Free to register as a therapist</span>
          </div>
        </div>
      </section>

      {/* ── FAQ ───────────────────────────────────────────────────────── */}
      <section className="landing-section-sm reveal">
        <div className="faq-grid" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 60px' }}>
          <div>
            <FAQ q="Is this actually private?" a="Audio is deleted immediately after transcription. Transcripts live on our servers, encrypted. We don't sell data or train on your entries. You can export or delete everything at any time." />
            <FAQ q="How is this different from a general AI chat?" a="It knows your history. Every reflection is grounded in your last 5 entries: the emotions, topics, and patterns that recur. A general chat has no memory. DreamLog does." />
            <FAQ q="Does it work in Hindi or Hinglish?" a="Yes. Your language is detected automatically. Reflections are generated in the same language. Hindi and Hinglish support is included from DreamLog+." />
          </div>
          <div>
            <FAQ q="Can I share this with my therapist?" a="Yes. From Settings, you generate a passcode-protected link. Your therapist sees mood trends and AI summaries, never raw transcripts or recordings." />
            <FAQ q="What happens if I say something in crisis?" a="Two-stage detection runs on every entry. If distress is detected, you get crisis resources immediately: iCall, Vandrevala Foundation, 988 (US). The entry is flagged and handled separately." />
          </div>
        </div>
      </section>

      <hr className="section-divider" />

      {/* ── Download CTA ──────────────────────────────────────────────── */}
      <section id="download" className="landing-section reveal" style={{ textAlign: 'center', padding: '100px 40px 80px' }}>
        <h2 className="serif" style={{ fontSize: 'clamp(2rem, 4vw, 3.2rem)', fontWeight: 300, margin: '0 0 20px', lineHeight: 1.2 }}>
          Your thoughts are<br /><em style={{ color: 'var(--gold)' }}>worth understanding.</em>
        </h2>
        <p style={{ fontSize: '0.95rem', color: 'var(--muted)', margin: '0 0 36px' }}>
          Available on iOS and Android. Free to start.
        </p>
        <DownloadButtons androidUrl={version.android_store_url} iosUrl={version.ios_store_url} />
        {version.ios_store_url === '#' && (
          <p style={{ marginTop: 16, fontSize: '0.78rem', color: 'var(--muted-2)' }}>
            Store listings coming soon. <a href="mailto:support@dreamlog.app" style={{ color: 'var(--muted)' }}>Get notified</a>
          </p>
        )}
      </section>

      {/* ── Footer ────────────────────────────────────────────────────── */}
      <footer style={{ borderTop: '1px solid var(--border)', padding: '48px 60px', maxWidth: 1320, margin: '0 auto' }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr auto auto auto', gap: 40, alignItems: 'start' }}>
          <div>
            <a href="/" style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
              <div style={{ width: 26, height: 26, borderRadius: 6, background: 'var(--gold)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '0.95rem', color: '#18150f' }}>D</div>
              <span className="serif" style={{ fontSize: '1rem', fontWeight: 600 }}>DreamLog</span>
            </a>
            <p style={{ fontSize: '0.8rem', color: 'var(--muted-2)', lineHeight: 1.7, margin: 0, maxWidth: 240 }}>
              Voice journaling with AI reflection. Built for honesty, not performance.
            </p>
          </div>
          <div>
            <div style={{ fontSize: '0.72rem', letterSpacing: '1.5px', textTransform: 'uppercase', color: 'var(--muted-2)', fontWeight: 600, marginBottom: 12 }}>Product</div>
            {[{ label: 'Features', href: '#features' }, { label: 'Therapy Mode', href: '#therapy' }, { label: 'Dream Decoder', href: '#dream' }, { label: 'Pricing', href: '#pricing' }, { label: 'For Therapists', href: '/login' }].map(l => (
              <a key={l.label} href={l.href} style={{ display: 'block', fontSize: '0.84rem', color: 'var(--muted)', marginBottom: 8 }}>{l.label}</a>
            ))}
          </div>
          <div>
            <div style={{ fontSize: '0.72rem', letterSpacing: '1.5px', textTransform: 'uppercase', color: 'var(--muted-2)', fontWeight: 600, marginBottom: 12 }}>Company</div>
            <a href="/privacy" style={{ display: 'block', fontSize: '0.84rem', color: 'var(--muted)', marginBottom: 8 }}>Privacy Policy</a>
            <a href="/terms" style={{ display: 'block', fontSize: '0.84rem', color: 'var(--muted)', marginBottom: 8 }}>Terms</a>
            <a href="mailto:support@dreamlog.app" style={{ display: 'block', fontSize: '0.84rem', color: 'var(--muted)', marginBottom: 8 }}>Support</a>
          </div>
          <div>
            <div style={{ fontSize: '0.72rem', letterSpacing: '1.5px', textTransform: 'uppercase', color: 'var(--muted-2)', fontWeight: 600, marginBottom: 12 }}>If you need help</div>
            {[{ label: 'iCall: 9152987821', href: 'tel:9152987821' }, { label: 'Vandrevala: 1860-2662-345', href: 'tel:18602662345' }, { label: '988 (US/Canada)', href: 'tel:988' }].map(r => (
              <a key={r.label} href={r.href} style={{ display: 'block', fontSize: '0.8rem', color: 'var(--muted-2)', marginBottom: 7 }}>{r.label}</a>
            ))}
          </div>
        </div>
        <div style={{ marginTop: 48, paddingTop: 24, borderTop: '1px solid var(--border)', fontSize: '0.75rem', color: 'var(--muted-2)' }}>
          © 2026 DreamLog · <a href="mailto:support@dreamlog.app" style={{ color: 'var(--muted-2)' }}>support@dreamlog.app</a>
        </div>
      </footer>

      <style>{`
        /* ── Tablet (960px) ── */
        @media (max-width: 960px) {
          .hero-mock { display: none !important; }
        }

        /* ── Tablet (860px) ── */
        @media (max-width: 860px) {
          .therapy-personas-grid { grid-template-columns: repeat(2, 1fr) !important; }
          .dream-grid            { grid-template-columns: 1fr !important; }
          .features-hero-grid    { grid-template-columns: repeat(2, 1fr) !important; }
          .pricing-grid          { grid-template-columns: 1fr !important; }
          .testimonials-grid     { grid-template-columns: 1fr !important; }
          .b2b-inner             { padding: 56px 40px !important; gap: 40px !important; }
        }

        /* ── Mobile (640px) ── */
        @media (max-width: 640px) {
          /* Nav */
          .landing-nav { padding: 0 20px !important; }
          .landing-nav > div:last-child a:not(.btn-primary) { display: none; }

          /* Section padding */
          .landing-section    { padding: 64px 20px 56px !important; }
          .landing-section-sm { padding: 40px 20px !important; }

          /* Hero flex row → column */
          .landing-section > div[style*="space-between"] { flex-direction: column !important; gap: 40px !important; }

          /* How it works: 3-col → 1-col */
          .how-grid { grid-template-columns: 1fr !important; gap: 32px !important; }
          .how-grid > div { padding-right: 0 !important; }

          /* Journal mirror: 2-col → 1-col */
          .journal-grid { grid-template-columns: 1fr !important; gap: 32px !important; }

          /* Therapy intro + orb: 2-col → 1-col */
          .therapy-intro-grid { grid-template-columns: 1fr !important; gap: 32px !important; }

          /* Therapy personas: 4-col → 2-col */
          .therapy-personas-grid { grid-template-columns: repeat(2, 1fr) !important; }

          /* Dream decoder: 2-col → 1-col */
          .dream-grid { grid-template-columns: 1fr !important; }

          /* Feature hero cards: 3-col → 1-col; fix border-radius */
          .features-hero-grid { grid-template-columns: 1fr !important; }
          .features-hero-grid .feat-hero:first-child { border-radius: 16px 16px 0 0 !important; }
          .features-hero-grid .feat-hero:last-child  { border-radius: 0 0 16px 16px !important; }
          .features-hero-grid .feat-hero:not(:first-child):not(:last-child) { border-radius: 0 !important; }

          /* Feature sub-grid: 2-col → 1-col */
          .features-sub-grid { grid-template-columns: 1fr !important; }
          .features-sub-grid > div { border-right: none !important; border-top: 1px solid var(--border) !important; }
          .features-sub-grid > div:first-child { border-top: none !important; }

          /* Testimonials: stacked */
          .testimonials-grid { grid-template-columns: 1fr !important; }

          /* Pricing: 3-col → 1-col */
          .pricing-grid { grid-template-columns: 1fr !important; }

          /* B2B: 2-col → 1-col */
          .b2b-inner { grid-template-columns: 1fr !important; padding: 48px 20px !important; gap: 32px !important; }

          /* For clinicians row: wrap */
          .landing-section-sm > div[style*="space-between"] { flex-direction: column !important; align-items: flex-start !important; gap: 20px !important; }

          /* FAQ: 2-col → 1-col */
          .faq-grid { grid-template-columns: 1fr !important; }

          /* Download buttons */
          #download > div { flex-direction: column !important; align-items: center; }

          /* Footer */
          footer > div:first-child { grid-template-columns: 1fr 1fr !important; gap: 28px !important; }
          footer { padding: 40px 20px !important; }
        }

        /* ── Small mobile (420px) ── */
        @media (max-width: 420px) {
          footer > div:first-child { grid-template-columns: 1fr !important; }
          .landing-nav > div:last-child .btn-primary { padding: 8px 14px !important; font-size: 0.8rem !important; }
          .therapy-personas-grid { grid-template-columns: 1fr !important; }
        }
      `}</style>
    </div>
  );
}
