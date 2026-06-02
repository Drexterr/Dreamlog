'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api, clearToken, getToken, type ClientSummary } from '../../lib/api';
import MoodBadge from '../../components/MoodBadge';
import { format, parseISO } from 'date-fns';

export default function DashboardPage() {
  const router = useRouter();
  const [clients, setClients] = useState<ClientSummary[]>([]);
  const [loading, setLoading] = useState(true);
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

  const handleSignOut = () => {
    clearToken();
    router.push('/login');
  };

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

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: '32px 24px' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 32 }}>
        <div>
          <h1 style={{ fontFamily: 'Georgia, serif', fontSize: 26, color: 'var(--brand)', margin: 0 }}>
            DreamLog
          </h1>
          <p style={{ color: 'var(--muted)', fontSize: 13, margin: '2px 0 0' }}>Therapist Portal</p>
        </div>
        <button className="btn-ghost" onClick={handleSignOut}>Sign out</button>
      </div>

      {/* Add client */}
      <div className="card" style={{ marginBottom: 32 }}>
        <h2 style={{ fontSize: 15, fontWeight: 600, margin: '0 0 12px', color: 'var(--text)' }}>
          Add a client
        </h2>
        <p style={{ color: 'var(--muted)', fontSize: 13, margin: '0 0 14px', lineHeight: 1.6 }}>
          Ask your client to go to <strong>Settings → Share User ID</strong> in the DreamLog app
          and send you their ID. Paste it below.
        </p>
        <div style={{ display: 'flex', gap: 10 }}>
          <input
            value={linkInput}
            onChange={e => setLinkInput(e.target.value)}
            placeholder="Client user ID (UUID)"
            style={{ flex: 1 }}
            onKeyDown={e => e.key === 'Enter' && handleLink()}
          />
          <button className="btn-primary" onClick={handleLink} disabled={linking}>
            {linking ? 'Linking…' : 'Add client'}
          </button>
        </div>
        {linkError && <p style={{ color: '#f87171', fontSize: 12, marginTop: 8 }}>{linkError}</p>}
      </div>

      {/* Clients list */}
      <h2 style={{ fontSize: 15, fontWeight: 600, margin: '0 0 14px', color: 'var(--text)' }}>
        Your clients
      </h2>

      {loading ? (
        <p style={{ color: 'var(--muted)' }}>Loading…</p>
      ) : clients.length === 0 ? (
        <div className="card" style={{ textAlign: 'center', padding: '40px 24px' }}>
          <p style={{ color: 'var(--muted)', margin: 0 }}>No clients yet. Add one above.</p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {clients.map(client => (
            <div
              key={client.client_id}
              className="card"
              style={{ display: 'flex', alignItems: 'center', gap: 16, cursor: 'pointer' }}
              onClick={() => router.push(`/dashboard/clients/${client.client_id}`)}
            >
              {/* Avatar */}
              <div style={{
                width: 44, height: 44, borderRadius: '50%',
                background: 'var(--brand)', display: 'flex',
                alignItems: 'center', justifyContent: 'center',
                flexShrink: 0, fontSize: 18, color: 'white', fontWeight: 700,
              }}>
                {client.name.charAt(0).toUpperCase()}
              </div>

              <div style={{ flex: 1 }}>
                <div style={{ fontWeight: 600, fontSize: 15 }}>{client.name}</div>
                <div style={{ color: 'var(--muted)', fontSize: 12, marginTop: 2 }}>
                  {client.entry_count} entries ·{' '}
                  {client.last_entry_at
                    ? `Last entry ${format(parseISO(client.last_entry_at), 'MMM d')}`
                    : 'No entries yet'}
                </div>
              </div>

              {client.avg_mood_30d != null && (
                <MoodBadge score={client.avg_mood_30d} label="30d avg" />
              )}

              <button
                className="btn-ghost"
                style={{ fontSize: 12, padding: '6px 12px' }}
                onClick={e => { e.stopPropagation(); handleUnlink(client.client_id, client.name); }}
              >
                Remove
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
