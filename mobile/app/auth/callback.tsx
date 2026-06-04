import { useEffect } from 'react';
import { View, ActivityIndicator, Text, StyleSheet } from 'react-native';
import { useRouter } from 'expo-router';
import { supabase } from '../../src/lib/supabase';
import { api, storeToken } from '../../src/api/client';

// Landing screen for Supabase email confirmation deep links.
// URL format: dreamlog://auth/callback#access_token=...  (implicit flow)
//          or dreamlog://auth/callback?code=...          (PKCE flow)
//
// src/lib/supabase.ts already handles the token exchange via Linking listeners.
// This screen just waits for onAuthStateChange to fire and then navigates away.
export default function AuthCallback() {
  const router = useRouter();

  useEffect(() => {
    const { data: { subscription } } = supabase.auth.onAuthStateChange(
      async (event, session) => {
        if ((event === 'SIGNED_IN' || event === 'USER_UPDATED') && session) {
          try {
            await storeToken(session.access_token);
            const user = await api.me();
            router.replace(user.goal ? '/(tabs)' : '/onboarding' as any);
          } catch {
            router.replace('/auth');
          }
        }
      }
    );
    return () => subscription.unsubscribe();
  }, []);

  return (
    <View style={styles.container}>
      <ActivityIndicator color="#7C3AED" size="large" />
      <Text style={styles.text}>Verifying your account…</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f0c1e',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 16,
  },
  text: {
    color: '#a78bfa',
    fontFamily: 'Nunito_400Regular',
    fontSize: 15,
  },
});
