import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SecureStore from 'expo-secure-store';

// Supabase session storage backed by the OS keychain / keystore instead of
// plaintext AsyncStorage: the session holds a long-lived refresh token, which
// must not sit unencrypted in app storage or device backups.
//
// SecureStore values are limited (2048 bytes on Android), so values are split
// into chunks of at most MAX_CHUNK_BYTES UTF-8 bytes - sized by bytes, not
// characters, so a session with non-ASCII metadata (e.g. a Hindi full_name)
// still fits.
//
// Writes are atomic (N13): chunks go under a fresh generation key
// `<key>.<gen>.<i>`, then ONE write of the pointer `<key>.ptr` = "<gen>:<count>"
// switches readers over, and only then is the previous generation deleted. A
// concurrent read or an app kill at any point sees either the old or the new
// complete value - never a half-written one that would sign the user out.
//
// Older builds stored `<key>.n` + `<key>.0..n-1`, and before that plaintext
// AsyncStorage; both are migrated on first read.

const MAX_CHUNK_BYTES = 2000;

// SecureStore keys allow only [A-Za-z0-9._-].
const safeKey = (key: string) => key.replace(/[^A-Za-z0-9._-]/g, '_');

const utf8Bytes = (cp: number) => (cp < 0x80 ? 1 : cp < 0x800 ? 2 : cp < 0x10000 ? 3 : 4);

/** Splits on code-point boundaries into chunks of at most maxBytes UTF-8 bytes. */
export function chunkByUtf8Bytes(value: string, maxBytes = MAX_CHUNK_BYTES): string[] {
  const parts: string[] = [];
  let current = '';
  let bytes = 0;
  for (const ch of value) {
    const n = utf8Bytes(ch.codePointAt(0) ?? 0);
    if (bytes + n > maxBytes && current) {
      parts.push(current);
      current = '';
      bytes = 0;
    }
    current += ch;
    bytes += n;
  }
  if (current || parts.length === 0) parts.push(current);
  return parts;
}

type Pointer = { gen: string; count: number };

async function readPointer(base: string): Promise<Pointer | null> {
  const raw = await SecureStore.getItemAsync(`${base}.ptr`);
  if (!raw) return null;
  const [gen, count] = raw.split(':');
  const n = Number(count);
  return gen && Number.isInteger(n) && n > 0 ? { gen, count: n } : null;
}

async function readGeneration(base: string, p: Pointer): Promise<string | null> {
  const parts = await Promise.all(Array.from({ length: p.count }, (_, i) => SecureStore.getItemAsync(`${base}.${p.gen}.${i}`)));
  return parts.every((x) => x !== null) ? parts.join('') : null;
}

async function deleteGeneration(base: string, p: Pointer): Promise<void> {
  await Promise.all(Array.from({ length: p.count }, (_, i) => SecureStore.deleteItemAsync(`${base}.${p.gen}.${i}`)));
}

// ── Legacy `<key>.n` chunk format (builds before N13) ────────────────────────
async function readLegacyChunks(base: string): Promise<string | null> {
  const count = Number(await SecureStore.getItemAsync(`${base}.n`)) || 0;
  if (count <= 0) return null;
  const parts = await Promise.all(Array.from({ length: count }, (_, i) => SecureStore.getItemAsync(`${base}.${i}`)));
  return parts.every((p) => p !== null) ? parts.join('') : null;
}

async function removeLegacyChunks(base: string): Promise<void> {
  const count = Number(await SecureStore.getItemAsync(`${base}.n`)) || 0;
  await Promise.all(Array.from({ length: count }, (_, i) => SecureStore.deleteItemAsync(`${base}.${i}`)));
  await SecureStore.deleteItemAsync(`${base}.n`);
}

// Writes for the same process are serialized so two refreshes can't race
// their pointer switches and generation cleanup.
let writeTail: Promise<void> = Promise.resolve();
function serialized(op: () => Promise<void>): Promise<void> {
  const next = writeTail.then(op, op);
  writeTail = next.catch(() => {});
  return next;
}

const newGeneration = () => `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 8)}`;

async function write(base: string, value: string): Promise<void> {
  const previous = await readPointer(base);
  const gen = newGeneration();
  const parts = chunkByUtf8Bytes(value);
  await Promise.all(parts.map((p, i) => SecureStore.setItemAsync(`${base}.${gen}.${i}`, p)));
  await SecureStore.setItemAsync(`${base}.ptr`, `${gen}:${parts.length}`); // the atomic switch
  if (previous && previous.gen !== gen) await deleteGeneration(base, previous);
  await removeLegacyChunks(base);
}

export const secureSessionStorage = {
  async getItem(key: string): Promise<string | null> {
    const base = safeKey(key);
    // A writer may switch the pointer and delete the old generation between
    // our two reads; then the pointer has moved and the new value is complete.
    for (let attempt = 0; attempt < 2; attempt++) {
      const p = await readPointer(base);
      if (!p) break;
      const value = await readGeneration(base, p);
      if (value !== null) return value;
      const now = await readPointer(base);
      if (!now || now.gen === p.gen) return null;
    }

    const legacyChunks = await readLegacyChunks(base);
    if (legacyChunks !== null) {
      await serialized(() => write(base, legacyChunks));
      return legacyChunks;
    }

    // One-time migration from the old plaintext AsyncStorage location.
    const legacy = await AsyncStorage.getItem(key);
    if (legacy !== null) {
      await secureSessionStorage.setItem(key, legacy);
      await AsyncStorage.removeItem(key);
    }
    return legacy;
  },

  setItem(key: string, value: string): Promise<void> {
    return serialized(() => write(safeKey(key), value));
  },

  removeItem(key: string): Promise<void> {
    const base = safeKey(key);
    return serialized(async () => {
      const p = await readPointer(base);
      await SecureStore.deleteItemAsync(`${base}.ptr`);
      if (p) await deleteGeneration(base, p);
      await removeLegacyChunks(base);
      await AsyncStorage.removeItem(key);
    });
  },
};
