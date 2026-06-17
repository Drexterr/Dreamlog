'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api, getToken, type ClientSummary } from '../../lib/api';
import PortalSidebar from '../../components/PortalSidebar';
import { differenceInDays, format, parseISO } from 'date-fns';

function moodColor(score: number): string {
  if (score >= 70) return 'var(--success)';
  if (score >= 45) return 'var(--gold)';
  return 'var(--danger)';
}

function trendBadge(client: ClientSummary) {
  if (client.avg_mood_30d == null) return null;
  if (client.avg_mood_30d < 45) return <span className="badge-declining">↘ Declining</span>;
  if (client.avg_mood_30d >= 65) return <span className="badge-improving">↗ Improving</span>;
  return <span className="badge-stable">— Stable</span>;
}

function dynamicHeadline(clients: ClientSummary[]): string {
  const activeThisWeek = clients.filter(
    c => c.last_entry_at && differenceInDays(new Date(), parseISO(c.last_entry_at)) <= 7
  ).length;
  if (activeThisWeek === 0) return 'No new entries this week.';
  const words = ['One', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine', 'Ten'];
  const word = activeThisWeek <= 10 ? words[activeThisWeek - 1] : String(activeThisWeek);
  return `${word} client${activeThisWeek === 1 ? '' : 's'} shared reflections this week.`;
}

function crossClientNote(clients: ClientSummary[]): string | null {
  const declining = clients.filter(c => c.avg_mood_30d != null && c.avg_mood_30d < 45);
  if (declining.length >= 3) {
    return `${declining.length} of your clients are showing mood scores below 45 this month. Worth a closer look before your next sessions.`;
  }
  const active = clients.filter(c => c.last_entry_at && differenceInDays(new Date(), parseISO(c.last_entry_at)) <= 3);
  if (active.length >= 2) {
    return `${active.length} clients journaled in the last 72 hours. Their entries are ready for you ahead of upcoming sessions.`;
  }
  if (clients.length > 0) {
    return `You have ${clients.length} linked client${clients.length === 1 ? '' : 's'}. Entries are anonymised — you only see AI summaries and mood trends.`;
  }
  return null;
}

export default function DashboardPage() {
  const router = useRouter();
  const [clients, setClients] = useState<ClientSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<'all' | 'attention' | 'improving' | 'stable'>('all');
  const [linkInput, setLinkInput] = useState('');
  const [linking, setLinking] = useState(false);
  const [linkError, setLinkError] = useState('');
  const [showAddClient, setShowAddClient] = useState(false);

  useEffect(() => {
    if (!getToken()) { router.replace('/login'); return; }
    api.listClients()
      .then(setClients)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [router]);

  const handleLink = async () => {
    if (!linkInput.trim()) return;
    setLinking(true);
    setLinkError('');
    try {
      await api.linkClient(linkInput.trim());
      const updated = await api.listClients();
      setClients(updated);
      setLinkInput('');
      setShowAddClient(false);
    } catch {
      setLinkError('Client not found or already linked. Ask your client to share their DreamLog user ID from Settings.');
    } finally {
      setLinking(false);
    }
  };

  const handleUnlink = async (clientId: string, name: string) => {
    if (!confirm(`Remove ${name} from your client list?`)) return;
    try {
      await api.unlinkClient(clientId);
      setClients(prev => prev.filter(c => c.client_id !== clientId));
    } catch {}
  };

  const stats = useMemo(() => {
    const withMood = clients.filter(c => c.avg_mood_30d != null);
    const avgMood = withMood.length
      ? (withMood.reduce((s, c) => s + (c.avg_mood_30d ?? 0), 0) / withMood.length).toFixed(1)
      : null;
    const activeThisWeek = clients.filter(
      c => c.last_entry_at && differenceInDays(new Date(), parseISO(c.last_entry_at)) <= 7
    ).length;
    const needAttention = clients.filter(c => c.avg_mood_30d != null && c.avg_mood_30d < 45).length;
    return { total: clients.length, activeThisWeek, avgMood, needAttention };
  }, [clients]);

  const attentionClients = useMemo(
    () => clients.filter(c => c.avg_mood_30d != null && c.avg_mood_30d < 50)
          .sort((a, b) => (a.avg_mood_30d ?? 99) - (b.avg_mood_30d ?? 99))
          .slice(0, 3),
    [clients]
  );

  const visibleClients = useMemo(() => {
    let list = [...clients];
    if (search.trim()) list = list.filter(c => c.name.toLowerCase().includes(search.toLowerCase()));
    if (filter === 'attention') list = list.filter(c => c.avg_mood_30d != null && c.avg_mood_30d < 45);
    if (filter === 'improving') list = list.filter(c => c.avg_mood_30d != null && c.avg_mood_30d >= 65);
    if (filter === 'stable')    list = list.filter(c => c.avg_mood_30d != null && c.avg_mood_30d >= 45 && c.avg_mood_30d < 65);
    return list.sort((a, b) => {
      if (!a.last_entry_at) return 1;
      if (!b.last_entry_at) return -1;
      return b.last_entry_at.localeCompare(a.last_entry_at);
    });
  }, [clients, search, filter]);

  const note = crossClientNote(clients);
  const today = new Date();
  const hour = today.getHours();
  const greeting = hour < 12 ? 'Good morning' : hour < 17 ? 'Good afternoon' : 'Good evening';

  return (
    <div className="portal-layout">
      <PortalSidebar />

      <main className="portal-main" style={{ padding: '0', minHeight: '100vh' }}>
        {/* Top bar */}
        <div style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          padding: '16px 32px', borderBottom: '1px solid var(--border)',
          position: 'sticky', top: 0, background: 'var(--bg)', zIndex: 10,
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, flex: 1 }}>
            <svg width="15" height="15" viewBox="0 0 15 15" fill="none" style={{ color: 'var(--muted-2)' }}>
              <circle cx="6.5" cy="6.5" r="5" stroke="currentColor" strokeWidth="1.4"/>
              <path d="M11 11l2.5 2.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
            </svg>
            <span style={{ color: 'var(--muted-2)', fontSize: '0.86rem' }}>Search clients, themes…</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <span style={{ color: 'var(--muted)', fontSize: '0.82rem' }}>
              {format(today, 'EEEE, MMMM d')}
            </span>
            <button
              className="btn-primary"
              style={{ padding: '8px 16px', fontSize: '0.82rem', borderRadius: 10 }}
              onClick={() => setShowAddClient(true)}
            >
              + Add Client
            </button>
          </div>
        </div>

        <div style={{ padding: '36px 32px 64px' }}>
          {/* Header */}
          <div className="fade-up" style={{ marginBottom: 28 }}>
            <div className="eyebrow" style={{ marginBottom: 8 }}>
              {greeting.toUpperCase()}
            </div>
            <h1 className="serif" style={{
              fontSize: 'clamp(1.6rem, 3vw, 2.4rem)', fontWeight: 300,
              color: 'var(--text)', margin: 0, lineHeight: 1.2,
            }}>
              {loading ? 'Loading your dashboard.' : dynamicHeadline(clients)}
            </h1>
          </div>

          {/* Stat cards */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(4, 1fr)',
            gap: 14, marginBottom: 28,
          }}>
            {[
              { label: 'TOTAL CLIENTS', value: loading ? '—' : String(stats.total), sub: 'Active in last 30 days' },
              { label: 'NEED ATTENTION', value: loading ? '—' : String(stats.needAttention), sub: `${stats.needAttention} clients trending down`, alert: !loading && stats.needAttention > 0 },
              { label: 'AVERAGE MOOD', value: loading ? '—' : (stats.avgMood ?? '—'), sub: '30-day weighted avg' },
              { label: 'ACTIVE THIS WEEK', value: loading ? '—' : String(stats.activeThisWeek), sub: 'Entries in last 7 days' },
            ].map((s, i) => (
              <div key={i} className="stat-card fade-up" style={{ animationDelay: `${i * 0.06}s` }}>
                <div className="eyebrow" style={{ marginBottom: 12 }}>{s.label}</div>
                <div className="serif" style={{
                  fontSize: '2.4rem', fontWeight: 300, lineHeight: 1,
                  color: s.alert ? 'var(--danger)' : 'var(--text)',
                  marginBottom: 6,
                }}>{s.value}</div>
                <div style={{ fontSize: '0.75rem', color: 'var(--muted-2)' }}>{s.sub}</div>
              </div>
            ))}
          </div>

          {/* Bottom two-col */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 360px', gap: 14, marginBottom: 32 }}>

            {/* Quietly worth noticing */}
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
                <div>
                  <div className="eyebrow" style={{ marginBottom: 6 }}>QUIETLY WORTH NOTICING</div>
                  <h2 style={{ fontSize: '1.1rem', fontWeight: 600, color: 'var(--text)', margin: 0 }}>
                    Clients with shifting mood
                  </h2>
                </div>
                {clients.length > 0 && (
                  <span
                    style={{ fontSize: '0.78rem', color: 'var(--muted)', cursor: 'pointer' }}
                    onClick={() => setFilter('attention')}
                  >
                    See all →
                  </span>
                )}
              </div>

              {loading ? (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                  {[0, 1].map(i => <div key={i} className="skeleton" style={{ height: 60 }} />)}
                </div>
              ) : attentionClients.length === 0 ? (
                <p style={{ color: 'var(--muted)', fontSize: '0.86rem', margin: '24px 0', textAlign: 'center' }}>
                  {clients.length === 0
                    ? 'No clients linked yet.'
                    : 'All clients are in a good range this month.'}
                </p>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  {attentionClients.map(client => (
                    <div
                      key={client.client_id}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 12,
                        padding: '14px 12px', borderRadius: 12, cursor: 'pointer',
                        transition: 'background 0.15s ease',
                      }}
                      className="card-hover"
                      onClick={() => router.push(`/dashboard/clients/${client.client_id}`)}
                    >
                      <div style={{
                        width: 36, height: 36, borderRadius: '50%',
                        background: 'var(--bg-card-2)', border: '1px solid var(--border)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        fontSize: 14, color: 'var(--muted)', fontWeight: 700, flexShrink: 0,
                        fontFamily: "'Cormorant Garamond', serif",
                      }}>
                        {client.name.charAt(0).toUpperCase()}
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontWeight: 600, fontSize: '0.9rem', color: 'var(--text)' }}>{client.name}</div>
                        <div style={{ fontSize: '0.75rem', color: 'var(--muted)', marginTop: 2 }}>
                          {client.entry_count} entries
                          {client.last_entry_at && ` · ${format(parseISO(client.last_entry_at), 'MMM d')}`}
                        </div>
                      </div>
                      <span className="badge-declining">↘ Declining</span>
                      <span className="serif" style={{
                        fontSize: '1.4rem', fontWeight: 300,
                        color: client.avg_mood_30d != null ? moodColor(client.avg_mood_30d) : 'var(--muted)',
                        minWidth: 36, textAlign: 'right',
                      }}>
                        {client.avg_mood_30d?.toFixed(1) ?? '—'}
                      </span>
                      <span style={{ color: 'var(--muted)', fontSize: '0.9rem' }}>→</span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Note from DreamLog */}
            <div className="card" style={{ background: 'rgba(201,169,110,0.04)', borderColor: 'rgba(201,169,110,0.12)' }}>
              <div className="eyebrow" style={{ marginBottom: 14, color: 'var(--gold)' }}>A NOTE FROM DREAMLOG</div>
              {loading ? (
                <div className="skeleton" style={{ height: 80 }} />
              ) : note ? (
                <>
                  <p className="serif" style={{
                    fontStyle: 'italic', fontSize: '1.05rem', lineHeight: 1.6,
                    color: 'var(--text)', margin: '0 0 16px',
                  }}>
                    &ldquo;{note}&rdquo;
                  </p>
                  <p style={{ fontSize: '0.75rem', color: 'var(--muted-2)', margin: 0, lineHeight: 1.5 }}>
                    — Weekly cross-client patterns, anonymised and consent-checked.
                  </p>
                </>
              ) : (
                <p style={{ color: 'var(--muted)', fontSize: '0.86rem', lineHeight: 1.6, margin: 0 }}>
                  Add your first client to start seeing cross-client patterns here.
                </p>
              )}
            </div>
          </div>

          {/* Full client list */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, gap: 12, flexWrap: 'wrap' }}>
            <div>
              <div className="eyebrow" style={{ marginBottom: 4 }}>YOUR CLIENTS</div>
              <h2 className="serif" style={{ fontSize: '1.5rem', fontWeight: 300, margin: 0, color: 'var(--text)' }}>
                {loading ? '—' : `${clients.length} client${clients.length !== 1 ? 's' : ''}${stats.needAttention > 0 ? ` · ${stats.needAttention} need a gentle check-in` : ''}`}
              </h2>
            </div>
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search by name…"
              style={{ maxWidth: 240, padding: '9px 14px', fontSize: '0.84rem' }}
            />
          </div>

          {/* Filter tabs */}
          <div style={{ display: 'flex', gap: 4, marginBottom: 14 }}>
            {([
              { key: 'all', label: 'All' },
              { key: 'attention', label: 'Needs attention' },
              { key: 'improving', label: 'Improving' },
              { key: 'stable', label: 'Stable' },
            ] as const).map(f => (
              <button
                key={f.key}
                onClick={() => setFilter(f.key)}
                style={{
                  padding: '6px 14px', borderRadius: 100, border: '1px solid',
                  fontSize: '0.8rem', fontWeight: 500, cursor: 'pointer',
                  fontFamily: 'inherit', transition: 'all 0.15s ease',
                  background: filter === f.key ? 'transparent' : 'transparent',
                  borderColor: filter === f.key ? 'var(--text)' : 'var(--border)',
                  color: filter === f.key ? 'var(--text)' : 'var(--muted)',
                }}
              >
                {f.label}
              </button>
            ))}
          </div>

          {/* Table header */}
          {!loading && visibleClients.length > 0 && (
            <div style={{
              display: 'grid',
              gridTemplateColumns: '1fr 120px 140px 160px 80px',
              gap: 16, padding: '0 16px 8px',
              borderBottom: '1px solid var(--border)',
            }}>
              {['CLIENT', '30-DAY MOOD', 'TREND', 'LAST ENTRY', 'ALERTS'].map(h => (
                <div key={h} className="eyebrow">{h}</div>
              ))}
            </div>
          )}

          {/* Client rows */}
          {loading ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 2, marginTop: 8 }}>
              {[0, 1, 2].map(i => <div key={i} className="skeleton" style={{ height: 68, borderRadius: 12 }} />)}
            </div>
          ) : visibleClients.length === 0 ? (
            <div style={{
              textAlign: 'center', padding: '56px 24px',
              background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 16, marginTop: 8,
            }}>
              <p style={{ color: 'var(--muted)', margin: 0, fontSize: '0.9rem' }}>
                {clients.length === 0
                  ? 'No clients linked. Add your first client above.'
                  : `No clients match "${search}".`}
              </p>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 2, marginTop: 4 }}>
              {visibleClients.map((client, i) => {
                const inactive = !client.last_entry_at || differenceInDays(new Date(), parseISO(client.last_entry_at)) > 14;
                const lowMood = client.avg_mood_30d != null && client.avg_mood_30d < 45;
                return (
                  <div
                    key={client.client_id}
                    className="card-hover fade-up"
                    style={{
                      display: 'grid',
                      gridTemplateColumns: '1fr 120px 140px 160px 80px',
                      gap: 16, alignItems: 'center',
                      padding: '16px 16px',
                      borderRadius: 12,
                      borderBottom: '1px solid var(--border)',
                      animationDelay: `${Math.min(i, 8) * 0.04}s`,
                      cursor: 'pointer',
                    }}
                    onClick={() => router.push(`/dashboard/clients/${client.client_id}`)}
                  >
                    {/* Client */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 12, minWidth: 0 }}>
                      <div style={{
                        width: 34, height: 34, borderRadius: '50%', flexShrink: 0,
                        background: 'rgba(201,169,110,0.1)', border: '1px solid rgba(201,169,110,0.15)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        fontSize: 14, color: 'var(--gold)', fontWeight: 700, fontFamily: "'Cormorant Garamond', serif",
                      }}>{client.name.charAt(0).toUpperCase()}</div>
                      <div style={{ minWidth: 0 }}>
                        <div style={{ fontWeight: 600, fontSize: '0.9rem', color: 'var(--text)', display: 'flex', alignItems: 'center', gap: 8 }}>
                          {client.name}
                          {inactive && <span className="chip-muted" style={{ fontSize: '0.6rem' }}>quiet</span>}
                        </div>
                        <div style={{ fontSize: '0.73rem', color: 'var(--muted-2)', marginTop: 2 }}>
                          {client.entry_count} entries
                        </div>
                      </div>
                    </div>

                    {/* Mood */}
                    <div>
                      <span className="serif" style={{
                        fontSize: '1.5rem', fontWeight: 300,
                        color: client.avg_mood_30d != null ? moodColor(client.avg_mood_30d) : 'var(--muted)',
                      }}>
                        {client.avg_mood_30d?.toFixed(1) ?? '—'}
                      </span>
                    </div>

                    {/* Trend */}
                    <div>{trendBadge(client)}</div>

                    {/* Last entry */}
                    <div style={{ fontSize: '0.82rem', color: 'var(--muted)' }}>
                      {client.last_entry_at
                        ? differenceInDays(new Date(), parseISO(client.last_entry_at)) === 0
                          ? 'Today'
                          : differenceInDays(new Date(), parseISO(client.last_entry_at)) === 1
                          ? 'Yesterday'
                          : `${differenceInDays(new Date(), parseISO(client.last_entry_at))} days ago`
                        : 'No entries yet'}
                    </div>

                    {/* Alerts */}
                    <div>
                      {lowMood && (
                        <span style={{
                          display: 'flex', alignItems: 'center', gap: 4,
                          fontSize: '0.68rem', color: 'var(--danger)', fontWeight: 600,
                        }}>
                          ⚠ LOW MOOD
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </main>

      {/* Add client modal */}
      {showAddClient && (
        <div
          style={{
            position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
            backdropFilter: 'blur(8px)', zIndex: 200,
            display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24,
          }}
          onClick={e => { if (e.target === e.currentTarget) setShowAddClient(false); }}
        >
          <div className="card fade-up" style={{ width: '100%', maxWidth: 440, padding: 32 }}>
            <h2 className="serif" style={{ fontSize: '1.6rem', fontWeight: 300, margin: '0 0 8px', color: 'var(--text)' }}>
              Add a client
            </h2>
            <p style={{ color: 'var(--muted)', fontSize: '0.86rem', margin: '0 0 24px', lineHeight: 1.6 }}>
              Ask your client to go to <strong style={{ color: 'var(--text)' }}>Settings → Share User ID</strong> in the DreamLog app and send you their ID.
            </p>
            <input
              value={linkInput}
              onChange={e => setLinkInput(e.target.value)}
              placeholder="Client user ID (UUID)"
              autoFocus
              onKeyDown={e => e.key === 'Enter' && handleLink()}
              style={{ marginBottom: 12 }}
            />
            {linkError && (
              <p style={{ color: 'var(--danger)', fontSize: '0.8rem', margin: '0 0 12px' }}>{linkError}</p>
            )}
            <div style={{ display: 'flex', gap: 10 }}>
              <button className="btn-ghost" onClick={() => setShowAddClient(false)} style={{ flex: 1 }}>Cancel</button>
              <button className="btn-primary" onClick={handleLink} disabled={linking} style={{ flex: 1, borderRadius: 12 }}>
                {linking ? <span className="spin" /> : 'Add client'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
