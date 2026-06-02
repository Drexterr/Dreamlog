import React, { useState } from 'react';
import { useAuth } from '../context/AuthContext';

export const Login: React.FC = () => {
  const { login, register } = useAuth();
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [credentials, setCredentials] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      if (mode === 'login') {
        await login(email, password);
      } else {
        await register(email, name, credentials);
      }
    } catch (err: any) {
      setError(err.message || 'An error occurred. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={styles.container}>
      {/* Background blobs */}
      <div style={{ ...styles.bgBlob, ...styles.bgBlob1 }} />
      <div style={{ ...styles.bgBlob, ...styles.bgBlob2 }} />

      <div style={styles.card}>
        <div style={styles.header}>
          <div style={styles.logo}>D</div>
          <h1 style={styles.title}>DreamLog</h1>
          <span style={styles.tag}>Clinic</span>
        </div>
        <p style={styles.subtitle}>Therapist Clinical Insights Portal</p>

        {/* Mode Switcher Tabs */}
        <div style={styles.tabGroup}>
          <button
            style={{
              ...styles.tab,
              ...(mode === 'login' ? styles.tabActive : {}),
            }}
            onClick={() => {
              setMode('login');
              setError('');
            }}
          >
            Sign In
          </button>
          <button
            style={{
              ...styles.tab,
              ...(mode === 'register' ? styles.tabActive : {}),
            }}
            onClick={() => {
              setMode('register');
              setError('');
            }}
          >
            Create Account
          </button>
        </div>

        {error && <div style={styles.errorAlert}>{error}</div>}

        <form onSubmit={handleSubmit} style={styles.form}>
          {mode === 'register' && (
            <div style={styles.formGroup}>
              <label style={styles.label}>Full Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Dr. Sarah Jenkins"
                required
                style={styles.input}
              />
            </div>
          )}

          <div style={styles.formGroup}>
            <label style={styles.label}>Professional Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="sjenkins@clinic.org"
              required
              style={styles.input}
            />
          </div>

          <div style={styles.formGroup}>
            <label style={styles.label}>Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              style={styles.input}
            />
          </div>

          {mode === 'register' && (
            <div style={styles.formGroup}>
              <label style={styles.label}>Credentials & License</label>
              <input
                type="text"
                value={credentials}
                onChange={(e) => setCredentials(e.target.value)}
                placeholder="Clinical Psychologist, PsyD"
                style={styles.input}
              />
            </div>
          )}

          <button type="submit" disabled={loading} style={styles.submitBtn}>
            {loading ? (
              <span style={styles.spinner} />
            ) : mode === 'login' ? (
              'Access Dashboard'
            ) : (
              'Register as Therapist'
            )}
          </button>
        </form>
      </div>
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  container: {
    width: '100vw',
    height: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: '#0b1329',
    position: 'relative',
    overflow: 'hidden',
  },
  bgBlob: {
    position: 'absolute',
    borderRadius: '50%',
    filter: 'blur(100px)',
    opacity: 0.18,
    pointerEvents: 'none',
  },
  bgBlob1: {
    width: '400px',
    height: '400px',
    background: '#00b4d8',
    top: '-10%',
    left: '-10%',
  },
  bgBlob2: {
    width: '350px',
    height: '350px',
    background: '#7B6FA0',
    bottom: '-5%',
    right: '-5%',
  },
  card: {
    width: '100%',
    maxWidth: '440px',
    background: 'rgba(28, 37, 65, 0.55)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: '24px',
    padding: '40px',
    backdropFilter: 'blur(20px)',
    boxShadow: '0 24px 60px rgba(0, 0, 0, 0.35)',
    textAlign: 'center',
    zIndex: 2,
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '12px',
    marginBottom: '10px',
  },
  logo: {
    width: '36px',
    height: '36px',
    borderRadius: '8px',
    background: 'linear-gradient(135deg, #00b4d8 0%, #0077b6 100%)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontFamily: "'Outfit', sans-serif",
    fontWeight: 700,
    fontSize: '1.2rem',
    color: '#fff',
    boxShadow: '0 0 15px rgba(0, 180, 216, 0.3)',
  },
  title: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '1.6rem',
    fontWeight: 600,
    color: '#f8fafc',
    letterSpacing: '0.5px',
  },
  tag: {
    fontSize: '0.7rem',
    textTransform: 'uppercase',
    letterSpacing: '1px',
    color: '#00b4d8',
    background: 'rgba(0, 180, 216, 0.1)',
    padding: '2px 8px',
    borderRadius: '4px',
    fontWeight: 700,
  },
  subtitle: {
    fontSize: '0.9rem',
    color: '#cbd5e1',
    marginBottom: '32px',
  },
  tabGroup: {
    display: 'flex',
    background: 'rgba(11, 19, 43, 0.6)',
    borderRadius: '12px',
    padding: '4px',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    marginBottom: '24px',
  },
  tab: {
    flex: 1,
    padding: '10px',
    background: 'transparent',
    border: 'none',
    color: '#64748b',
    fontFamily: 'inherit',
    fontSize: '0.85rem',
    fontWeight: 600,
    cursor: 'pointer',
    borderRadius: '8px',
    transition: 'all 0.2s ease',
  },
  tabActive: {
    background: '#00b4d8',
    color: '#0b1329',
    boxShadow: '0 4px 10px rgba(0, 180, 216, 0.2)',
  },
  errorAlert: {
    background: 'rgba(239, 68, 68, 0.12)',
    border: '1px solid rgba(239, 68, 68, 0.25)',
    color: '#fca5a5',
    fontSize: '0.8rem',
    padding: '10px 14px',
    borderRadius: '10px',
    marginBottom: '20px',
    textAlign: 'left',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
    textAlign: 'left',
  },
  formGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  label: {
    fontSize: '0.75rem',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
    color: '#cbd5e1',
    fontWeight: 600,
  },
  input: {
    background: 'rgba(11, 19, 43, 0.6)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: '10px',
    padding: '12px',
    color: '#f8fafc',
    fontFamily: 'inherit',
    fontSize: '0.9rem',
    outline: 'none',
    transition: 'all 0.2s ease',
  },
  submitBtn: {
    background: '#00b4d8',
    border: 'none',
    color: '#0b1329',
    padding: '14px',
    borderRadius: '12px',
    fontFamily: 'inherit',
    fontSize: '0.95rem',
    fontWeight: 700,
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: '12px',
    transition: 'all 0.2s ease',
    boxShadow: '0 4px 12px rgba(0, 180, 216, 0.2)',
  },
  spinner: {
    width: '20px',
    height: '20px',
    border: '2px solid rgba(11, 19, 43, 0.1)',
    borderTopColor: '#0b1329',
    borderRadius: '50%',
    display: 'inline-block',
    animation: 'spin 1s infinite linear',
  },
};
