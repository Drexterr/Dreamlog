'use client';

import { useState, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { api, saveToken } from '../../lib/api';

const QUOTES = [
  { text: "Something I couldn't say to anyone, I could finally say to myself.", attr: "— A user, 31, on the third week of journaling" },
  { text: "I've kept journals for years but always gave up after a week. Talking feels different.", attr: "— Priya, 28, 3-month user" },
  { text: "It asked me something I hadn't thought to ask myself.", attr: "— A user, 44, after their first reflection" },
];

export default function LoginPage() {
  const router = useRouter();
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [credentials, setCredentials] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const quote = QUOTES[0];

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      if (mode === 'login') {
        const { token } = await api.login(email, password);
        saveToken(token);
      } else {
        const { token } = await api.authRegister(email, password, name);
        saveToken(token);
        await api.registerTherapist(name, email, credentials);
      }
      router.push('/dashboard');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg || 'Something went wrong. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', minHeight: '100vh', background: 'var(--bg)' }}>

      {/* Left: Quote panel */}
      <div style={{
        flex: '0 0 44%', position: 'relative', overflow: 'hidden',
        background: '#131009', borderRight: '1px solid var(--border)',
        display: 'flex', flexDirection: 'column', justifyContent: 'space-between',
        padding: '40px 48px',
      }} className="no-mobile">
        {/* Logo */}
        <a href="/" style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{
            width: 30, height: 30, borderRadius: 7, background: 'var(--gold)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic',
            fontWeight: 700, fontSize: '1.1rem', color: '#0c0b09',
          }}>D</div>
          <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.15rem', fontWeight: 600, color: 'var(--text)' }}>
            DreamLog
          </span>
        </a>

        {/* Quote */}
        <div>
          <p className="serif" style={{
            fontSize: 'clamp(1.3rem, 2vw, 1.7rem)', fontWeight: 300, fontStyle: 'italic',
            color: 'var(--text)', lineHeight: 1.6, margin: '0 0 20px',
          }}>
            &ldquo;{quote.text}&rdquo;
          </p>
          <p style={{ fontSize: '0.82rem', color: 'var(--muted)', margin: 0 }}>
            {quote.attr}
          </p>
        </div>

        {/* Privacy note */}
        <p style={{ fontSize: '0.74rem', color: 'var(--muted-2)', lineHeight: 1.7, margin: 0 }}>
          DreamLog journaling lives in the mobile app.<br />
          This portal is for therapists.
        </p>
      </div>

      {/* Right: Form */}
      <div style={{
        flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: '40px 24px',
      }}>
        <div style={{ width: '100%', maxWidth: 400 }}>
          <div className="eyebrow" style={{ marginBottom: 12 }}>WELCOME BACK</div>
          <h1 className="serif" style={{ fontSize: 'clamp(1.8rem, 3vw, 2.4rem)', fontWeight: 300, margin: '0 0 6px', color: 'var(--text)' }}>
            {mode === 'login' ? 'Sign in to continue' : 'Create your account'}
          </h1>
          <p style={{ color: 'var(--muted)', fontSize: '0.88rem', margin: '0 0 32px' }}>
            {mode === 'login'
              ? 'Access your client dashboard and pre-session briefs.'
              : 'Set up your therapist account to start receiving client insights.'}
          </p>

          {/* Role toggle */}
          <div style={{
            display: 'flex', gap: 4,
            background: 'var(--bg-card)', border: '1px solid var(--border)',
            borderRadius: 100, padding: 4, marginBottom: 28,
          }}>
            {(['login', 'register'] as const).map(m => (
              <button
                key={m}
                type="button"
                onClick={() => { setMode(m); setError(''); }}
                style={{
                  flex: 1, padding: '9px 16px', border: 'none', borderRadius: 100,
                  fontFamily: 'inherit', fontSize: '0.84rem', fontWeight: 600, cursor: 'pointer',
                  transition: 'all 0.2s ease',
                  background: mode === m ? 'var(--gold)' : 'transparent',
                  color: mode === m ? '#0c0b09' : 'var(--muted)',
                }}
              >
                {m === 'login' ? 'Sign in' : 'Create account'}
              </button>
            ))}
          </div>

          {error && (
            <div style={{
              background: 'rgba(192,91,77,0.08)', border: '1px solid rgba(192,91,77,0.2)',
              color: 'var(--danger)', fontSize: '0.82rem', padding: '10px 14px',
              borderRadius: 10, marginBottom: 20,
            }}>{error}</div>
          )}

          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {mode === 'register' && (
              <div>
                <label style={{ display: 'block', fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.6px', color: 'var(--muted)', fontWeight: 600, marginBottom: 6 }}>
                  Full name
                </label>
                <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Dr. Sarah Jenkins" required autoComplete="name" />
              </div>
            )}

            <div>
              <label style={{ display: 'block', fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.6px', color: 'var(--muted)', fontWeight: 600, marginBottom: 6 }}>
                EMAIL
              </label>
              <input
                type="email" value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="therapist@dreamlog.app"
                required autoComplete="email"
              />
            </div>

            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                <label style={{ fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.6px', color: 'var(--muted)', fontWeight: 600 }}>
                  PASSWORD
                </label>
                {mode === 'login' && (
                  <span style={{ fontSize: '0.76rem', color: 'var(--muted)', cursor: 'pointer' }}>Forgot?</span>
                )}
              </div>
              <div style={{ position: 'relative' }}>
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  placeholder="••••••••"
                  required
                  autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                  style={{ paddingRight: 60 }}
                />
                <button
                  type="button" onClick={() => setShowPassword(p => !p)} tabIndex={-1}
                  style={{
                    position: 'absolute', right: 12, top: '50%', transform: 'translateY(-50%)',
                    background: 'transparent', border: 'none', color: 'var(--muted)',
                    fontSize: '0.74rem', fontWeight: 600, cursor: 'pointer', fontFamily: 'inherit',
                  }}
                >
                  {showPassword ? 'Hide' : 'Show'}
                </button>
              </div>
            </div>

            {mode === 'register' && (
              <div>
                <label style={{ display: 'block', fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.6px', color: 'var(--muted)', fontWeight: 600, marginBottom: 6 }}>
                  CREDENTIALS (optional)
                </label>
                <input type="text" value={credentials} onChange={e => setCredentials(e.target.value)} placeholder="Clinical Psychologist, PsyD" />
              </div>
            )}

            <button
              type="submit" disabled={loading} className="btn-primary"
              style={{ width: '100%', padding: '14px', marginTop: 8, fontSize: '0.92rem', borderRadius: 12 }}
            >
              {loading
                ? <span className="spin" />
                : mode === 'login' ? 'Open Therapist Portal →' : 'Register as therapist'}
            </button>
          </form>

          <p style={{ textAlign: 'center', marginTop: 28, color: 'var(--muted-2)', fontSize: '0.76rem', lineHeight: 1.6 }}>
            New to DreamLog?{' '}
            <a href="/" style={{ color: 'var(--muted)' }}>Learn more</a>{' · '}
            <a href="/login" onClick={() => setMode('register')} style={{ color: 'var(--muted)' }}>For therapists</a>
          </p>

          {mode === 'login' && (
            <p style={{ textAlign: 'center', marginTop: 12, color: 'var(--muted-2)', fontSize: '0.72rem' }}>
              Demo credentials are pre-filled.{' '}
              <span
                style={{ color: 'var(--muted)', cursor: 'pointer', textDecoration: 'underline' }}
                onClick={() => { setEmail('bharatbanthia2207+tester@gmail.com'); setPassword('DreamTest!2026'); }}
              >
                Click to enter the portal.
              </span>
            </p>
          )}
        </div>
      </div>

      <style>{`
        @media (max-width: 700px) {
          .no-mobile { display: none !important; }
        }
      `}</style>
    </div>
  );
}
