'use client';

import { useEffect, useState, type ReactNode } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { clearToken, getToken, notesApi, type Therapist } from '../lib/api';
import { Brand, SiteFrame } from './SiteFrame';
export default function PortalShell({
  title,
  children,
  dirty = false
}: {
  title: string;
  children: ReactNode;
  dirty?: boolean;
}) {
  const path = usePathname(),
    router = useRouter();
  const [therapist, setTherapist] = useState<Therapist | null>(null);
  const [authError, setAuthError] = useState(false);
  useEffect(() => {
    if (!getToken()) {
      router.replace('/login');
      return;
    }
    let active = true;
    notesApi.me().then(r => {
      if (active) setTherapist(r.therapist);
    }).catch(e => {
      if (!active) return;
      if (e?.response?.status === 401 || e?.response?.status === 403) {
        clearToken();
        router.replace('/login');
      } else setAuthError(true);
    });
    return () => {
      active = false;
    };
  }, [router]);
  useEffect(() => {
    const guard = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = '';
      }
    };
    window.addEventListener('beforeunload', guard);
    return () => window.removeEventListener('beforeunload', guard);
  }, [dirty]);
  return <SiteFrame><a className="skip" href="#main">Skip to content</a><div className="portal"><aside className="sidebar"><Brand /><nav aria-label="Therapist workspace"><a href="/dashboard" className={path === '/dashboard' || path.startsWith('/dashboard/clients') ? 'active' : ''}>Clients</a><a href="/dashboard/notes" className={path.startsWith('/dashboard/notes') ? 'active' : ''}>Session notes</a><a href="/therapists">Portal guide</a></nav><div className="sidebar-bottom"><strong>{therapist?.name || 'Therapist workspace'}</strong><p>{therapist?.credentials}</p><button className="text-btn" onClick={() => {
            if (dirty && !confirm('Leave without saving your notes?')) return;
            clearToken();
            router.replace('/login');
          }}>Sign out</button></div></aside><div className="portal-body"><header className="portal-header"><span>Therapist workspace / {title}</span><a href="/privacy">Privacy</a></header><main className="portal-main" id="main" tabIndex={-1}>{authError && <p className="notice" role="alert">Account details are temporarily unavailable. <button className="text-btn" onClick={() => location.reload()}>Retry</button></p>}{children}</main></div></div></SiteFrame>;
}
