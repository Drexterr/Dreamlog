'use client';

import { usePathname, useRouter } from 'next/navigation';
import { clearToken } from '../lib/api';

const NAV = [
  {
    key: 'dashboard',
    label: 'Dashboard',
    href: '/dashboard',
    icon: (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <rect x="1" y="1" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.4"/>
        <rect x="9" y="1" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.4"/>
        <rect x="1" y="9" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.4"/>
        <rect x="9" y="9" width="6" height="6" rx="1.5" stroke="currentColor" strokeWidth="1.4"/>
      </svg>
    ),
  },
  {
    key: 'clients',
    label: 'Clients',
    href: '/dashboard',
    icon: (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <circle cx="6" cy="5" r="3" stroke="currentColor" strokeWidth="1.4"/>
        <path d="M1 14c0-2.761 2.239-5 5-5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
        <circle cx="12" cy="6" r="2.5" stroke="currentColor" strokeWidth="1.4"/>
        <path d="M10 14c0-2.209 1.343-4 3-4" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
      </svg>
    ),
  },
  {
    key: 'insights',
    label: 'Insights',
    href: null,
    soon: true,
    icon: (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M2 12l4-4 3 3 5-7" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round"/>
      </svg>
    ),
  },
  {
    key: 'settings',
    label: 'Settings',
    href: null,
    soon: true,
    icon: (
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <circle cx="8" cy="8" r="2.5" stroke="currentColor" strokeWidth="1.4"/>
        <path d="M8 1v2M8 13v2M1 8h2M13 8h2M3.05 3.05l1.41 1.41M11.54 11.54l1.41 1.41M3.05 12.95l1.41-1.41M11.54 4.46l1.41-1.41" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
      </svg>
    ),
  },
];

interface PortalSidebarProps {
  therapistName?: string;
  therapistCredentials?: string;
}

export default function PortalSidebar({ therapistName, therapistCredentials }: PortalSidebarProps) {
  const pathname = usePathname();
  const router = useRouter();

  const isActive = (key: string) => {
    if (key === 'clients' && pathname.startsWith('/dashboard/clients/')) return true;
    if (key === 'dashboard' && pathname === '/dashboard') return true;
    return false;
  };

  const handleSignOut = () => {
    clearToken();
    router.push('/login');
  };

  const initial = therapistName?.charAt(0).toUpperCase() ?? 'T';

  return (
    <aside className="portal-sidebar no-print">
      {/* Logo */}
      <div style={{ padding: '20px 20px 16px', borderBottom: '1px solid var(--border)' }}>
        <a href="/dashboard" style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{
            width: 30, height: 30, borderRadius: 8,
            background: 'var(--gold)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontFamily: "'Cormorant Garamond', serif",
            fontStyle: 'italic', fontWeight: 700, fontSize: '1.1rem', color: '#0c0b09',
          }}>D</div>
          <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.15rem', fontWeight: 600, color: 'var(--text)', letterSpacing: 0.3 }}>
            DreamLog
          </span>
        </a>
        <div style={{
          marginTop: 8, fontSize: '0.62rem', letterSpacing: '1.5px',
          textTransform: 'uppercase', color: 'var(--muted-2)', fontWeight: 600,
        }}>
          PORTAL
        </div>
      </div>

      {/* Search hint */}
      <div style={{ padding: '14px 16px 8px' }}>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8,
          background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)',
          borderRadius: 10, padding: '8px 12px', color: 'var(--muted-2)', fontSize: '0.8rem',
        }}>
          <svg width="13" height="13" viewBox="0 0 13 13" fill="none">
            <circle cx="5.5" cy="5.5" r="4" stroke="currentColor" strokeWidth="1.4"/>
            <path d="M9 9l2.5 2.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
          </svg>
          Search clients…
        </div>
      </div>

      {/* Nav */}
      <nav style={{ flex: 1, padding: '8px 0' }}>
        {NAV.map(item => (
          <div
            key={item.key}
            className={`sidebar-nav-item${isActive(item.key) ? ' active' : ''}${item.soon ? ' soon' : ''}`}
            style={item.soon ? { opacity: 0.35, cursor: 'default' } : {}}
            onClick={() => {
              if (item.href && !item.soon) router.push(item.href);
            }}
          >
            {item.icon}
            <span>{item.label}</span>
            {item.soon && (
              <span style={{
                marginLeft: 'auto', fontSize: '0.6rem', letterSpacing: '1px',
                textTransform: 'uppercase', color: 'var(--muted-2)', fontWeight: 600,
              }}>soon</span>
            )}
          </div>
        ))}
      </nav>

      {/* User + sign out */}
      <div style={{ padding: '16px', borderTop: '1px solid var(--border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
          <div style={{
            width: 34, height: 34, borderRadius: '50%',
            background: 'var(--gold-light)', border: '1px solid rgba(201,169,110,0.25)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 14, color: 'var(--gold)', fontWeight: 700, fontFamily: "'Cormorant Garamond', serif", flexShrink: 0,
          }}>
            {initial}
          </div>
          <div style={{ minWidth: 0 }}>
            <div style={{ fontSize: '0.82rem', color: 'var(--text)', fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {therapistName ?? 'Therapist'}
            </div>
            {therapistCredentials && (
              <div style={{ fontSize: '0.7rem', color: 'var(--muted-2)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {therapistCredentials}
              </div>
            )}
          </div>
        </div>
        <button
          onClick={handleSignOut}
          style={{
            display: 'flex', alignItems: 'center', gap: 8,
            width: '100%', background: 'transparent', border: 'none',
            color: 'var(--muted-2)', fontSize: '0.8rem', cursor: 'pointer',
            padding: '6px 8px', borderRadius: 8, fontFamily: 'inherit',
            transition: 'color 0.15s ease',
          }}
          onMouseEnter={e => (e.currentTarget.style.color = 'var(--text)')}
          onMouseLeave={e => (e.currentTarget.style.color = 'var(--muted-2)')}
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
            <path d="M5 2H2a1 1 0 00-1 1v8a1 1 0 001 1h3M10 10l3-3-3-3M13 7H5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          Sign out
        </button>
      </div>
    </aside>
  );
}
