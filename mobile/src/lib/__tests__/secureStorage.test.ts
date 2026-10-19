import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SecureStore from 'expo-secure-store';
import { secureSessionStorage } from '../secureStorage';

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
    expect(secure.get('sb-ref-auth-token.n')).toBe('2');
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
});
