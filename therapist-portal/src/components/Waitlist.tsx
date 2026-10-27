'use client';

import { useEffect, useRef, useState } from 'react';

export default function Waitlist() {
  const dialog = useRef<HTMLDialogElement>(null);
  const request = useRef<AbortController | null>(null);
  const opener = useRef<HTMLElement | null>(null);
  const [status, setStatus] = useState<'idle' | 'saving' | 'success' | 'error'>('idle');
  const [message, setMessage] = useState('');
  useEffect(() => {
    const open = (event: MouseEvent) => {
      const target = (event.target as Element).closest<HTMLAnchorElement>('[data-open-waitlist]');
      if (!target || !dialog.current || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
      event.preventDefault();
      opener.current = target;
      dialog.current.showModal();
    };
    document.addEventListener('click', open);
    if (window.location.hash === '#waitlist') dialog.current?.showModal();
    return () => { document.removeEventListener('click', open); request.current?.abort(); };
  }, []);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (request.current) return;
    const form = new FormData(event.currentTarget);
    const controller = new AbortController();
    request.current = controller;
    const timeout = window.setTimeout(() => controller.abort(), 12000);
    setStatus('saving');
    setMessage('');
    try {
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'https://dreamlog-production-f9e2.up.railway.app'}/waitlist`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: String(form.get('email') || '').trim(), website: form.get('website') || '' }),
        signal: controller.signal,
      });
      if (!response.ok) {
        throw new Error(response.status === 429 ? 'Too many attempts. Please try again in a minute.' : response.status === 400 ? 'Check your email address and try again.' : "We couldn't save your place. Please try again.");
      }
      setStatus('success');
    } catch (error) {
      setStatus('error');
      setMessage(error instanceof Error && error.name !== 'AbortError' ? error.message : 'The connection took too long. Please try again.');
    } finally { window.clearTimeout(timeout); request.current = null; }
  }

  return <dialog id="waitlist" className="waitlist-dialog" ref={dialog} aria-labelledby="waitlist-title" onClose={() => opener.current?.focus({ preventScroll: true })}>
    <button className="waitlist-close" type="button" aria-label="Close waitlist" onClick={() => dialog.current?.close()}>×</button>
    <span className="waitlist-brand">Ode</span>
    <h2 id="waitlist-title">{status === 'success' ? "You're on the list." : 'A little space, soon.'}</h2>
    {status === 'success' ? <div role="status"><p>We’ll email you when Ode is ready for you.</p><button className="button button-amber" type="button" onClick={() => dialog.current?.close()}>Back to Ode <span aria-hidden="true">↗</span></button></div> : <>
      <p>Leave your email. We’ll let you know when Ode launches on iOS and Android.</p>
      <form onSubmit={submit} aria-busy={status === 'saving'}>
        <label htmlFor="waitlist-email">Email address</label>
        <input id="waitlist-email" name="email" type="email" autoComplete="email" inputMode="email" maxLength={254} placeholder="you@example.com" required aria-describedby="waitlist-note waitlist-error" readOnly={status === 'saving'} />
        <div className="waitlist-honey" aria-hidden="true"><label htmlFor="waitlist-website">Website</label><input id="waitlist-website" name="website" type="text" tabIndex={-1} autoComplete="off" /></div>
        <p id="waitlist-error" className="waitlist-error" role="alert">{message}</p>
        <button className="button button-amber" type="submit" disabled={status === 'saving'}>{status === 'saving' ? 'Saving your place…' : 'Join the waitlist'} <span aria-hidden="true">↗</span></button>
        <p id="waitlist-note" className="waitlist-note">By joining, you agree to receive an Ode launch email. No account needed. <a href="/privacy">Privacy policy</a></p>
      </form>
    </>}
  </dialog>;
}
