import Link from 'next/link';

export default function TherapistsPage() {
  return (
    <div style={{ background: '#18150f', color: '#e8ddd0', minHeight: '100vh', fontFamily: "'Nunito', sans-serif" }}>
      <nav style={{ maxWidth: 1100, margin: '0 auto', padding: '24px 40px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Link href="/" style={{ display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none', color: '#e8ddd0' }}>
          <div style={{ width: 28, height: 28, borderRadius: 7, background: '#c8955a', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1rem', color: '#18150f' }}>D</div>
          <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.1rem', fontWeight: 600 }}>DreamLog</span>
        </Link>
        <Link href="/login" style={{ background: '#c8955a', color: '#18150f', borderRadius: 100, padding: '9px 20px', fontSize: '0.84rem', fontWeight: 700, textDecoration: 'none' }}>
          Open Therapist Portal
        </Link>
      </nav>

      <main style={{ maxWidth: 1100, margin: '0 auto', padding: '60px 40px 120px' }}>

        {/* Hero */}
        <div style={{ maxWidth: 720, marginBottom: 80 }}>
          <div style={{ fontSize: '0.7rem', letterSpacing: '2.5px', color: 'rgba(232,221,208,0.3)', textTransform: 'uppercase', marginBottom: 20 }}>For mental health professionals</div>
          <h1 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(2.6rem, 5.5vw, 4.2rem)', fontWeight: 300, margin: '0 0 24px', lineHeight: 1.1, letterSpacing: '-0.02em' }}>
            Your clients arrive<br />already knowing<br /><em style={{ color: '#c8955a' }}>what they feel.</em>
          </h1>
          <p style={{ fontSize: '1rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: '0 0 36px', maxWidth: 560 }}>
            DreamLog is a voice journaling app your clients use between sessions. When they choose to share, you receive a structured AI brief — mood trends, recurring themes, and the questions they couldn&apos;t put into words — before you even say hello.
          </p>
          <div style={{ display: 'flex', gap: 14, flexWrap: 'wrap' }}>
            <Link href="/login" style={{ background: '#c8955a', color: '#18150f', borderRadius: 12, padding: '13px 26px', fontWeight: 700, fontSize: '0.9rem', textDecoration: 'none' }}>
              Register as a therapist — free
            </Link>
            <a href="mailto:support@dreamlog.app" style={{ border: '1px solid rgba(255,255,255,0.15)', color: '#e8ddd0', borderRadius: 12, padding: '13px 26px', fontSize: '0.9rem', textDecoration: 'none' }}>
              Talk to us →
            </a>
          </div>
          <p style={{ margin: '14px 0 0', fontSize: '0.78rem', color: 'rgba(232,221,208,0.3)' }}>No cost to register. No contract. No minimum client count.</p>
        </div>

        {/* How it works */}
        <div style={{ borderTop: '1px solid rgba(255,255,255,0.07)', paddingTop: 64, marginBottom: 80 }}>
          <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(1.6rem, 3vw, 2.4rem)', fontWeight: 300, margin: '0 0 48px', lineHeight: 1.2 }}>
            How it works between sessions
          </h2>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 40 }}>
            {[
              {
                step: '01',
                title: 'Client journals daily',
                body: 'Your client speaks for 2 minutes or 20 — whatever arrives. DreamLog transcribes, analyses, and stores a structured reflection after every entry. No effort from you.',
              },
              {
                step: '02',
                title: 'Client generates a share link',
                body: 'From Settings, they create a passcode-protected link that expires in 72 hours. They share the link and passcode directly with you — you see nothing without their explicit action.',
              },
              {
                step: '03',
                title: 'You open the portal',
                body: 'Your therapist dashboard shows an AI-generated pre-session brief: 30-day mood trend, top emotions, recurring themes, and 5 recent entry summaries. In under 60 seconds.',
              },
            ].map(s => (
              <div key={s.step}>
                <div style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '0.7rem', color: 'rgba(200,149,90,0.5)', letterSpacing: 2, marginBottom: 20 }}>{s.step}</div>
                <div style={{ fontWeight: 700, fontSize: '0.95rem', color: '#e8ddd0', marginBottom: 10 }}>{s.title}</div>
                <p style={{ fontSize: '0.87rem', color: 'rgba(232,221,208,0.5)', lineHeight: 1.8, margin: 0 }}>{s.body}</p>
              </div>
            ))}
          </div>
        </div>

        {/* What you see / don't see */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, marginBottom: 80 }}>
          <div style={{ background: 'rgba(90,147,103,0.06)', border: '1px solid rgba(90,147,103,0.2)', borderRadius: 16, padding: '32px 28px' }}>
            <div style={{ fontSize: '0.68rem', letterSpacing: '2px', color: 'rgba(90,147,103,0.7)', textTransform: 'uppercase', marginBottom: 20 }}>What you see</div>
            {[
              '30-day average mood score and trend direction',
              'Top 3 recurring emotions (frequency + intensity)',
              'Recurring topics and themes across entries',
              'AI-generated 3-sentence pre-session brief',
              '5 recent entry summaries (AI-written, not raw transcripts)',
              'Total entries in the period',
            ].map(item => (
              <div key={item} style={{ display: 'flex', gap: 12, alignItems: 'flex-start', marginBottom: 12 }}>
                <span style={{ color: '#5a9367', flexShrink: 0, marginTop: 2, fontSize: '0.8rem' }}>✓</span>
                <span style={{ fontSize: '0.85rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.5 }}>{item}</span>
              </div>
            ))}
          </div>
          <div style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.07)', borderRadius: 16, padding: '32px 28px' }}>
            <div style={{ fontSize: '0.68rem', letterSpacing: '2px', color: 'rgba(232,221,208,0.3)', textTransform: 'uppercase', marginBottom: 20 }}>What you never see</div>
            {[
              'Raw audio recordings — deleted after transcription',
              'Full transcripts of any session',
              'Therapy Mode conversation messages',
              'Entries the client has not chosen to share',
              'Any data without an active, client-generated share link',
              'Anything after a share link expires (72 hours)',
            ].map(item => (
              <div key={item} style={{ display: 'flex', gap: 12, alignItems: 'flex-start', marginBottom: 12 }}>
                <span style={{ color: 'rgba(232,221,208,0.2)', flexShrink: 0, marginTop: 2, fontSize: '0.8rem' }}>✕</span>
                <span style={{ fontSize: '0.85rem', color: 'rgba(232,221,208,0.35)', lineHeight: 1.5 }}>{item}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Beta invite */}
        <div style={{ background: 'rgba(200,149,90,0.05)', border: '1px solid rgba(200,149,90,0.18)', borderRadius: 20, padding: '48px 52px', marginBottom: 80 }}>
          <div style={{ fontSize: '0.7rem', letterSpacing: '2.5px', color: 'rgba(200,149,90,0.5)', textTransform: 'uppercase', marginBottom: 16 }}>Early access · Beta</div>
          <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(1.6rem, 3vw, 2.4rem)', fontWeight: 300, margin: '0 0 16px', lineHeight: 1.2 }}>
            We&apos;re inviting therapists to test DreamLog with their clients.
          </h2>
          <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.8, margin: '0 0 28px', maxWidth: 560 }}>
            If you see potential in voice journaling as a between-session tool, we&apos;d like to partner with you. Early access is free — and your feedback directly shapes what gets built next.
          </p>
          <div style={{ display: 'flex', gap: 14, flexWrap: 'wrap' }}>
            <Link href="/login" style={{ background: '#c8955a', color: '#18150f', borderRadius: 12, padding: '13px 26px', fontWeight: 700, fontSize: '0.9rem', textDecoration: 'none' }}>
              Register your practice
            </Link>
            <a href="mailto:support@dreamlog.app?subject=Therapist Beta Interest" style={{ border: '1px solid rgba(200,149,90,0.25)', color: '#c8955a', borderRadius: 12, padding: '13px 26px', fontSize: '0.9rem', textDecoration: 'none' }}>
              Email us first →
            </a>
          </div>
        </div>

        {/* FAQ */}
        <div style={{ borderTop: '1px solid rgba(255,255,255,0.07)', paddingTop: 56 }}>
          <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(1.4rem, 2.5vw, 2rem)', fontWeight: 300, margin: '0 0 36px' }}>Questions therapists ask</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
            {[
              { q: 'Is it HIPAA-compliant?', a: 'DreamLog uses encrypted transit (HTTPS) and encrypted storage. Audio is never retained. We are building toward full HIPAA compliance — reach us at support@dreamlog.app for a current data-processing assessment.' },
              { q: 'Does it claim to be therapy?', a: 'No. Every session and Therapy Mode interaction includes a clear disclaimer. DreamLog is an AI-assisted reflection tool, not a licensed therapeutic service. We are explicit about this in the product and in our Terms.' },
              { q: 'What happens if a client discloses crisis content?', a: 'Two-stage automated detection runs on every journal entry and every Therapy Mode message. If crisis content is flagged, the user immediately sees hotline resources: iCall, Vandrevala Foundation, and 988. Entries flagged as crisis are excluded from mood analytics and handled separately. You are not a first responder in this system — the system handles it.' },
              { q: 'Can I recommend this to all my clients?', a: 'DreamLog is suitable for adults who want a private journaling tool. It is not designed for clients in acute psychiatric crisis, active psychosis, or who require clinical monitoring. Use your professional judgement — the same way you would recommend a book or a journaling prompt.' },
              { q: 'Is the therapist portal free?', a: 'Yes. Registering as a therapist and accessing client briefs costs nothing. Clients pay for their own DreamLog plan; they can share with any registered therapist regardless of their plan tier.' },
            ].map((item, i) => (
              <div key={item.q} style={{ padding: '22px 0', borderBottom: '1px solid rgba(255,255,255,0.06)', borderTop: i === 0 ? '1px solid rgba(255,255,255,0.06)' : 'none' }}>
                <div style={{ fontSize: '0.95rem', fontWeight: 600, color: '#e8ddd0', marginBottom: 10 }}>{item.q}</div>
                <p style={{ fontSize: '0.87rem', color: 'rgba(232,221,208,0.5)', margin: 0, lineHeight: 1.75 }}>{item.a}</p>
              </div>
            ))}
          </div>
        </div>
      </main>

      <footer style={{ borderTop: '1px solid rgba(255,255,255,0.07)', padding: '28px 40px', maxWidth: 1100, margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 16 }}>
        <span style={{ fontSize: '0.74rem', color: 'rgba(232,221,208,0.3)' }}>© 2026 DreamLog</span>
        <div style={{ display: 'flex', gap: 24 }}>
          {[['Home', '/'], ['Pricing', '/pricing'], ['About', '/about'], ['Teams', '/teams'], ['Privacy', '/privacy'], ['Terms', '/terms']].map(([label, href]) => (
            <a key={label} href={href} style={{ fontSize: '0.8rem', color: 'rgba(232,221,208,0.4)', textDecoration: 'none' }}>{label}</a>
          ))}
        </div>
      </footer>

      <style>{`
        @media (max-width: 640px) {
          main { padding: 40px 20px 80px !important; }
          nav { padding: 20px !important; }
          div[style*="grid-template-columns: repeat(3, 1fr)"] { grid-template-columns: 1fr !important; gap: 32px !important; }
          div[style*="grid-template-columns: 1fr 1fr"] { grid-template-columns: 1fr !important; }
        }
      `}</style>
    </div>
  );
}
