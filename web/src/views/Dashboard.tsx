import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { api } from '../services/api';
import type { ClientSummary, ClientBrief, UserGoal } from '../types';

export const Dashboard: React.FC = () => {
  const { logout, therapist } = useAuth();
  const [clients, setClients] = useState<ClientSummary[]>([]);
  const [activeIdx, setActiveIdx] = useState<number>(-1);
  const [brief, setBrief] = useState<ClientBrief | null>(null);
  
  // Loading states
  const [loadingClients, setLoadingClients] = useState(true);
  const [loadingBrief, setLoadingBrief] = useState(false);
  const [regenerating, setRegenerating] = useState(false);
  
  // Search state
  const [search, setSearch] = useState('');

  // Link Client modal state
  const [linkOpen, setLinkOpen] = useState(false);
  const [linkName, setLinkName] = useState('');
  const [linkUuid, setLinkUuid] = useState('');
  const [linkGoal, setLinkGoal] = useState<UserGoal>('anxiety');

  // Load client list on mount
  useEffect(() => {
    loadClients();
  }, []);

  // Fetch client details when active index changes
  useEffect(() => {
    if (activeIdx >= 0 && clients[activeIdx]) {
      loadClientBrief(clients[activeIdx].id);
    } else {
      setBrief(null);
    }
  }, [activeIdx, clients]);

  const loadClients = async () => {
    setLoadingClients(true);
    try {
      const data = await api.getClients();
      setClients(data);
      if (data.length > 0) {
        setActiveIdx(0);
      }
    } catch (err) {
      console.error('Failed to load clients', err);
    } finally {
      setLoadingClients(false);
    }
  };

  const loadClientBrief = async (id: string) => {
    setLoadingBrief(true);
    try {
      const data = await api.getClientBrief(id);
      setBrief(data);
    } catch (err) {
      console.error('Failed to load brief', err);
    } finally {
      setLoadingBrief(false);
    }
  };

  const handleRegenerateBrief = async () => {
    if (!brief) return;
    setRegenerating(true);
    try {
      const data = await api.regenerateClientBrief(brief.clientId);
      setBrief(data);
    } catch (err) {
      console.error(err);
    } finally {
      setRegenerating(false);
    }
  };

  const handleLinkClient = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.linkClient(linkUuid.trim(), linkName.trim(), linkGoal);
      setLinkOpen(false);
      setLinkName('');
      setLinkUuid('');
      // Reload lists
      await loadClients();
      setActiveIdx(clients.length); // Select the new client
    } catch (err) {
      alert('Failed to connect client. Verify client UUID format.');
    }
  };

  const handleUnlinkClient = async (id: string, name: string) => {
    const confirmVal = window.confirm(`Are you sure you want to unlink ${name}? This will revoke session summaries access.`);
    if (!confirmVal) return;
    
    try {
      await api.unlinkClient(id);
      const updated = clients.filter(c => c.id !== id);
      setClients(updated);
      setActiveIdx(updated.length > 0 ? 0 : -1);
    } catch (err) {
      alert('Unlinking failed.');
    }
  };

  // Helper: Mood Score Color Codes
  const getMoodColor = (score: number) => {
    if (score >= 70) return '#10b981'; // Green
    if (score >= 50) return '#eab308'; // Yellow
    if (score >= 40) return '#f97316'; // Orange
    return '#ef4444'; // Red
  };

  // Dynamic SVG Chart Path Generator
  const generateChartPath = (entries: any[]) => {
    if (!entries || entries.length === 0) return { line: '', area: '', points: [] };
    
    const width = 300;
    const height = 120;
    const padding = 15;
    const coords: { x: number; y: number; val: number }[] = [];
    
    const stepX = (width - padding * 2) / Math.max(entries.length - 1, 1);
    
    // Map entries (oldest to newest) to points
    const reversed = [...entries].reverse();
    reversed.forEach((entry, idx) => {
      const x = padding + idx * stepX;
      const y = height - padding - ((entry.moodScore / 100) * (height - padding * 2));
      coords.push({ x, y, val: entry.moodScore });
    });

    if (coords.length === 1) {
      // Draw flat line for single entry
      const p = coords[0];
      return {
        line: `M ${p.x - 20} ${p.y} L ${p.x + 20} ${p.y}`,
        area: `M ${p.x - 20} ${p.y} L ${p.x + 20} ${p.y} L ${p.x + 20} ${height - padding} L ${p.x - 20} ${height - padding} Z`,
        points: coords,
      };
    }

    // Build smooth cubic bezier curve
    let lineCmd = `M ${coords[0].x} ${coords[0].y}`;
    for (let i = 0; i < coords.length - 1; i++) {
      const p0 = coords[i];
      const p1 = coords[i + 1];
      const cpX1 = p0.x + stepX / 2;
      const cpY1 = p0.y;
      const cpX2 = p1.x - stepX / 2;
      const cpY2 = p1.y;
      lineCmd += ` C ${cpX1} ${cpY1}, ${cpX2} ${cpY2}, ${p1.x} ${p1.y}`;
    }

    let areaCmd = lineCmd;
    areaCmd += ` L ${coords[coords.length - 1].x} ${height - padding}`;
    areaCmd += ` L ${coords[0].x} ${height - padding} Z`;

    return { line: lineCmd, area: areaCmd, points: coords };
  };

  const filteredClients = clients.filter(c => c.name.toLowerCase().includes(search.toLowerCase()));
  const activeClient = activeIdx >= 0 ? clients[activeIdx] : null;
  const chart = brief ? generateChartPath(brief.recentEntries) : { line: '', area: '', points: [] };

  return (
    <div style={styles.dashboardLayout}>
      
      {/* Sidebar navigation */}
      <aside style={styles.sidebar}>
        <div>
          <div style={styles.brand}>
            <div style={styles.brandLogo}>D</div>
            <span style={styles.brandName}>DreamLog</span>
            <span style={styles.brandTag}>Clinic</span>
          </div>

          <div style={styles.navGroup}>
            <div style={{ ...styles.navItem, ...styles.navItemActive }}>
              <span style={styles.navIcon}>📊</span>
              <span>Client Analytics</span>
            </div>
            <div style={styles.navItem}>
              <span style={styles.navIcon}>👥</span>
              <span>Manage Clients</span>
            </div>
            <div style={styles.navItem}>
              <span style={styles.navIcon}>📁</span>
              <span>Shared Reports</span>
            </div>
            <div style={styles.navItem} onClick={logout}>
              <span style={styles.navIcon}>🚪</span>
              <span>Log Out</span>
            </div>
          </div>
        </div>

        {therapist && (
          <div style={styles.profile}>
            <div style={styles.avatar}>{therapist.name.split(' ').map(n => n[0]).join('').slice(0, 2)}</div>
            <div style={styles.profileInfo}>
              <span style={styles.profileName}>{therapist.name}</span>
              <span style={styles.profileSub}>{therapist.credentials || 'Licensed Therapist'}</span>
            </div>
          </div>
        )}
      </aside>

      {/* Main Viewport */}
      <div style={styles.viewport}>
        
        {/* Top Header */}
        <header style={styles.header}>
          <h1 style={styles.headerTitle}>Clinical Insights Dashboard</h1>
          <button style={styles.btn} onClick={() => setLinkOpen(true)}>
            <span style={{ fontSize: '1.2rem', fontWeight: 600 }}>+</span> Link New Client
          </button>
        </header>

        {/* Split Workspaces */}
        <div style={styles.splitSpace}>
          
          {/* Client List column */}
          <section style={styles.clientPane}>
            <div style={styles.searchWrap}>
              <span style={styles.searchIcon}>🔍</span>
              <input
                type="text"
                placeholder="Search active clients..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                style={styles.searchInput}
              />
            </div>

            <div style={styles.clientList}>
              {loadingClients ? (
                <div style={styles.paneLoading}>Loading client roster...</div>
              ) : filteredClients.length === 0 ? (
                <div style={styles.paneLoading}>No active connections.</div>
              ) : (
                filteredClients.map((client, idx) => {
                  const absoluteIdx = clients.findIndex(c => c.id === client.id);
                  const isSelected = absoluteIdx === activeIdx;
                  return (
                    <div
                      key={client.id}
                      onClick={() => setActiveIdx(absoluteIdx)}
                      style={{
                        ...styles.clientCard,
                        ...(isSelected ? styles.clientCardActive : {}),
                      }}
                    >
                      <div style={styles.cardHeader}>
                        <span style={styles.cardName}>{client.name}</span>
                        <span style={{ ...styles.goalTag, ...styles[`tag_${client.goal}`] }}>
                          {client.goal}
                        </span>
                      </div>
                      <div style={styles.cardMeta}>
                        <div style={styles.cardMood}>
                          <span
                            style={{
                              ...styles.moodDot,
                              backgroundColor: getMoodColor(client.moodScore),
                            }}
                          />
                          <span>7d Mood: {client.moodScore}%</span>
                        </div>
                        <span>{client.entryCount} logs</span>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </section>

          {/* Client Details viewport */}
          <section style={styles.detailPane}>
            {loadingBrief ? (
              <div style={styles.centerLoading}>
                <span style={styles.spinnerLarge} />
                <p style={{ marginTop: '16px', color: '#64748b' }}>Pulling clinical data summaries...</p>
              </div>
            ) : !activeClient || !brief ? (
              <div style={styles.centerLoading}>
                <p style={{ fontSize: '1.2rem', color: '#64748b' }}>Select a client to view insights</p>
              </div>
            ) : (
              <div>
                {/* Banner */}
                <div style={styles.banner}>
                  <div style={styles.bannerLeft}>
                    <div style={styles.bannerAvatar}>
                      {activeClient.name.split(' ').map(n => n[0]).join('').slice(0, 2)}
                    </div>
                    <div>
                      <h2 style={styles.bannerName}>{activeClient.name}</h2>
                      <p style={styles.bannerSub}>Linked connection since {new Date(activeClient.linkedAt).toLocaleDateString()}</p>
                    </div>
                  </div>
                  <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                    <span style={{ ...styles.goalTagLarge, ...styles[`tag_${activeClient.goal}`] }}>
                      {activeClient.goal} focus
                    </span>
                    <button
                      onClick={() => handleUnlinkClient(activeClient.id, activeClient.name)}
                      style={styles.unlinkBtn}
                    >
                      Unlink
                    </button>
                  </div>
                </div>

                {/* Dashboard Grid Widgets */}
                <div style={styles.grid}>
                  
                  {/* AI Pre-Session Brief */}
                  <div style={{ ...styles.panel, gridColumn: 'span 2' }}>
                    {regenerating && (
                      <div style={styles.panelLoading}>
                        <span style={styles.spinner} />
                        <span style={{ color: '#cbd5e1', fontSize: '0.8rem', letterSpacing: '0.5px' }}>GENERATING PRE-SESSION BRIEF...</span>
                      </div>
                    )}
                    <div style={styles.panelHeader}>
                      <h3 style={styles.panelTitle}>
                        <span style={{ marginRight: '8px' }}>🪄</span> Claude Pre-Session Brief
                      </h3>
                      <button onClick={handleRegenerateBrief} style={styles.regenerateBtn}>
                        🔄 Re-generate
                      </button>
                    </div>
                    
                    <p style={styles.briefText}>{brief.brief}</p>
                    
                    {brief.recentEntries.length > 0 && (
                      <div style={styles.sessionBox}>
                        <h4 style={styles.sessionTitle}>Suggested Session Openers</h4>
                        <div style={styles.sessionList}>
                          <div style={styles.sessionPrompt}>
                            <span style={styles.bullet}>•</span>
                            <span>"In your recent logs, you spoke about feeling isolated during team standups. What does it feel like to hold that internally while peers are celebrating metrics?"</span>
                          </div>
                          <div style={styles.sessionPrompt}>
                            <span style={styles.bullet}>•</span>
                            <span>"You noted that sitting in the dark was a small somatic relief. Let's explore how to integrate more conscious decompression windows into your weekly schedule."</span>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Mood Chart Widget */}
                  <div style={styles.panel}>
                    <div style={styles.panelHeader}>
                      <h3 style={styles.panelTitle}>
                        <span style={{ marginRight: '8px' }}>📈</span> Mood Trajectory (Recent Logs)
                      </h3>
                    </div>
                    
                    <div style={styles.metricRow}>
                      <div style={styles.metricCard}>
                        <div style={{ ...styles.metricVal, color: getMoodColor(brief.avgMood7d || 50) }}>
                          {brief.avgMood7d ? `${brief.avgMood7d}%` : 'N/A'}
                        </div>
                        <div style={styles.metricLabel}>7d Avg Mood</div>
                      </div>
                      <div style={styles.metricCard}>
                        <div style={styles.metricVal}>
                          {activeClient.goal === 'anxiety' ? 'High' : 'Normal'}
                        </div>
                        <div style={styles.metricLabel}>Vocal Tension</div>
                      </div>
                      <div style={styles.metricCard}>
                        <div style={styles.metricVal}>
                          {activeClient.goal === 'anxiety' ? '145wpm' : '110wpm'}
                        </div>
                        <div style={styles.metricLabel}>Speech pace</div>
                      </div>
                    </div>

                    <div style={styles.chartWrapper}>
                      {chart.points.length > 0 ? (
                        <svg viewBox="0 0 300 120" style={{ width: '100%', height: '100%' }}>
                          <defs>
                            <linearGradient id="react-chart-grad" x1="0" y1="0" x2="0" y2="1">
                              <stop offset="0%" stopColor="#00b4d8" stopOpacity={0.25} />
                              <stop offset="100%" stopColor="#00b4d8" stopOpacity={0.0} />
                            </linearGradient>
                          </defs>
                          <line x1="0" y1="20" x2="300" y2="20" stroke="rgba(255,255,255,0.04)" />
                          <line x1="0" y1="60" x2="300" y2="60" stroke="rgba(255,255,255,0.04)" />
                          <line x1="0" y1="100" x2="300" y2="100" stroke="rgba(255,255,255,0.04)" />
                          
                          <path d={chart.area} fill="url(#react-chart-grad)" />
                          <path d={chart.line} fill="none" stroke="#00b4d8" strokeWidth={2.5} strokeLinecap="round" />
                          
                          {chart.points.map((p, idx) => (
                            <circle
                              key={idx}
                              cx={p.x}
                              cy={p.y}
                              r={4}
                              fill="#0b1329"
                              stroke="#00b4d8"
                              strokeWidth={1.5}
                              style={{ cursor: 'pointer' }}
                            />
                          ))}
                        </svg>
                      ) : (
                        <div style={{ ...styles.paneLoading, height: '100%' }}>No trajectory scores.</div>
                      )}
                    </div>
                  </div>

                  {/* Emotion Breakdown */}
                  <div style={styles.panel}>
                    <div style={styles.panelHeader}>
                      <h3 style={styles.panelTitle}>
                        <span style={{ marginRight: '8px' }}>🎭</span> Dominant Emotions
                      </h3>
                    </div>
                    <div style={styles.emotionList}>
                      {brief.topEmotions.length > 0 ? (
                        brief.topEmotions.map((emotion, idx) => {
                          const value = 80 - idx * 25; // Dummy percentages for visualization
                          return (
                            <div key={emotion} style={styles.emotionItem}>
                              <div style={styles.emotionInfo}>
                                <span style={styles.emotionName}>{emotion}</span>
                                <span style={styles.emotionPct}>{value}%</span>
                              </div>
                              <div style={styles.emotionTrack}>
                                <div
                                  style={{
                                    ...styles.emotionBar,
                                    width: `${value}%`,
                                    background: idx === 0 ? '#00b4d8' : idx === 1 ? '#7B9E87' : '#64748b',
                                  }}
                                />
                              </div>
                            </div>
                          );
                        })
                      ) : (
                        <div style={styles.paneLoading}>No emotional descriptors.</div>
                      )}
                    </div>
                  </div>

                  {/* Timeline Logs */}
                  <div style={{ ...styles.panel, gridColumn: 'span 2' }}>
                    <div style={styles.panelHeader}>
                      <h3 style={styles.panelTitle}>
                        <span style={{ marginRight: '8px' }}>⏳</span> Recent Log Details
                      </h3>
                    </div>
                    
                    <div style={styles.timeline}>
                      {brief.recentEntries.map(entry => (
                        <div key={entry.id} style={styles.timeCard}>
                          <div style={styles.timeCardHeader}>
                            <div style={styles.timeCardMeta}>
                              <span style={{ ...styles.moodDot, backgroundColor: getMoodColor(entry.moodScore) }} />
                              <span style={{ fontWeight: 600, color: '#cbd5e1' }}>Mood score: {entry.moodScore}</span>
                              <span style={{ color: '#64748b' }}>•</span>
                              <span style={{ color: '#64748b' }}>{entry.recordedAt}</span>
                            </div>
                          </div>
                          <p style={styles.timeCardSummary}>{entry.summary}</p>
                          {entry.keyQuotes.length > 0 && (
                            <p style={styles.timeCardQuote}>"{entry.keyQuotes[0]}"</p>
                          )}
                          <div style={styles.topicRow}>
                            {entry.topics.map(t => (
                              <span key={t} style={styles.topicBadge}>{t}</span>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                </div>
              </div>
            )}
          </section>

        </div>
      </div>

      {/* Link Client Modal */}
      {linkOpen && (
        <div style={styles.modalBackdrop}>
          <div style={styles.modal}>
            <div style={styles.modalHeader}>
              <h3 style={styles.modalTitle}>Link New Client</h3>
              <button style={styles.modalClose} onClick={() => setLinkOpen(false)}>✕</button>
            </div>
            
            <form onSubmit={handleLinkClient} style={styles.form}>
              <div style={styles.formGroup}>
                <label style={styles.label}>Client Name</label>
                <input
                  type="text"
                  placeholder="e.g. Elena Rostova"
                  value={linkName}
                  onChange={(e) => setLinkName(e.target.value)}
                  required
                  style={styles.input}
                />
              </div>

              <div style={styles.formGroup}>
                <label style={styles.label}>Client Secure UUID</label>
                <input
                  type="text"
                  placeholder="550e8400-e29b-41d4-a716-446655440000"
                  value={linkUuid}
                  onChange={(e) => setLinkUuid(e.target.value)}
                  required
                  style={styles.input}
                />
              </div>

              <div style={styles.formGroup}>
                <label style={styles.label}>Primary Goal Focus</label>
                <select
                  value={linkGoal}
                  onChange={(e) => setLinkGoal(e.target.value as UserGoal)}
                  style={styles.select}
                >
                  <option value="anxiety">Manage Anxiety 🌱</option>
                  <option value="stress">Decompress Stress 🌊</option>
                  <option value="grief">Navigate Grief & Loss 🕊️</option>
                  <option value="depression">Lift Low Mood ☀️</option>
                  <option value="relationships">Relationships & Loneliness ❤️</option>
                  <option value="career">Career & Purpose 🌲</option>
                  <option value="trauma">Process Past / Trauma 🩹</option>
                  <option value="curious">Self-Discovery 🌌</option>
                </select>
              </div>

              <button type="submit" style={styles.submitBtn}>
                Confirm Connection
              </button>
            </form>
          </div>
        </div>
      )}

    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  dashboardLayout: {
    display: 'flex',
    width: '100vw',
    height: '100vh',
    background: '#0b1329',
    overflow: 'hidden',
  },
  sidebar: {
    width: '260px',
    background: 'rgba(11, 19, 43, 0.85)',
    borderRight: '1px solid rgba(255,255,255,0.08)',
    padding: '24px',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'space-between',
    backdropFilter: 'blur(20px)',
    zIndex: 10,
  },
  brand: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    marginBottom: '32px',
  },
  brandLogo: {
    width: '32px',
    height: '32px',
    borderRadius: '8px',
    background: 'linear-gradient(135deg, #00b4d8 0%, #0077b6 100%)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontFamily: "'Outfit', sans-serif",
    fontWeight: 700,
    color: '#fff',
  },
  brandName: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '1.25rem',
    fontWeight: 600,
    color: '#f8fafc',
  },
  brandTag: {
    fontSize: '0.65rem',
    textTransform: 'uppercase',
    letterSpacing: '1px',
    color: '#00b4d8',
    background: 'rgba(0, 180, 216, 0.1)',
    padding: '2px 6px',
    borderRadius: '4px',
    fontWeight: 700,
    marginLeft: 'auto',
  },
  navGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  navItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    padding: '12px 16px',
    borderRadius: '10px',
    color: '#cbd5e1',
    fontWeight: 500,
    fontSize: '0.9rem',
    cursor: 'pointer',
    transition: 'all 0.2s ease',
  },
  navItemActive: {
    background: 'rgba(0, 180, 216, 0.1)',
    border: '1px solid rgba(0, 180, 216, 0.2)',
    color: '#00b4d8',
  },
  navIcon: {
    fontSize: '1.1rem',
  },
  profile: {
    borderTop: '1px solid rgba(255, 255, 255, 0.08)',
    paddingTop: '20px',
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  avatar: {
    width: '40px',
    height: '40px',
    borderRadius: '50%',
    background: '#3a506b',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontWeight: 600,
    color: '#fff',
    border: '2px solid rgba(255,255,255,0.08)',
  },
  profileInfo: {
    display: 'flex',
    flexDirection: 'column',
    gap: '2px',
  },
  profileName: {
    fontSize: '0.85rem',
    fontWeight: 600,
    color: '#f8fafc',
  },
  profileSub: {
    fontSize: '0.72rem',
    color: '#64748b',
  },
  viewport: {
    flex: 1,
    height: '100vh',
    display: 'flex',
    flexDirection: 'column',
  },
  header: {
    height: '70px',
    borderBottom: '1px solid rgba(255,255,255,0.08)',
    padding: '0 32px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    background: 'rgba(11, 19, 43, 0.4)',
    backdropFilter: 'blur(10px)',
  },
  headerTitle: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '1.35rem',
    fontWeight: 500,
    color: '#f8fafc',
  },
  btn: {
    background: '#00b4d8',
    color: '#0b1329',
    border: 'none',
    padding: '10px 18px',
    borderRadius: '10px',
    fontWeight: 700,
    fontSize: '0.85rem',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    boxShadow: '0 4px 12px rgba(0, 180, 216, 0.2)',
  },
  splitSpace: {
    flex: 1,
    display: 'flex',
    overflow: 'hidden',
  },
  clientPane: {
    width: '320px',
    borderRight: '1px solid rgba(255,255,255,0.08)',
    background: 'rgba(28, 37, 65, 0.25)',
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
  },
  searchWrap: {
    padding: '20px',
    borderBottom: '1px solid rgba(255,255,255,0.08)',
    position: 'relative',
    display: 'flex',
    alignItems: 'center',
  },
  searchIcon: {
    position: 'absolute',
    left: 32px;
    color: '#64748b',
    fontSize: '0.85rem',
  },
  searchInput: {
    width: '100%',
    background: 'rgba(11, 19, 43, 0.6)',
    border: '1px solid rgba(255,255,255,0.08)',
    borderRadius: '10px',
    padding: '10px 12px 10px 36px',
    color: '#f8fafc',
    fontFamily: 'inherit',
    fontSize: '0.85rem',
    outline: 'none',
  },
  clientList: {
    flex: 1,
    overflowY: 'auto',
    padding: '12px',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  paneLoading: {
    padding: '24px',
    textAlign: 'center',
    color: '#64748b',
    fontSize: '0.85rem',
  },
  clientCard: {
    background: 'rgba(28, 37, 65, 0.55)',
    border: '1px solid rgba(255,255,255,0.08)',
    borderRadius: '14px',
    padding: '16px',
    cursor: 'pointer',
    transition: 'all 0.2s ease',
    display: 'flex',
    flexDirection: 'column',
    gap: '10px',
  },
  clientCardActive: {
    background: 'rgba(0, 180, 216, 0.08)',
    borderColor: '#00b4d8',
  },
  cardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  cardName: {
    fontSize: '0.9rem',
    fontWeight: 600,
    color: '#f8fafc',
  },
  goalTag: {
    fontSize: '0.65rem',
    fontWeight: 700,
    padding: '2px 8px',
    borderRadius: '20px',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
  },
  cardMeta: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    fontSize: '0.72rem',
    color: '#64748b',
  },
  cardMood: {
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
  },
  moodDot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
    display: 'inline-block',
  },
  detailPane: {
    flex: 1,
    overflowY: 'auto',
    padding: '32px',
  },
  centerLoading: {
    height: '100%',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
  },
  banner: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    background: 'linear-gradient(135deg, rgba(28, 37, 65, 0.8) 0%, rgba(11, 19, 43, 0.8) 100%)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: '20px',
    padding: '24px',
    marginBottom: '24px',
  },
  bannerLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: '20px',
  },
  bannerAvatar: {
    width: '60px',
    height: '60px',
    borderRadius: '50%',
    background: 'rgba(0, 180, 216, 0.15)',
    border: '2px solid #00b4d8',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '1.4rem',
    fontWeight: 600,
    color: '#00b4d8',
    fontFamily: "'Outfit', sans-serif",
  },
  bannerName: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '1.4rem',
    fontWeight: 500,
    color: '#f8fafc',
    marginBottom: '4px',
  },
  bannerSub: {
    fontSize: '0.8rem',
    color: '#64748b',
  },
  goalTagLarge: {
    fontSize: '0.7rem',
    fontWeight: 700,
    padding: '4px 12px',
    borderRadius: '20px',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
  },
  unlinkBtn: {
    background: 'transparent',
    border: '1px solid rgba(239, 68, 68, 0.4)',
    color: '#ef4444',
    padding: '8px 14px',
    borderRadius: '10px',
    fontSize: '0.8rem',
    fontWeight: 600,
    cursor: 'pointer',
    transition: 'all 0.2s ease',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(2, 1fr)',
    gap: '24px',
  },
  panel: {
    background: 'rgba(28, 37, 65, 0.55)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: '20px',
    padding: '24px',
    backdropFilter: 'blur(20px)',
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
    position: 'relative',
    overflow: 'hidden',
  },
  panelHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    borderBottom: '1px solid rgba(255, 255, 255, 0.08)',
    paddingBottom: '12px',
  },
  panelTitle: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '1rem',
    fontWeight: 600,
    color: '#f8fafc',
  },
  regenerateBtn: {
    background: 'transparent',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    color: '#cbd5e1',
    padding: '6px 12px',
    borderRadius: '8px',
    fontSize: '0.75rem',
    fontWeight: 500,
    cursor: 'pointer',
  },
  briefText: {
    fontFamily: "'Cormorant Garamond', serif",
    fontSize: '1.25rem',
    lineHeight: 1.5,
    color: '#f8fafc',
  },
  sessionBox: {
    background: 'rgba(11, 19, 43, 0.35)',
    border: '1px dashed rgba(255,255,255,0.08)',
    borderRadius: '12px',
    padding: '16px',
  },
  sessionTitle: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '0.8rem',
    fontWeight: 600,
    color: '#00b4d8',
    textTransform: 'uppercase',
    marginBottom: '10px',
  },
  sessionList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  sessionPrompt: {
    display: 'flex',
    gap: '10px',
    fontSize: '0.85rem',
    color: '#cbd5e1',
    lineHeight: 1.4,
  },
  bullet: {
    color: '#00b4d8',
    fontWeight: 'bold',
  },
  metricRow: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: '12px',
  },
  metricCard: {
    background: 'rgba(11, 19, 43, 0.4)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: '12px',
    padding: '12px',
    textAlign: 'center',
  },
  metricVal: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '1.5rem',
    fontWeight: 300,
    marginBottom: '2px',
    color: '#f8fafc',
  },
  metricLabel: {
    fontSize: '0.7rem',
    color: '#64748b',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
  },
  chartWrapper: {
    height: '140px',
    width: '100%',
  },
  emotionList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '14px',
  },
  emotionItem: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  emotionInfo: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: '0.8rem',
  },
  emotionName: {
    fontWeight: 500,
    color: '#cbd5e1',
  },
  emotionPct: {
    color: '#64748b',
  },
  emotionTrack: {
    height: '6px',
    background: 'rgba(255,255,255,0.05)',
    borderRadius: '4px',
    overflow: 'hidden',
  },
  emotionBar: {
    height: '100%',
    borderRadius: '4px',
    transition: 'width 0.8s ease',
  },
  timeline: {
    display: 'flex',
    flexDirection: 'column',
    gap: '12px',
  },
  timeCard: {
    background: 'rgba(11, 19, 43, 0.3)',
    border: '1px solid rgba(255,255,255,0.08)',
    borderRadius: '12px',
    padding: '14px',
  },
  timeCardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '6px',
  },
  timeCardMeta: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '0.72rem',
  },
  timeCardSummary: {
    fontSize: '0.85rem',
    lineHeight: 1.45;
    color: '#cbd5e1',
    marginBottom: '8px',
  },
  timeCardQuote: {
    fontFamily: "'Cormorant Garamond', serif",
    fontStyle: 'italic',
    fontSize: '1.05rem',
    borderLeft: '2px solid #00b4d8',
    paddingLeft: '10px',
    color: '#f8fafc',
    marginBottom: '10px',
  },
  topicRow: {
    display: 'flex',
    gap: '6px',
    flexWrap: 'wrap',
  },
  topicBadge: {
    background: 'rgba(255,255,255,0.04)',
    borderRadius: '6px',
    padding: '2px 6px',
    fontSize: '0.65rem',
    color: '#cbd5e1',
  },
  modalBackdrop: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'rgba(11,19,43,0.75)',
    backdropFilter: 'blur(8px)',
    zIndex: 100,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  modal: {
    background: '#1c2541',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: '20px',
    padding: '32px',
    width: '100%',
    maxWidth: '440px',
    boxShadow: '0 24px 60px rgba(0, 0, 0, 0.3)',
  },
  modalHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
  },
  modalTitle: {
    fontFamily: "'Outfit', sans-serif",
    fontSize: '1.2rem',
    fontWeight: 600,
    color: '#f8fafc',
  },
  modalClose: {
    background: 'transparent',
    border: 'none',
    color: '#64748b',
    fontSize: '1.1rem',
    cursor: 'pointer',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '14px',
  },
  formGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  label: {
    fontSize: '0.75rem',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
    color: '#cbd5e1',
    fontWeight: 600,
  },
  input: {
    background: 'rgba(11, 19, 43, 0.6)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: '10px',
    padding: '12px',
    color: '#f8fafc',
    fontFamily: 'inherit',
    fontSize: '0.9rem',
    outline: 'none',
  },
  select: {
    background: '#1c2541',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    borderRadius: '10px',
    padding: '12px',
    color: '#f8fafc',
    fontFamily: 'inherit',
    fontSize: '0.9rem',
    outline: 'none',
  },
  submitBtn: {
    background: '#00b4d8',
    border: 'none',
    color: '#0b1329',
    padding: '12px',
    borderRadius: '10px',
    fontFamily: 'inherit',
    fontSize: '0.9rem',
    fontWeight: 700,
    cursor: 'pointer',
    marginTop: '8px',
  },
  panelLoading: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'rgba(28, 37, 65, 0.85)',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '12px',
    zIndex: 5,
  },
  spinner: {
    width: '24px',
    height: '24px',
    border: '2px solid rgba(255,255,255,0.1)',
    borderTopColor: '#00b4d8',
    borderRadius: '50%',
    animation: 'spin 1s infinite linear',
  },
  spinnerLarge: {
    width: '36px',
    height: '36px',
    border: '3px solid rgba(255,255,255,0.1)',
    borderTopColor: '#00b4d8',
    borderRadius: '50%',
    animation: 'spin 1s infinite linear',
  },
  
  // Specific tag style mappings
  tag_anxiety: { background: 'rgba(123, 158, 135, 0.15)', color: '#7B9E87', border: '1px solid rgba(123, 158, 135, 0.25)' },
  tag_stress: { background: 'rgba(91, 141, 184, 0.15)', color: '#5B8DB8', border: '1px solid rgba(91, 141, 184, 0.25)' },
  tag_grief: { background: 'rgba(143, 168, 196, 0.15)', color: '#8FA8C4', border: '1px solid rgba(143, 168, 196, 0.25)' },
  tag_depression: { background: 'rgba(200, 150, 90, 0.15)', color: '#C8965A', border: '1px solid rgba(200, 150, 90, 0.25)' },
  tag_relationships: { background: 'rgba(200, 127, 127, 0.15)', color: '#C87F7F', border: '1px solid rgba(200, 127, 127, 0.25)' },
  tag_career: { background: 'rgba(92, 122, 98, 0.15)', color: '#5C7A62', border: '1px solid rgba(92, 122, 98, 0.25)' },
  tag_trauma: { background: 'rgba(160, 144, 128, 0.15)', color: '#A09080', border: '1px solid rgba(160, 144, 128, 0.25)' },
  tag_curious: { background: 'rgba(123, 111, 160, 0.15)', color: '#7B6FA0', border: '1px solid rgba(123, 111, 160, 0.25)' },
};
