'use client';

import { useRouter } from 'next/navigation';
import { clearToken } from '../lib/api';

export default function PortalHeader() {
  const router = useRouter();

  const handleSignOut = () => {
    clearToken();
    router.push('/login');
  };

  return (
    <header className="no-print" style={styles.header}>
      <a href="/dashboard" style={styles.brand}>
        <span style={styles.logo}>D</span>
        <span>
          <span className="serif" style={styles.name}>DreamLog</span>
          <span style={styles.tag}>Clinic</span>
        </span>
      </a>
      <button className="btn-ghost" style={{ padding: '8px 16px', fontSize: '0.82rem' }} onClick={handleSignOut}>
        Sign out
      </button>
    </header>
  );
}

const styles: Record<string, React.CSSProperties> = {
  header: {
    position: 'sticky', top: 0, zIndex: 50,
    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    padding: '14px 24px',
    background: 'rgba(250,248,245,0.88)', backdropFilter: 'blur(14px)',
    borderBottom: '1px solid var(--border)',
  },
  brand: { display: 'flex', alignItems: 'center', gap: '10px', textDecoration: 'none', color: 'var(--text)' },
  logo: {
    width: '30px', height: '30px', borderRadius: '7px', background: 'var(--brand)',
    display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
    fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700,
    fontSize: '1.15rem', color: '#FAF8F5',
  },
  name: { fontSize: '1.25rem', fontWeight: 600, fontStyle: 'italic', marginRight: '8px' },
  tag: {
    fontSize: '0.62rem', textTransform: 'uppercase', letterSpacing: '1px',
    color: 'var(--brand)', background: 'var(--brand-tint)',
    padding: '2px 7px', borderRadius: '4px', fontWeight: 700, verticalAlign: 'middle',
  },
};
