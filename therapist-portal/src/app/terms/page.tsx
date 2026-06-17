'use client';

export default function TermsPage() {
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
        <h1 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(2rem, 4vw, 3rem)', fontWeight: 300, margin: '0 0 12px', lineHeight: 1.1 }}>Terms of Service</h1>
        <p style={{ fontSize: '0.84rem', color: 'var(--muted)', margin: '0 0 64px' }}>Last updated: June 2026</p>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 48 }}>

          <Section title="By using DreamLog, you're agreeing to this">
            <P>When you create an account or open the app, you&apos;re accepting these Terms. That goes for everyone - personal users, therapists, and businesses. If you don&apos;t agree, please don&apos;t use the app.</P>
          </Section>

          <Section title="What DreamLog is and what it isn't">
            <P>DreamLog is a voice journaling app with AI-powered emotional reflection. It is not a therapist. It is not a doctor. It cannot diagnose you with anything, and nothing it says should be treated as medical advice or clinical guidance.</P>
            <P>Therapy Mode is an AI-assisted conversation that draws on your journal history - it&apos;s named carefully to make clear it&apos;s not actual clinical therapy. Nothing in the app replaces a real mental health professional.</P>
            <P>If you&apos;re in crisis, please reach out to a professional. You can find crisis resources anytime inside the app under Settings → Get help now.</P>
          </Section>

          <Section title="Your account">
            <Item label="Age">You need to be at least 13 to use DreamLog. If you&apos;re under 18, you&apos;re confirming you have a parent or guardian&apos;s permission.</Item>
            <Item label="Accuracy">Please give us accurate information when signing up, and keep your login details safe - that&apos;s your responsibility.</Item>
            <Item label="One account per person">You can&apos;t create multiple accounts to get around plan limits.</Item>
            <Item label="Closing accounts">We can suspend or close accounts that break these Terms - we&apos;ll try to give notice when we can. You can delete your own account anytime in Settings.</Item>
          </Section>

          <Section title="Your content is yours">
            <P>Everything you record or write in DreamLog belongs to you. We ask only for a limited licence to process it - meaning we can transcribe it, run AI analysis on it, store it, and deliver it to your devices. That&apos;s it.</P>
            <P>We don&apos;t claim ownership of your journal entries. We don&apos;t use your content to train AI. We don&apos;t sell it.</P>
            <P>You are responsible for what you record. Please don&apos;t use DreamLog to store or send anything illegal, harassing, or that violates someone else&apos;s rights.</P>
          </Section>

          <Section title="Subscriptions and payments">
            <Item label="Plans">DreamLog has a Free plan and two paid options - DreamLog+ and DreamLog Pro. Paid plans are 30-day passes. They don&apos;t auto-renew. You pay once, get 30 days, and buy again if you want more.</Item>
            <Item label="Therapy sessions">Sessions can be purchased individually or are included in Pro. Your very first session is always free. The charge goes through when the session begins.</Item>
            <Item label="Refunds">Because access starts the moment you purchase, we can&apos;t refund unused days. If a technical issue stopped you from accessing the app, email us at <a href="mailto:support@dreamlog.app" style={{ color: 'var(--gold)' }}>support@dreamlog.app</a> within 7 days.</Item>
            <Item label="Price changes">If prices change, we&apos;ll give you at least 14 days&apos; notice by email or in-app.</Item>
            <Item label="App Store / Play purchases">If you buy through Apple or Google, their billing rules apply on top of ours.</Item>
          </Section>

          <Section title="For therapists">
            <P>The Therapist Portal is for licensed mental health professionals only. By registering, you&apos;re confirming you hold the appropriate credentials in your country or region.</P>
            <P>You can only see a client&apos;s data if they&apos;ve explicitly shared it with you via a passcode-protected link. You cannot go beyond what they&apos;ve shared, and you cannot store, redistribute, or use their data for anything outside direct clinical care.</P>
            <P>You&apos;re responsible for keeping your portal login secure and for handling client data in line with the professional and legal standards that apply to you - including HIPAA where relevant.</P>
          </Section>

          <Section title="Crisis detection">
            <P>Every journal entry and therapy session message is automatically scanned for signs of distress in two stages. If something is flagged, crisis resources appear in-app right away.</P>
            <P>This is not a replacement for emergency services. If you or someone else is in immediate danger, call emergency services - 112 in India, 911 in the US - or a crisis helpline immediately.</P>
            <P>The system is designed to err on the side of caution: it would rather show resources when they&apos;re not needed than miss a genuine crisis.</P>
          </Section>

          <Section title="What you can't do">
            <ul style={{ paddingLeft: 20, color: 'var(--muted)', lineHeight: 2.2, fontSize: '0.9rem', margin: 0 }}>
              <li>Try to reverse-engineer, decompile, or extract the app&apos;s source code.</li>
              <li>Use bots or automated tools to scrape or bulk-access the service.</li>
              <li>Attempt to access another user&apos;s data.</li>
              <li>Use DreamLog in ways that break the law.</li>
              <li>Impersonate someone else or misrepresent your professional credentials to get therapist access.</li>
            </ul>
          </Section>

          <Section title="Intellectual property">
            <P>The DreamLog name, logo, design, and all original content we create belong to us. Please don&apos;t use them without permission.</P>
            <P>Your journal content stays yours. AI-generated reflections are made for your personal use - we don&apos;t claim copyright over them.</P>
          </Section>

          <Section title="No guarantees">
            <P>DreamLog is provided as-is. We can&apos;t promise the service will always be available, error-free, or that every AI reflection will be perfectly accurate for your situation.</P>
            <P>AI-generated reflections are not clinical assessments. They can be wrong, incomplete, or miss important context. Please don&apos;t make major personal, medical, or legal decisions based solely on what DreamLog tells you.</P>
          </Section>

          <Section title="Our liability is limited">
            <P>To the extent the law allows, DreamLog and its operators are not liable for indirect, incidental, or consequential damages from your use of the service - including emotional distress, data loss, or decisions made based on AI output.</P>
            <P>If we are found liable for something, the maximum we owe is what you paid us in the 30 days before the issue arose.</P>
          </Section>

          <Section title="Legal stuff">
            <P>These Terms are governed by the laws of India. Any disputes go to the courts of Mumbai, Maharashtra.</P>
          </Section>

          <Section title="If these Terms change">
            <P>We&apos;ll tell you about any significant changes at least 14 days before they take effect, by email or in-app notice. If you keep using DreamLog after that, it means you&apos;ve accepted the updated Terms.</P>
          </Section>

          <Section title="Contact">
            <P>Questions about these Terms: <a href="mailto:support@dreamlog.app" style={{ color: 'var(--gold)' }}>support@dreamlog.app</a></P>
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

function Item({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '180px 1fr', gap: '0 24px', alignItems: 'start' }}>
      <div style={{ fontSize: '0.78rem', fontWeight: 600, color: 'var(--text)', paddingTop: 3, letterSpacing: '0.01em' }}>{label}</div>
      <p style={{ fontSize: '0.9rem', color: 'var(--muted)', lineHeight: 1.85, margin: 0 }}>{children}</p>
    </div>
  );
}
