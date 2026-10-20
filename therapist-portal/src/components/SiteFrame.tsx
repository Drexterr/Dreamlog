'use client';

import { useEffect, useRef, type ReactNode } from 'react';
const palettes: Record<string, [string, string, string]> = {
  curious: ['#18150f', '#d8ad73', '#80542e'],
  anxiety: ['#1A2B20', '#97bca1', '#42664c'],
  stress: ['#0F1E2E', '#94b9db', '#3d6283'],
  grief: ['#1C1820', '#bbacd1', '#6a5484'],
  depression: ['#1A160A', '#dbc166', '#746019'],
  relationships: ['#221518', '#dda6a6', '#8b4d4d'],
  career: ['#0C1A1A', '#7ec2c2', '#336b6b'],
  trauma: ['#1C1410', '#d9a58c', '#8d4e31']
};
export function SiteFrame({
  children
}: {
  children: ReactNode;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    let goal = 'curious';
    try {
      goal = localStorage.getItem('ode-proposal-goal-theme') || goal;
    } catch {}
    const [bg, amber, accent] = palettes[goal] || palettes.curious;
    ref.current?.style.setProperty('--bg', bg);
    ref.current?.style.setProperty('--amber', amber);
    ref.current?.style.setProperty('--accent', accent);
  }, []);
  return <div className="ode-site" ref={ref}>{children}</div>;
}
export function Brand({
  href = '/'
}: {
  href?: string;
}) {
  return <a className="brand" href={href}><svg viewBox="0 0 52 26" fill="none" aria-hidden="true"><path d="M2 14C6 3 9 23 14 13S22 21 27 13S34 18 40 13" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" /><circle cx="48" cy="13" r="2" fill="currentColor" /></svg>Ode</a>;
}
export function SiteHeader() {
  return <header className="site-head"><nav className="nav wrap" aria-label="Main navigation"><Brand /><div className="nav-links"><a href="/therapists">Therapists</a><a href="/pricing">Plans</a><a className="secondary-nav" href="/about">About</a><a className="secondary-nav" href="/teams">Teams</a><a className="btn small" href="/login">Therapist portal ↗</a></div></nav></header>;
}
export function SiteFooter() {
  return <footer className="site-foot"><div className="wrap"><div className="foot-grid"><Brand /><div className="foot-links">{[['Home', '/'], ['Plans', '/pricing'], ['Therapists', '/therapists'], ['Teams', '/teams'], ['About', '/about'], ['Privacy', '/privacy'], ['Terms', '/terms'], ['Support', 'mailto:talktoode.dev@gmail.com']].map(([name, href]) => <a key={href} href={href}>{name}</a>)}</div></div><p>Ode supports reflection. It is not clinical therapy or an emergency service.</p></div></footer>;
}
