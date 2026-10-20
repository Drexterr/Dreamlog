'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { api, getToken, type ClientSummary } from '../../lib/api';
import PortalShell from '../../components/PortalShell';
export default function DashboardPage() {
  const router = useRouter();
  const [clients, setClients] = useState<ClientSummary[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(''),
    [search, setSearch] = useState(''),
    [filter, setFilter] = useState('all'),
    [clientId, setClientId] = useState(''),
    [linkError, setLinkError] = useState(''),
    [status, setStatus] = useState(''),
    [busy, setBusy] = useState(false);
  const lock = useRef(false),
    dialog = useRef<HTMLDialogElement>(null),
    mounted = useRef(false);
  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const r = await api.listClients();
      if (mounted.current) setClients(r);
    } catch {
      if (mounted.current) setError('Could not load clients. Please try again.');
    } finally {
      if (mounted.current) setLoading(false);
    }
  }, []);
  useEffect(() => {
    mounted.current = true;
    if (!getToken()) {
      router.replace('/login');
      return;
    }
    void load();
    return () => {
      mounted.current = false;
    };
  }, [load, router]);
  const visible = useMemo(() => clients.filter(c => c.name.toLowerCase().includes(search.trim().toLowerCase()) && (filter === 'all' || (filter === 'active' ? c.entry_count > 0 : c.entry_count === 0))), [clients, search, filter]);
  const active = clients.filter(c => c.last_entry_at && Date.now() - new Date(c.last_entry_at).getTime() < 7 * 86400000).length;
  async function link(e: FormEvent) {
    e.preventDefault();
    if (lock.current || !clientId.trim()) return;
    lock.current = true;
    setBusy(true);
    setLinkError('');
    try {
      await api.linkClient(clientId.trim());
      if (!mounted.current) return;
      setClientId('');
      setStatus('Link request sent. Your client must approve it in the Ode app before their information appears here.');
      dialog.current?.close();
      await load();
    } catch {
      if (mounted.current) setLinkError('Could not link this client. Check their Ode user ID and that sharing is approved.');
    } finally {
      lock.current = false;
      if (mounted.current) setBusy(false);
    }
  }
  return <PortalShell title="Clients"><div className="page-heading"><div><h1>Your clients.</h1><p>A little context before your next conversation.</p></div><button className="btn" onClick={() => {
        setLinkError('');
        dialog.current?.showModal();
      }}>Link client +</button></div>{status && <p role="status" className="notice">{status}</p>}<div className="stats-strip"><div><strong>{loading ? '—' : clients.length}</strong><span>Linked clients</span></div><div><strong>{loading ? '—' : active}</strong><span>Journaled this week</span></div><div><strong>30 days</strong><span>Current overview</span></div></div><div className="notice"><p><strong>Client choice comes first.</strong> Briefs reflect shared entries. They are AI interpretations, not clinical assessments.</p></div><div className="toolbar"><div className="tabs" aria-label="Filter clients">{[['all', 'All clients'], ['active', 'With entries'], ['empty', 'No entries yet']].map(([id, label]) => <button key={id} aria-pressed={filter === id} onClick={() => setFilter(id)}>{label}</button>)}</div><input aria-label="Search clients" placeholder="Search clients…" value={search} onChange={e => setSearch(e.target.value)} /></div>{error ? <div className="notice" role="alert"><p>{error}</p><button className="text-btn" onClick={() => void load()}>Try again</button></div> : loading ? <p role="status" className="empty">Loading your clients…</p> : visible.length ? <div className="table-scroll"><table className="client-table"><thead><tr><th scope="col">Client</th><th scope="col">Last journal</th><th scope="col">Entries</th><th scope="col">30-day mood</th><th scope="col">Brief</th></tr></thead><tbody>{visible.map(c => <tr key={c.client_id}><td data-label="Client"><a href={`/dashboard/clients/${encodeURIComponent(c.client_id)}`}><span className="initial" aria-hidden="true">{c.name.split(/\s+/).map(n => n[0]).join('').slice(0, 2)}</span>{c.name}</a></td><td data-label="Last journal">{c.last_entry_at ? new Date(c.last_entry_at).toLocaleDateString(undefined, {
                month: 'short',
                day: 'numeric'
              }) : 'No entries yet'}</td><td data-label="Entries">{c.entry_count}</td><td data-label="30-day mood">{c.avg_mood_30d == null ? 'Not available' : `${Math.round(c.avg_mood_30d)} / 100`}</td><td data-label="Brief"><a href={`/dashboard/clients/${encodeURIComponent(c.client_id)}`}>{c.entry_count ? 'Read brief ↗' : 'View client ↗'}</a></td></tr>)}</tbody></table></div> : <div className="empty">{clients.length ? 'No clients match. Try another name or filter.' : 'No linked clients yet. Ask your client to share their Ode user ID from Settings, then link them here.'}</div>}<p className="table-footer">Shared summaries and mood data only. Journal-derived scores are not clinical assessments.</p><dialog ref={dialog} aria-labelledby="link-heading" onCancel={e => {
      if (busy) e.preventDefault();
    }}><form onSubmit={link}><h2 id="link-heading">Link a client</h2><p className="quiet small-note">Ask your client for their Ode user ID from Settings. Their approval is required before shared information is available.</p><label htmlFor="client-id">Ode user ID</label><input id="client-id" value={clientId} onChange={e => setClientId(e.target.value)} required disabled={busy} />{linkError && <p className="error" role="alert">{linkError}</p>}<div className="dialog-actions"><button className="text-btn" type="button" disabled={busy} onClick={() => dialog.current?.close()}>Cancel</button><button className="btn" disabled={busy}>{busy ? 'Linking…' : 'Link client'}</button></div></form></dialog></PortalShell>;
}
