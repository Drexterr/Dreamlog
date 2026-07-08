'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  notesApi,
  getToken,
  type ExternalClient,
  type ClientSession,
} from '../../../lib/api';
import PortalSidebar from '../../../components/PortalSidebar';

const STATUS_LABEL: Record<string, string> = {
  pending: 'Extracting…',
  processing: 'Extracting…',
  completed: 'Ready',
  failed: 'Failed',
};

// Session Notes workspace: manage external clients, upload photographed
// session notes (OCR → editable bullets), edit, and AI-summarize.
export default function SessionNotesPage() {
  const router = useRouter();
  const [consented, setConsented] = useState<boolean | null>(null);
  const [clients, setClients] = useState<ExternalClient[]>([]);
  const [selected, setSelected] = useState<ExternalClient | null>(null);
  const [sessions, setSessions] = useState<ClientSession[]>([]);
  const [openSession, setOpenSession] = useState<ClientSession | null>(null);
  const [bullets, setBullets] = useState<string[]>([]);
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [newClientName, setNewClientName] = useState('');
  const [showAddClient, setShowAddClient] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const pollTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const loadClients = useCallback(async () => {
    const list = await notesApi.listExternalClients().catch(() => []);
    setClients(list);
    return list;
  }, []);

  useEffect(() => {
    if (!getToken()) {
      router.replace('/login');
      return;
    }
    notesApi
      .me()
      .then(resp => setConsented(resp.client_consent_accepted))
      .catch(() => router.replace('/dashboard'));
    loadClients();
  }, [router, loadClients]);

  // Load sessions when a client is selected.
  useEffect(() => {
    if (!selected) {
      setSessions([]);
      return;
    }
    notesApi
      .listSessions({ external_client_id: selected.id })
      .then(setSessions)
      .catch(() => setSessions([]));
  }, [selected]);

  // Poll an open session while OCR runs.
  useEffect(() => {
    if (pollTimer.current) clearTimeout(pollTimer.current);
    if (!openSession || (openSession.status !== 'pending' && openSession.status !== 'processing')) return;
    pollTimer.current = setTimeout(async () => {
      const fresh = await notesApi.getSession(openSession.id).catch(() => null);
      if (fresh) {
        setOpenSession(fresh);
        setBullets(fresh.bullets);
        setSessions(prev => prev.map(s => (s.id === fresh.id ? fresh : s)));
      }
    }, 3000);
    return () => {
      if (pollTimer.current) clearTimeout(pollTimer.current);
    };
  }, [openSession]);

  const handleAcceptConsent = async () => {
    await notesApi.acceptConsent().catch(() => {});
    setConsented(true);
  };

  const handleAddClient = async () => {
    const name = newClientName.trim();
    if (!name) return;
    setBusy(true);
    try {
      await notesApi.createExternalClient(name);
      setNewClientName('');
      setShowAddClient(false);
      await loadClients();
    } catch {
      setError('Could not add client.');
    } finally {
      setBusy(false);
    }
  };

  const handleUpload = async (file: File) => {
    if (!selected) return;
    const allowed = ['image/jpeg', 'image/png', 'image/webp'];
    if (!allowed.includes(file.type)) {
      setError('Please upload a JPEG, PNG, or WebP photo.');
      return;
    }
    if (file.size > 10 * 1024 * 1024) {
      setError('Photo must be under 10 MB.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const { upload_url, image_key } = await notesApi.presignNote(file.name, file.type);
      const put = await fetch(upload_url, {
        method: 'PUT',
        headers: { 'Content-Type': file.type },
        body: file,
      });
      if (!put.ok) throw new Error('upload failed');
      const session = await notesApi.createSession({
        external_client_id: selected.id,
        image_key,
      });
      setSessions(prev => [session, ...prev]);
      setOpenSession(session);
      setBullets(session.bullets);
      setDirty(false);
    } catch {
      setError('Upload failed. Please try again.');
    } finally {
      setBusy(false);
      if (fileInput.current) fileInput.current.value = '';
    }
  };

  const openSessionDetail = async (s: ClientSession) => {
    const fresh = await notesApi.getSession(s.id).catch(() => s);
    setOpenSession(fresh);
    setBullets(fresh.bullets.length ? fresh.bullets : ['']);
    setDirty(false);
    setError('');
  };

  const handleSaveBullets = async () => {
    if (!openSession) return;
    const cleaned = bullets.map(b => b.trim()).filter(Boolean);
    if (!cleaned.length) {
      setError('Notes cannot be empty.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const updated = await notesApi.updateBullets(openSession.id, cleaned);
      setOpenSession(updated);
      setBullets(updated.bullets);
      setDirty(false);
      setSessions(prev => prev.map(s => (s.id === updated.id ? updated : s)));
    } catch {
      setError('Save failed.');
    } finally {
      setBusy(false);
    }
  };

  const handleSummarize = async () => {
    if (!openSession) return;
    if (dirty) {
      setError('Save your edits first.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const updated = await notesApi.summarize(openSession.id);
      setOpenSession(updated);
      setSessions(prev => prev.map(s => (s.id === updated.id ? updated : s)));
    } catch {
      setError('Summarization failed.');
    } finally {
      setBusy(false);
    }
  };

  const handleDeleteSession = async () => {
    if (!openSession) return;
    if (!confirm('Delete these session notes permanently?')) return;
    await notesApi.deleteSession(openSession.id).catch(() => {});
    setSessions(prev => prev.filter(s => s.id !== openSession.id));
    setOpenSession(null);
    loadClients();
  };

  return (
    <div className="flex min-h-screen" style={{ background: 'var(--bg, #0f0c1e)' }}>
      <PortalSidebar />
      <main className="flex-1 p-8 overflow-y-auto">
        <h1 className="text-2xl font-semibold mb-1" style={{ color: 'var(--text-primary, #fff)' }}>
          Session notes
        </h1>
        <p className="text-sm mb-6" style={{ color: 'var(--text-muted, #9ca3af)' }}>
          Photograph your handwritten notes — DreamLog extracts them into editable bullets and can
          summarize each session. Notes are encrypted at rest; photos are deleted after extraction.
        </p>

        {consented === false && (
          <div className="rounded-xl border border-purple-500/40 bg-purple-500/10 p-5 mb-6 max-w-2xl">
            <h2 className="font-semibold mb-2" style={{ color: 'var(--text-primary, #fff)' }}>
              Client data responsibility
            </h2>
            <p className="text-sm mb-4" style={{ color: 'var(--text-muted, #9ca3af)' }}>
              Before adding clients: you confirm you have your clients&apos; consent to store session
              notes, you are responsible for the information you upload, and you&apos;ll use
              identifiers your clients are comfortable with (first name or initials recommended).
            </p>
            <button
              onClick={handleAcceptConsent}
              className="rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-sm font-medium px-4 py-2"
            >
              I understand and accept
            </button>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-[280px_1fr] gap-6">
          {/* Client list */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-semibold uppercase tracking-wide" style={{ color: 'var(--text-muted, #9ca3af)' }}>
                Clients
              </h2>
              <button
                onClick={() => setShowAddClient(v => !v)}
                disabled={consented !== true}
                className="text-xs rounded-md bg-purple-600 hover:bg-purple-500 disabled:opacity-40 text-white px-3 py-1.5"
              >
                + Add
              </button>
            </div>
            {showAddClient && (
              <div className="mb-3 flex gap-2">
                <input
                  value={newClientName}
                  onChange={e => setNewClientName(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleAddClient()}
                  placeholder="First name or initials"
                  className="flex-1 rounded-md bg-white/5 border border-white/10 px-3 py-1.5 text-sm text-white placeholder:text-white/30"
                />
                <button onClick={handleAddClient} disabled={busy} className="text-xs rounded-md bg-purple-600 text-white px-3">
                  Save
                </button>
              </div>
            )}
            <ul className="space-y-2">
              {clients.map(c => (
                <li key={c.id}>
                  <button
                    onClick={() => {
                      setSelected(c);
                      setOpenSession(null);
                    }}
                    className={`w-full text-left rounded-lg border px-3 py-2.5 transition ${
                      selected?.id === c.id
                        ? 'border-purple-500 bg-purple-500/15'
                        : 'border-white/10 bg-white/5 hover:bg-white/10'
                    }`}
                  >
                    <span className="block text-sm font-medium text-white">{c.name}</span>
                    <span className="block text-xs text-white/40">
                      {c.session_count} session{c.session_count === 1 ? '' : 's'}
                    </span>
                  </button>
                </li>
              ))}
              {clients.length === 0 && (
                <li className="text-xs text-white/40 border border-dashed border-white/10 rounded-lg p-4">
                  No clients yet. Add your first client to start keeping notes.
                </li>
              )}
            </ul>
          </section>

          {/* Sessions + detail */}
          <section>
            {!selected ? (
              <div className="border border-dashed border-white/10 rounded-xl p-10 text-center text-sm text-white/40">
                Select a client to view or add session notes.
              </div>
            ) : (
              <>
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-lg font-medium text-white">{selected.name}</h2>
                  <div className="flex gap-2">
                    <input
                      ref={fileInput}
                      type="file"
                      accept="image/jpeg,image/png,image/webp"
                      className="hidden"
                      onChange={e => e.target.files?.[0] && handleUpload(e.target.files[0])}
                    />
                    <button
                      onClick={() => fileInput.current?.click()}
                      disabled={busy || consented !== true}
                      className="rounded-lg bg-purple-600 hover:bg-purple-500 disabled:opacity-40 text-white text-sm font-medium px-4 py-2"
                    >
                      {busy ? 'Working…' : '📷 Upload note photo'}
                    </button>
                  </div>
                </div>

                {error && <p className="text-sm text-red-400 mb-3">{error}</p>}

                <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
                  {/* Session list */}
                  <ul className="space-y-2">
                    {sessions.map(s => (
                      <li key={s.id}>
                        <button
                          onClick={() => openSessionDetail(s)}
                          className={`w-full text-left rounded-lg border px-4 py-3 transition ${
                            openSession?.id === s.id
                              ? 'border-purple-500 bg-purple-500/15'
                              : 'border-white/10 bg-white/5 hover:bg-white/10'
                          }`}
                        >
                          <div className="flex justify-between">
                            <span className="text-sm font-medium text-white">{s.session_date}</span>
                            <span
                              className={`text-xs ${
                                s.status === 'completed'
                                  ? 'text-emerald-400'
                                  : s.status === 'failed'
                                    ? 'text-red-400'
                                    : 'text-white/40'
                              }`}
                            >
                              {STATUS_LABEL[s.status]}
                            </span>
                          </div>
                          {s.bullets[0] && (
                            <span className="block text-xs text-white/50 mt-1 line-clamp-2">• {s.bullets[0]}</span>
                          )}
                        </button>
                      </li>
                    ))}
                    {sessions.length === 0 && (
                      <li className="text-xs text-white/40 border border-dashed border-white/10 rounded-lg p-4">
                        No sessions for {selected.name} yet. Upload a photo of your notes to create one.
                      </li>
                    )}
                  </ul>

                  {/* Session detail */}
                  {openSession && (
                    <div className="rounded-xl border border-white/10 bg-white/5 p-5">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-sm font-semibold text-white">{openSession.session_date}</h3>
                        <button onClick={handleDeleteSession} className="text-xs text-red-400 hover:text-red-300">
                          Delete
                        </button>
                      </div>

                      {(openSession.status === 'pending' || openSession.status === 'processing') && (
                        <p className="text-sm text-white/50">Reading your notes… this usually takes a few seconds.</p>
                      )}

                      {openSession.status === 'failed' && (
                        <p className="text-sm text-red-400">
                          Extraction failed: {openSession.error_msg || 'photo unreadable'}. Delete and retake the photo.
                        </p>
                      )}

                      {openSession.status === 'completed' && (
                        <>
                          {openSession.summary ? (
                            <div className="rounded-lg border border-purple-500/40 bg-purple-500/10 p-3 mb-4">
                              <p className="text-[10px] uppercase tracking-wider text-white/40 mb-1">✦ AI summary</p>
                              <p className="text-sm text-white/80 leading-relaxed">{openSession.summary}</p>
                            </div>
                          ) : (
                            <button
                              onClick={handleSummarize}
                              disabled={busy}
                              className="mb-4 rounded-lg border border-purple-500 text-white text-xs font-medium px-3 py-2 hover:bg-purple-500/10 disabled:opacity-40"
                            >
                              ✦ Summarize with AI
                            </button>
                          )}

                          <div className="space-y-2">
                            {bullets.map((b, i) => (
                              <div key={i} className="flex gap-2 items-start">
                                <span className="text-white/40 mt-2">•</span>
                                <textarea
                                  value={b}
                                  rows={2}
                                  onChange={e => {
                                    const v = e.target.value;
                                    setBullets(prev => prev.map((x, j) => (j === i ? v : x)));
                                    setDirty(true);
                                  }}
                                  className="flex-1 rounded-md bg-white/5 border border-white/10 px-3 py-2 text-sm text-white resize-y"
                                />
                                <button
                                  onClick={() => {
                                    setBullets(prev => prev.filter((_, j) => j !== i));
                                    setDirty(true);
                                  }}
                                  className="text-white/40 hover:text-red-400 mt-2"
                                >
                                  ✕
                                </button>
                              </div>
                            ))}
                            <button
                              onClick={() => {
                                setBullets(prev => [...prev, '']);
                                setDirty(true);
                              }}
                              className="text-xs text-white/50 hover:text-white"
                            >
                              + Add a note
                            </button>
                          </div>

                          {dirty && (
                            <button
                              onClick={handleSaveBullets}
                              disabled={busy}
                              className="mt-4 rounded-lg bg-purple-600 hover:bg-purple-500 disabled:opacity-40 text-white text-sm font-medium px-4 py-2"
                            >
                              Save changes
                            </button>
                          )}

                          {openSession.raw_text && (
                            <details className="mt-4">
                              <summary className="text-xs text-white/40 cursor-pointer">Original extracted text</summary>
                              <p className="text-xs text-white/50 whitespace-pre-wrap mt-2">{openSession.raw_text}</p>
                            </details>
                          )}
                        </>
                      )}
                    </div>
                  )}
                </div>
              </>
            )}
          </section>
        </div>
      </main>
    </div>
  );
}
