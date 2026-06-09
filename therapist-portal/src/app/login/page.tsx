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

      <a href="/" style={styles.backBtn}>← Back to Home</a>

      <div style={styles.card}>
        <div style={styles.header}>
          <div style={styles.logo}>D</div>
          <h1 style={styles.title}>DreamLog</h1>
          <span style={styles.tag}>Clinic</span>
        </div>
        <p style={styles.subtitle}>Therapist Clinical Insights Portal</p>

        <div style={styles.tabGroup}>
          <button style={{ ...styles.tab, ...(mode === 'login' ? styles.tabActive : {}) }} onClick={() => { setMode('login'); setError(''); }}>
            Sign In
          </button>
          <button style={{ ...styles.tab, ...(mode === 'register' ? styles.tabActive : {}) }} onClick={() => { setMode('register'); setError(''); }}>
            Create Account
          </button>
        </div>

        {error && <div style={styles.errorAlert}>{error}</div>}

        <form onSubmit={handleSubmit} style={styles.form}>
          {mode === 'register' && (
            <div style={styles.formGroup}>
              <label style={styles.label}>Full Name</label>
              <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="Dr. Sarah Jenkins" required style={styles.input} />
            </div>
          )}

          <div style={styles.formGroup}>
            <label style={styles.label}>Professional Email</label>
            <input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="sjenkins@clinic.org" required style={styles.input} />
          </div>

          <div style={styles.formGroup}>
            <label style={styles.label}>Password</label>
            <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="••••••••" required style={styles.input} />
          </div>

          {mode === 'register' && (
            <div style={styles.formGroup}>
              <label style={styles.label}>Credentials &amp; License</label>
              <input type="text" value={credentials} onChange={e => setCredentials(e.target.value)} placeholder="Clinical Psychologist, PsyD" style={styles.input} />
            </div>
          )}

          <button type="submit" disabled={loading} style={styles.submitBtn}>
            {loading
              ? <span style={styles.spinner} />
              : mode === 'login' ? 'Access Dashboard' : 'Register as Therapist'
            }
          </button>
        </form>
      </div>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: { width: '100vw', height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#FAF8F5', position: 'relative', overflow: 'hidden', fontFamily: "'Plus Jakarta Sans', sans-serif" },
  backBtn: { position: 'absolute', top: '32px', left: '32px', background: 'transparent', border: 'none', color: '#7E8280', fontSize: '0.9rem', fontWeight: 500, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '6px', zIndex: 10, textDecoration: 'none' },
  bgBlob: { position: 'absolute', borderRadius: '50%', filter: 'blur(100px)', opacity: 0.35, pointerEvents: 'none' },
  bgBlob1: { width: '450px', height: '450px', background: '#E6E8E3', top: '-10%', left: '-10%' },
  bgBlob2: { width: '400px', height: '400px', background: '#EAE6DF', bottom: '-5%', right: '-5%' },
  card: { width: '100%', maxWidth: '440px', background: '#FFFFFF', border: '1px solid rgba(42,44,43,0.08)', borderRadius: '24px', padding: '40px', boxShadow: '0 20px 48px rgba(42,44,43,0.03)', textAlign: 'center', zIndex: 2 },
  header: { display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '12px', marginBottom: '10px' },
  logo: { width: '36px', height: '36px', borderRadius: '8px', background: '#788574', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1.4rem', color: '#FAF8F5', boxShadow: '0 4px 10px rgba(120,133,116,0.25)' },
  title: { fontFamily: "'Cormorant Garamond', serif", fontSize: '1.8rem', fontWeight: 600, color: '#2A2C2B', letterSpacing: '0.5px' },
  tag: { fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '1px', color: '#788574', background: 'rgba(120,133,116,0.1)', padding: '2px 8px', borderRadius: '4px', fontWeight: 700 },
  subtitle: { fontSize: '0.9rem', color: '#7E8280', marginBottom: '32px' },
  tabGroup: { display: 'flex', background: 'rgba(120,133,116,0.04)', borderRadius: '12px', padding: '4px', border: '1px solid rgba(42,44,43,0.05)', marginBottom: '24px' },
  tab: { flex: 1, padding: '10px', background: 'transparent', border: 'none', color: '#7E8280', fontFamily: 'inherit', fontSize: '0.85rem', fontWeight: 600, cursor: 'pointer', borderRadius: '8px', transition: 'all 0.25s ease' },
  tabActive: { background: '#788574', color: '#FAF8F5', boxShadow: '0 4px 10px rgba(120,133,116,0.2)' },
  errorAlert: { background: 'rgba(217,83,79,0.08)', border: '1px solid rgba(217,83,79,0.2)', color: '#c9302c', fontSize: '0.8rem', padding: '10px 14px', borderRadius: '10px', marginBottom: '20px', textAlign: 'left' },
  form: { display: 'flex', flexDirection: 'column', gap: '16px', textAlign: 'left' },
  formGroup: { display: 'flex', flexDirection: 'column', gap: '6px' },
  label: { fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.5px', color: '#7E8280', fontWeight: 600 },
  input: { background: '#FAF8F5', border: '1px solid rgba(42,44,43,0.12)', borderRadius: '10px', padding: '12px', color: '#2A2C2B', fontFamily: 'inherit', fontSize: '0.9rem', outline: 'none', width: '100%', boxSizing: 'border-box' },
  submitBtn: { background: '#2A2C2B', border: 'none', color: '#FAF8F5', padding: '14px', borderRadius: '12px', fontFamily: 'inherit', fontSize: '0.95rem', fontWeight: 700, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', marginTop: '12px', transition: 'all 0.2s ease', boxShadow: '0 4px 12px rgba(42,44,43,0.12)', width: '100%' },
  spinner: { width: '20px', height: '20px', border: '2px solid rgba(250,248,245,0.2)', borderTopColor: '#FAF8F5', borderRadius: '50%', display: 'inline-block', animation: 'spin 1s infinite linear' },
};
