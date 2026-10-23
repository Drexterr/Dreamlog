import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SecureStore from 'expo-secure-store';
import { chunkByUtf8Bytes, secureSessionStorage } from '../secureStorage';

jest.mock('expo-secure-store', () => {
  const store = new Map<string, string>();
  return {
    __store: store,
    getItemAsync: jest.fn(async (k: string) => store.get(k) ?? null),
    setItemAsync: jest.fn(async (k: string, v: string) => { store.set(k, v); }),
    deleteItemAsync: jest.fn(async (k: string) => { store.delete(k); }),
  };
});
const secure = (SecureStore as unknown as { __store: Map<string, string> }).__store;

beforeEach(async () => { secure.clear(); await AsyncStorage.clear(); });

// M8: the Supabase session (incl. refresh token) lives in the keychain, chunked.
describe('secure session storage', () => {
  it('round-trips a value larger than one SecureStore chunk', async () => {
    const session = JSON.stringify({ access_token: 'a'.repeat(3000), refresh_token: 'r' });
    await secureSessionStorage.setItem('sb-ref-auth-token', session);
    expect(await secureSessionStorage.getItem('sb-ref-auth-token')).toBe(session);
    expect(secure.get('sb-ref-auth-token.ptr')).toMatch(/:2$/);
    expect(await AsyncStorage.getItem('sb-ref-auth-token')).toBeNull();
  });
  it('migrates a legacy plaintext session out of AsyncStorage', async () => {
    await AsyncStorage.setItem('sb-ref-auth-token', '{"refresh_token":"legacy"}');
    expect(await secureSessionStorage.getItem('sb-ref-auth-token')).toBe('{"refresh_token":"legacy"}');
    expect(await AsyncStorage.getItem('sb-ref-auth-token')).toBeNull();
    expect(await secureSessionStorage.getItem('sb-ref-auth-token')).toBe('{"refresh_token":"legacy"}');
  });
  it('removes every chunk', async () => {
    await secureSessionStorage.setItem('k', 'x'.repeat(4000));
    await secureSessionStorage.removeItem('k');
    expect(secure.size).toBe(0);
    expect(await secureSessionStorage.getItem('k')).toBeNull();
  });

  // N13: chunks are sized by UTF-8 bytes, so non-ASCII sessions stay under
  // SecureStore's 2048-byte Android limit.
  it('keeps every chunk within 2000 UTF-8 bytes for non-ASCII values', async () => {
    const session = JSON.stringify({ user_metadata: { full_name: 'अनन्या शर्मा '.repeat(200) }, refresh_token: 'r' });
    await secureSessionStorage.setItem('k', session);
    for (const [key, v] of secure) {
      if (!key.endsWith('.ptr')) expect(Buffer.byteLength(v, 'utf8')).toBeLessThanOrEqual(2000);
    }
    expect(await secureSessionStorage.getItem('k')).toBe(session);
  });
  it('never splits a surrogate pair', () => {
    const parts = chunkByUtf8Bytes('😀'.repeat(1000), 10);
    expect(parts.join('')).toBe('😀'.repeat(1000));
    // No chunk may end in a high surrogate (code unit 0xD800-0xDBFF).
    expect(parts.every((p) => { const c = p.charCodeAt(p.length - 1); return c < 0xd800 || c > 0xdbff; })).toBe(true);
  });
  // N13: a write interrupted before the pointer switch leaves the old value.
  it('keeps the previous session if a write dies before switching the pointer', async () => {
    await secureSessionStorage.setItem('k', 'old-session');
    const setItem = SecureStore.setItemAsync as jest.Mock;
    const original = setItem.getMockImplementation();
    setItem.mockImplementation(async (k: string, v: string) => {
      if (k.endsWith('.ptr')) throw new Error('app killed');
      secure.set(k, v);
    });
    await expect(secureSessionStorage.setItem('k', 'new-session')).rejects.toThrow('app killed');
    setItem.mockImplementation(original);
    expect(await secureSessionStorage.getItem('k')).toBe('old-session');
  });
  // N13: reads concurrent with rewrites always see a complete session.
  it('never returns null to a read racing a rewrite', async () => {
    await secureSessionStorage.setItem('k', 'v0'.repeat(1500));
    const reads: Promise<string | null>[] = [];
    const writes: Promise<void>[] = [];
    for (let i = 1; i <= 10; i++) {
      writes.push(secureSessionStorage.setItem('k', `v${i}`.repeat(1500)));
      reads.push(secureSessionStorage.getItem('k'));
    }
    await Promise.all(writes);
    for (const r of await Promise.all(reads)) expect(r).not.toBeNull();
    expect(await secureSessionStorage.getItem('k')).toBe('v10'.repeat(1500));
    // Old generations are cleaned up: one pointer plus the current chunks.
    const ptr = secure.get('k.ptr')!;
    expect(secure.size).toBe(1 + Number(ptr.split(':')[1]));
  });
  it('migrates the previous chunk format', async () => {
    secure.set('k.n', '2'); secure.set('k.0', 'hello '); secure.set('k.1', 'world');
    expect(await secureSessionStorage.getItem('k')).toBe('hello world');
    expect(secure.has('k.n')).toBe(false);
    expect(secure.get('k.ptr')).toBeDefined();
    expect(await secureSessionStorage.getItem('k')).toBe('hello world');
  });
});
