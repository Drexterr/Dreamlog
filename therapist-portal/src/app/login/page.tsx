'use client';

import { useState, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { api, saveToken } from '../../lib/api';

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
      setError(msg || 'An error occurred. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={styles.container}>
      <div style={{ ...styles.bgBlob, ...styles.bgBlob1 }} />
      <div style={{ ...styles.bgBlob, ...styles.bgBlob2 }} />

      <a href="/" style={styles.backBtn}>← Back to home</a>

      <div className="card fade-up" style={styles.card}>
        <div style={styles.header}>
          <div style={styles.logo}>D</div>
          <h1 className="serif" style={styles.title}>DreamLog</h1>
          <span style={styles.tag}>Clinic</span>
        </div>
        <p style={styles.subtitle}>Therapist Clinical Insights Portal</p>

        <div style={styles.tabGroup}>
          <button
            type="button"
            style={{ ...styles.tab, ...(mode === 'login' ? styles.tabActive : {}) }}
            onClick={() => { setMode('login'); setError(''); }}
          >
            Sign in
          </button>
          <button
            type="button"
            style={{ ...styles.tab, ...(mode === 'register' ? styles.tabActive : {}) }}
            onClick={() => { setMode('register'); setError(''); }}
          >
            Create account
          </button>
        </div>

        {error && <div style={styles.errorAlert}>{error}</div>}

        <form onSubmit={handleSubmit} style={styles.form}>
          {mode === 'register' && (
            <div style={styles.formGroup}>
              <label style={styles.label}>Full name</label>
              <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Dr. Sarah Jenkins" required autoComplete="name" />
            </div>
          )}

          <div style={styles.formGroup}>
            <label style={styles.label}>Professional email</label>
            <input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="sjenkins@clinic.org" required autoComplete="email" />
          </div>

          <div style={styles.formGroup}>
            <label style={styles.label}>Password</label>
            <div style={{ position: 'relative' }}>
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                style={{ paddingRight: 64 }}
              />
              <button
                type="button"
                onClick={() => setShowPassword(p => !p)}
                style={styles.showBtn}
                tabIndex={-1}
              >
                {showPassword ? 'Hide' : 'Show'}
              </button>
            </div>
          </div>

          {mode === 'register' && (
            <div style={styles.formGroup}>
              <label style={styles.label}>Credentials &amp; license</label>
              <input type="text" value={credentials} onChange={e => setCredentials(e.target.value)} placeholder="Clinical Psychologist, PsyD" />
            </div>
          )}

          <button type="submit" disabled={loading} className="btn-dark" style={{ width: '100%', padding: '14px', marginTop: 8, fontSize: '0.95rem' }}>
            {loading
              ? <span className="spin" />
              : mode === 'login' ? 'Access dashboard' : 'Register as therapist'
            }
          </button>
        </form>

        <p style={styles.footNote}>
          Clients link to you from the DreamLog app. You only ever see AI summaries and mood trends — never raw recordings.
        </p>
      </div>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: { width: '100vw', minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg)', position: 'relative', overflow: 'hidden', padding: '24px' },
  backBtn: { position: 'absolute', top: '32px', left: '32px', color: 'var(--muted)', fontSize: '0.9rem', fontWeight: 500, zIndex: 10, textDecoration: 'none' },
  bgBlob: { position: 'absolute', borderRadius: '50%', filter: 'blur(100px)', opacity: 0.35, pointerEvents: 'none' },
  bgBlob1: { width: '450px', height: '450px', background: '#E6E8E3', top: '-10%', left: '-10%' },
  bgBlob2: { width: '400px', height: '400px', background: '#EAE6DF', bottom: '-5%', right: '-5%' },
  card: { width: '100%', maxWidth: '440px', borderRadius: '24px', padding: '40px', boxShadow: 'var(--shadow-lg)', textAlign: 'center', zIndex: 2 },
  header: { display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '12px', marginBottom: '10px' },
  logo: { width: '36px', height: '36px', borderRadius: '8px', background: 'var(--brand)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1.4rem', color: '#FAF8F5', boxShadow: '0 4px 10px rgba(120,133,116,0.25)' },
  title: { fontSize: '1.8rem', fontWeight: 600, color: 'var(--text)', letterSpacing: '0.5px' },
  tag: { fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '1px', color: 'var(--brand)', background: 'var(--brand-tint)', padding: '2px 8px', borderRadius: '4px', fontWeight: 700 },
  subtitle: { fontSize: '0.9rem', color: 'var(--muted)', marginBottom: '32px', marginTop: 0 },
  tabGroup: { display: 'flex', background: 'rgba(120,133,116,0.06)', borderRadius: '12px', padding: '4px', border: '1px solid var(--border)', marginBottom: '24px' },
  tab: { flex: 1, padding: '10px', background: 'transparent', border: 'none', color: 'var(--muted)', fontFamily: 'inherit', fontSize: '0.85rem', fontWeight: 600, cursor: 'pointer', borderRadius: '8px', transition: 'all 0.25s ease' },
  tabActive: { background: 'var(--brand)', color: '#FAF8F5', boxShadow: '0 4px 10px rgba(120,133,116,0.2)' },
  errorAlert: { background: 'rgba(192,91,77,0.08)', border: '1px solid rgba(192,91,77,0.25)', color: 'var(--danger)', fontSize: '0.8rem', padding: '10px 14px', borderRadius: '10px', marginBottom: '20px', textAlign: 'left' },
  form: { display: 'flex', flexDirection: 'column', gap: '16px', textAlign: 'left' },
  formGroup: { display: 'flex', flexDirection: 'column', gap: '6px' },
  label: { fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '0.6px', color: 'var(--muted)', fontWeight: 600 },
  showBtn: { position: 'absolute', right: '10px', top: '50%', transform: 'translateY(-50%)', background: 'transparent', border: 'none', color: 'var(--brand)', fontSize: '0.74rem', fontWeight: 700, cursor: 'pointer', padding: '4px 6px', fontFamily: 'inherit' },
  footNote: { fontSize: '0.74rem', color: 'var(--muted)', lineHeight: 1.6, marginTop: '24px', marginBottom: 0 },
};
