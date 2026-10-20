'use client';

import { useRef, useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { api, clearToken, notesApi, saveToken } from '../../lib/api';
import { SiteFooter, SiteFrame, SiteHeader } from '../../components/SiteFrame';
export default function LoginPage() {
  const router = useRouter();
  const [register, setRegister] = useState(false),
    [email, setEmail] = useState(''),
    [password, setPassword] = useState(''),
    [name, setName] = useState(''),
    [credentials, setCredentials] = useState(''),
    [show, setShow] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(''),
    [accountCreated, setAccountCreated] = useState(false);
  const lock = useRef(false);
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (lock.current) return;
    lock.current = true;
    setBusy(true);
    setError('');
    try {
      if (register) {
        if (!accountCreated) {
          const r = await api.authRegister(email.trim(), password, name.trim());
          saveToken(r.token);
          setAccountCreated(true);
        }
        await api.registerTherapist(name.trim(), email.trim(), credentials.trim());
      } else {
        const r = await api.login(email.trim(), password);
        saveToken(r.token);
        await notesApi.me();
      }
      router.push('/dashboard');
    } catch {
      if (!register) clearToken();
      setError(register ? 'Could not complete registration. Check your details and try again. If your account already exists, sign in or contact support.' : 'Could not sign in to a therapist account. Check your details and try again.');
    } finally {
      lock.current = false;
      setBusy(false);
    }
  }
  return <SiteFrame><SiteHeader /><main id="main" className="login"><section className="login-story"><div><p className="kicker" style={{
            color: 'var(--amber)'
          }}>Ode for therapists</p><h1>Make room for<br />the conversation.</h1><p>Client context and session notes, in one quiet workspace.</p><svg className="breath-line" viewBox="0 0 400 100" fill="none" aria-hidden="true"><path d="M2 50C22 3 42 97 62 50S102 82 122 50S162 70 182 50S222 58 242 50S282 53 322 50" stroke="currentColor" strokeWidth="2" /><circle cx="355" cy="50" r="3" fill="currentColor" /></svg></div><p className="story-bottom small-note">Journaling happens in the mobile app.<br />This portal is for mental health professionals.</p></section><section className="login-form-wrap"><form className="login-form" onSubmit={submit}><h2>{register ? 'Create your account.' : 'Welcome back.'}</h2><p className="quiet">{register ? 'Set up your therapist workspace.' : 'Sign in to your therapist workspace.'}</p>{register && <><div className="field"><label htmlFor="full-name">Full name</label><input id="full-name" value={name} onChange={e => setName(e.target.value)} autoComplete="name" required disabled={busy || accountCreated} /></div><div className="field"><label htmlFor="credentials">Professional credentials</label><input id="credentials" value={credentials} onChange={e => setCredentials(e.target.value)} required disabled={busy} /></div></>}<div className="field"><label htmlFor="email">Email address</label><input id="email" type="email" value={email} onChange={e => setEmail(e.target.value)} autoComplete="username" required disabled={busy || accountCreated} /></div><div className="field"><label htmlFor="password">Password</label><div className="password-field"><input id="password" type={show ? 'text' : 'password'} value={password} onChange={e => setPassword(e.target.value)} autoComplete={register ? 'new-password' : 'current-password'} required disabled={busy || accountCreated} /><button type="button" aria-label={show ? 'Hide password' : 'Show password'} onClick={() => setShow(!show)}>{show ? 'Hide' : 'Show'}</button></div></div>{error && <p className="error" role="alert">{error}</p>}<button className="btn full" disabled={busy} type="submit">{busy ? 'Please wait…' : register ? 'Create therapist account →' : 'Sign in →'}</button><p className="small-note">{register ? 'Already registered?' : 'New to the portal?'} <button disabled={busy} type="button" className="text-btn" onClick={() => {
              setRegister(!register);
              setError('');
              setAccountCreated(false);
              clearToken();
            }}>{register ? 'Sign in' : 'Create account'}</button></p><p className="small-note quiet">By creating an account you confirm your professional credentials and agree to the <a href="/terms">Terms of Service</a>. Read our <a href="/privacy">Privacy Policy</a>.</p><a className="small-note" href="mailto:talktoode.dev@gmail.com">Need help accessing your account?</a></form></section></main><SiteFooter /></SiteFrame>;
}
