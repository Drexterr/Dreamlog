'use client';

export default function PrivacyPage() {
  return (
    <div style={{ background: 'var(--bg)', color: 'var(--text)', minHeight: '100vh' }}>
      {/* Nav */}
      <nav style={{ borderBottom: '1px solid var(--border)', padding: '0 60px', height: 64, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <a href="/" style={{ display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none' }}>
          <div style={{ width: 28, height: 28, borderRadius: 7, background: 'var(--gold)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', fontWeight: 700, fontSize: '1rem', color: '#18150f' }}>D</div>
          <span style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.1rem', fontWeight: 600 }}>DreamLog</span>
        </a>
        <a href="/" style={{ fontSize: '0.84rem', color: 'var(--muted)', textDecoration: 'none' }}>← Back to home</a>
      </nav>

      <div style={{ maxWidth: 740, margin: '0 auto', padding: '72px 40px 120px' }}>
        <div style={{ fontSize: '0.6rem', letterSpacing: '2.5px', textTransform: 'uppercase', color: 'var(--gold)', fontWeight: 600, marginBottom: 20 }}>LEGAL</div>
        <h1 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(2rem, 4vw, 3rem)', fontWeight: 300, margin: '0 0 12px', lineHeight: 1.1 }}>Privacy Policy</h1>
        <p style={{ fontSize: '0.84rem', color: 'var(--muted)', margin: '0 0 64px' }}>Last updated: June 2026</p>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 48 }}>

          <Section title="What DreamLog Actually Is">
            <P>DreamLog is a voice journaling app. You talk, we turn it into text, reflect it back to you, and help you make sense of your inner life over time. The whole thing is built on one core belief: your thoughts belong to you. Not us. Not brands. Not data brokers. You.</P>
            <P>This document tells you exactly what we collect, why we need it, how long we hold onto it, and what you can do about any of it.</P>
          </Section>

          <Section title="What We Collect and Why">
            <Item label="Voice recordings">When you record something, the audio goes straight to secure cloud storage without ever touching our servers. Once it has been transcribed, the audio file is deleted permanently. We genuinely don&apos;t keep it. There&apos;s no archive of your voice sitting somewhere.</Item>
            <Item label="Transcripts">The text version of what you said is stored on our servers. This is the heart of DreamLog - it&apos;s what powers your journal history, your reflections, and the emotional patterns we track over time.</Item>
            <Item label="AI-generated content">The reflections, mood scores, emotional tags, key quotes, nudge messages, and dream interpretations we generate - these get saved alongside your entry so your history stays intact.</Item>
            <Item label="Your account details">Just your email and display name. If you sign in with Google or Apple, we only see what you explicitly authorise - nothing more.</Item>
            <Item label="Your notification token">If you turn on morning nudges, we store a device token to deliver them. You can switch it off anytime in Settings and it&apos;s gone.</Item>
            <Item label="Basic usage data">Things like when you recorded, how long the session was, and which version of the app you&apos;re using. We don&apos;t track what you tap, how you scroll, or anything behavioural beyond the app itself.</Item>
          </Section>

          <Section title="What We Don't Collect">
            <P>No location. No contacts. No photos. No tracking pixels. No advertising SDKs quietly hoovering up your data in the background. We don&apos;t sell your data for advertising - not now, not ever.</P>
          </Section>

          <Section title="How We Actually Use Your Data">
            <Item label="To write your reflection">Your latest transcript, along with your five most recent entries for context, gets sent to Anthropic&apos;s Claude API. That&apos;s what generates your personalised reflection. Anthropic doesn&apos;t use API data for training by default.</Item>
            <Item label="To spot if you're struggling">Every transcript is automatically checked for signs of distress - first by keyword matching, and if it&apos;s unclear, by a follow-up AI check. If something concerning is detected, crisis resources appear immediately. No human reads your entries.</Item>
            <Item label="To get better over time">The more you use DreamLog, the more it understands your recurring patterns, themes, and emotional language. This all lives in your account and never leaves it.</Item>
            <Item label="To send your morning nudge">If you opt into notifications, we craft a short message based on your last entry and send it at whatever time you choose via Firebase. It&apos;s entirely optional.</Item>
            <Item label="For Therapist Share">If you choose to share a link with your therapist, they get an anonymised view of your mood trends and AI summaries - never your actual transcripts or recordings. The link is passcode-protected and expires after 72 hours automatically.</Item>
          </Section>

          <Section title="How Long We Keep Things">
            <Item label="Audio recordings">Deleted immediately after transcription - not recoverable.</Item>
            <Item label="Transcripts & analyses">Until you delete the entry or your account.</Item>
            <Item label="Share links">72 hours, then permanently removed.</Item>
            <Item label="Account data">Until you request deletion.</Item>
          </Section>

          <Section title="The Services We Work With">
            <P>We use a small number of trusted external providers to run DreamLog. Here&apos;s who they are and what they do with your data:</P>
            <Item label="Anthropic (Claude)" href="https://www.anthropic.com/privacy">Generates your reflections and handles crisis detection. They don&apos;t train on API submissions.</Item>
            <Item label="OpenAI (Whisper)" href="https://openai.com/policies/privacy-policy">Transcribes your audio. Audio is sent to them only for this purpose.</Item>
            <Item label="Cloudflare R2">Holds your audio temporarily while transcription is happening, then it&apos;s deleted.</Item>
            <Item label="Supabase" href="https://supabase.com/privacy">Manages your account login, including social sign-in.</Item>
            <Item label="Firebase (Google)">Delivers your push notifications if you&apos;ve opted in.</Item>
            <Item label="Azure Speech Services" href="https://privacy.microsoft.com">If you use Therapy Mode with AI voice, your assistant&apos;s responses are processed here.</Item>
            <Item label="Stripe" href="https://stripe.com/privacy">Handles all payments. We never see your card details - Stripe takes care of that entirely.</Item>
          </Section>

          <Section title="A Note on Mental Health Data">
            <P>Journal entries and mood data are sensitive. We treat them that way:</P>
            <ul style={{ paddingLeft: 20, color: 'var(--muted)', lineHeight: 2.2, fontSize: '0.9rem', margin: 0 }}>
              <li>No DreamLog employee reads your entries.</li>
              <li>Crisis detection is automated - no human review.</li>
              <li>Therapist Share is opt-in, anonymised, and passcode-protected.</li>
              <li>We don&apos;t use your emotional data to target you with ads.</li>
              <li>We don&apos;t share it with insurers, employers, or anyone else.</li>
            </ul>
          </Section>

          <Section title="Children">
            <P>DreamLog isn&apos;t for anyone under 13. If you think a child has made an account, email us at <a href="mailto:support@dreamlog.app" style={{ color: 'var(--gold)' }}>support@dreamlog.app</a> and we&apos;ll delete it straight away.</P>
          </Section>

          <Section title="Your Rights">
            <P>You&apos;re in control. Here&apos;s what you can do at any time:</P>
            <ul style={{ paddingLeft: 20, color: 'var(--muted)', lineHeight: 2.2, fontSize: '0.9rem', margin: 0 }}>
              <li><strong style={{ color: 'var(--text)' }}>Export your data</strong> - Settings → Export my data. You&apos;ll get a PDF of your full journal history.</li>
              <li><strong style={{ color: 'var(--text)' }}>Delete an entry</strong> - Tap and delete from the journal screen. It&apos;s gone permanently.</li>
              <li><strong style={{ color: 'var(--text)' }}>Delete your account</strong> - Settings → Delete account. Everything goes - entries, analyses, account data. This can&apos;t be undone.</li>
              <li><strong style={{ color: 'var(--text)' }}>Turn off notifications</strong> - Settings → Notifications, or through your phone&apos;s system settings.</li>
              <li><strong style={{ color: 'var(--text)' }}>Revoke therapist access</strong> - Share links expire on their own after 72 hours, or contact us to kill one early.</li>
            </ul>
            <P>For a full copy of everything we hold about you, or for any GDPR requests, email <a href="mailto:support@dreamlog.app" style={{ color: 'var(--gold)' }}>support@dreamlog.app</a>.</P>
          </Section>

          <Section title="Security">
            <P>All data moves over HTTPS. Passwords are hashed, never stored as plain text. Tokens expire. Audio is stored in private buckets with time-limited access only. Database access is locked down to our application servers only.</P>
            <P>No system is completely airtight. If you find something, please let us know at <a href="mailto:support@dreamlog.app" style={{ color: 'var(--gold)' }}>support@dreamlog.app</a> before making it public.</P>
          </Section>

          <Section title="If This Policy Changes">
            <P>If we make any significant changes, we&apos;ll let you know by email or in-app - at least 14 days before they kick in. The date at the top of this page always shows you what version you&apos;re reading.</P>
          </Section>

          <Section title="Contact">
            <P>Questions, requests, or concerns: <a href="mailto:support@dreamlog.app" style={{ color: 'var(--gold)' }}>support@dreamlog.app</a></P>
          </Section>

        </div>
      </div>

      <footer style={{ borderTop: '1px solid var(--border)', padding: '32px 60px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 16 }}>
        <span style={{ fontSize: '0.78rem', color: 'var(--muted-2)' }}>© 2026 DreamLog</span>
        <div style={{ display: 'flex', gap: 24 }}>
          <a href="/privacy" style={{ fontSize: '0.78rem', color: 'var(--muted)', textDecoration: 'none' }}>Privacy Policy</a>
          <a href="/terms" style={{ fontSize: '0.78rem', color: 'var(--muted)', textDecoration: 'none' }}>Terms</a>
          <a href="mailto:support@dreamlog.app" style={{ fontSize: '0.78rem', color: 'var(--muted)', textDecoration: 'none' }}>Support</a>
        </div>
      </footer>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.35rem', fontWeight: 400, color: 'var(--text)', margin: '0 0 20px', letterSpacing: '-0.01em' }}>{title}</h2>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>{children}</div>
    </div>
  );
}

function P({ children }: { children: React.ReactNode }) {
  return <p style={{ fontSize: '0.9rem', color: 'var(--muted)', lineHeight: 1.85, margin: 0 }}>{children}</p>;
}

function Item({ label, href, children }: { label: string; href?: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '180px 1fr', gap: '0 24px', alignItems: 'start' }}>
      <div style={{ fontSize: '0.78rem', fontWeight: 600, paddingTop: 3, letterSpacing: '0.01em' }}>
        {href ? (
          <a href={href} target="_blank" rel="noreferrer" style={{ color: 'var(--gold)', textDecoration: 'none' }}>{label} ↗</a>
        ) : (
          <span style={{ color: 'var(--text)' }}>{label}</span>
        )}
      </div>
      <p style={{ fontSize: '0.9rem', color: 'var(--muted)', lineHeight: 1.85, margin: 0 }}>{children}</p>
    </div>
  );
}
