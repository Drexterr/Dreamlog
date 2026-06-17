'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { api, getToken, type ClientBrief } from '../../../../lib/api';
import PortalSidebar from '../../../../components/PortalSidebar';
import { format, parseISO } from 'date-fns';
import {
  ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid,
} from 'recharts';

function moodColor(score: number): string {
  if (score >= 70) return 'var(--success)';
  if (score >= 45) return 'var(--gold)';
  return 'var(--danger)';
}

function moodLabel(score: number): string {
  if (score >= 71) return 'High';
  if (score >= 46) return 'Moderate';
  if (score >= 26) return 'Low';
  return 'Very low';
}

function TrendIcon({ trend }: { trend: string }) {
  if (trend === 'improving') return <span style={{ color: 'var(--success)', fontSize: '1.1rem' }}>↗</span>;
  if (trend === 'declining') return <span style={{ color: 'var(--danger)', fontSize: '1.1rem' }}>↘</span>;
  if (trend === 'stable')    return <span style={{ color: 'var(--muted)', fontSize: '1.1rem' }}>→</span>;
  return <span style={{ color: 'var(--muted-2)', fontSize: '0.8rem' }}>—</span>;
}

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
      .catch(() => setError('Could not load client brief.'))
      .finally(() => { setLoading(false); setRefreshing(false); });
  }, [clientId]);

  useEffect(() => {
    if (!getToken()) { router.replace('/login'); return; }
    if (!clientId) { router.replace('/dashboard'); return; }
    load();
  }, [load, router, clientId]);

  if (loading) {
    return (
      <div className="portal-layout">
        <PortalSidebar />
        <main className="portal-main" style={{ padding: '40px 40px 64px' }}>
          <div className="skeleton" style={{ height: 16, width: 100, marginBottom: 28, borderRadius: 8 }} />
          <div style={{ display: 'flex', gap: 16, alignItems: 'center', marginBottom: 32 }}>
            <div className="skeleton" style={{ width: 52, height: 52, borderRadius: '50%' }} />
            <div>
              <div className="skeleton" style={{ height: 24, width: 180, marginBottom: 8, borderRadius: 6 }} />
              <div className="skeleton" style={{ height: 14, width: 260, borderRadius: 6 }} />
            </div>
          </div>
          <div className="skeleton" style={{ height: 120, borderRadius: 16, marginBottom: 16 }} />
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginBottom: 16 }}>
            {[0,1,2].map(i => <div key={i} className="skeleton" style={{ height: 110, borderRadius: 16 }} />)}
          </div>
          <div className="skeleton" style={{ height: 200, borderRadius: 16 }} />
          <p style={{ textAlign: 'center', color: 'var(--muted)', fontSize: '0.82rem', marginTop: 24 }}>
            Generating pre-session brief…
          </p>
        </main>
      </div>
    );
  }

  if (error || !brief) {
    return (
      <div className="portal-layout">
        <PortalSidebar />
        <main className="portal-main" style={{ padding: '60px 40px', textAlign: 'center' }}>
          <p style={{ color: 'var(--danger)', marginBottom: 16 }}>{error || 'Client not found.'}</p>
          <div style={{ display: 'flex', gap: 10, justifyContent: 'center' }}>
            <button className="btn-ghost" onClick={() => router.push('/dashboard')}>← Back</button>
            <button className="btn-primary" onClick={() => load()} style={{ borderRadius: 12 }}>Retry</button>
          </div>
        </main>
      </div>
    );
  }

  const chartData = [...brief.recent_entries]
    .sort((a, b) => a.date.localeCompare(b.date))
    .map(e => ({
      date: format(parseISO(e.date), 'MMM d'),
      mood: e.mood_score,
    }));

  return (
    <div className="portal-layout">
      <PortalSidebar />

      <main className="portal-main" style={{ minHeight: '100vh' }}>
        {/* Top nav */}
        <div className="no-print" style={{
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          padding: '14px 32px', borderBottom: '1px solid var(--border)',
          position: 'sticky', top: 0, background: 'var(--bg)', zIndex: 10,
        }}>
          <button
            className="btn-ghost"
            style={{ fontSize: '0.82rem', padding: '7px 14px' }}
            onClick={() => router.push('/dashboard')}
          >
            ← All clients
          </button>
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              className="btn-ghost"
              style={{ fontSize: '0.82rem', padding: '7px 14px' }}
              onClick={() => load(true)}
              disabled={refreshing}
            >
              {refreshing ? <span className="spin-gold" /> : 'Regenerate'}
            </button>
            <button
              className="btn-ghost"
              style={{ fontSize: '0.82rem', padding: '7px 14px' }}
              onClick={() => window.print()}
            >
              Print
            </button>
          </div>
        </div>

        <div style={{ padding: '36px 40px 64px', maxWidth: 820 }}>
          {/* Client header */}
          <div className="fade-up" style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 32 }}>
            <div style={{
              width: 52, height: 52, borderRadius: '50%',
              background: 'var(--gold-light)', border: '1px solid rgba(201,169,110,0.2)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 22, color: 'var(--gold)', fontWeight: 700, flexShrink: 0,
              fontFamily: "'Cormorant Garamond', serif",
            }}>
              {brief.client_name.charAt(0).toUpperCase()}
            </div>
            <div>
              <h1 className="serif" style={{ margin: 0, fontSize: '1.9rem', fontWeight: 300, color: 'var(--text)' }}>
                {brief.client_name}
              </h1>
              <p style={{ margin: '3px 0 0', color: 'var(--muted)', fontSize: '0.8rem' }}>
                {brief.entry_count} total entries · brief generated {format(parseISO(brief.generated_at), 'MMM d, h:mm a')}
              </p>
            </div>
          </div>

          {/* Pre-session brief */}
          <div className="card fade-up" style={{
            marginBottom: 16, borderLeft: '3px solid var(--gold)',
            background: 'rgba(201,169,110,0.04)', animationDelay: '0.05s',
          }}>
            <div className="eyebrow-gold" style={{ marginBottom: 12 }}>PRE-SESSION BRIEF</div>
            <p className="serif" style={{
              margin: 0, fontSize: '1.1rem', lineHeight: 1.75,
              color: 'var(--text)', fontStyle: 'italic',
            }}>
              {brief.brief}
            </p>
          </div>

          {/* Stat cards */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginBottom: 16 }}>
            {/* 7-day mood */}
            <div className="card" style={{ textAlign: 'center' }}>
              <div className="eyebrow" style={{ marginBottom: 10 }}>7-DAY AVG MOOD</div>
              {brief.avg_mood_7d != null ? (
                <>
                  <div className="serif" style={{
                    fontSize: '2.6rem', fontWeight: 300, lineHeight: 1,
                    color: moodColor(brief.avg_mood_7d), marginBottom: 4,
                  }}>{brief.avg_mood_7d}</div>
                  <div style={{ fontSize: '0.74rem', color: 'var(--muted)', marginBottom: 12 }}>{moodLabel(brief.avg_mood_7d)}</div>
                  <div style={{ height: 4, background: 'var(--bg-card-2)', borderRadius: 2, overflow: 'hidden' }}>
                    <div style={{
                      height: '100%', width: `${brief.avg_mood_7d}%`,
                      background: moodColor(brief.avg_mood_7d), borderRadius: 2,
                      transition: 'width 0.8s ease',
                    }} />
                  </div>
                </>
              ) : (
                <div style={{ color: 'var(--muted)', padding: '16px 0', fontSize: '0.9rem' }}>No data</div>
              )}
            </div>

            {/* Trend */}
            <div className="card" style={{ textAlign: 'center' }}>
              <div className="eyebrow" style={{ marginBottom: 10 }}>MOOD TREND</div>
              <div style={{ fontSize: '2rem', marginBottom: 4 }}>
                <TrendIcon trend={brief.mood_trend} />
              </div>
              <div style={{ fontSize: '0.8rem', color: 'var(--muted)', textTransform: 'capitalize' }}>
                {brief.mood_trend.replace('_', ' ')}
              </div>
            </div>

            {/* Top emotions */}
            <div className="card">
              <div className="eyebrow" style={{ marginBottom: 10 }}>TOP EMOTIONS</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {brief.top_emotions.slice(0, 3).map(e => (
                  <span key={e} className="chip" style={{ alignSelf: 'flex-start' }}>{e}</span>
                ))}
                {brief.top_emotions.length === 0 && (
                  <span style={{ color: 'var(--muted)', fontSize: '0.85rem' }}>—</span>
                )}
              </div>
            </div>
          </div>

          {/* Mood chart */}
          {chartData.length >= 2 && (
            <div className="card" style={{ marginBottom: 16 }}>
              <div className="eyebrow" style={{ marginBottom: 16 }}>MOOD ACROSS RECENT ENTRIES</div>
              <div style={{ width: '100%', height: 200 }}>
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData} margin={{ top: 6, right: 8, left: -22, bottom: 0 }}>
                    <defs>
                      <linearGradient id="moodFill" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="#C9A96E" stopOpacity={0.2} />
                        <stop offset="100%" stopColor="#C9A96E" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid stroke="rgba(255,255,255,0.04)" vertical={false} />
                    <XAxis dataKey="date" tick={{ fontSize: 11, fill: '#968E87' }} axisLine={false} tickLine={false} />
                    <YAxis domain={[0, 100]} tick={{ fontSize: 11, fill: '#968E87' }} axisLine={false} tickLine={false} />
                    <Tooltip
                      contentStyle={{ background: '#1c1a17', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 10, fontSize: 12 }}
                      labelStyle={{ color: '#EDE7DD', fontWeight: 600 }}
                      itemStyle={{ color: '#C9A96E' }}
                    />
                    <Area
                      type="monotone" dataKey="mood"
                      stroke="#C9A96E" strokeWidth={2}
                      fill="url(#moodFill)"
                      dot={{ r: 3, fill: '#C9A96E', stroke: '#C9A96E', strokeWidth: 1 }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>
          )}

          {/* Recent entries */}
          {brief.recent_entries.length > 0 && (
            <>
              <div className="eyebrow" style={{ margin: '24px 0 12px' }}>RECENT ENTRIES</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                {brief.recent_entries.map((entry, i) => (
                  <div key={i} className="card" style={{ display: 'flex', gap: 16 }}>
                    {/* Mood circle */}
                    <div style={{
                      width: 44, height: 44, borderRadius: '50%', flexShrink: 0,
                      border: `2px solid ${moodColor(entry.mood_score)}`,
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      background: 'var(--bg)',
                    }}>
                      <span className="serif" style={{ fontSize: '1rem', fontWeight: 300, color: moodColor(entry.mood_score) }}>
                        {entry.mood_score}
                      </span>
                    </div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: '0.72rem', color: 'var(--muted)', marginBottom: 5, fontWeight: 600, letterSpacing: '0.3px' }}>
                        {format(parseISO(entry.date), 'EEEE, MMMM d')}
                      </div>
                      <p style={{ margin: 0, fontSize: '0.88rem', lineHeight: 1.65, color: 'var(--text)' }}>{entry.summary}</p>
                      {entry.key_quote && (
                        <p className="serif" style={{ margin: '10px 0 0', fontSize: '0.95rem', color: 'var(--muted)', fontStyle: 'italic', lineHeight: 1.5 }}>
                          &ldquo;{entry.key_quote}&rdquo;
                        </p>
                      )}
                      {entry.topics.length > 0 && (
                        <div style={{ marginTop: 10, display: 'flex', flexWrap: 'wrap', gap: 5 }}>
                          {entry.topics.map(t => <span key={t} className="chip-muted">{t}</span>)}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}

          <p style={{ marginTop: 36, fontSize: '0.72rem', color: 'var(--muted-2)', lineHeight: 1.8 }}>
            This brief is generated from AI-summarised journal entries only. Raw transcripts, voice recordings, and crisis entries
            are never included. The client consented to share this data with you via the DreamLog app. Your access is
            consent-gated and revocable by each client at any time.
          </p>
        </div>
      </main>
    </div>
  );
}
