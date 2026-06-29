import Link from 'next/link';

export default function TeamsPage() {
  return (
    <div style={{ background: '#18150f', color: '#e8ddd0', minHeight: '100vh', fontFamily: "'Nunito', sans-serif" }}>
      <nav style={{ maxWidth: 1100, margin: '0 auto', padding: '24px 40px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Link href="/" style={{ display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none', color: '#e8ddd0' }}>
          <div style={{ width: 28, height: 28, borderRadius: 7, background: '#c8955a', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1rem', color: '#18150f' }}>D</div>
          <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.1rem', fontWeight: 600 }}>DreamLog</span>
        </Link>
        <a href="mailto:support@dreamlog.app?subject=Team Wellness Enquiry" style={{ background: '#c8955a', color: '#18150f', borderRadius: 100, padding: '9px 20px', fontSize: '0.84rem', fontWeight: 700, textDecoration: 'none' }}>
          Talk to us →
        </a>
      </nav>

      <main style={{ maxWidth: 1100, margin: '0 auto', padding: '60px 40px 120px' }}>

        {/* Hero */}
        <div style={{ maxWidth: 760, marginBottom: 80 }}>
          <div style={{ fontSize: '0.7rem', letterSpacing: '2.5px', color: 'rgba(232,221,208,0.3)', textTransform: 'uppercase', marginBottom: 20 }}>Corporate wellness</div>
          <h1 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(2.6rem, 5.5vw, 4.2rem)', fontWeight: 300, margin: '0 0 24px', lineHeight: 1.1, letterSpacing: '-0.02em' }}>
            DreamLog for teams<br /><em style={{ color: '#c8955a' }}>who actually care.</em>
          </h1>
          <p style={{ fontSize: '1rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: '0 0 36px', maxWidth: 600 }}>
            Aggregate, anonymized emotional wellbeing insights across your team. Individual journals stay private — always. HR sees patterns, never people.
          </p>
          <div style={{ display: 'flex', gap: 14, flexWrap: 'wrap' }}>
            <a href="mailto:support@dreamlog.app?subject=Team Wellness Enquiry" style={{ background: '#c8955a', color: '#18150f', borderRadius: 12, padding: '13px 26px', fontWeight: 700, fontSize: '0.9rem', textDecoration: 'none' }}>
              Start a conversation
            </a>
            <Link href="/pricing" style={{ border: '1px solid rgba(255,255,255,0.15)', color: '#e8ddd0', borderRadius: 12, padding: '13px 26px', fontSize: '0.9rem', textDecoration: 'none' }}>
              See individual pricing
            </Link>
          </div>
          <p style={{ margin: '14px 0 0', fontSize: '0.78rem', color: 'rgba(232,221,208,0.3)' }}>₹199 per employee / month · minimum 50 employees · annual contract</p>
        </div>

        {/* The problem */}
        <div style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.07)', borderRadius: 16, padding: '40px 44px', marginBottom: 20 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 48, alignItems: 'center' }}>
            <div>
              <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(1.5rem, 2.5vw, 2rem)', fontWeight: 300, margin: '0 0 16px', lineHeight: 1.2 }}>The cost you&apos;re not measuring</h2>
              <p style={{ fontSize: '0.9rem', color: 'rgba(232,221,208,0.5)', lineHeight: 1.8, margin: 0 }}>
                Untreated mental health issues cost Indian companies an estimated ₹1.3 lakh per employee per year in lost productivity, absenteeism, and attrition. That number is invisible until it isn&apos;t.
              </p>
              <p style={{ fontSize: '0.9rem', color: 'rgba(232,221,208,0.5)', lineHeight: 1.8, margin: '16px 0 0' }}>
                India&apos;s 2024 Occupational Safety and Health Code makes mental health support mandatory for specific employee categories. DreamLog helps you comply — and actually help.
              </p>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {[
                { stat: '₹1.3L', label: 'Annual productivity loss per employee with untreated stress (WHO, 2024)' },
                { stat: '67%', label: 'Indian employers who increased digital wellness budgets in 2024 (APA)' },
                { stat: '₹199', label: 'What DreamLog costs per employee per month' },
              ].map(item => (
                <div key={item.stat} style={{ display: 'flex', gap: 20, alignItems: 'flex-start', paddingBottom: 16, borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
                  <div style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.8rem', fontWeight: 300, color: '#c8955a', lineHeight: 1, flexShrink: 0, minWidth: 72 }}>{item.stat}</div>
                  <p style={{ fontSize: '0.82rem', color: 'rgba(232,221,208,0.45)', margin: 0, lineHeight: 1.6 }}>{item.label}</p>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* What HR sees */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, marginBottom: 80 }}>
          <div style={{ background: 'rgba(200,149,90,0.04)', border: '1px solid rgba(200,149,90,0.15)', borderRadius: 16, padding: '32px 28px' }}>
            <h3 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.3rem', fontWeight: 300, margin: '0 0 20px', fontStyle: 'italic' }}>What HR sees</h3>
            {[
              'Weekly aggregate mood trend (team-level, anonymized)',
              'Top 3 emotional themes across the team this week',
              'Percentage of employees actively journaling',
              'Alert when team mood drops below a configurable threshold',
              'Monthly wellness report for leadership',
            ].map(item => (
              <div key={item} style={{ display: 'flex', gap: 12, marginBottom: 12, alignItems: 'flex-start' }}>
                <span style={{ color: '#c8955a', flexShrink: 0, marginTop: 2, fontSize: '0.8rem' }}>✓</span>
                <span style={{ fontSize: '0.85rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.5 }}>{item}</span>
              </div>
            ))}
          </div>
          <div style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.07)', borderRadius: 16, padding: '32px 28px' }}>
            <h3 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.3rem', fontWeight: 300, margin: '0 0 20px', fontStyle: 'italic' }}>What HR never sees</h3>
            {[
              'Any individual employee\'s journal entry or transcript',
              'Which employee said what',
              'Audio recordings (deleted after transcription)',
              'Mood scores tied to individual identities',
              'Any data the employee has not explicitly shared',
            ].map(item => (
              <div key={item} style={{ display: 'flex', gap: 12, marginBottom: 12, alignItems: 'flex-start' }}>
                <span style={{ color: 'rgba(232,221,208,0.2)', flexShrink: 0, marginTop: 2, fontSize: '0.8rem' }}>✕</span>
                <span style={{ fontSize: '0.85rem', color: 'rgba(232,221,208,0.35)', lineHeight: 1.5 }}>{item}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Features for teams */}
        <div style={{ borderTop: '1px solid rgba(255,255,255,0.07)', paddingTop: 64, marginBottom: 80 }}>
          <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(1.6rem, 3vw, 2.4rem)', fontWeight: 300, margin: '0 0 48px', lineHeight: 1.2 }}>
            What your employees get
          </h2>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 20 }}>
            {[
              { title: 'Daily voice journaling', body: 'Speak for 2 minutes. No typing, no prompts. An AI reflection is ready within minutes.' },
              { title: 'AI-powered reflection', body: 'Pattern recognition across entries. What keeps coming up. What they might not have noticed themselves.' },
              { title: 'Therapy Mode', body: 'A private AI companion conversation when they need to process something beyond journaling.' },
              { title: 'Morning nudge', body: 'A personalised daily push notification drawn from what they said the day before.' },
              { title: 'Crisis detection', body: 'Two-stage automated detection on every entry. Hotline resources appear immediately if distress is flagged.' },
              { title: 'Hindi + Hinglish', body: 'Full support for Hindi and Hinglish — auto-detected, no setting required.' },
            ].map(f => (
              <div key={f.title} style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.07)', borderRadius: 14, padding: '24px 22px' }}>
                <div style={{ fontWeight: 700, fontSize: '0.9rem', color: '#e8ddd0', marginBottom: 8 }}>{f.title}</div>
                <p style={{ fontSize: '0.83rem', color: 'rgba(232,221,208,0.5)', margin: 0, lineHeight: 1.7 }}>{f.body}</p>
              </div>
            ))}
          </div>
        </div>

        {/* CTA */}
        <div style={{ background: 'rgba(200,149,90,0.05)', border: '1px solid rgba(200,149,90,0.18)', borderRadius: 20, padding: '52px 56px', textAlign: 'center' }}>
          <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(1.8rem, 3.5vw, 2.8rem)', fontWeight: 300, margin: '0 0 16px', lineHeight: 1.2 }}>
            Start with a 30-day pilot.
          </h2>
          <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.5)', margin: '0 0 32px', lineHeight: 1.75, maxWidth: 480, marginLeft: 'auto', marginRight: 'auto' }}>
            We offer a free 30-day pilot for teams of 25–100 employees. No contract, no commitment. See the dashboard. Hear from your employees.
          </p>
          <a href="mailto:support@dreamlog.app?subject=Team Pilot Request" style={{ display: 'inline-block', background: '#c8955a', color: '#18150f', borderRadius: 12, padding: '14px 32px', fontWeight: 700, fontSize: '0.95rem', textDecoration: 'none' }}>
            Request a pilot →
          </a>
          <p style={{ margin: '16px 0 0', fontSize: '0.78rem', color: 'rgba(232,221,208,0.3)' }}>We&apos;ll get back to you within 48 hours.</p>
        </div>
      </main>

      <footer style={{ borderTop: '1px solid rgba(255,255,255,0.07)', padding: '28px 40px', maxWidth: 1100, margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 16 }}>
        <span style={{ fontSize: '0.74rem', color: 'rgba(232,221,208,0.3)' }}>© 2026 DreamLog · Built for honesty, not performance.</span>
        <div style={{ display: 'flex', gap: 24 }}>
          {[['Home', '/'], ['Pricing', '/pricing'], ['Therapists', '/therapists'], ['About', '/about'], ['Privacy', '/privacy'], ['Terms', '/terms']].map(([label, href]) => (
            <a key={label} href={href} style={{ fontSize: '0.8rem', color: 'rgba(232,221,208,0.4)', textDecoration: 'none' }}>{label}</a>
          ))}
        </div>
      </footer>

      <style>{`
        @media (max-width: 640px) {
          main { padding: 40px 20px 80px !important; }
          nav { padding: 20px !important; }
          div[style*="grid-template-columns: 1fr 1fr"] { grid-template-columns: 1fr !important; }
          div[style*="grid-template-columns: repeat(3, 1fr)"] { grid-template-columns: 1fr !important; }
          div[style*="padding: '52px 56px'"] { padding: 36px 24px !important; }
        }
      `}</style>
    </div>
  );
}
