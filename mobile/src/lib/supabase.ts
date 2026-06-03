import { createClient } from '@supabase/supabase-js';
import AsyncStorage from '@react-native-async-storage/async-storage';
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
