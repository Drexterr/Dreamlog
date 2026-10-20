'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api, getToken, type ClientBrief } from '../../../../lib/api';
import PortalShell from '../../../../components/PortalShell';
export default function ClientBriefContent() {
  const router = useRouter();
  const [id, setId] = useState(''),
    [brief, setBrief] = useState<ClientBrief | null>(null),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(''),
    [unlinking, setUnlinking] = useState(false);
  const generation = useRef(0),
    lock = useRef(false);
  useEffect(() => {
    if (!getToken()) {
      router.replace('/login');
      return;
    }
    const value = window.location.pathname.split('/').filter(Boolean)[2];
    if (!value || value === '_') {
      router.replace('/dashboard');
      return;
    }
    setId(decodeURIComponent(value));
  }, [router]);
  const load = useCallback(async () => {
    if (!id || lock.current) return;
    lock.current = true;
    const run = ++generation.current;
    setLoading(true);
    setError('');
    try {
      const result = await api.getClientBrief(id);
      if (generation.current === run) setBrief(result);
    } catch {
      if (generation.current === run) setError('Could not load this shared client brief. Access may have expired or been revoked.');
    } finally {
      lock.current = false;
      if (generation.current === run) setLoading(false);
    }
  }, [id]);
  useEffect(() => {
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  async function unlink() {
    if (unlinking || !confirm('Unlink this client? Their journal will not be deleted.')) return;
    setUnlinking(true);
    try {
      await api.unlinkClient(id);
      router.push('/dashboard');
    } catch {
      setError('Could not unlink this client. Please try again.');
    } finally {
      setUnlinking(false);
    }
  }
  if (loading && !brief) return <PortalShell title="Client brief"><p className="empty" role="status">Preparing the shared client brief…</p></PortalShell>;
  if (!brief) return <PortalShell title="Client brief"><p className="error" role="alert">{error}</p><div className="actions"><a href="/dashboard">← All clients</a><button className="btn" onClick={() => void load()}>Try again</button></div></PortalShell>;
  const entries = [...brief.recent_entries].sort((a, b) => a.date.localeCompare(b.date));
  const points = entries.map((e, i) => `${20 + (entries.length > 1 ? i / (entries.length - 1) : 0.5) * 520},${145 - Math.max(0, Math.min(100, e.mood_score)) * 1.2}`).join(' ');
  return <PortalShell title="Client brief"><div className="page-heading"><div><a className="small-note" href="/dashboard">← All clients</a><h1 style={{
          marginTop: 15
        }}>{brief.client_name}</h1><p>{brief.entry_count} shared entries · generated {new Date(brief.generated_at).toLocaleString()}</p></div><div className="actions no-print"><button className="text-btn" disabled={loading} onClick={() => void load()}>{loading ? 'Refreshing…' : 'Refresh brief'}</button><button className="btn outline small" onClick={() => window.print()}>Print brief</button></div></div>{error && <p className="error" role="alert">{error}</p>}<div className="brief-layout"><article><h2>Pre-session brief</h2><div className="brief-prose"><p>{brief.brief}</p></div>{entries.length > 0 && <><h2>Mood across recent entries</h2><p className="small-note quiet">Journal-derived scores, 0–100. These are not clinical assessments.</p><svg className="chart" viewBox="0 0 560 170" role="img" aria-label={`Recent journal mood scores: ${entries.map(e => Math.round(e.mood_score)).join(', ')}`}><path d="M20 25H540M20 85H540M20 145H540" stroke="#cbbfac" strokeWidth=".7" fill="none" /><polyline points={points} fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round" />{entries.map((e, i) => <circle key={i} cx={20 + (entries.length > 1 ? i / (entries.length - 1) : .5) * 520} cy={145 - Math.max(0, Math.min(100, e.mood_score)) * 1.2} r="3" fill="var(--accent)"><title>{new Date(e.date).toLocaleDateString()}: {e.mood_score}/100</title></circle>)}</svg><div className="chart-labels"><span>{new Date(entries[0].date).toLocaleDateString()}</span><span>{new Date(entries[entries.length - 1].date).toLocaleDateString()}</span></div></>}<h2 style={{
          marginTop: 40
        }}>Recent shared summaries</h2>{[...brief.recent_entries].map((e, i) => <div className="entry" key={`${e.date}-${i}`}><strong>{new Date(e.date).toLocaleDateString()}</strong><p>{e.summary}</p>{e.topics.length > 0 && <p className="small-note">{e.topics.join(' · ')}</p>}</div>)}{!brief.recent_entries.length && <p className="empty">No shared journal summaries yet.</p>}</article><aside className="brief-side"><dl className="meta-list"><div><dt>7-day average mood</dt><dd>{brief.avg_mood_7d == null ? 'Not available' : `${Math.round(brief.avg_mood_7d)} / 100`}</dd></div><div><dt>Mood trend</dt><dd>{brief.mood_trend === 'insufficient_data' ? 'Not enough data' : brief.mood_trend}</dd></div><div><dt>Recurring emotions</dt><dd>{brief.top_emotions.length ? brief.top_emotions.join(' · ') : 'Not available'}</dd></div></dl><div className="notice"><strong>Shared context only</strong><p>Full transcripts and recordings are excluded. Use your professional judgment when reading AI interpretations.</p></div><p className="small-note no-print" style={{
          marginTop: 26
        }}>Notes for your own consultations are organised separately in <a href="/dashboard/notes">Session notes →</a>.</p><button className="text-btn no-print" disabled={unlinking} onClick={() => void unlink()}>{unlinking ? 'Unlinking…' : 'Unlink client'}</button></aside></div></PortalShell>;
}
