import { createClient } from '@supabase/supabase-js';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { Linking } from 'react-native';
import { storeToken, clearToken } from '../api/client';

const supabaseUrl = process.env.EXPO_PUBLIC_SUPABASE_URL!;
const supabaseAnonKey = process.env.EXPO_PUBLIC_SUPABASE_ANON_KEY!;

export const supabase = createClient(supabaseUrl, supabaseAnonKey, {
  auth: {
    storage: AsyncStorage,
    autoRefreshToken: true,
    persistSession: true,
    detectSessionInUrl: false,
  },
});

// Keep the axios bearer token in sync with Supabase session changes.
// This means the rest of the app (api/client.ts) never needs to know about Supabase.
supabase.auth.onAuthStateChange(async (_event, session) => {
  if (session?.access_token) {
    await storeToken(session.access_token);
  } else {
    await clearToken();
  }
});

// Parse auth tokens from a Supabase deep link callback URL and establish the session.
// Supabase sends tokens in the URL fragment for implicit flow:
//   dreamlog://#access_token=...&refresh_token=...&type=signup
// or a code param for PKCE flow:
//   dreamlog://?code=...
async function handleDeepLink(url: string) {
  const fragment = url.split('#')[1] ?? url.split('?')[1] ?? '';
  if (!fragment) return;
  // Split at the FIRST '=' only so base64-padded tokens are preserved.
  const params: Record<string, string> = {};
  fragment.split('&').forEach(pair => {
    const eqIdx = pair.indexOf('=');
    if (eqIdx === -1) return;
    const k = decodeURIComponent(pair.slice(0, eqIdx));
    const v = decodeURIComponent(pair.slice(eqIdx + 1));
    params[k] = v;
  });
  if (params.access_token && params.refresh_token) {
    await supabase.auth.setSession({
      access_token: params.access_token,
      refresh_token: params.refresh_token,
    });
  } else if (params.code) {
    await supabase.auth.exchangeCodeForSession(params.code);
  }
}

// Resolves once the initial deep link (if any) has been fully processed.
// _layout.tsx awaits this before calling getSession() so the session is
// guaranteed to be set before we decide where to navigate.
export const deepLinkReady: Promise<void> = Linking.getInitialURL()
  .then(url => { if (url) return handleDeepLink(url); })
  .catch(() => {});

// Handle confirmation link while the app is already running in the background.
Linking.addEventListener('url', ({ url }) => { handleDeepLink(url); });
