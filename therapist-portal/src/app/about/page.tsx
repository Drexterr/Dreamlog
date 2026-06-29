import Link from 'next/link';

export default function AboutPage() {
  return (
    <div style={{ background: '#18150f', color: '#e8ddd0', minHeight: '100vh', fontFamily: "'Nunito', sans-serif" }}>
      <nav style={{ maxWidth: 960, margin: '0 auto', padding: '24px 40px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Link href="/" style={{ display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none', color: '#e8ddd0' }}>
          <div style={{ width: 28, height: 28, borderRadius: 7, background: '#c8955a', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1rem', color: '#18150f' }}>D</div>
          <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.1rem', fontWeight: 600 }}>DreamLog</span>
        </Link>
        <Link href="/#download" style={{ background: '#c8955a', color: '#18150f', borderRadius: 100, padding: '9px 20px', fontSize: '0.84rem', fontWeight: 700, textDecoration: 'none' }}>
          Download Free
        </Link>
      </nav>

      <main style={{ maxWidth: 720, margin: '0 auto', padding: '60px 40px 120px' }}>

        {/* Header */}
        <div style={{ marginBottom: 72 }}>
          <div style={{ fontSize: '0.7rem', letterSpacing: '2.5px', color: 'rgba(232,221,208,0.3)', textTransform: 'uppercase', marginBottom: 20 }}>Our story</div>
          <h1 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(2.8rem, 6vw, 4.5rem)', fontWeight: 300, margin: '0 0 24px', lineHeight: 1.1, letterSpacing: '-0.02em' }}>
            We built the journal<br />we needed.
          </h1>
          <p style={{ fontSize: '1.05rem', color: 'rgba(232,221,208,0.6)', lineHeight: 1.85, margin: 0 }}>
            DreamLog started as a personal experiment. The question was simple: what if you could speak your day aloud, and something genuinely thoughtful listened back?
          </p>
        </div>

        {/* Body */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 32 }}>
          <div style={{ borderLeft: '1px solid rgba(200,149,90,0.25)', paddingLeft: 28 }}>
            <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.5rem', fontWeight: 300, margin: '0 0 12px', fontStyle: 'italic' }}>The problem with journaling</h2>
            <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: 0 }}>
              Most people who try journaling stop. Not because they don&apos;t want to reflect — because the friction of writing makes it feel like homework. You&apos;re tired. You sit down. You stare at a blank page. You don&apos;t know where to start. You close the app.
            </p>
            <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: '16px 0 0' }}>
              Speaking is different. When you talk, things come out that wouldn&apos;t survive the journey from thought to typed word. Honesty, frustration, the quiet worry you didn&apos;t realise was there — these surface in voice in a way they don&apos;t on a keyboard.
            </p>
          </div>

          <div style={{ borderLeft: '1px solid rgba(200,149,90,0.25)', paddingLeft: 28 }}>
            <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.5rem', fontWeight: 300, margin: '0 0 12px', fontStyle: 'italic' }}>What we built</h2>
            <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: 0 }}>
              DreamLog transcribes what you say, cross-references it with your recent entries, and reflects back what it notices — the patterns, the recurring names, the emotions you mentioned without labelling. It doesn&apos;t tell you what to do. It shows you what you&apos;re already thinking.
            </p>
            <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: '16px 0 0' }}>
              We added Dream Decoder for the images your mind makes at night — using both Jungian depth-psychology and Vedic Svapna Shastra, because not everyone thinks in the same symbolic vocabulary. We added Therapy Mode for when you need more than a mirror — a companion that has read everything you&apos;ve said and comes to the conversation prepared.
            </p>
          </div>

          <div style={{ borderLeft: '1px solid rgba(200,149,90,0.25)', paddingLeft: 28 }}>
            <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.5rem', fontWeight: 300, margin: '0 0 12px', fontStyle: 'italic' }}>What we won&apos;t do</h2>
            <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: 0 }}>
              DreamLog is not a therapy replacement. It doesn&apos;t diagnose, prescribe, or pretend to be a licensed professional. Every session and reflection includes a clear disclaimer. Crisis detection runs on every entry — if you mention something serious, crisis resources appear immediately. This is non-negotiable.
            </p>
            <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: '16px 0 0' }}>
              We also don&apos;t sell your data. Your audio is deleted the moment it&apos;s transcribed. We don&apos;t train our models on your entries. We don&apos;t show you ads. The product is the product — you pay for it, or you use the generous free tier, and that&apos;s the entire relationship.
            </p>
          </div>

          <div style={{ borderLeft: '1px solid rgba(200,149,90,0.25)', paddingLeft: 28 }}>
            <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.5rem', fontWeight: 300, margin: '0 0 12px', fontStyle: 'italic' }}>Where we are now</h2>
            <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: 0 }}>
              DreamLog is in early access — currently being tested with a small group of users and mental health professionals. We&apos;re building slowly, intentionally, and in conversation with the people who use it.
            </p>
            <p style={{ fontSize: '0.95rem', color: 'rgba(232,221,208,0.55)', lineHeight: 1.85, margin: '16px 0 0' }}>
              If you&apos;re a therapist interested in recommending DreamLog to clients, or a company looking at team wellness, we&apos;d like to talk. Reach us at{' '}
              <a href="mailto:support@dreamlog.app" style={{ color: '#c8955a', textDecoration: 'none' }}>support@dreamlog.app</a>.
            </p>
          </div>
        </div>

        {/* Pull quote */}
        <div style={{ marginTop: 72, background: 'rgba(200,149,90,0.05)', border: '1px solid rgba(200,149,90,0.15)', borderRadius: 16, padding: '36px 40px' }}>
          <p style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(1.3rem, 2.5vw, 1.7rem)', fontWeight: 300, fontStyle: 'italic', lineHeight: 1.6, color: '#e8ddd0', margin: '0 0 16px' }}>
            &ldquo;Your thoughts are worth understanding. That&apos;s the whole premise.&rdquo;
          </p>
          <span style={{ fontSize: '0.8rem', color: 'rgba(232,221,208,0.35)' }}>— The DreamLog team</span>
        </div>

        {/* CTAs */}
        <div style={{ marginTop: 56, display: 'flex', gap: 14, flexWrap: 'wrap' }}>
          <Link href="/#download" style={{ background: '#c8955a', color: '#18150f', borderRadius: 12, padding: '13px 26px', fontWeight: 700, fontSize: '0.9rem', textDecoration: 'none' }}>
            Start journaling free
          </Link>
          <Link href="/therapists" style={{ border: '1px solid rgba(255,255,255,0.15)', color: '#e8ddd0', borderRadius: 12, padding: '13px 26px', fontSize: '0.9rem', textDecoration: 'none' }}>
            For therapists →
          </Link>
        </div>
      </main>

      <footer style={{ borderTop: '1px solid rgba(255,255,255,0.07)', padding: '28px 40px', maxWidth: 960, margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 16 }}>
        <span style={{ fontSize: '0.74rem', color: 'rgba(232,221,208,0.3)' }}>© 2026 DreamLog · Built for honesty, not performance.</span>
        <div style={{ display: 'flex', gap: 24 }}>
          {[['Home', '/'], ['Pricing', '/pricing'], ['Privacy', '/privacy'], ['Terms', '/terms']].map(([label, href]) => (
            <a key={label} href={href} style={{ fontSize: '0.8rem', color: 'rgba(232,221,208,0.4)', textDecoration: 'none' }}>{label}</a>
          ))}
        </div>
      </footer>
    </div>
  );
}
