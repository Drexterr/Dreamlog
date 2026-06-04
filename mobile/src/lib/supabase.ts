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
  const params = Object.fromEntries(
    fragment.split('&').map(pair => {
      const [k = '', v = ''] = pair.split('=');
      return [decodeURIComponent(k), decodeURIComponent(v)];
    })
  );
  if (params.access_token && params.refresh_token) {
    await supabase.auth.setSession({
      access_token: params.access_token,
      refresh_token: params.refresh_token,
    });
  } else if (params.code) {
    await supabase.auth.exchangeCodeForSession(params.code);
  }
}

// Handle app launched directly from the email confirmation link.
Linking.getInitialURL().then(url => { if (url) handleDeepLink(url); });
// Handle confirmation link while the app is already running.
Linking.addEventListener('url', ({ url }) => { handleDeepLink(url); });
