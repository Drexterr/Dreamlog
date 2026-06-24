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
    <div className={featured ? 'pricing-featured' : 'pricing-card'} style={{ background: featured ? 'rgba(212,165,106,0.07)' : 'var(--bg-card)', border: `1px solid ${featured ? 'rgba(212,165,106,0.35)' : 'var(--border)'}`, borderRadius: 20, padding: '28px 24px', display: 'flex', flexDirection: 'column', position: 'relative' }}>
      {featured && (
        <div style={{ position: 'absolute', top: -12, left: '50%', transform: 'translateX(-50%)', background: 'var(--gold)', color: '#18150f', fontSize: '0.65rem', fontWeight: 800, letterSpacing: '1.5px', textTransform: 'uppercase', padding: '5px 14px', borderRadius: 100, whiteSpace: 'nowrap' }}>Most popular</div>
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

/* ── Insight ticker ───────────────────────────────────────────────────────── */
function InsightTicker() {
  const items = [
    'You mentioned "tired" in 9 of your last 14 entries',
    'Your entries are 40% longer on weekends',
    "You've brought up your sister three weeks in a row",
    'Tone is noticeably lighter when you record before noon',
    'Recurring theme this month: feeling behind',
    'You asked a question at the end of 6 out of 7 entries',
    'The word "fine" appeared 31 times — usually at the start',
    'Most hopeful entry: last Tuesday, 8:47am',
  ];
  const all = [...items, ...items];
  return (
    <div style={{
      borderTop: '1px solid var(--border)',
      borderBottom: '1px solid var(--border)',
      overflow: 'hidden',
      padding: '15px 0',
      cursor: 'default',
      maskImage: 'linear-gradient(90deg, transparent 0%, black 8%, black 92%, transparent 100%)',
      WebkitMaskImage: 'linear-gradient(90deg, transparent 0%, black 8%, black 92%, transparent 100%)',
    }}>
      <div className="ticker-track" style={{ display: 'inline-flex', whiteSpace: 'nowrap' }}>
        {all.map((item, i) => (
          <span key={i} style={{
            fontSize: '0.76rem',
            color: 'var(--muted)',
            padding: '0 40px',
            display: 'inline-flex',
            alignItems: 'center',
            gap: '40px',
            flexShrink: 0,
          }}>
            <span style={{ color: 'var(--gold)', opacity: 0.35, fontSize: '0.42rem' }}>◆</span>
            {item}
          </span>
        ))}
      </div>
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

      <a href="#main-content" className="skip-link">Skip to main content</a>

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
          <a href="/login" className="nav-link" style={{ padding: '8px 14px', fontSize: '0.86rem', color: 'var(--muted)', fontWeight: 500 }}>Therapist Portal</a>
          <a href={dlHref} className="btn-primary" style={{ padding: '9px 20px', fontSize: '0.84rem', borderRadius: 100 }}>Download Free</a>
        </div>
      </nav>

      {/* ── Hero ──────────────────────────────────────────────────────── */}
      <section id="main-content" className="landing-section" style={{ paddingTop: 160, paddingBottom: 100, position: 'relative', overflow: 'hidden', minHeight: '100dvh', display: 'flex', alignItems: 'center' }}>
        {/* Ambient glow — top-left bleed */}
        <div aria-hidden="true" style={{
          position: 'absolute', top: '-10%', left: '-5%',
          width: '55%', height: '80%',
          background: 'radial-gradient(ellipse 60% 70% at 30% 40%, rgba(200,149,90,0.06) 0%, transparent 65%)',
          pointerEvents: 'none',
        }} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 60, position: 'relative', width: '100%' }}>
          {/* Left: copy — takes 55% */}
          <div style={{ flex: '0 0 55%', maxWidth: 640 }}>
            <h1 className="serif" style={{ fontSize: 'clamp(3.8rem, 6.5vw, 7rem)', fontWeight: 300, lineHeight: 1.05, margin: '0 0 28px', color: 'var(--text)', animation: 'heroIn 0.8s cubic-bezier(0.16,1,0.3,1) both', animationDelay: '0.1s', letterSpacing: '-0.02em' }}>
              Your thoughts,<br /><em style={{ color: 'var(--gold)' }}>out loud.</em>
            </h1>
            <p style={{ fontSize: '1.05rem', color: 'var(--muted)', lineHeight: 1.75, margin: '0 0 36px', maxWidth: 440, animation: 'heroIn 0.8s cubic-bezier(0.16,1,0.3,1) both', animationDelay: '0.22s' }}>
              Record your day in your own voice. DreamLog transcribes it, cross-references your last five entries, and shows you what you keep coming back to.
            </p>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', animation: 'heroIn 0.8s cubic-bezier(0.16,1,0.3,1) both', animationDelay: '0.34s' }}>
              <a href={dlHref} className="btn-primary" style={{ padding: '14px 28px', fontSize: '0.92rem' }}>Start journaling free</a>
              <a href="/login" className="btn-ghost" style={{ padding: '14px 28px', fontSize: '0.92rem' }}>Therapist portal</a>
            </div>
          </div>
          {/* Right: phone mockup with floating cards */}
          <div className="hero-mock" style={{ flex: 1, display: 'flex', justifyContent: 'center', animation: 'heroIn 1s cubic-bezier(0.16,1,0.3,1) both', animationDelay: '0.3s' }}>
            <AppMockup />
          </div>
        </div>
      </section>


      <InsightTicker />

      {/* ── How it works ──────────────────────────────────────────────── */}
      <section className="landing-section reveal" id="how">
        <h2 className="serif" style={{ fontSize: 'clamp(1.8rem, 3.5vw, 2.6rem)', fontWeight: 300, margin: '0 0 72px', maxWidth: 420 }}>
          Three steps.<br />Then it connects the dots.
        </h2>
        <div className="how-grid stagger" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 0, position: 'relative' }}>
          {/* connecting line */}
          <div style={{ position: 'absolute', top: 18, left: '16.66%', right: '16.66%', height: 1, background: 'linear-gradient(90deg, transparent, var(--border-mid), var(--border-mid), transparent)', pointerEvents: 'none' }} />
          {[
            { n: '1', title: 'Speak freely', body: 'Open DreamLog and talk. No typing, no prompts. Two minutes or an hour — it just records.' },
            { n: '2', title: 'AI reflects', body: "Your words get transcribed and cross-referenced with your last five entries. That's where the patterns come from." },
            { n: '3', title: 'See the pattern', body: 'The same names keep appearing. The same worries. The same small hope. You start to see what actually matters to you.' },
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
              No scores. No streaks. No productivity guilt. Just a summary of what you actually said this week — the names, the feelings, the question worth sitting with.
            </p>
            <p style={{ fontSize: '0.95rem', color: 'var(--muted)', lineHeight: 1.8, margin: '16px 0 0' }}>
              The card on the right shows what a Tuesday reflection looks like. DreamLog picked up the morning walk, a quiet worry about <span style={{ color: 'var(--text)' }}>family</span>, and left one question worth carrying into Wednesday.
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
        {/* Split panel: text left, orb fills entire right */}
        <div style={{
          display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 0,
          background: 'var(--bg-card)', border: '1px solid var(--border)',
          borderRadius: 24, overflow: 'hidden', marginBottom: 44,
        }}>
          {/* Left: copy */}
          <div style={{ padding: '56px 52px', display: 'flex', flexDirection: 'column', justifyContent: 'center', borderRight: '1px solid var(--border)' }}>
            <h2 className="serif" style={{ fontSize: 'clamp(2rem, 3.8vw, 3.4rem)', fontWeight: 300, margin: '0 0 20px', lineHeight: 1.05, letterSpacing: '-0.025em' }}>
              A conversation<br />that already knows<br /><em style={{ color: 'var(--gold)' }}>your story.</em>
            </h2>
            <p style={{ fontSize: '0.95rem', color: 'var(--muted)', lineHeight: 1.8, margin: '0 0 28px', maxWidth: 380 }}>
              Not a chatbot. A companion that has read everything you&apos;ve said to it, and comes prepared. Up to an hour. Voice or text. Four distinct tones.
            </p>
            <p style={{ fontSize: '0.78rem', color: 'var(--muted-2)', margin: 0, lineHeight: 1.8 }}>
              {p('therapy')} per session · included monthly with Pro<br /><em>not a replacement for therapy</em>
            </p>
          </div>
          {/* Right: orb fills the whole panel */}
          <div style={{
            position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center',
            minHeight: 420, padding: '60px 40px', overflow: 'hidden',
            background: 'radial-gradient(ellipse 70% 70% at 50% 50%, rgba(200,149,90,0.04) 0%, transparent 70%)',
          }}>
            <div aria-hidden="true" style={{ position: 'absolute', inset: 0, margin: 'auto', width: 380, height: 380, borderRadius: '50%', border: '1px solid rgba(200,149,90,0.06)', animation: 'breathIdle 7s ease-in-out infinite', pointerEvents: 'none' }} />
            <div aria-hidden="true" style={{ position: 'absolute', inset: 0, margin: 'auto', width: 300, height: 300, borderRadius: '50%', border: '1px solid rgba(200,149,90,0.09)', animation: 'breathIdle 7s ease-in-out 1s infinite', pointerEvents: 'none' }} />
            <div aria-hidden="true" style={{ position: 'absolute', inset: 0, margin: 'auto', width: 220, height: 220, borderRadius: '50%', border: '1px solid rgba(200,149,90,0.12)', animation: 'breathIdle 7s ease-in-out 2s infinite', pointerEvents: 'none' }} />
            <BreathingOrb />
          </div>
        </div>
        <div className="therapy-personas-grid stagger" style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 14 }}>
          {[
            { icon: <IconHeart />, name: 'Comforting', sub: 'Warm, validating, feelings-first. Holds space before it asks anything.' },
            { icon: <IconCompass />, name: 'Rational', sub: 'Structured and Socratic. Helps you think clearly without judgment.' },
            { icon: <IconLens />, name: 'CBT-Informed', sub: 'Names thought patterns gently. Asks what the evidence actually says.' },
            { icon: <IconBreath />, name: 'Mindful', sub: 'Grounding and present. Works with breath, not despite it.' },
          ].map(persona => (
            <div key={persona.name} className="card" style={{ padding: '22px 20px' }}>
              <div style={{ color: 'var(--gold)', marginBottom: 14 }}>{persona.icon}</div>
              <div style={{ fontWeight: 600, fontSize: '0.92rem', color: 'var(--text)', marginBottom: 9 }}>{persona.name}</div>
              <p style={{ fontSize: '0.82rem', color: 'var(--muted)', margin: 0, lineHeight: 1.65 }}>{persona.sub}</p>
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
        {/* Single split-panel — not two equal cards */}
        <div className="dream-split" style={{
          display: 'grid', gridTemplateColumns: '1fr 1px 1fr', gap: 0,
          background: 'var(--bg-card)', border: '1px solid var(--border)',
          borderRadius: 20, overflow: 'hidden',
        }}>
          {/* Jungian */}
          <div className="dream-panel" style={{ padding: '48px 44px', position: 'relative', overflow: 'hidden' }}>
            <div aria-hidden="true" style={{ position: 'absolute', right: -24, top: -16, fontSize: '9rem', color: 'rgba(200,149,90,0.05)', fontFamily: 'Georgia, serif', userSelect: 'none', lineHeight: 1, pointerEvents: 'none' }}>Ψ</div>
            <div style={{ fontSize: '0.58rem', letterSpacing: '2.5px', color: 'var(--muted-2)', textTransform: 'uppercase', marginBottom: 20 }}>Psychological</div>
            <h3 className="serif" style={{ fontSize: '1.7rem', fontWeight: 300, color: 'var(--text)', margin: '0 0 16px', fontStyle: 'italic', lineHeight: 1.2 }}>Jungian lens</h3>
            <p style={{ fontSize: '0.88rem', color: 'var(--muted)', margin: '0 0 28px', lineHeight: 1.8 }}>
              Read your dream as an inner conversation. The shadows, the figures, the unresolved themes — surfaced gently, without prescription.
            </p>
            <p className="serif" style={{ fontStyle: 'italic', fontSize: '1rem', color: 'var(--text)', margin: 0, lineHeight: 1.7, opacity: 0.8 }}>
              &ldquo;The man you couldn&apos;t find in the dream might be the part of you that hasn&apos;t been allowed to speak.&rdquo;
            </p>
          </div>
          {/* Divider */}
          <div style={{ background: 'var(--border)' }} />
          {/* Vedic */}
          <div className="dream-panel" style={{ padding: '48px 44px', position: 'relative', overflow: 'hidden', background: 'rgba(200,149,90,0.02)' }}>
            <div aria-hidden="true" style={{ position: 'absolute', right: -20, top: -10, fontSize: '9rem', color: 'rgba(200,149,90,0.05)', fontFamily: 'Georgia, serif', userSelect: 'none', lineHeight: 1, pointerEvents: 'none' }}>ॐ</div>
            <div style={{ fontSize: '0.58rem', letterSpacing: '2.5px', color: 'var(--muted-2)', textTransform: 'uppercase', marginBottom: 20 }}>Indian symbolic</div>
            <h3 className="serif" style={{ fontSize: '1.7rem', fontWeight: 300, color: 'var(--text)', margin: '0 0 16px', fontStyle: 'italic', lineHeight: 1.2 }}>Vedic lens</h3>
            <p style={{ fontSize: '0.88rem', color: 'var(--muted)', margin: '0 0 28px', lineHeight: 1.8 }}>
              Interpret your dream through traditional Indian symbolism: rivers, doors, animals, time of night. Cultural, careful, contextual.
            </p>
            <p className="serif" style={{ fontStyle: 'italic', fontSize: '1rem', color: 'var(--text)', margin: 0, lineHeight: 1.7, opacity: 0.8 }}>
              &ldquo;The river that wouldn&apos;t carry you may signify a passage you have not yet asked permission to cross.&rdquo;
            </p>
          </div>
        </div>
      </section>

      <hr className="section-divider" />

      {/* ── Features ──────────────────────────────────────────────────── */}
      <section className="landing-section reveal" id="features">
        <h2 className="serif" style={{ fontSize: 'clamp(1.8rem, 3vw, 2.5rem)', fontWeight: 300, margin: '0 0 48px' }}>
          What&apos;s inside<br />DreamLog.
        </h2>

        {/* Bento feature grid — 2-col top, 1 wide bottom */}
        <div className="features-bento" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, marginBottom: 2 }}>
          <div className="feat-hero reveal" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: '16px 0 0 0', padding: '36px 32px 40px', transitionDelay: '0s' }}>
            <div className="feat-icon" style={{ color: 'var(--gold)', marginBottom: 20, opacity: 0.75 }}><IconMic /></div>
            <div style={{ fontWeight: 700, fontSize: '1.05rem', color: 'var(--text)', marginBottom: 12, letterSpacing: '-0.01em' }}>Voice journaling</div>
            <p style={{ fontSize: '0.88rem', color: 'var(--muted)', margin: 0, lineHeight: 1.8 }}>Speak naturally. No typing, no prompts. DreamLog captures your thoughts the way they actually arrive — unfiltered, in your own voice.</p>
          </div>
          <div className="feat-hero reveal" style={{ background: 'rgba(200,149,90,0.04)', border: '1px solid rgba(200,149,90,0.18)', borderRadius: '0 16px 0 0', padding: '36px 32px 40px', transitionDelay: '0.1s' }}>
            <div className="feat-icon" style={{ color: 'var(--gold)', marginBottom: 20, opacity: 0.75 }}><IconSparkle /></div>
            <div style={{ fontWeight: 700, fontSize: '1.05rem', color: 'var(--text)', marginBottom: 12, letterSpacing: '-0.01em' }}>AI emotional reflection</div>
            <p style={{ fontSize: '0.88rem', color: 'var(--muted)', margin: 0, lineHeight: 1.8 }}>Your entries become a private mirror. Mood, themes, recurring phrases — the quiet things you didn't say out loud, surfaced gently after every recording.</p>
          </div>
          <div className="feat-hero feat-wide reveal" style={{ gridColumn: '1 / -1', background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: '0 0 16px 16px', padding: '28px 36px', transitionDelay: '0.2s' }}>
            <div className="feat-icon" style={{ color: 'var(--gold)', opacity: 0.75 }}><IconChat /></div>
            <div style={{ flex: 1 }}>
              <div style={{ fontWeight: 700, fontSize: '1.05rem', color: 'var(--text)', marginBottom: 10, letterSpacing: '-0.01em' }}>Therapy mode</div>
              <p style={{ fontSize: '0.88rem', color: 'var(--muted)', margin: 0, lineHeight: 1.75 }}>A companion conversation that already understands your emotional history. Up to an hour, voice or text, four distinct tones.</p>
            </div>
            <a href="#therapy" style={{ flexShrink: 0, fontSize: '0.8rem', color: 'var(--gold)', fontWeight: 600, letterSpacing: 0.3, whiteSpace: 'nowrap', borderBottom: '1px solid rgba(212,165,106,0.3)', paddingBottom: 2, transition: 'border-color 0.2s ease', textDecoration: 'none' }}>Explore Therapy Mode →</a>
          </div>
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

      {/* ── Pull quote — woven in, not a testimonials box ─────────────── */}
      <div className="reveal" style={{ maxWidth: 1320, margin: '0 auto', padding: '0 60px 80px' }}>
        <div style={{ borderTop: '1px solid var(--border)', paddingTop: 64 }}>
          <div className="serif" style={{ fontSize: 'clamp(0.7rem, 0.9vw, 0.85rem)', fontWeight: 300, letterSpacing: '0.05em', color: 'var(--muted-2)', marginBottom: 24, fontStyle: 'normal' }}>— Rohit, 41, after his first weekly review</div>
          <p className="serif reveal-left" style={{ fontSize: 'clamp(1.6rem, 3vw, 2.4rem)', fontWeight: 300, fontStyle: 'italic', lineHeight: 1.55, color: 'var(--text)', margin: '0 0 40px', maxWidth: 820 }}>
            &ldquo;It noticed I kept mentioning my father without realising it. Three entries in a row. Nobody had ever pointed that out before.&rdquo;
          </p>
          <div style={{ display: 'flex', gap: 48, flexWrap: 'wrap' }}>
            {[
              { quote: '"Talking feels different. Something about hearing my own voice makes the reflection land."', attr: 'Priya, 28 — 3-month user' },
              { quote: '"It says what I actually said. That\'s a harder and better thing."', attr: 'Ananya, 34 — six weeks in' },
            ].map(t => (
              <div key={t.attr} style={{ flex: '1 1 260px' }}>
                <p className="serif" style={{ fontStyle: 'italic', fontSize: '0.95rem', color: 'var(--muted)', margin: '0 0 10px', lineHeight: 1.7 }}>&ldquo;{t.quote}&rdquo;</p>
                <span style={{ fontSize: '0.76rem', color: 'var(--muted-2)' }}>{t.attr}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* ── Pricing ───────────────────────────────────────────────────── */}
      <section className="landing-section reveal" id="pricing">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: 24, marginBottom: 44, flexWrap: 'wrap' }}>
          <h2 className="serif" style={{ fontSize: 'clamp(2.2rem, 4vw, 3.6rem)', fontWeight: 300, margin: 0, lineHeight: 1.05, letterSpacing: '-0.02em' }}>Honest<br />pricing.</h2>
          <p style={{ fontSize: '0.9rem', color: 'var(--muted)', margin: 0, maxWidth: 260, lineHeight: 1.7, textAlign: 'right' }}>Most people start Free. You never need to upgrade. It&apos;s allowed.</p>
        </div>
        {/* Free — horizontal starter row, breaks the 3-tower pattern */}
        <div className="pricing-free-row" style={{ display: 'flex', alignItems: 'center', gap: 20, background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 16, padding: '18px 28px', marginBottom: 12, flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, flexShrink: 0 }}>
            <span style={{ fontSize: '0.8rem', fontWeight: 600, color: 'var(--muted)' }}>Free</span>
            <span className="serif" style={{ fontSize: '1.7rem', fontWeight: 300, color: 'var(--text)', opacity: ready ? 1 : 0, transition: 'opacity 0.3s' }}>{p('free')}</span>
            <span style={{ fontSize: '0.75rem', color: 'var(--muted-2)' }}>forever</span>
          </div>
          <div style={{ flex: 1, display: 'flex', gap: 20, flexWrap: 'wrap' }}>
            {['10 entries / month', 'AI reflection', '7-day mood chart', '3-turn follow-up', 'Crisis detection'].map(f => (
              <span key={f} style={{ fontSize: '0.8rem', color: 'var(--muted)', display: 'inline-flex', gap: 7, alignItems: 'center' }}>
                <span style={{ color: 'var(--gold)', fontSize: '0.7rem' }}>✓</span>{f}
              </span>
            ))}
          </div>
          <a href={dlHref} className="btn-ghost" style={{ padding: '9px 22px', whiteSpace: 'nowrap', flexShrink: 0, fontSize: '0.84rem' }}>Download Free</a>
        </div>

        {/* Plus + Pro — 2 columns, Plus naturally dominates with more features */}
        <div className="pricing-grid" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <PricingCard name="DreamLog+" price={p('plus')} period={per} sub="Unlimited entries. All modes. The complete journaling product." features={['Unlimited entries', 'All 5 journaling modes', 'Dream Decoder (Jungian + Vedic)', 'Life Graph & Mood History', 'Weekly + Annual Reviews', 'Life Chapters', 'PDF export', 'Therapist share (5/month)']} cta="Get DreamLog+" downloadHref={dlHref} featured ready={ready} />
          <PricingCard name="DreamLog Pro" price={p('pro')} period={per} sub="Everything in Plus, plus one therapy session every month." features={['Everything in Plus', '1 Therapy Session / month', `Extra sessions at ${currency === 'INR' ? '₹299' : currency === 'EUR' ? '€3.99' : '$4.99'}`, 'Unlimited therapist share', 'Priority processing']} cta="Get DreamLog Pro" downloadHref={dlHref} ready={ready} />
        </div>
        <p style={{ marginTop: 14, fontSize: '0.76rem', color: 'var(--muted-2)', textAlign: 'right' }}>
          30-day passes · managed in-app · no auto-renew
        </p>
      </section>

      {/* ── Corporate Wellness ────────────────────────────────────────── */}
      <section style={{ background: 'var(--bg-card-2)', borderTop: '1px solid var(--border)', borderBottom: '1px solid var(--border)' }}>
        <div className="b2b-inner" style={{ maxWidth: 1320, margin: '0 auto', padding: '80px 60px', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 60, alignItems: 'start' }}>
          <div>
            <h2 className="serif" style={{ fontSize: 'clamp(1.6rem, 3vw, 2.4rem)', fontWeight: 300, margin: '0 0 16px', lineHeight: 1.15, color: 'var(--text)' }}>
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
            <h2 className="serif" style={{ fontSize: 'clamp(1.5rem, 2.5vw, 2rem)', fontWeight: 300, margin: '0 0 14px' }}>
              For the therapists<br />who send their clients here.
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
          <div className="faq-col stagger">
            <div className="faq-item">
              <h3>Is this actually private?</h3>
              <p>Audio is deleted immediately after transcription. Transcripts live on our servers, encrypted. We don't sell data or train on your entries. You can export or delete everything at any time.</p>
            </div>
            <div className="faq-item">
              <h3>How is this different from a general AI chat?</h3>
              <p>It knows your history. Every reflection is grounded in your last 5 entries — the emotions, topics, and patterns that recur. A general chat has no memory. DreamLog does.</p>
            </div>
            <div className="faq-item">
              <h3>Does it work in Hindi or Hinglish?</h3>
              <p>Yes. Your language is detected automatically. Reflections are generated in the same language. Hindi and Hinglish support is included from DreamLog+.</p>
            </div>
          </div>
          <div className="faq-col stagger">
            <div className="faq-item">
              <h3>Can I share this with my therapist?</h3>
              <p>Yes. From Settings, you generate a passcode-protected link. Your therapist sees mood trends and AI summaries, never raw transcripts or recordings.</p>
            </div>
            <div className="faq-item">
              <h3>What happens if I say something in crisis?</h3>
              <p>Two-stage detection runs on every entry. If distress is detected, you get crisis resources immediately: iCall, Vandrevala Foundation, 988 (US). The entry is flagged and handled separately.</p>
            </div>
          </div>
        </div>
      </section>

      <hr className="section-divider" />

      {/* ── Download CTA ──────────────────────────────────────────────── */}
      <section id="download" className="landing-section reveal">
        <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 40, alignItems: 'end', flexWrap: 'wrap' }}>
          <h2 className="serif" style={{ fontSize: 'clamp(3rem, 6vw, 6.5rem)', fontWeight: 300, margin: 0, lineHeight: 1.0, letterSpacing: '-0.025em' }}>
            Your thoughts<br />are worth<br /><em style={{ color: 'var(--gold)' }}>understanding.</em>
          </h2>
          <div style={{ paddingBottom: 8, display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 16 }}>
            <DownloadButtons androidUrl={version.android_store_url} iosUrl={version.ios_store_url} />
            {version.ios_store_url === '#' && (
              <p style={{ margin: 0, fontSize: '0.78rem', color: 'var(--muted-2)', textAlign: 'right' }}>
                Store listings coming soon.{' '}
                <a href="mailto:support@dreamlog.app" style={{ color: 'var(--muted)' }}>Get notified</a>
              </p>
            )}
            <p style={{ margin: 0, fontSize: '0.82rem', color: 'var(--muted-2)' }}>Available on iOS and Android. Free to start.</p>
          </div>
        </div>
      </section>

      {/* ── Footer — minimal, no link-farm ───────────────────────────── */}
      <footer style={{ borderTop: '1px solid var(--border)', padding: '36px 60px', maxWidth: 1320, margin: '0 auto' }}>
        {/* Top row: brand left, nav links right */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 32, flexWrap: 'wrap' }}>
          <a href="/" style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <div style={{ width: 26, height: 26, borderRadius: 6, background: 'var(--gold)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '0.95rem', color: '#18150f' }}>D</div>
            <span className="serif" style={{ fontSize: '1rem', fontWeight: 600 }}>DreamLog</span>
          </a>
          <div style={{ display: 'flex', gap: 28, flexWrap: 'wrap', alignItems: 'center' }}>
            {[
              { label: 'Features', href: '#features' },
              { label: 'Pricing', href: '#pricing' },
              { label: 'Therapist Portal', href: '/login' },
              { label: 'Privacy', href: '/privacy' },
              { label: 'Terms', href: '/terms' },
              { label: 'Support', href: 'mailto:support@dreamlog.app' },
            ].map(l => (
              <a key={l.label} href={l.href} style={{ fontSize: '0.82rem', color: 'var(--muted)', transition: 'color 0.15s ease' }}
                onMouseEnter={e => (e.currentTarget.style.color = 'var(--text)')}
                onMouseLeave={e => (e.currentTarget.style.color = 'var(--muted)')}
              >{l.label}</a>
            ))}
          </div>
        </div>

        {/* Bottom row: copyright left, crisis lines right */}
        <div style={{ marginTop: 24, paddingTop: 20, borderTop: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
          <span style={{ fontSize: '0.74rem', color: 'var(--muted-2)' }}>© 2026 DreamLog · Built for honesty, not performance.</span>
          <div style={{ display: 'flex', gap: 20, flexWrap: 'wrap' }}>
            <span style={{ fontSize: '0.7rem', color: 'var(--muted-2)', opacity: 0.6 }}>If you need help:</span>
            {[
              { label: 'iCall 9152987821', href: 'tel:9152987821' },
              { label: 'Vandrevala 1860-2662-345', href: 'tel:18602662345' },
              { label: '988', href: 'tel:988' },
            ].map(r => (
              <a key={r.label} href={r.href} style={{ fontSize: '0.7rem', color: 'var(--muted-2)' }}>{r.label}</a>
            ))}
          </div>
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
          .b2b-inner             { padding: 56px 40px !important; gap: 40px !important; }
        }

        /* ── Mobile (640px) ── */
        @media (max-width: 640px) {
          /* Nav */
          .landing-nav { padding: 0 20px !important; }
          .landing-nav > div:last-child a:not(.btn-primary) { display: none; }

          /* Section padding */
          .landing-section    { padding: 72px 20px !important; }
          .landing-section-sm { padding: 48px 20px !important; }

          /* Hero flex row → column */
          .landing-section > div[style*="space-between"] { flex-direction: column !important; gap: 40px !important; }

          /* How it works: 3-col → 1-col */
          .how-grid { grid-template-columns: 1fr !important; gap: 32px !important; }
          .how-grid > div { padding-right: 0 !important; }

          /* Journal mirror: 2-col → 1-col */
          .journal-grid { grid-template-columns: 1fr !important; gap: 32px !important; }

          /* Therapy split panel: 2-col → 1-col */
          .therapy-intro-grid { grid-template-columns: 1fr !important; gap: 32px !important; }
          #therapy > div:first-child { grid-template-columns: 1fr !important; }
          #therapy > div:first-child > div:first-child { border-right: none !important; border-bottom: 1px solid var(--border) !important; padding: 36px 28px !important; }
          #therapy > div:first-child > div:last-child { min-height: 280px !important; padding: 40px 20px !important; }
          #therapy > div:first-child > div:last-child > div:nth-child(1),
          #therapy > div:first-child > div:last-child > div:nth-child(2),
          #therapy > div:first-child > div:last-child > div:nth-child(3) { width: 200px !important; height: 200px !important; }

          /* Therapy personas: 4-col → 2-col */
          .therapy-personas-grid { grid-template-columns: repeat(2, 1fr) !important; }

          /* Dream split: columns → rows */
          .dream-split { grid-template-columns: 1fr !important; }
          .dream-split > div:nth-child(2) { height: 1px !important; width: 100% !important; }
          .dream-panel { padding: 36px 28px !important; }

          /* Feature bento: 2-col top + wide bottom → 1-col */
          .features-bento { grid-template-columns: 1fr !important; }
          .features-bento .feat-hero:nth-child(1) { border-radius: 16px 16px 0 0 !important; }
          .features-bento .feat-hero:nth-child(2) { border-radius: 0 !important; }
          .features-bento .feat-hero:nth-child(3) { border-radius: 0 0 16px 16px !important; }
          .feat-wide { flex-direction: column !important; align-items: flex-start !important; gap: 16px !important; }

          /* Feature sub-grid: 2-col → 1-col */
          .features-sub-grid { grid-template-columns: 1fr !important; }
          .features-sub-grid > div { border-right: none !important; border-top: 1px solid var(--border) !important; }
          .features-sub-grid > div:first-child { border-top: none !important; }

          /* Pricing: 2-col → 1-col */
          .pricing-grid { grid-template-columns: 1fr !important; }
          .pricing-free-row { flex-direction: column !important; align-items: flex-start !important; gap: 16px !important; }
          .pricing-free-row > a { align-self: stretch; text-align: center; }

          /* B2B: 2-col → 1-col */
          .b2b-inner { grid-template-columns: 1fr !important; padding: 48px 20px !important; gap: 32px !important; }

          /* For clinicians row: wrap */
          .landing-section-sm > div[style*="space-between"] { flex-direction: column !important; align-items: flex-start !important; gap: 20px !important; }

          /* FAQ: 2-col → 1-col */
          .faq-grid { grid-template-columns: 1fr !important; }

          /* Download buttons */
          #download > div { grid-template-columns: 1fr !important; gap: 32px !important; }
          #download > div > div:last-child { align-items: flex-start !important; }

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
