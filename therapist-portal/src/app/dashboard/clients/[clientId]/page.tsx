'use client';

import { useEffect, useState } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { api, getToken, type ClientBrief } from '../../../../lib/api';
import MoodBadge from '../../../../components/MoodBadge';
import MoodTrendIcon from '../../../../components/MoodTrendIcon';
import { format, parseISO } from 'date-fns';

export default function ClientBriefPage() {
  const router = useRouter();
  const { clientId } = useParams<{ clientId: string }>();
  const [brief, setBrief] = useState<ClientBrief | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!getToken()) { router.replace('/login'); return; }
    api.getClientBrief(clientId)
      .then(setBrief)
      .catch(() => setError('Could not load client brief. Try again.'))
      .finally(() => setLoading(false));
  }, [clientId, router]);

  if (loading) {
    return (
      <div style={{ maxWidth: 760, margin: '0 auto', padding: '60px 24px', textAlign: 'center', color: 'var(--muted)' }}>
        Generating pre-session brief…
      </div>
    );
  }

  if (error || !brief) {
    return (
      <div style={{ maxWidth: 760, margin: '0 auto', padding: '60px 24px', textAlign: 'center' }}>
        <p style={{ color: '#f87171' }}>{error || 'Client not found.'}</p>
        <button className="btn-ghost" onClick={() => router.push('/dashboard')} style={{ marginTop: 16 }}>
          ← Back
        </button>
      </div>
    );
  }

  const moodLabel = (score: number) =>
    score >= 71 ? 'High' : score >= 46 ? 'Moderate' : score >= 26 ? 'Low' : 'Very Low';

  return (
    <div style={{ maxWidth: 760, margin: '0 auto', padding: '32px 24px' }}>

      {/* Back nav */}
      <button
        className="btn-ghost"
        style={{ marginBottom: 24, fontSize: 13 }}
        onClick={() => router.push('/dashboard')}
      >
        ← All clients
      </button>

      {/* Client header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 28 }}>
        <div style={{
          width: 54, height: 54, borderRadius: '50%',
          background: 'var(--brand)', display: 'flex',
          alignItems: 'center', justifyContent: 'center',
          fontSize: 22, color: 'white', fontWeight: 700, flexShrink: 0,
        }}>
          {brief.client_name.charAt(0).toUpperCase()}
        </div>
        <div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>{brief.client_name}</h1>
          <p style={{ margin: '2px 0 0', color: 'var(--muted)', fontSize: 13 }}>
            {brief.entry_count} total entries ·{' '}
            Generated {format(parseISO(brief.generated_at), 'MMM d, h:mm a')}
          </p>
        </div>
      </div>

      {/* Pre-session brief card */}
      <div className="card" style={{ marginBottom: 20, borderColor: 'var(--brand)' }}>
        <div style={{ fontSize: 10, letterSpacing: 1.2, color: 'var(--brand)', fontWeight: 700, marginBottom: 12 }}>
          PRE-SESSION BRIEF
        </div>
        <p style={{ margin: 0, fontSize: 15, lineHeight: 1.75, color: 'var(--text)' }}>
          {brief.brief}
        </p>
      </div>

      {/* Stats row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginBottom: 20 }}>
        <div className="card" style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 10, color: 'var(--muted)', marginBottom: 8 }}>7-DAY AVG MOOD</div>
          {brief.avg_mood_7d != null ? (
            <>
              <div style={{ fontSize: 28, fontWeight: 700, color: moodColor(brief.avg_mood_7d) }}>
                {brief.avg_mood_7d}
              </div>
              <div style={{ fontSize: 11, color: 'var(--muted)' }}>{moodLabel(brief.avg_mood_7d)}</div>
            </>
          ) : (
            <div style={{ color: 'var(--muted)', fontSize: 14 }}>—</div>
          )}
        </div>
        <div className="card" style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 10, color: 'var(--muted)', marginBottom: 8 }}>MOOD TREND</div>
          <div style={{ fontSize: 22 }}>
            <MoodTrendIcon trend={brief.mood_trend} />
          </div>
          <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 4 }}>
            {brief.mood_trend.replace('_', ' ')}
          </div>
        </div>
        <div className="card" style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 10, color: 'var(--muted)', marginBottom: 8 }}>TOP EMOTIONS</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {brief.top_emotions.slice(0, 3).map(e => (
              <span key={e} style={{ fontSize: 11, color: 'var(--text)', textTransform: 'capitalize' }}>{e}</span>
            ))}
            {brief.top_emotions.length === 0 && (
              <span style={{ color: 'var(--muted)', fontSize: 12 }}>—</span>
            )}
          </div>
        </div>
      </div>

      {/* Recent entries */}
      {brief.recent_entries.length > 0 && (
        <>
          <h2 style={{ fontSize: 13, fontWeight: 700, color: 'var(--muted)', letterSpacing: 1, margin: '0 0 12px' }}>
            RECENT ENTRIES
          </h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {brief.recent_entries.map((entry, i) => (
              <div key={i} className="card" style={{ display: 'flex', gap: 14 }}>
                <MoodBadge score={entry.mood_score} />
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 11, color: 'var(--muted)', marginBottom: 4 }}>
                    {format(parseISO(entry.date), 'EEEE, MMM d')}
                  </div>
                  <p style={{ margin: 0, fontSize: 13, lineHeight: 1.6, color: 'var(--text)' }}>
                    {entry.summary}
                  </p>
                  {entry.key_quote && (
                    <p style={{ margin: '8px 0 0', fontSize: 12, color: 'var(--muted)', fontStyle: 'italic' }}>
                      "{entry.key_quote}"
                    </p>
                  )}
                  {entry.topics.length > 0 && (
                    <div style={{ marginTop: 6, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                      {entry.topics.map(t => (
                        <span key={t} style={{
                          fontSize: 10, background: 'var(--border)', color: 'var(--muted)',
                          borderRadius: 4, padding: '2px 8px',
                        }}>
                          {t}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      <p style={{ marginTop: 32, fontSize: 11, color: 'var(--muted)', lineHeight: 1.7 }}>
        This brief is generated from AI-summarised journal entries only. Raw transcripts, voice recordings,
        and crisis entries are never included. The client consented to share this data with you via the DreamLog app.
      </p>
    </div>
  );
}

function moodColor(score: number): string {
  if (score >= 71) return '#4ade80';
  if (score >= 46) return '#facc15';
  if (score >= 26) return '#fb923c';
  return '#f87171';
}
