'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { api, getToken, type ClientBrief } from '../../../../lib/api';
import MoodBadge from '../../../../components/MoodBadge';
import MoodTrendIcon from '../../../../components/MoodTrendIcon';
import PortalHeader from '../../../../components/PortalHeader';
import { format, parseISO } from 'date-fns';
import {
  ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid,
} from 'recharts';

export default function ClientBriefContent() {
  const router = useRouter();
  const params = useParams<{ clientId?: string[] }>();
  const clientId = params.clientId?.[0] ?? '';
  const [brief, setBrief] = useState<ClientBrief | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback((asRefresh = false) => {
    if (asRefresh) setRefreshing(true); else setLoading(true);
    setError('');
    api.getClientBrief(clientId)
      .then(setBrief)
      .catch(() => setError('Could not load client brief. Try again.'))
      .finally(() => { setLoading(false); setRefreshing(false); });
  }, [clientId]);

  useEffect(() => {
    if (!getToken()) { router.replace('/login'); return; }
    load();
  }, [load, router]);

  if (loading) {
    return (
      <div style={{ minHeight: '100vh' }}>
        <PortalHeader />
        <div style={{ maxWidth: 800, margin: '0 auto', padding: '32px 24px' }}>
          <div className="skeleton" style={{ height: 20, width: 120, marginBottom: 24 }} />
          <div style={{ display: 'flex', gap: 16, alignItems: 'center', marginBottom: 28 }}>
            <div className="skeleton" style={{ width: 54, height: 54, borderRadius: '50%' }} />
            <div style={{ flex: 1 }}>
              <div className="skeleton" style={{ height: 22, width: 180, marginBottom: 8 }} />
              <div className="skeleton" style={{ height: 14, width: 260 }} />
            </div>
          </div>
          <div className="skeleton" style={{ height: 130, borderRadius: 16, marginBottom: 20 }} />
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 12, marginBottom: 20 }}>
            {[0, 1, 2].map(i => <div key={i} className="skeleton" style={{ height: 110, borderRadius: 16 }} />)}
          </div>
          <div className="skeleton" style={{ height: 220, borderRadius: 16 }} />
          <p style={{ textAlign: 'center', color: 'var(--muted)', fontSize: '0.82rem', marginTop: 24 }}>
            Generating pre-session brief with Claude…
          </p>
        </div>
      </div>
    );
  }

  if (error || !brief) {
    return (
      <div style={{ minHeight: '100vh' }}>
        <PortalHeader />
        <div style={{ maxWidth: 760, margin: '0 auto', padding: '60px 24px', textAlign: 'center' }}>
          <p style={{ color: 'var(--danger)' }}>{error || 'Client not found.'}</p>
          <div style={{ display: 'flex', gap: 10, justifyContent: 'center', marginTop: 16 }}>
            <button className="btn-ghost" onClick={() => router.push('/dashboard')}>← Back</button>
            <button className="btn-primary" onClick={() => load()}>Retry</button>
          </div>
        </div>
      </div>
    );
  }

  const moodLabel = (score: number) =>
    score >= 71 ? 'High' : score >= 46 ? 'Moderate' : score >= 26 ? 'Low' : 'Very low';

  const chartData = [...brief.recent_entries]
    .sort((a, b) => a.date.localeCompare(b.date))
    .map(e => ({
      date: format(parseISO(e.date), 'MMM d'),
      mood: e.mood_score,
    }));

  return (
    <div style={{ minHeight: '100vh' }}>
      <PortalHeader />

      <div style={{ maxWidth: 800, margin: '0 auto', padding: '32px 24px 64px' }}>
        <div className="no-print" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24, flexWrap: 'wrap', gap: 10 }}>
          <button className="btn-ghost" style={{ fontSize: '0.82rem', padding: '8px 16px' }} onClick={() => router.push('/dashboard')}>
            ← All clients
          </button>
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn-ghost" style={{ fontSize: '0.82rem', padding: '8px 16px' }} onClick={() => load(true)} disabled={refreshing}>
              {refreshing ? <span className="spin-dark" /> : 'Regenerate'}
            </button>
            <button className="btn-ghost" style={{ fontSize: '0.82rem', padding: '8px 16px' }} onClick={() => window.print()}>
              Print
            </button>
          </div>
        </div>

        {/* Client header */}
        <div className="fade-up" style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 28 }}>
          <div style={styles.avatar}>{brief.client_name.charAt(0).toUpperCase()}</div>
          <div>
            <h1 className="serif" style={{ margin: 0, fontSize: '1.8rem', fontWeight: 600 }}>{brief.client_name}</h1>
            <p style={{ margin: '2px 0 0', color: 'var(--muted)', fontSize: '0.82rem' }}>
              {brief.entry_count} total entries · brief generated {format(parseISO(brief.generated_at), 'MMM d, h:mm a')}
            </p>
          </div>
        </div>

        {/* Pre-session brief */}
        <div className="card fade-up" style={{ marginBottom: 20, borderLeft: '3px solid var(--brand)', animationDelay: '0.05s' }}>
          <div className="eyebrow" style={{ marginBottom: 12 }}>Pre-session brief</div>
          <p className="serif" style={{ margin: 0, fontSize: '1.15rem', lineHeight: 1.7, color: 'var(--text)', fontStyle: 'italic' }}>
            {brief.brief}
          </p>
        </div>

        {/* Stat cards */}
        <div style={styles.statGrid}>
          <div className="card" style={{ textAlign: 'center' }}>
            <div style={styles.statLabel}>7-day avg mood</div>
            {brief.avg_mood_7d != null ? (
              <>
                <div className="serif" style={{ fontSize: '2.2rem', fontWeight: 600, color: moodColor(brief.avg_mood_7d) }}>{brief.avg_mood_7d}</div>
                <div style={{ fontSize: '0.74rem', color: 'var(--muted)' }}>{moodLabel(brief.avg_mood_7d)}</div>
                <div style={styles.moodBarTrack}>
                  <div style={{ ...styles.moodBarFill, width: `${brief.avg_mood_7d}%`, background: moodColor(brief.avg_mood_7d) }} />
                </div>
              </>
            ) : (
              <div style={{ color: 'var(--muted)', fontSize: '1rem', padding: '12px 0' }}>—</div>
            )}
          </div>
          <div className="card" style={{ textAlign: 'center' }}>
            <div style={styles.statLabel}>Mood trend</div>
            <div style={{ fontSize: '1.6rem', margin: '6px 0 2px' }}><MoodTrendIcon trend={brief.mood_trend} /></div>
            <div style={{ fontSize: '0.78rem', color: 'var(--muted)', textTransform: 'capitalize' }}>{brief.mood_trend.replace('_', ' ')}</div>
          </div>
          <div className="card" style={{ textAlign: 'center' }}>
            <div style={styles.statLabel}>Top emotions</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, alignItems: 'center', marginTop: 6 }}>
              {brief.top_emotions.slice(0, 3).map(e => <span key={e} className="chip">{e}</span>)}
              {brief.top_emotions.length === 0 && <span style={{ color: 'var(--muted)', fontSize: '0.85rem' }}>—</span>}
            </div>
          </div>
        </div>

        {/* Mood chart */}
        {chartData.length >= 2 && (
          <div className="card" style={{ marginBottom: 20 }}>
            <div className="eyebrow" style={{ marginBottom: 14 }}>Mood across recent entries</div>
            <div style={{ width: '100%', height: 200 }}>
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData} margin={{ top: 6, right: 8, left: -22, bottom: 0 }}>
                  <defs>
                    <linearGradient id="moodFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="#788574" stopOpacity={0.22} />
                      <stop offset="100%" stopColor="#788574" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid stroke="rgba(42,44,43,0.06)" vertical={false} />
                  <XAxis dataKey="date" tick={{ fontSize: 11, fill: '#7E8280' }} axisLine={false} tickLine={false} />
                  <YAxis domain={[0, 100]} tick={{ fontSize: 11, fill: '#7E8280' }} axisLine={false} tickLine={false} />
                  <Tooltip
                    contentStyle={{ background: '#2A2C2B', border: 'none', borderRadius: 10, fontSize: 12 }}
                    labelStyle={{ color: '#FAF8F5', fontWeight: 600 }}
                    itemStyle={{ color: '#FAF8F5' }}
                  />
                  <Area type="monotone" dataKey="mood" stroke="#788574" strokeWidth={2.5} fill="url(#moodFill)" dot={{ r: 3, fill: '#FFFFFF', stroke: '#788574', strokeWidth: 2 }} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>
        )}

        {/* Recent entries */}
        {brief.recent_entries.length > 0 && (
          <>
            <div className="eyebrow" style={{ margin: '0 0 12px' }}>Recent entries</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {brief.recent_entries.map((entry, i) => (
                <div key={i} className="card" style={{ display: 'flex', gap: 14 }}>
                  <MoodBadge score={entry.mood_score} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: '0.74rem', color: 'var(--muted)', marginBottom: 4, fontWeight: 600 }}>
                      {format(parseISO(entry.date), 'EEEE, MMM d')}
                    </div>
                    <p style={{ margin: 0, fontSize: '0.86rem', lineHeight: 1.6, color: 'var(--text)' }}>{entry.summary}</p>
                    {entry.key_quote && (
                      <p className="serif" style={{ margin: '8px 0 0', fontSize: '0.95rem', color: 'var(--muted)', fontStyle: 'italic' }}>
                        “{entry.key_quote}”
                      </p>
                    )}
                    {entry.topics.length > 0 && (
                      <div style={{ marginTop: 8, display: 'flex', flexWrap: 'wrap', gap: 5 }}>
                        {entry.topics.map(t => <span key={t} className="chip-muted">{t}</span>)}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </>
        )}

        <p style={{ marginTop: 32, fontSize: '0.72rem', color: 'var(--muted)', lineHeight: 1.7 }}>
          This brief is generated from AI-summarised journal entries only. Raw transcripts, voice recordings,
          and crisis entries are never included. The client consented to share this data with you via the DreamLog app.
        </p>
      </div>
    </div>
  );
}

function moodColor(score: number): string {
  if (score >= 71) return '#5A9367';
  if (score >= 46) return '#B08A3E';
  if (score >= 26) return '#C0703D';
  return '#C05B4D';
}

const styles: Record<string, React.CSSProperties> = {
  avatar: {
    width: 54, height: 54, borderRadius: '50%', background: 'var(--brand)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    fontSize: 22, color: '#FAF8F5', fontWeight: 700, flexShrink: 0,
    fontFamily: "'Cormorant Garamond', serif",
  },
  statGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(170px, 1fr))', gap: 12, marginBottom: 20 },
  statLabel: { fontSize: '0.66rem', color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '1px', fontWeight: 700, marginBottom: 6 },
  moodBarTrack: { height: 5, background: 'rgba(42,44,43,0.07)', borderRadius: 3, overflow: 'hidden', marginTop: 10 },
  moodBarFill: { height: '100%', borderRadius: 3, transition: 'width 0.6s ease' },
};
