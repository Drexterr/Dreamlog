import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SecureStore from 'expo-secure-store';

// Supabase session storage backed by the OS keychain / keystore instead of
// plaintext AsyncStorage: the session holds a long-lived refresh token, which
// must not sit unencrypted in app storage or device backups.
//
// SecureStore values are limited (~2 KB on Android), so values are split into
// chunks: `<key>.n` holds the chunk count and `<key>.0..n-1` the parts.
// Sessions written by older builds to AsyncStorage are migrated on first read.

const CHUNK = 1800;

// SecureStore keys allow only [A-Za-z0-9._-].
const safeKey = (key: string) => key.replace(/[^A-Za-z0-9._-]/g, '_');

async function removeChunks(base: string): Promise<void> {
  const count = Number(await SecureStore.getItemAsync(`${base}.n`)) || 0;
  await Promise.all(Array.from({ length: count }, (_, i) => SecureStore.deleteItemAsync(`${base}.${i}`)));
  await SecureStore.deleteItemAsync(`${base}.n`);
}

export const secureSessionStorage = {
  async getItem(key: string): Promise<string | null> {
    const base = safeKey(key);
    const count = Number(await SecureStore.getItemAsync(`${base}.n`)) || 0;
    if (count > 0) {
      const parts = await Promise.all(Array.from({ length: count }, (_, i) => SecureStore.getItemAsync(`${base}.${i}`)));
      if (parts.every((p) => p !== null)) return parts.join('');
      return null;
    }
    // One-time migration from the old plaintext AsyncStorage location.
    const legacy = await AsyncStorage.getItem(key);
    if (legacy !== null) {
      await secureSessionStorage.setItem(key, legacy);
      await AsyncStorage.removeItem(key);
    }
    return legacy;
  },

  async setItem(key: string, value: string): Promise<void> {
    const base = safeKey(key);
    await removeChunks(base);
    const parts: string[] = [];
    for (let i = 0; i < value.length; i += CHUNK) parts.push(value.slice(i, i + CHUNK));
    await Promise.all(parts.map((p, i) => SecureStore.setItemAsync(`${base}.${i}`, p)));
    await SecureStore.setItemAsync(`${base}.n`, String(parts.length));
  },

  async removeItem(key: string): Promise<void> {
    await removeChunks(safeKey(key));
    await AsyncStorage.removeItem(key);
  },
};
