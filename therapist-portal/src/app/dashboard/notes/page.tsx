'use client';

import { useCallback, useEffect, useRef, useState, type ChangeEvent } from 'react';
import { useRouter } from 'next/navigation';
import { getToken, notesApi, type ExternalClient, type ClientSession } from '../../../lib/api';
import PortalShell from '../../../components/PortalShell';
export default function SessionNotesPage() {
  const router = useRouter();
  const [consent, setConsent] = useState<boolean | null>(null),
    [clients, setClients] = useState<ExternalClient[]>([]),
    [selected, setSelected] = useState<ExternalClient | null>(null),
    [sessions, setSessions] = useState<ClientSession[]>([]),
    [session, setSession] = useState<ClientSession | null>(null),
    [text, setText] = useState(''),
    [dirty, setDirty] = useState(false),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(''),
    [status, setStatus] = useState(''),
    [newName, setNewName] = useState(''),
    [adding, setAdding] = useState(false),
    [sessionsLoading, setSessionsLoading] = useState(false);
  const alive = useRef(false),
    generation = useRef(0),
    lock = useRef(false),
    file = useRef<HTMLInputElement>(null);
  const initial = useCallback(async () => {
    setError('');
    setLoading(true);
    try {
      const [me, list] = await Promise.all([notesApi.me(), notesApi.listExternalClients()]);
      if (!alive.current) return;
      setConsent(me.client_consent_accepted);
      setClients(list);
      setSelected(current => current && list.some(c => c.id === current.id) ? current : list.find(c => !c.archived) || null);
    } catch {
      if (alive.current) setError('Could not load the notes workspace. Please try again.');
    } finally {
      if (alive.current) setLoading(false);
    }
  }, []);
  useEffect(() => {
    alive.current = true;
    if (!getToken()) {
      router.replace('/login');
      return;
    }
    void initial();
    return () => {
      alive.current = false;
      generation.current++;
    };
  }, [initial, router]);
  useEffect(() => {
    const run = ++generation.current;
    setSessions([]);
    setSession(null);
    setText('');
    setDirty(false);
    setSessionsLoading(!!selected);
    if (!selected) return;
    notesApi.listSessions({
      external_client_id: selected.id
    }).then(list => {
      if (!alive.current || run !== generation.current) return;
      setSessions(list);
      const first = list[0];
      if (first) {
        setSession(first);
        setText(first.bullets.join('\n'));
      }
    }).catch(() => {
      if (alive.current && run === generation.current) setError('Could not load sessions for this client. Select the client again to retry.');
    }).finally(() => {
      if (alive.current && run === generation.current) setSessionsLoading(false);
    });
  }, [selected]);
  useEffect(() => {
    if (!session || !['pending', 'processing'].includes(session.status)) return;
    let active = true;
    const current = session.id;
    const timer = setTimeout(async () => {
      try {
        const fresh = await notesApi.getSession(current);
        if (!active || !alive.current) return;
        setSession(fresh);
        setText(fresh.bullets.join('\n'));
        setSessions(list => list.map(s => s.id === fresh.id ? fresh : s));
      } catch {
        if (active && alive.current) setError('Extraction status is temporarily unavailable. Reopen the session to retry.');
      }
    }, 3000);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [session]);
  const canLeave = () => !dirty || confirm('Discard unsaved changes to these notes?');
  async function action(work: () => Promise<void>) {
    if (lock.current) return;
    lock.current = true;
    setBusy(true);
    setError('');
    setStatus('');
    try {
      await work();
    } catch {
      if (alive.current) setError('That action could not be completed. Your notes have not been discarded. Please try again.');
    } finally {
      lock.current = false;
      if (alive.current) setBusy(false);
    }
  }
  function update(s: ClientSession) {
    setSession(s);
    setText(s.bullets.join('\n'));
    setDirty(false);
    setSessions(list => list.some(x => x.id === s.id) ? list.map(x => x.id === s.id ? s : x) : [s, ...list]);
  }
  async function accept() {
    await action(async () => {
      const r = await notesApi.acceptConsent();
      if (alive.current) setConsent(r.client_consent_accepted);
    });
  }
  async function add() {
    if (!newName.trim() || !consent) return;
    await action(async () => {
      const c = await notesApi.createExternalClient(newName.trim());
      if (!alive.current) return;
      setClients(list => [c, ...list]);
      setSelected(c);
      setAdding(false);
      setNewName('');
    });
  }
  async function open(s: ClientSession) {
    if (!canLeave()) return;
    await action(async () => {
      const fresh = await notesApi.getSession(s.id);
      if (alive.current) update(fresh);
    });
  }
  async function save() {
    if (!consent || !selected || sessionsLoading) return;
    const bullets = text.split('\n').map(b => b.trim()).filter(Boolean);
    if (!bullets.length) {
      setError('Write at least one note before saving.');
      return;
    }
    await action(async () => {
      const saved = session ? await notesApi.updateBullets(session.id, bullets) : await notesApi.createSession({
        external_client_id: selected.id,
        bullets
      });
      if (alive.current) {
        update(saved);
        setStatus('Notes saved.');
      }
    });
  }
  async function upload(e: ChangeEvent<HTMLInputElement>) {
    const photo = e.target.files?.[0];
    if (!photo || !selected || !consent) return;
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(photo.type) || photo.size > 10 * 1024 * 1024) {
      setError('Choose a JPEG, PNG or WebP image under 10 MB.');
      e.target.value = '';
      return;
    }
    if (!canLeave()) {
      e.target.value = '';
      return;
    }
    await action(async () => {
      const presign = await notesApi.presignNote(photo.name, photo.type);
      const response = await fetch(presign.upload_url, {
        method: 'PUT',
        headers: {
          'Content-Type': photo.type
        },
        body: photo
      });
      if (!response.ok) throw new Error('upload failed');
      const created = await notesApi.createSession({
        external_client_id: selected.id,
        image_key: presign.image_key
      });
      if (alive.current) {
        update(created);
        setStatus('Photo uploaded. Extracting your notes…');
      }
    });
    if (file.current) file.current.value = '';
  }
  async function summarize() {
    if (!session) return;
    if (dirty) {
      setError('Save your edits before generating a summary.');
      return;
    }
    await action(async () => {
      const fresh = await notesApi.summarize(session.id);
      if (alive.current) {
        update(fresh);
        setStatus('Summary updated.');
      }
    });
  }
  async function remove() {
    if (!session || !confirm('Permanently delete these session notes?')) return;
    const id = session.id;
    await action(async () => {
      await notesApi.deleteSession(id);
      if (!alive.current) return;
      setSessions(list => list.filter(s => s.id !== id));
      setSession(null);
      setText('');
      setDirty(false);
      setStatus('Session notes deleted.');
    });
  }
  const processing = !!session && ['pending', 'processing'].includes(session.status);
  return <PortalShell title="Session notes" dirty={dirty}><div className="page-heading"><div><h1>Session notes.</h1><p>Bring your notes into a clear, editable record.</p></div><button className="btn small" disabled={!consent || busy || dirty} onClick={() => setAdding(!adding)}>Add client +</button></div>{error && <div className="notice" role="alert"><p className="error">{error}</p>{loading === false && consent === null && <button className="text-btn" onClick={() => void initial()}>Retry workspace</button>}</div>}{loading && <p role="status">Loading the notes workspace…</p>}{consent === false && <section className="consent-demo"><h2>Client data responsibility</h2><p>Before adding clients, confirm that you have their consent to store session notes, that you are responsible for the information you upload, and that you will use identifiers they are comfortable with. First names or initials are recommended.</p><button className="btn" disabled={busy} onClick={() => void accept()}>I understand and accept</button></section>}{adding && <form className="note-tools" onSubmit={e => {
      e.preventDefault();
      void add();
    }}><label htmlFor="new-client">Client name or initials</label><input id="new-client" value={newName} onChange={e => setNewName(e.target.value)} required disabled={busy} maxLength={200} /><button className="btn small" disabled={busy}>Add client</button><button className="text-btn" type="button" disabled={busy} onClick={() => setAdding(false)}>Cancel</button></form>}<div className="notes-layout"><aside><h2 style={{
          fontSize: 15
        }}>Clients</h2><div className="notes-clients">{clients.filter(c => !c.archived).map(c => <button className="note-client" key={c.id} aria-pressed={selected?.id === c.id} disabled={busy} onClick={() => {
            if (canLeave()) {
              setError('');
              setSelected({
                ...c
              });
            }
          }}>{c.name}<span className="subtext">{c.session_count} session{c.session_count === 1 ? '' : 's'}</span></button>)}</div>{!clients.length && !loading && <p className="small-note quiet">Add a client to start organising your session notes.</p>}</aside><section className="note-edit">{selected ? <><div className="page-heading"><div><h2 style={{
                marginBottom: 4
              }}>{selected.name}</h2><p>{session ? new Date(session.session_date).toLocaleDateString() : 'New session'}</p></div><span className="review-pill">{dirty ? 'Unsaved changes' : session ? processing ? 'Extracting…' : session.status === 'failed' ? 'Extraction failed' : 'Saved' : 'New note'}</span></div><div className="session-list" aria-label="Sessions">{sessions.map(s => <button key={s.id} className="session-choice" disabled={busy} aria-pressed={session?.id === s.id} onClick={() => void open(s)}>{new Date(s.session_date).toLocaleDateString()} · {s.status === 'completed' ? 'Ready' : s.status}</button>)}</div><div className="note-tools"><button className="text-btn" disabled={!consent || busy || sessionsLoading} onClick={() => {
              if (canLeave()) {
                setSession(null);
                setText('');
                setDirty(false);
                setStatus('');
              }
            }}>New session +</button>{session && <button className="text-btn" disabled={busy} onClick={() => void remove()}>Delete session</button>}</div>{session?.status === 'failed' && <p role="alert" className="error">We could not extract this image. You can upload another photo or type your notes below.</p>}<label htmlFor="note-text">Session bullets · one per line</label><textarea id="note-text" value={text} onChange={e => {
            setText(e.target.value);
            setDirty(true);
            setStatus('');
          }} disabled={!consent || busy || processing || sessionsLoading} placeholder="Write your session notes…" /><div className="actions"><button className="btn" disabled={!consent || busy || processing || sessionsLoading} onClick={() => void save()}>{busy ? 'Please wait…' : 'Save notes'}</button><button className="btn outline" disabled={!session || !consent || busy || processing || dirty || sessionsLoading} onClick={() => void summarize()}>Generate summary</button></div><div className="note-upload"><label htmlFor="note-photo">Upload a note photo</label><input ref={file} id="note-photo" type="file" accept="image/jpeg,image/png,image/webp" disabled={!consent || busy || processing || sessionsLoading} onChange={e => void upload(e)} /><p className="small-note quiet">JPEG, PNG or WebP · up to 10 MB. Photos are deleted after extraction. Review extracted text before saving changes.</p></div>{session?.summary && <section className="notice"><h3>AI session summary</h3><p>{session.summary}</p><p className="small-note">Check this interpretation against your original notes and professional judgment.</p></section>}</> : <p className="empty">{loading ? 'Loading…' : 'Select a client to view their sessions.'}</p>}{status && <p className="status-message" role="status">{status}</p>}</section></div></PortalShell>;
}
