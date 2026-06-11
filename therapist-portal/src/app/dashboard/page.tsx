'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api, getToken, type ClientSummary } from '../../lib/api';
import MoodBadge from '../../components/MoodBadge';
import PortalHeader from '../../components/PortalHeader';
import { Sprout } from '../../components/icons';
import { differenceInDays, format, parseISO } from 'date-fns';

export default function DashboardPage() {
  const router = useRouter();
  const [clients, setClients] = useState<ClientSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [linkInput, setLinkInput] = useState('');
  const [linking, setLinking] = useState(false);
  const [linkError, setLinkError] = useState('');

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
    } catch {
      setLinkError('Client not found or already linked. Ask your client to share their DreamLog user ID.');
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
      ? Math.round(withMood.reduce((sum, c) => sum + (c.avg_mood_30d ?? 0), 0) / withMood.length)
      : null;
    const activeThisWeek = clients.filter(
      c => c.last_entry_at && differenceInDays(new Date(), parseISO(c.last_entry_at)) <= 7
    ).length;
    const needAttention = clients.filter(c => c.avg_mood_30d != null && c.avg_mood_30d < 46).length;
    return { total: clients.length, activeThisWeek, avgMood, needAttention };
  }, [clients]);

  const visibleClients = useMemo(() => {
    const filtered = search.trim()
      ? clients.filter(c => c.name.toLowerCase().includes(search.trim().toLowerCase()))
      : clients;
    return [...filtered].sort((a, b) => {
      if (!a.last_entry_at) return 1;
      if (!b.last_entry_at) return -1;
      return b.last_entry_at.localeCompare(a.last_entry_at);
    });
  }, [clients, search]);

  return (
    <div style={{ minHeight: '100vh' }}>
      <PortalHeader />

      <div style={{ maxWidth: 940, margin: '0 auto', padding: '32px 24px 64px' }}>
        <div className="fade-up" style={{ marginBottom: 28 }}>
          <span className="eyebrow">therapist portal</span>
          <h1 className="serif" style={{ fontSize: 'clamp(1.8rem, 3.5vw, 2.4rem)', fontWeight: 300, margin: '6px 0 4px' }}>
            Your clients
          </h1>
          <p style={{ color: 'var(--muted)', fontSize: '0.9rem', margin: 0 }}>
            A high-level picture of the week between sessions — never raw recordings.
          </p>
        </div>

        {/* Stats overview */}
        <div style={styles.statsGrid}>
          {[
            { label: 'Clients', value: loading ? '—' : String(stats.total) },
            { label: 'Active this week', value: loading ? '—' : String(stats.activeThisWeek) },
            { label: 'Avg 30-day mood', value: loading || stats.avgMood == null ? '—' : String(stats.avgMood) },
            { label: 'May need attention', value: loading ? '—' : String(stats.needAttention), alert: !loading && stats.needAttention > 0 },
          ].map(stat => (
            <div key={stat.label} className="card" style={{ textAlign: 'center', padding: '18px 12px' }}>
              <div className="serif" style={{ fontSize: '1.9rem', fontWeight: 600, color: stat.alert ? 'var(--danger)' : 'var(--text)' }}>
                {stat.value}
              </div>
              <div style={{ fontSize: '0.72rem', color: 'var(--muted)', marginTop: 2 }}>{stat.label}</div>
            </div>
          ))}
        </div>

        {/* Add client */}
        <div className="card" style={{ marginBottom: 32 }}>
          <h2 style={{ fontSize: '0.95rem', fontWeight: 600, margin: '0 0 8px' }}>Add a client</h2>
          <p style={{ color: 'var(--muted)', fontSize: '0.82rem', margin: '0 0 14px', lineHeight: 1.6 }}>
            Ask your client to go to <strong>Settings → Share User ID</strong> in the DreamLog app
            and send you their ID. Paste it below.
          </p>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            <input
              value={linkInput}
              onChange={e => setLinkInput(e.target.value)}
              placeholder="Client user ID (UUID)"
              style={{ flex: '1 1 260px' }}
              onKeyDown={e => e.key === 'Enter' && handleLink()}
            />
            <button className="btn-primary" onClick={handleLink} disabled={linking}>
              {linking ? <span className="spin" /> : 'Add client'}
            </button>
          </div>
          {linkError && <p style={{ color: 'var(--danger)', fontSize: '0.78rem', marginTop: 10, marginBottom: 0 }}>{linkError}</p>}
        </div>

        {/* Client list */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 16, marginBottom: 14, flexWrap: 'wrap' }}>
          <h2 style={{ fontSize: '0.95rem', fontWeight: 600, margin: 0 }}>
            Client list {!loading && clients.length > 0 && <span style={{ color: 'var(--muted)', fontWeight: 400 }}>({visibleClients.length})</span>}
          </h2>
          {clients.length > 3 && (
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search clients…"
              style={{ maxWidth: 220, padding: '8px 12px', fontSize: '0.84rem' }}
            />
          )}
        </div>

        {loading ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {[0, 1, 2].map(i => <div key={i} className="skeleton" style={{ height: 84, borderRadius: 16 }} />)}
          </div>
        ) : clients.length === 0 ? (
          <div className="card" style={{ textAlign: 'center', padding: '56px 24px' }}>
            <div style={{ color: 'var(--brand)', marginBottom: 12, display: 'flex', justifyContent: 'center' }}><Sprout size={34} /></div>
            <p style={{ fontWeight: 600, margin: '0 0 6px' }}>No clients linked yet</p>
            <p style={{ color: 'var(--muted)', fontSize: '0.85rem', margin: 0, lineHeight: 1.6 }}>
              When a client shares their DreamLog user ID with you, add it above to see
              their mood trends and pre-session briefs.
            </p>
          </div>
        ) : visibleClients.length === 0 ? (
          <div className="card" style={{ textAlign: 'center', padding: '32px 24px' }}>
            <p style={{ color: 'var(--muted)', margin: 0, fontSize: '0.88rem' }}>No clients match “{search}”.</p>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {visibleClients.map((client, i) => {
              const inactive = !client.last_entry_at || differenceInDays(new Date(), parseISO(client.last_entry_at)) > 14;
              return (
                <div
                  key={client.client_id}
                  className="card card-hover fade-up client-row"
                  style={{ display: 'flex', alignItems: 'center', gap: 16, animationDelay: `${Math.min(i, 8) * 0.05}s` }}
                  onClick={() => router.push(`/dashboard/clients/${client.client_id}`)}
                >
                  <div style={styles.avatar}>{client.name.charAt(0).toUpperCase()}</div>

                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                      <span style={{ fontWeight: 600, fontSize: '0.95rem' }}>{client.name}</span>
                      {inactive && <span className="chip-muted">quiet for 2+ weeks</span>}
                    </div>
                    <div style={{ color: 'var(--muted)', fontSize: '0.78rem', marginTop: 3 }}>
                      {client.entry_count} entries ·{' '}
                      {client.last_entry_at
                        ? `last entry ${format(parseISO(client.last_entry_at), 'MMM d')}`
                        : 'no entries yet'}
                      {' · '}linked {format(parseISO(client.linked_at), 'MMM d, yyyy')}
                    </div>
                  </div>

                  {client.avg_mood_30d != null && (
                    <MoodBadge score={client.avg_mood_30d} label="30d avg" />
                  )}

                  <button
                    className="btn-danger-ghost"
                    onClick={e => { e.stopPropagation(); handleUnlink(client.client_id, client.name); }}
                  >
                    Remove
                  </button>

                  <span style={{ color: 'var(--muted)', fontSize: '1.1rem' }}>›</span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
    gap: 14,
    marginBottom: 24,
  },
  avatar: {
    width: 44, height: 44, borderRadius: '50%',
    background: 'var(--brand)', display: 'flex',
    alignItems: 'center', justifyContent: 'center',
    flexShrink: 0, fontSize: 17, color: '#FAF8F5', fontWeight: 700,
    fontFamily: "'Cormorant Garamond', serif",
  },
};
