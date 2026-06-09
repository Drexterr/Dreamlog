'use client';

import React, { useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';

function useInView(threshold = 0.15) {
  const ref = useRef<HTMLDivElement>(null);
  const [inView, setInView] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) { setInView(true); observer.disconnect(); } },
      { threshold }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [threshold]);

  return { ref, inView };
}

function useCounter(target: number, inView: boolean, duration = 1600) {
  const [count, setCount] = useState(0);
  useEffect(() => {
    if (!inView) return;
    let start = 0;
    const step = target / (duration / 16);
    const timer = setInterval(() => {
      start += step;
      if (start >= target) { setCount(target); clearInterval(timer); }
      else setCount(Math.floor(start));
    }, 16);
    return () => clearInterval(timer);
  }, [inView, target, duration]);
  return count;
}

export default function LandingPage() {
  const router = useRouter();
  const goToPortal = () => router.push('/login');

  const [isBreatheActive, setIsBreatheActive] = useState(true);
  const [breathPhase, setBreathPhase] = useState<'inhale' | 'hold' | 'exhale'>('inhale');
  const breathIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [activeTab, setActiveTab] = useState<'record' | 'reflect' | 'radar'>('record');
  const [isRecording, setIsRecording] = useState(false);
  const [recordTime, setRecordTime] = useState(0);
  const [transcriptionText, setTranscriptionText] = useState('');
  const recordingIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [reflectionText, setReflectionText] = useState('');
  const [reflectionIndex, setReflectionIndex] = useState(0);
  const reflectionTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [activeFaq, setActiveFaq] = useState<number | null>(null);
  const [hoveredDataPoint, setHoveredDataPoint] = useState<{ x: number; y: number; mood: number; date: string } | null>(null);
  const [selectedTopic, setSelectedTopic] = useState<string>('Resilience');
  const [menuOpen, setMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  const statsRef = useInView();
  const entries = useCounter(22847, statsRef.inView);
  const streak = useCounter(97, statsRef.inView);
  const mins = useCounter(3, statsRef.inView);

  const featuresRef = useInView(0.1);
  const quoteRef = useInView(0.2);
  const therapistRef = useInView(0.1);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 40);
    window.addEventListener('scroll', onScroll);
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  useEffect(() => {
    if (!isBreatheActive) {
      if (breathIntervalRef.current) clearInterval(breathIntervalRef.current);
      return;
    }
    let step = 0;
    setBreathPhase('inhale');
    breathIntervalRef.current = setInterval(() => {
      step = (step + 1) % 6;
      if (step === 0) setBreathPhase('inhale');
      else if (step === 2) setBreathPhase('hold');
      else if (step === 4) setBreathPhase('exhale');
    }, 1000);
    return () => { if (breathIntervalRef.current) clearInterval(breathIntervalRef.current); };
  }, [isBreatheActive]);

  const breathLabels = { inhale: 'breathe in', hold: 'hold still', exhale: 'let it go' };
  const breathWords = { inhale: 'Inhale', hold: 'Hold', exhale: 'Exhale' };

  const fullTranscriptText = "I had a really busy day today... felt overwhelmed but I managed to finish my tasks. I think I need a break and some quiet time.";

  const startMockRecording = () => {
    if (isRecording) {
      setIsRecording(false);
      if (recordingIntervalRef.current) clearInterval(recordingIntervalRef.current);
      setTimeout(() => setActiveTab('reflect'), 800);
    } else {
      setIsRecording(true);
      setRecordTime(0);
      setTranscriptionText('');
      recordingIntervalRef.current = setInterval(() => setRecordTime((t) => t + 1), 1000);
    }
  };

  useEffect(() => {
    if (isRecording && recordTime > 1) {
      const words = fullTranscriptText.split(' ');
      const wordsToShow = Math.min(Math.floor(recordTime * 2.5), words.length);
      setTranscriptionText(words.slice(0, wordsToShow).join(' ') + (wordsToShow < words.length ? '...' : ''));
      if (wordsToShow >= words.length) {
        setIsRecording(false);
        if (recordingIntervalRef.current) clearInterval(recordingIntervalRef.current);
        setTimeout(() => setActiveTab('reflect'), 1500);
      }
    }
  }, [recordTime, isRecording]);

  const targetReflection = "It sounds like you carried a heavy load today, pushing through even when it felt like too much. In the middle of all that doing - what's one small thing you could let go of tonight?";

  useEffect(() => {
    if (activeTab === 'reflect') { setReflectionText(''); setReflectionIndex(0); }
  }, [activeTab]);

  useEffect(() => {
    if (activeTab === 'reflect' && reflectionIndex < targetReflection.length) {
      reflectionTimeoutRef.current = setTimeout(() => {
        setReflectionText((prev) => prev + targetReflection[reflectionIndex]);
        setReflectionIndex((idx) => idx + 1);
      }, 22);
    }
    return () => { if (reflectionTimeoutRef.current) clearTimeout(reflectionTimeoutRef.current); };
  }, [activeTab, reflectionIndex]);

  useEffect(() => {
    return () => {
      if (recordingIntervalRef.current) clearInterval(recordingIntervalRef.current);
      if (reflectionTimeoutRef.current) clearTimeout(reflectionTimeoutRef.current);
    };
  }, []);

  const moodData = [
    { x: 40, y: 150, mood: 65, date: 'Mon' },
    { x: 100, y: 120, mood: 72, date: 'Tue' },
    { x: 160, y: 160, mood: 58, date: 'Wed' },
    { x: 220, y: 90, mood: 80, date: 'Thu' },
    { x: 280, y: 110, mood: 75, date: 'Fri' },
    { x: 340, y: 70, mood: 88, date: 'Sat' },
    { x: 400, y: 80, mood: 85, date: 'Sun' },
  ];

  const topicsList = [
    { name: 'Resilience', count: 12, desc: "Shows up most when you write about work - pushing through even when you really didn't want to." },
    { name: 'Self-Care', count: 8, desc: "You mention rest, walks, and switching off notifications. It's more present than you think." },
    { name: 'Ambition', count: 15, desc: "Goals, future plans, that feeling of wanting to build something. It's your most consistent thread." },
    { name: 'Gratitude', count: 6, desc: "Small things - a good meal, a friend who texted. You notice them even on hard days." },
  ];

  const faqs = [
    {
      q: "Do I have to write anything?",
      a: "Nope. You just talk. DreamLog uses Whisper to convert speech to text, so you can record a rambling 10-minute brain dump or a quiet two-minute check-in - whatever feels right. Most people find it easier than a blank page.",
    },
    {
      q: "What does the AI actually say back?",
      a: "It doesn't give you advice or diagnoses. It reflects - it notices what you said, validates how you're feeling, and usually ends with a question to help you think a little deeper. Warm, not clinical. Think of it like journaling with a good listener.",
    },
    {
      q: "Can I share this with my therapist?",
      a: "Yes. You can generate a passcode-protected link that expires after 72 hours. It shows your therapist a high-level picture - mood trends, recurring themes, how you've been feeling week to week. It doesn't expose your raw recordings unless you choose to include them.",
    },
    {
      q: "Is this actually private?",
      a: "Everything is encrypted. We don't sell your data, and we don't use your entries to train any public models. Your words stay yours.",
    },
  ];

  return (
    <div style={s.page}>
      <style dangerouslySetInnerHTML={{ __html: css }} />

      <div className="blob blob-1" />
      <div className="blob blob-2" />
      <div className="blob blob-3" />

      {/* Header */}
      <header className={`site-header ${scrolled ? 'scrolled' : ''}`}>
        <div style={s.logoWrap}>
          <span style={s.logoLeaf}>🍃</span>
          <span style={s.logoName}>DreamLog</span>
        </div>
        <nav className="desktop-nav">
          <a href="#features" className="nav-link">Features</a>
          <a href="#breathing" className="nav-link">Breathe</a>
          <a href="#showcase" className="nav-link">How it works</a>
          <a href="#therapist" className="nav-link">For therapists</a>
        </nav>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <button onClick={goToPortal} className="btn-dark">Therapist portal →</button>
          <button className="hamburger" onClick={() => setMenuOpen(!menuOpen)} aria-label="menu">
            <span /><span /><span />
          </button>
        </div>
      </header>

      {menuOpen && (
        <div className="mobile-menu">
          <a href="#features" className="mobile-link" onClick={() => setMenuOpen(false)}>Features</a>
          <a href="#breathing" className="mobile-link" onClick={() => setMenuOpen(false)}>Breathe</a>
          <a href="#showcase" className="mobile-link" onClick={() => setMenuOpen(false)}>How it works</a>
          <a href="#therapist" className="mobile-link" onClick={() => setMenuOpen(false)}>For therapists</a>
          <button onClick={() => { goToPortal(); setMenuOpen(false); }} className="mobile-link" style={{ background: 'none', border: 'none', fontFamily: 'inherit', cursor: 'pointer', textAlign: 'left', padding: '14px 28px' }}>Therapist portal</button>
        </div>
      )}

      {/* Hero */}
      <section style={s.hero}>
        <div className="fade-in-up" style={s.heroLeft}>
          <span style={s.eyebrow}>voice journaling · ai reflection</span>
          <h1 style={s.heroHeading}>
            Your thoughts,<br />
            <em style={s.heroItalic}>out loud.</em>
          </h1>
          <p style={s.heroPara}>
            Talk for two minutes or twenty. DreamLog listens, transcribes, and reflects back
            something worth sitting with - grounded in everything you&apos;ve shared before.
          </p>
          <div style={{ display: 'flex', gap: '14px', flexWrap: 'wrap' }}>
            <a href="#showcase" className="btn-primary">See how it works</a>
            <button onClick={goToPortal} className="btn-ghost">Therapist portal</button>
          </div>
          <p style={s.heroNote}>No account needed to explore · Free to start</p>
        </div>

        <div
          id="breathing"
          className={`breathing-card ${isBreatheActive ? 'breathing-active' : ''}`}
          onClick={() => setIsBreatheActive(p => !p)}
          style={s.breathCard}
        >
          <div className="breath-ring" />
          <div className="breath-ring-2" />
          <div className="breath-orb">
            <span className="breath-word">{isBreatheActive ? breathWords[breathPhase] : 'Pause'}</span>
          </div>
          <p className="breath-label">{isBreatheActive ? breathLabels[breathPhase] : 'tap to resume'}</p>
          <span style={s.breathHint}>tap to {isBreatheActive ? 'pause' : 'start'}</span>
        </div>
      </section>

      {/* Stats */}
      <div ref={statsRef.ref} style={s.statsRow}>
        <div style={s.statItem}>
          <span className={`stat-num ${statsRef.inView ? 'count-in' : ''}`}>
            {entries.toLocaleString()}
          </span>
          <span style={s.statLabel}>entries reflected on</span>
        </div>
        <div style={s.statDivider} />
        <div style={s.statItem}>
          <span className={`stat-num ${statsRef.inView ? 'count-in' : ''}`}>{mins} min</span>
          <span style={s.statLabel}>average recording</span>
        </div>
        <div style={s.statDivider} />
        <div style={s.statItem}>
          <span className={`stat-num ${statsRef.inView ? 'count-in' : ''}`}>{streak} days</span>
          <span style={s.statLabel}>longest streak</span>
        </div>
      </div>

      {/* Features */}
      <section id="features" style={s.section} ref={featuresRef.ref}>
        <div style={s.sectionHead}>
          <span style={s.sectionEyebrow}>what it does</span>
          <h2 style={s.sectionHeading}>Built around how people actually think</h2>
          <p style={s.sectionSub}>Not how they&apos;re supposed to journal.</p>
        </div>
        <div style={s.featureGrid}>
          {[
            { icon: '🎙️', title: 'Just talk', body: "No cursor blinking at you. No word count. Hit record and say whatever's on your mind - Whisper picks it up even when you trail off mid-sentence." },
            { icon: '🪞', title: 'It asks back', body: "Claude reads your entry and reflects something specific to you, not a generic affirmation. It usually ends with one question you didn't think to ask yourself." },
            { icon: '🌱', title: 'Notice patterns', body: "After a few weeks, you start seeing things - which topics come up when you're stressed, what your mood actually looks like on Mondays, what you keep circling back to." },
            { icon: '🔒', title: 'Private by design', body: "Audio is deleted the moment it's transcribed. Entries are encrypted. You can share a summary with your therapist, but nothing leaves without you choosing it." },
          ].map((f, i) => (
            <div key={f.title} className={`feature-card ${featuresRef.inView ? 'card-visible' : ''}`} style={{ animationDelay: `${i * 0.1}s` }}>
              <span style={s.featureIcon}>{f.icon}</span>
              <h3 style={s.featureTitle}>{f.title}</h3>
              <p style={s.featureBody}>{f.body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Showcase */}
      <section id="showcase" style={{ ...s.section, background: 'rgba(235, 236, 233, 0.3)', borderTop: '1px solid rgba(42,44,43,0.06)', borderBottom: '1px solid rgba(42,44,43,0.06)' }}>
        <div style={{ textAlign: 'center', marginBottom: '44px' }}>
          <span style={s.sectionEyebrow}>try it</span>
          <h2 style={s.sectionHeading}>See what it actually feels like</h2>
          <p style={s.sectionSub}>Click through the three steps - recording, reflection, patterns.</p>
        </div>

        <div style={s.simulator}>
          <div style={s.tabs}>
            {[
              { key: 'record', label: '① Record' },
              { key: 'reflect', label: '② Reflection' },
              { key: 'radar', label: '③ Patterns' },
            ].map(tab => (
              <button key={tab.key} onClick={() => setActiveTab(tab.key as 'record' | 'reflect' | 'radar')} className={`sim-tab ${activeTab === tab.key ? 'sim-tab-active' : ''}`}>
                {tab.label}
              </button>
            ))}
          </div>

          <div style={s.screenWrap}>
            {activeTab === 'record' && (
              <div style={s.screen}>
                <div style={s.screenBar}>
                  <span style={{ fontWeight: 600, fontSize: '0.82rem' }}>Voice Journal</span>
                  <span style={s.onlineDot} />
                </div>
                <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '28px', gap: '18px' }}>
                  <div style={s.timer}>
                    {Math.floor(recordTime / 60).toString().padStart(2, '0')}:{(recordTime % 60).toString().padStart(2, '0')}
                  </div>
                  <div style={{ height: '36px', display: 'flex', alignItems: 'center', justifyContent: 'center', width: '100%' }}>
                    {isRecording
                      ? <div style={{ display: 'flex', gap: '3px', alignItems: 'center' }}>
                          {[...Array(12)].map((_, i) => <div key={i} className="wave-bar" style={{ animationDelay: `${i * 0.08}s` }} />)}
                        </div>
                      : <div style={{ height: '2px', width: '100px', background: 'rgba(42,44,43,0.12)', borderRadius: '1px' }} />
                    }
                  </div>
                  <button onClick={startMockRecording} className={`rec-btn ${isRecording ? 'rec-btn-active' : ''}`}>
                    {isRecording ? '■' : '●'}
                  </button>
                  <p style={{ fontSize: '0.77rem', color: 'var(--muted)', fontWeight: 500 }}>
                    {isRecording ? 'recording - tap to finish' : 'tap to start'}
                  </p>
                  {transcriptionText && (
                    <div style={s.transcript}>
                      <span style={{ fontSize: '0.68rem', color: 'var(--accent)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.5px', display: 'block', marginBottom: '4px' }}>Whisper · live</span>
                      <p style={{ fontSize: '0.82rem', color: 'var(--dark)', lineHeight: '1.45' }}>"{transcriptionText}"</p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {activeTab === 'reflect' && (
              <div style={s.screen}>
                <div style={s.screenBar}>
                  <span style={{ fontWeight: 600, fontSize: '0.82rem' }}>Reflection</span>
                  <span style={{ fontSize: '0.68rem', background: 'rgba(120,133,116,0.1)', color: 'var(--accent)', padding: '2px 8px', borderRadius: '4px', fontWeight: 600 }}>Claude</span>
                </div>
                <div style={{ flex: 1, padding: '20px', display: 'flex', flexDirection: 'column', gap: '14px' }}>
                  <div style={s.bubbleUser}>
                    <span style={s.bubbleLabel}>your entry</span>
                    <p style={{ fontSize: '0.87rem', lineHeight: '1.5' }}>"{fullTranscriptText}"</p>
                  </div>
                  <div style={s.bubbleAI}>
                    <span style={{ ...s.bubbleLabel, color: 'var(--accent)' }}>reflection</span>
                    <p style={{ fontSize: '0.92rem', fontFamily: "'Cormorant Garamond', serif", fontStyle: 'italic', lineHeight: '1.55', color: 'var(--dark)' }}>
                      {reflectionText}
                      {reflectionIndex < targetReflection.length && <span className="cursor">|</span>}
                    </p>
                  </div>
                  {reflectionIndex >= targetReflection.length && (
                    <button onClick={() => setActiveTab('radar')} className="btn-primary" style={{ marginTop: 'auto', fontSize: '0.82rem', padding: '10px 20px' }}>
                      See patterns →
                    </button>
                  )}
                </div>
              </div>
            )}

            {activeTab === 'radar' && (
              <div style={s.screen}>
                <div style={s.screenBar}>
                  <span style={{ fontWeight: 600, fontSize: '0.82rem' }}>Your patterns</span>
                  <span style={{ fontSize: '0.75rem', color: 'var(--muted)' }}>this week</span>
                </div>
                <div style={{ flex: 1, padding: '18px', display: 'flex', flexDirection: 'column', gap: '14px' }}>
                  <div style={s.chartBox}>
                    <svg viewBox="0 0 440 190" style={{ width: '100%', height: '100%' }}>
                      <defs>
                        <linearGradient id="moodGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor="#788574" stopOpacity="0.15" />
                          <stop offset="100%" stopColor="#788574" stopOpacity="0" />
                        </linearGradient>
                      </defs>
                      <line x1="40" y1="50" x2="400" y2="50" stroke="rgba(42,44,43,0.05)" />
                      <line x1="40" y1="100" x2="400" y2="100" stroke="rgba(42,44,43,0.05)" />
                      <line x1="40" y1="150" x2="400" y2="150" stroke="rgba(42,44,43,0.05)" />
                      <path d="M 40 150 L 100 120 L 160 160 L 220 90 L 280 110 L 340 70 L 400 80 L 400 170 L 40 170 Z" fill="url(#moodGrad)" />
                      <path className="chart-line" d="M 40 150 L 100 120 L 160 160 L 220 90 L 280 110 L 340 70 L 400 80" fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
                      {moodData.map((pt, i) => (
                        <circle key={i} cx={pt.x} cy={pt.y}
                          r={hoveredDataPoint?.date === pt.date ? '6' : '4'}
                          fill="white" stroke="var(--accent)" strokeWidth="2"
                          style={{ cursor: 'pointer', transition: 'r 0.2s' }}
                          onMouseEnter={() => setHoveredDataPoint(pt)}
                          onMouseLeave={() => setHoveredDataPoint(null)}
                        />
                      ))}
                      {moodData.map((pt, i) => (
                        <text key={i} x={pt.x} y="185" textAnchor="middle" fontSize="10" fill="var(--muted)" fontFamily="inherit">{pt.date}</text>
                      ))}
                    </svg>
                    {hoveredDataPoint && (
                      <div style={{ ...s.tooltip, left: `${hoveredDataPoint.x - 45}px`, top: `${hoveredDataPoint.y - 52}px` }}>
                        <strong>{hoveredDataPoint.mood}</strong> · {hoveredDataPoint.date}
                      </div>
                    )}
                  </div>
                  <div>
                    <p style={{ fontSize: '0.7rem', color: 'var(--muted)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '8px' }}>recurring themes</p>
                    <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', marginBottom: '10px' }}>
                      {topicsList.map(t => (
                        <button key={t.name} onClick={() => setSelectedTopic(t.name)} className={`topic-pill ${selectedTopic === t.name ? 'topic-pill-active' : ''}`}>
                          {t.name} · {t.count}
                        </button>
                      ))}
                    </div>
                    <div style={s.topicDesc}>
                      <p style={{ fontSize: '0.8rem', color: 'var(--dark)', lineHeight: '1.5' }}>
                        {topicsList.find(t => t.name === selectedTopic)?.desc}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </section>

      {/* Quote */}
      <div ref={quoteRef.ref} className={`quote-section ${quoteRef.inView ? 'quote-visible' : ''}`} style={s.quoteWrap}>
        <div style={s.quoteInner}>
          <span style={s.quoteOpen}>&ldquo;</span>
          <blockquote style={s.quoteText}>
            I&apos;ve kept journals for years but always gave up after a week. Talking feels different - it&apos;s like the words come out before my inner critic can edit them.
          </blockquote>
          <cite style={s.quoteAuthor}>- Priya, 28, using DreamLog for 3 months</cite>
        </div>
      </div>

      {/* Therapist */}
      <section id="therapist" ref={therapistRef.ref} style={s.therapistSection}>
        <div className={`therapist-content ${therapistRef.inView ? 'slide-in-left' : ''}`} style={s.therapistLeft}>
          <span style={s.sectionEyebrow}>for clinicians</span>
          <h2 style={s.sectionHeading}>A window into the week before the session</h2>
          <p style={{ fontSize: '0.97rem', color: 'var(--muted)', lineHeight: '1.7', marginBottom: '24px' }}>
            DreamLog isn&apos;t trying to replace therapy. It&apos;s the space between sessions -
            where clients process in real time. You can see a high-level picture of
            how they&apos;ve been feeling without reading every word they wrote.
          </p>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '14px', marginBottom: '28px' }}>
            {[
              { title: 'Passcode-protected links', desc: 'Clients generate a share link. It expires in 72 hours. No account needed on your end.' },
              { title: 'Mood trends, not raw entries', desc: 'You see emotional trajectory, top themes, and a Claude-generated brief - not a word-for-word transcript.' },
              { title: 'Crisis detection built in', desc: 'Two-stage screening runs on every entry. If something flags, the client sees crisis resources immediately.' },
            ].map(item => (
              <div key={item.title} style={s.bulletRow}>
                <span style={s.checkmark}>✓</span>
                <div>
                  <strong style={{ fontSize: '0.92rem', display: 'block', marginBottom: '2px' }}>{item.title}</strong>
                  <span style={{ fontSize: '0.88rem', color: 'var(--muted)' }}>{item.desc}</span>
                </div>
              </div>
            ))}
          </div>
          <button onClick={goToPortal} className="btn-primary">Open therapist portal</button>
        </div>

        <div className={`therapist-card-wrap ${therapistRef.inView ? 'slide-in-right' : ''}`} style={s.therapistRight}>
          <div style={s.clinicCard}>
            <div style={s.clinicCardHeader}>
              <div>
                <p style={{ fontWeight: 600, fontSize: '0.95rem', marginBottom: '2px' }}>Client: Noah M.</p>
                <p style={{ fontSize: '0.72rem', color: 'var(--muted)' }}>Shared · expires in 48h</p>
              </div>
              <span style={s.activeBadge}>Active</span>
            </div>
            <div style={s.clinicMetrics}>
              <div style={s.metricRow}>
                <span style={{ fontSize: '0.8rem', color: 'var(--muted)' }}>Mood this week</span>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <div style={s.moodBar}><div style={{ ...s.moodFill, width: '76%' }} /></div>
                  <span style={{ fontSize: '0.85rem', fontWeight: 600 }}>76</span>
                </div>
              </div>
              <div style={s.metricRow}>
                <span style={{ fontSize: '0.8rem', color: 'var(--muted)' }}>Dominant emotion</span>
                <span style={{ fontSize: '0.85rem', fontWeight: 600 }}>Cautious hope</span>
              </div>
              <div style={s.metricRow}>
                <span style={{ fontSize: '0.8rem', color: 'var(--muted)' }}>Entries this week</span>
                <span style={{ fontSize: '0.85rem', fontWeight: 600 }}>5</span>
              </div>
            </div>
            <div style={s.clinicBrief}>
              <p style={{ fontSize: '0.68rem', color: 'var(--muted)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '8px' }}>Pre-session brief</p>
              <p style={{ fontSize: '0.82rem', lineHeight: '1.55', fontStyle: 'italic', color: 'var(--dark)' }}>
                &ldquo;Noah is building capacity to set limits at work. Mood lifted mid-week after a difficult Monday. Sleep is still mentioned as a secondary stressor.&rdquo;
              </p>
            </div>
            <div style={{ display: 'flex', gap: '6px', marginTop: '14px', flexWrap: 'wrap' }}>
              {['Work limits', 'Sleep', 'Resilience'].map(tag => <span key={tag} style={s.tagPill}>{tag}</span>)}
            </div>
          </div>
        </div>
      </section>

      {/* FAQ */}
      <section style={s.faqSection}>
        <h2 style={{ ...s.sectionHeading, textAlign: 'center', marginBottom: '48px' }}>Things people ask</h2>
        <div style={{ maxWidth: '700px', margin: '0 auto' }}>
          {faqs.map((faq, i) => (
            <div key={i} className="faq-row" style={{ borderBottom: '1px solid rgba(42,44,43,0.07)', padding: '20px 0' }}>
              <button onClick={() => setActiveFaq(activeFaq === i ? null : i)} style={s.faqBtn}>
                <span style={{ fontSize: '1.02rem', fontWeight: 500, color: 'var(--dark)', textAlign: 'left' }}>{faq.q}</span>
                <span style={{ ...s.faqPlus, transform: activeFaq === i ? 'rotate(45deg)' : 'rotate(0)' }}>+</span>
              </button>
              <div style={{ overflow: 'hidden', maxHeight: activeFaq === i ? '200px' : '0', opacity: activeFaq === i ? 1 : 0, transition: 'all 0.35s cubic-bezier(0.16, 1, 0.3, 1)', marginTop: activeFaq === i ? '10px' : '0' }}>
                <p style={{ fontSize: '0.9rem', color: 'var(--muted)', lineHeight: '1.65' }}>{faq.a}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* CTA Banner */}
      <div style={s.ctaBanner}>
        <h2 style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '2.6rem', fontWeight: 300, color: 'var(--dark)', marginBottom: '12px' }}>
          Start with one recording.
        </h2>
        <p style={{ fontSize: '0.97rem', color: 'var(--muted)', marginBottom: '28px', maxWidth: '440px', margin: '0 auto 28px' }}>
          Two minutes. No setup. Just press record and say what&apos;s on your mind.
        </p>
        <div style={{ display: 'flex', gap: '14px', justifyContent: 'center', flexWrap: 'wrap' }}>
          <a href="#showcase" className="btn-primary">Try the demo</a>
          <button onClick={goToPortal} className="btn-ghost">Therapist portal →</button>
        </div>
      </div>

      {/* Footer */}
      <footer style={s.footer}>
        <div style={s.footerInner}>
          <div>
            <p style={{ fontFamily: "'Cormorant Garamond', serif", fontSize: '1.4rem', fontStyle: 'italic', fontWeight: 600, marginBottom: '6px' }}>DreamLog</p>
            <p style={{ fontSize: '0.78rem', color: 'var(--muted)' }}>A mindful space for the things you need to say.</p>
          </div>
          <div style={{ display: 'flex', gap: '28px', flexWrap: 'wrap' }}>
            <a href="#features" className="footer-link">Features</a>
            <a href="#breathing" className="footer-link">Breathe</a>
            <a href="#showcase" className="footer-link">Demo</a>
            <a href="#therapist" className="footer-link">Clinicians</a>
          </div>
        </div>
        <p style={{ fontSize: '0.75rem', color: 'var(--muted)', textAlign: 'center', marginTop: '32px', paddingTop: '24px', borderTop: '1px solid rgba(42,44,43,0.06)' }}>
          © 2026 DreamLog · Built with care
        </p>
      </footer>
    </div>
  );
}

const css = `
:root {
  --bg: #FAF8F5;
  --dark: #2A2C2B;
  --muted: #7E8280;
  --accent: #788574;
  --accent-light: #EBECE9;
  --border: rgba(42,44,43,0.08);
  --card: #FFFFFF;
}

.blob { position: fixed; border-radius: 50%; filter: blur(80px); pointer-events: none; z-index: 0; }
.blob-1 { width: 500px; height: 500px; background: #E3E8DF; top: -120px; left: -100px; opacity: 0.5; animation: drift 16s ease-in-out infinite; }
.blob-2 { width: 420px; height: 420px; background: #EDE9E2; bottom: 10%; right: -80px; opacity: 0.45; animation: drift 22s ease-in-out infinite reverse; }
.blob-3 { width: 300px; height: 300px; background: #E8EAE5; top: 55%; left: 40%; opacity: 0.3; animation: drift 28s ease-in-out infinite 4s; }
@keyframes drift { 0%,100% { transform: translate(0,0) scale(1); } 33% { transform: translate(30px,-20px) scale(1.04); } 66% { transform: translate(-20px,15px) scale(0.97); } }

.site-header { position: fixed; top: 0; left: 0; right: 0; padding: 20px 6%; display: flex; justify-content: space-between; align-items: center; transition: all 0.35s ease; z-index: 200; }
.site-header.scrolled { background: rgba(250,248,245,0.88); backdrop-filter: blur(16px); border-bottom: 1px solid rgba(42,44,43,0.06); padding: 14px 6%; }
.desktop-nav { display: flex; gap: 28px; }
.nav-link { color: var(--muted); text-decoration: none; font-size: 0.88rem; font-weight: 500; transition: color 0.25s; position: relative; }
.nav-link:hover { color: var(--dark); }
.nav-link::after { content: ''; position: absolute; bottom: -3px; left: 0; width: 0; height: 1.5px; background: var(--accent); transition: width 0.3s ease; }
.nav-link:hover::after { width: 100%; }

.hamburger { display: none; flex-direction: column; gap: 5px; background: none; border: none; cursor: pointer; padding: 4px; }
.hamburger span { display: block; width: 22px; height: 1.5px; background: var(--dark); border-radius: 2px; transition: all 0.3s; }
.mobile-menu { position: fixed; top: 60px; left: 0; right: 0; background: rgba(250,248,245,0.97); backdrop-filter: blur(16px); border-bottom: 1px solid rgba(42,44,43,0.08); z-index: 199; padding: 8px 0; }
.mobile-link { display: block; padding: 14px 28px; color: var(--dark); text-decoration: none; font-size: 0.95rem; font-weight: 500; border-bottom: 1px solid rgba(42,44,43,0.05); }

.btn-primary, .btn-ghost, .btn-dark { border: none; border-radius: 30px; cursor: pointer; font-family: inherit; font-weight: 500; transition: all 0.3s cubic-bezier(0.16,1,0.3,1); text-decoration: none; display: inline-flex; align-items: center; justify-content: center; }
.btn-primary { background: var(--accent); color: #FAF8F5; padding: 13px 28px; font-size: 0.9rem; }
.btn-primary:hover { background: var(--dark); transform: translateY(-2px); box-shadow: 0 8px 20px rgba(42,44,43,0.1); }
.btn-ghost { background: transparent; color: var(--dark); border: 1px solid rgba(42,44,43,0.14); padding: 13px 28px; font-size: 0.9rem; }
.btn-ghost:hover { background: var(--accent-light); border-color: var(--accent); transform: translateY(-2px); }
.btn-dark { background: var(--dark); color: #FAF8F5; padding: 9px 20px; font-size: 0.84rem; border-radius: 30px; }
.btn-dark:hover { background: var(--accent); transform: translateY(-1px); }

.fade-in-up { animation: fadeUp 0.9s cubic-bezier(0.16,1,0.3,1) forwards; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(28px); } to { opacity: 1; transform: translateY(0); } }

.breathing-card { position: relative; width: 280px; height: 280px; border-radius: 50%; display: flex; flex-direction: column; align-items: center; justify-content: center; cursor: pointer; user-select: none; flex-shrink: 0; }
.breath-ring { position: absolute; inset: 0; border-radius: 50%; border: 1.5px solid rgba(120,133,116,0.25); transition: all 6s ease-in-out; }
.breath-ring-2 { position: absolute; inset: 24px; border-radius: 50%; background: rgba(235,236,233,0.6); transition: all 6s ease-in-out; }
.breath-orb { width: 110px; height: 110px; border-radius: 50%; background: var(--accent); display: flex; align-items: center; justify-content: center; z-index: 2; box-shadow: 0 12px 30px rgba(120,133,116,0.18); transition: all 6s ease-in-out; }
.breath-word { font-family: 'Cormorant Garamond', serif; font-size: 1.3rem; font-style: italic; font-weight: 300; color: #FAF8F5; }
.breath-label { position: absolute; bottom: 28px; font-size: 0.75rem; color: var(--muted); letter-spacing: 1px; text-transform: lowercase; transition: opacity 0.5s; }
.breathing-active .breath-ring { animation: ring-breathe 6s ease-in-out infinite; }
.breathing-active .breath-ring-2 { animation: inner-breathe 6s ease-in-out infinite; }
.breathing-active .breath-orb { animation: orb-breathe 6s ease-in-out infinite; }
@keyframes ring-breathe { 0%,100% { transform: scale(1); opacity: 0.4; } 33% { transform: scale(1.35); opacity: 0.08; } 66% { transform: scale(1.35); opacity: 0.08; } }
@keyframes inner-breathe { 0%,100% { transform: scale(1); } 33% { transform: scale(1.18); } 66% { transform: scale(1.18); } }
@keyframes orb-breathe { 0%,100% { transform: scale(1); } 33% { transform: scale(1.08); box-shadow: 0 16px 40px rgba(120,133,116,0.25); } 66% { transform: scale(1.08); } }

.stat-num { font-family: 'Cormorant Garamond', serif; font-size: 2.2rem; font-weight: 600; color: var(--dark); display: block; opacity: 0; transform: translateY(10px); transition: none; }
.stat-num.count-in { animation: countIn 0.6s cubic-bezier(0.16,1,0.3,1) forwards; }
@keyframes countIn { to { opacity: 1; transform: translateY(0); } }

.feature-card { background: var(--card); border: 1px solid var(--border); border-radius: 20px; padding: 28px; opacity: 0; transform: translateY(20px); transition: opacity 0.5s, transform 0.5s, box-shadow 0.4s, border-color 0.4s; }
.feature-card.card-visible { opacity: 1; transform: translateY(0); }
.feature-card:hover { transform: translateY(-5px) !important; box-shadow: 0 16px 32px rgba(42,44,43,0.05); border-color: rgba(120,133,116,0.2); }

.sim-tab { flex: 1; padding: 14px; background: transparent; border: none; font-family: inherit; font-size: 0.86rem; font-weight: 600; color: var(--muted); cursor: pointer; transition: all 0.25s; outline: none; }
.sim-tab:hover { color: var(--dark); }
.sim-tab-active { background: white; color: var(--dark); border-bottom: 2.5px solid var(--accent); }

.rec-btn { width: 62px; height: 62px; border-radius: 50%; border: none; background: var(--accent); color: white; font-size: 1.3rem; cursor: pointer; display: flex; align-items: center; justify-content: center; transition: all 0.3s; box-shadow: 0 4px 16px rgba(120,133,116,0.28); }
.rec-btn:hover { transform: scale(1.08); }
.rec-btn.rec-btn-active { background: #c0392b; animation: rec-pulse 1.8s ease-in-out infinite; }
@keyframes rec-pulse { 0%,100% { box-shadow: 0 0 0 0 rgba(192,57,43,0.3); } 50% { box-shadow: 0 0 0 14px rgba(192,57,43,0); } }

.wave-bar { width: 4px; background: var(--accent); border-radius: 2px; animation: wave 1.1s ease-in-out infinite; transform-origin: center; height: 4px; }
@keyframes wave { 0%,100% { transform: scaleY(0.3); } 50% { transform: scaleY(1); height: 28px; } }

.chart-line { stroke-dasharray: 800; stroke-dashoffset: 800; animation: drawLine 1.2s ease-out 0.2s forwards; }
@keyframes drawLine { to { stroke-dashoffset: 0; } }

.topic-pill { background: transparent; border: 1px solid rgba(42,44,43,0.1); border-radius: 14px; padding: 3px 10px; font-size: 0.72rem; color: var(--muted); cursor: pointer; font-family: inherit; transition: all 0.2s; }
.topic-pill:hover { border-color: var(--accent); color: var(--accent); }
.topic-pill-active { background: rgba(120,133,116,0.12); border-color: var(--accent); color: var(--accent); }

.quote-section { opacity: 0; transform: translateY(20px); transition: all 0.7s ease; }
.quote-section.quote-visible { opacity: 1; transform: translateY(0); }
.therapist-content { opacity: 0; transform: translateX(-24px); transition: all 0.7s cubic-bezier(0.16,1,0.3,1); }
.therapist-content.slide-in-left { opacity: 1; transform: translateX(0); }
.therapist-card-wrap { opacity: 0; transform: translateX(24px); transition: all 0.7s cubic-bezier(0.16,1,0.3,1) 0.15s; }
.therapist-card-wrap.slide-in-right { opacity: 1; transform: translateX(0); }

.cursor { color: var(--accent); animation: blink 0.8s step-end infinite; }
@keyframes blink { 50% { opacity: 0; } }

.faq-row:hover { border-color: rgba(120,133,116,0.2) !important; }
.footer-link { color: var(--muted); text-decoration: none; font-size: 0.86rem; transition: color 0.2s; }
.footer-link:hover { color: var(--dark); }

@media (max-width: 900px) { .desktop-nav { display: none; } .hamburger { display: flex; } }
@media (max-width: 700px) { .desktop-nav { display: none; } }
`;

const s: Record<string, React.CSSProperties> = {
  page: { width: '100vw', minHeight: '100vh', overflowX: 'hidden', background: '#FAF8F5', color: '#2A2C2B', fontFamily: "'Plus Jakarta Sans', sans-serif", lineHeight: 1.6, position: 'relative', zIndex: 1, WebkitFontSmoothing: 'antialiased' },
  logoWrap: { display: 'flex', alignItems: 'center', gap: '8px', zIndex: 1 },
  logoLeaf: { fontSize: '1.3rem' },
  logoName: { fontFamily: "'Cormorant Garamond', serif", fontSize: '1.6rem', fontWeight: 600, fontStyle: 'italic', color: '#2A2C2B' },
  hero: { minHeight: '100vh', padding: '120px 6% 80px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '60px', maxWidth: '1300px', margin: '0 auto', flexWrap: 'wrap' },
  heroLeft: { flex: '1 1 420px', display: 'flex', flexDirection: 'column', gap: '22px' },
  eyebrow: { fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '2px', color: '#788574', fontWeight: 600 },
  heroHeading: { fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(2.8rem, 5vw, 4.2rem)', fontWeight: 300, lineHeight: 1.12, color: '#2A2C2B', margin: 0 },
  heroItalic: { fontStyle: 'italic', color: '#788574' },
  heroPara: { fontSize: '1rem', color: '#7E8280', lineHeight: 1.7, maxWidth: '460px', margin: 0 },
  heroNote: { fontSize: '0.78rem', color: '#7E8280', margin: 0 },
  breathCard: { flex: '0 0 auto', position: 'relative' },
  breathHint: { position: 'absolute', top: '14px', fontSize: '0.65rem', color: '#7E8280', opacity: 0.6, textTransform: 'lowercase', letterSpacing: '0.5px' },
  statsRow: { display: 'flex', justifyContent: 'center', alignItems: 'center', gap: '0', padding: '40px 6%', background: 'rgba(235,236,233,0.4)', borderTop: '1px solid rgba(42,44,43,0.06)', borderBottom: '1px solid rgba(42,44,43,0.06)', flexWrap: 'wrap' },
  statItem: { flex: '1 1 180px', textAlign: 'center', padding: '12px 20px' },
  statLabel: { fontSize: '0.78rem', color: '#7E8280', display: 'block', marginTop: '4px' },
  statDivider: { width: '1px', height: '40px', background: 'rgba(42,44,43,0.08)', flexShrink: 0 },
  section: { padding: '90px 6%', maxWidth: '1300px', margin: '0 auto' },
  sectionHead: { textAlign: 'center', marginBottom: '56px' },
  sectionEyebrow: { fontSize: '0.72rem', textTransform: 'uppercase', letterSpacing: '2px', color: '#788574', fontWeight: 600 },
  sectionHeading: { fontFamily: "'Cormorant Garamond', serif", fontSize: 'clamp(2rem, 3.5vw, 2.8rem)', fontWeight: 300, color: '#2A2C2B', margin: '8px 0 0' },
  sectionSub: { fontSize: '0.95rem', color: '#7E8280', marginTop: '8px' },
  featureGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '24px' },
  featureIcon: { fontSize: '1.5rem', display: 'block', marginBottom: '12px' },
  featureTitle: { fontFamily: "'Cormorant Garamond', serif", fontSize: '1.65rem', fontWeight: 400, color: '#2A2C2B', marginBottom: '10px' },
  featureBody: { fontSize: '0.9rem', color: '#7E8280', lineHeight: 1.65 },
  simulator: { maxWidth: '820px', margin: '0 auto', background: 'white', border: '1px solid rgba(42,44,43,0.08)', borderRadius: '24px', overflow: 'hidden', boxShadow: '0 20px 50px rgba(42,44,43,0.04)' },
  tabs: { display: 'flex', background: 'rgba(120,133,116,0.04)', borderBottom: '1px solid rgba(42,44,43,0.06)' },
  screenWrap: { padding: '36px', background: '#FAF8F5', display: 'flex', justifyContent: 'center' },
  screen: { width: '100%', maxWidth: '420px', background: 'white', border: '1px solid rgba(42,44,43,0.08)', borderRadius: '18px', overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: '420px', boxShadow: '0 8px 24px rgba(0,0,0,0.02)' },
  screenBar: { padding: '14px 18px', borderBottom: '1px solid rgba(42,44,43,0.05)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  onlineDot: { width: '7px', height: '7px', borderRadius: '50%', background: '#788574' },
  timer: { fontSize: '2.4rem', fontFamily: "'Plus Jakarta Sans', monospace", fontWeight: 300, color: '#2A2C2B' },
  transcript: { width: '100%', background: 'rgba(120,133,116,0.04)', border: '1px solid rgba(120,133,116,0.1)', borderRadius: '10px', padding: '12px' },
  bubbleUser: { alignSelf: 'flex-end', background: 'rgba(120,133,116,0.08)', border: '1px solid rgba(120,133,116,0.12)', borderRadius: '14px 14px 4px 14px', padding: '12px 14px', maxWidth: '86%' },
  bubbleAI: { alignSelf: 'flex-start', background: 'white', border: '1px solid rgba(42,44,43,0.07)', borderRadius: '14px 14px 14px 4px', padding: '14px 16px', maxWidth: '92%', boxShadow: '0 4px 12px rgba(0,0,0,0.015)' },
  bubbleLabel: { fontSize: '0.65rem', color: '#7E8280', fontWeight: 700, letterSpacing: '0.5px', display: 'block', marginBottom: '5px', textTransform: 'uppercase' },
  chartBox: { height: '170px', position: 'relative', background: 'rgba(250,248,245,0.6)', borderRadius: '10px', padding: '8px', border: '1px solid rgba(42,44,43,0.04)' },
  tooltip: { position: 'absolute', background: '#2A2C2B', color: 'white', padding: '5px 10px', borderRadius: '7px', fontSize: '0.72rem', pointerEvents: 'none', whiteSpace: 'nowrap', zIndex: 10, boxShadow: '0 4px 12px rgba(0,0,0,0.15)' },
  topicDesc: { padding: '10px 12px', background: 'rgba(120,133,116,0.05)', borderRadius: '8px', borderLeft: '2px solid #788574' } as React.CSSProperties,
  quoteWrap: { padding: '80px 6%', background: '#F3F1ED', borderTop: '1px solid rgba(42,44,43,0.06)', borderBottom: '1px solid rgba(42,44,43,0.06)' },
  quoteInner: { maxWidth: '680px', margin: '0 auto', textAlign: 'center', position: 'relative' },
  quoteOpen: { fontFamily: "'Cormorant Garamond', serif", fontSize: '5rem', color: 'rgba(120,133,116,0.18)', lineHeight: 0.5, display: 'block', marginBottom: '16px' },
  quoteText: { fontFamily: "'Cormorant Garamond', serif", fontSize: '1.75rem', fontWeight: 300, fontStyle: 'italic', color: '#2A2C2B', lineHeight: 1.45, marginBottom: '20px' },
  quoteAuthor: { fontSize: '0.83rem', color: '#7E8280', fontStyle: 'normal' },
  therapistSection: { padding: '100px 6%', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '80px', alignItems: 'center', maxWidth: '1300px', margin: '0 auto' },
  therapistLeft: { display: 'flex', flexDirection: 'column', gap: '0' },
  therapistRight: { display: 'flex', justifyContent: 'center' },
  bulletRow: { display: 'flex', gap: '12px', alignItems: 'flex-start', marginBottom: '14px' },
  checkmark: { color: '#788574', fontWeight: 'bold', marginTop: '2px', flexShrink: 0 },
  clinicCard: { width: '100%', maxWidth: '420px', background: 'white', border: '1px solid rgba(42,44,43,0.08)', borderRadius: '22px', padding: '26px', boxShadow: '0 20px 44px rgba(42,44,43,0.04)' },
  clinicCardHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', paddingBottom: '16px', borderBottom: '1px solid rgba(42,44,43,0.06)' },
  activeBadge: { fontSize: '0.7rem', background: 'rgba(120,133,116,0.1)', color: '#788574', padding: '3px 10px', borderRadius: '12px', fontWeight: 600 },
  clinicMetrics: { padding: '14px 0', display: 'flex', flexDirection: 'column', gap: '10px', borderBottom: '1px solid rgba(42,44,43,0.06)' },
  metricRow: { display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  moodBar: { width: '80px', height: '6px', background: 'rgba(42,44,43,0.07)', borderRadius: '3px', overflow: 'hidden' },
  moodFill: { height: '100%', background: '#788574', borderRadius: '3px' },
  clinicBrief: { paddingTop: '14px' },
  tagPill: { fontSize: '0.68rem', background: 'rgba(42,44,43,0.04)', color: '#7E8280', padding: '3px 10px', borderRadius: '10px', border: '1px solid rgba(42,44,43,0.05)' },
  faqSection: { padding: '90px 6%', maxWidth: '1100px', margin: '0 auto' },
  faqBtn: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%', background: 'transparent', border: 'none', cursor: 'pointer', padding: '0', gap: '16px' },
  faqPlus: { fontSize: '1.4rem', color: '#7E8280', flexShrink: 0, transition: 'transform 0.3s ease', display: 'inline-block' },
  ctaBanner: { padding: '90px 6%', textAlign: 'center', background: 'rgba(235,236,233,0.3)', borderTop: '1px solid rgba(42,44,43,0.06)' },
  footer: { padding: '56px 6% 36px', borderTop: '1px solid rgba(42,44,43,0.06)', background: '#FAF8F5' },
  footerInner: { display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', maxWidth: '1300px', margin: '0 auto', flexWrap: 'wrap', gap: '24px', paddingBottom: '28px', borderBottom: '1px solid rgba(42,44,43,0.06)' },
};
