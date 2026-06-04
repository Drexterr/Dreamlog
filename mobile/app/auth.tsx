import { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  Alert,
  ScrollView,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { supabase } from '../src/lib/supabase';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';

type Mode = 'login' | 'register';

export default function AuthScreen() {
  const [mode, setMode] = useState<Mode>('login');
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const router = useRouter();
  const { colors } = useTheme();

  const reset = (next: Mode) => {
    setMode(next);
    setEmail('');
    setName('');
    setPassword('');
  };

  const handleSubmit = async () => {
    const emailTrimmed = email.trim();
    const passwordTrimmed = password.trim();
    const nameTrimmed = name.trim();

    if (!emailTrimmed || !passwordTrimmed) {
      Alert.alert('Missing fields', 'Email and password are required.');
      return;
    }
    if (mode === 'register' && !nameTrimmed) {
      Alert.alert('Missing fields', 'Name is required.');
      return;
    }
    if (mode === 'register' && passwordTrimmed.length < 6) {
      Alert.alert('Weak password', 'Password must be at least 6 characters.');
      return;
    }

    setLoading(true);
    try {
      if (mode === 'register') {
        const { data, error } = await supabase.auth.signUp({
          email: emailTrimmed,
          password: passwordTrimmed,
          options: {
            data: { full_name: nameTrimmed },
            emailRedirectTo: 'dreamlog://auth/callback',
          },
        });

        if (error) throw error;

        if (!data.session) {
          // Email confirmation required — Supabase sent a confirmation email.
          Alert.alert(
            'Check your email',
            'We sent a confirmation link to ' + emailTrimmed + '. Click it to activate your account, then sign in.',
          );
          reset('login');
          return;
        }

        // Session available immediately (email confirmation disabled in Supabase).
        // onAuthStateChange already stored the token via storeToken().
        router.replace('/onboarding' as any);
      } else {
        const { error } = await supabase.auth.signInWithPassword({
          email: emailTrimmed,
          password: passwordTrimmed,
        });

        if (error) throw error;

        // onAuthStateChange has stored the token — safe to call api.me() now.
        const user = await api.me();
        router.replace(user.goal ? '/(tabs)' : '/onboarding' as any);
      }
    } catch (err: any) {
      const message = err?.message ?? (mode === 'login' ? 'Invalid email or password.' : 'Registration failed.');
      Alert.alert('Error', message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.kav}
      >
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <View style={[styles.orb, { backgroundColor: colors.purple600, shadowColor: colors.purple500 }]} />
          <Text style={[styles.title, { color: colors.textPrimary }]}>DreamLog</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>Your AI listener that remembers</Text>

          {/* Tab switcher */}
          <View style={[styles.tabs, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <TouchableOpacity
              style={[styles.tab, mode === 'login' && { backgroundColor: colors.purple600 }]}
              onPress={() => reset('login')}
            >
              <Text style={[styles.tabText, { color: colors.textMuted }, mode === 'login' && styles.tabTextActive]}>Sign in</Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={[styles.tab, mode === 'register' && { backgroundColor: colors.purple600 }]}
              onPress={() => reset('register')}
            >
              <Text style={[styles.tabText, { color: colors.textMuted }, mode === 'register' && styles.tabTextActive]}>Create account</Text>
            </TouchableOpacity>
          </View>

          <View style={[styles.card, { backgroundColor: colors.card, borderColor: colors.border }]}>
            {mode === 'register' && (
              <TextInput
                style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint, color: colors.textPrimary }]}
                value={name}
                onChangeText={setName}
                placeholder="Your name"
                placeholderTextColor={colors.textFaint}
                autoCapitalize="words"
                autoCorrect={false}
                returnKeyType="next"
              />
            )}

            <TextInput
              style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint, color: colors.textPrimary }]}
              value={email}
              onChangeText={setEmail}
              placeholder="Email"
              placeholderTextColor={colors.textFaint}
              keyboardType="email-address"
              autoCapitalize="none"
              autoCorrect={false}
              returnKeyType="next"
            />

            <TextInput
              style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint, color: colors.textPrimary }]}
              value={password}
              onChangeText={setPassword}
              placeholder={mode === 'register' ? 'Password (min 6 characters)' : 'Password'}
              placeholderTextColor={colors.textFaint}
              secureTextEntry
              autoCapitalize="none"
              autoCorrect={false}
              returnKeyType="done"
              onSubmitEditing={handleSubmit}
            />

            <TouchableOpacity
              style={[styles.button, { backgroundColor: colors.purple600, shadowColor: colors.purple500 }, loading && styles.buttonLoading]}
              onPress={handleSubmit}
              disabled={loading}
              activeOpacity={0.8}
            >
              {loading ? (
                <ActivityIndicator color="#fff" size="small" />
              ) : (
                <Text style={styles.buttonText}>
                  {mode === 'login' ? 'Sign in' : 'Create account'}
                </Text>
              )}
            </TouchableOpacity>
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  kav: { flex: 1 },
  scroll: {
    alignItems: 'center',
    padding: 28,
    paddingTop: 60,
  },

  orb: {
    width: 80,
    height: 80,
    borderRadius: 40,
    marginBottom: 24,
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 0.6,
    shadowRadius: 24,
    elevation: 12,
    opacity: 0.85,
  },

  title: {
    fontSize: 32,
    fontFamily: 'CormorantGaramond_300Light',
    letterSpacing: 1,
    marginBottom: 6,
  },
  subtitle: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.5,
    marginBottom: 32,
  },

  tabs: {
    flexDirection: 'row',
    width: '100%',
    borderRadius: 14,
    borderWidth: 1,
    padding: 4,
    marginBottom: 16,
  },
  tab: {
    flex: 1,
    paddingVertical: 10,
    alignItems: 'center',
    borderRadius: 10,
  },
  tabText: {
    fontSize: 14,
    fontFamily: 'Nunito_600SemiBold',
  },
  tabTextActive: {
    color: '#fff',
  },

  card: {
    width: '100%',
    borderRadius: 20,
    borderWidth: 1,
    padding: 24,
    gap: 12,
  },

  input: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 14,
    fontFamily: 'Nunito_400Regular',
    fontSize: 15,
  },

  button: {
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 4,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 6,
  },
  buttonLoading: { opacity: 0.6 },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 0.5,
  },
});
