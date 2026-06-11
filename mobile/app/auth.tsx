import { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  ActivityIndicator,
  Modal,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { GoogleSignin, statusCodes, isErrorWithCode } from '@react-native-google-signin/google-signin';
import * as AppleAuthentication from 'expo-apple-authentication';
import { supabase } from '../src/lib/supabase';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';

GoogleSignin.configure({
  webClientId: process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID,
  scopes: ['email', 'profile'],
});

type Mode = 'login' | 'register';

function GoogleLogo() {
  return (
    <View style={googleLogoStyles.container}>
      <View style={googleLogoStyles.blue} />
      <View style={googleLogoStyles.bar} />
      <Text style={googleLogoStyles.letter}>G</Text>
    </View>
  );
}

const googleLogoStyles = StyleSheet.create({
  container: {
    width: 20,
    height: 20,
    borderRadius: 10,
    backgroundColor: '#fff',
    alignItems: 'center',
    justifyContent: 'center',
  },
  blue: {
    position: 'absolute',
    width: 20,
    height: 20,
    borderRadius: 10,
    borderWidth: 3,
    borderColor: '#4285F4',
    borderRightColor: '#34A853',
    borderBottomColor: '#FBBC05',
    transform: [{ rotate: '-45deg' }],
  },
  bar: {
    position: 'absolute',
    right: 0,
    top: '50%',
    width: 8,
    height: 3,
    backgroundColor: '#4285F4',
    marginTop: -1.5,
  },
  letter: {
    fontSize: 11,
    fontWeight: '700',
    color: '#4285F4',
    lineHeight: 20,
  },
});

export default function AuthScreen() {
  const [mode, setMode] = useState<Mode>('login');
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [emailModal, setEmailModal] = useState('');
  const router = useRouter();
  const { colors } = useTheme();

  const reset = (next: Mode) => {
    setMode(next);
    setEmail('');
    setName('');
    setPassword('');
    setError('');
  };

  const handleGoogleSignIn = async () => {
    setError('');
    setLoading(true);
    try {
      await GoogleSignin.hasPlayServices();
      const response = await GoogleSignin.signIn();
      if (response.type === 'cancelled') return;

      const idToken = response.data?.idToken;
      if (!idToken) throw new Error('No ID token from Google');

      const { error: signInError } = await supabase.auth.signInWithIdToken({
        provider: 'google',
        token: idToken,
      });
      if (signInError) throw signInError;

      const user = await api.me();
      router.replace(user.goal ? '/(tabs)' : '/onboarding' as any);
    } catch (err: any) {
      if (isErrorWithCode(err) && err.code === statusCodes.SIGN_IN_CANCELLED) return;
      setError(err?.message ?? 'Google sign-in failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleAppleSignIn = async () => {
    setError('');
    setLoading(true);
    try {
      const credential = await AppleAuthentication.signInAsync({
        requestedScopes: [
          AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
          AppleAuthentication.AppleAuthenticationScope.EMAIL,
        ],
      });

      if (!credential.identityToken) throw new Error('No identity token from Apple');

      const { error: signInError } = await supabase.auth.signInWithIdToken({
        provider: 'apple',
        token: credential.identityToken,
      });
      if (signInError) throw signInError;

      const user = await api.me();
      router.replace(user.goal ? '/(tabs)' : '/onboarding' as any);
    } catch (err: any) {
      if (err?.code === 'ERR_REQUEST_CANCELED') return;
      setError(err?.message ?? 'Apple sign-in failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async () => {
    const emailTrimmed = email.trim();
    const passwordTrimmed = password.trim();
    const nameTrimmed = name.trim();

    if (!emailTrimmed || !passwordTrimmed) {
      setError('Email and password are required.');
      return;
    }
    if (mode === 'register' && !nameTrimmed) {
      setError('Name is required.');
      return;
    }
    if (mode === 'register' && passwordTrimmed.length < 6) {
      setError('Password must be at least 6 characters.');
      return;
    }

    setError('');
    setLoading(true);
    try {
      if (mode === 'register') {
        const { data, error: signUpError } = await supabase.auth.signUp({
          email: emailTrimmed,
          password: passwordTrimmed,
          options: {
            data: { full_name: nameTrimmed },
            emailRedirectTo: 'dreamlog://auth/callback',
          },
        });

        if (signUpError) throw signUpError;

        if (!data.session) {
          setEmailModal(emailTrimmed);
          reset('login');
          return;
        }

        router.replace('/onboarding' as any);
      } else {
        const { error: signInError } = await supabase.auth.signInWithPassword({
          email: emailTrimmed,
          password: passwordTrimmed,
        });

        if (signInError) throw signInError;

        const user = await api.me();
        router.replace(user.goal ? '/(tabs)' : '/onboarding' as any);
      }
    } catch (err: any) {
      setError(err?.message ?? (mode === 'login' ? 'Invalid email or password.' : 'Registration failed.'));
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

          {/* Google Sign-In */}
          <TouchableOpacity
            style={[styles.googleBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
            onPress={handleGoogleSignIn}
            disabled={loading}
            activeOpacity={0.8}
          >
            {loading ? (
              <ActivityIndicator color={colors.textMuted} size="small" />
            ) : (
              <>
                <GoogleLogo />
                <Text style={[styles.googleBtnText, { color: colors.textPrimary }]}>Continue with Google</Text>
              </>
            )}
          </TouchableOpacity>

          {/* Sign in with Apple - iOS only (App Store Guideline 4.8) */}
          {Platform.OS === 'ios' && (
            <AppleAuthentication.AppleAuthenticationButton
              buttonType={AppleAuthentication.AppleAuthenticationButtonType.SIGN_IN}
              buttonStyle={AppleAuthentication.AppleAuthenticationButtonStyle.WHITE}
              cornerRadius={14}
              style={styles.appleBtn}
              onPress={handleAppleSignIn}
            />
          )}

          {/* Divider */}
          <View style={styles.divider}>
            <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
            <Text style={[styles.dividerText, { color: colors.textMuted }]}>or</Text>
            <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
          </View>

          <View style={[styles.card, { backgroundColor: colors.card, borderColor: colors.border }]}>
            {mode === 'register' && (
              <TextInput
                style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: error ? '#ef4444' : colors.borderFaint, color: colors.textPrimary }]}
                value={name}
                onChangeText={v => { setName(v); setError(''); }}
                placeholder="Your name"
                placeholderTextColor={colors.textFaint}
                autoCapitalize="words"
                autoCorrect={false}
                returnKeyType="next"
              />
            )}

            <TextInput
              style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: error ? '#ef4444' : colors.borderFaint, color: colors.textPrimary }]}
              value={email}
              onChangeText={v => { setEmail(v); setError(''); }}
              placeholder="Email"
              placeholderTextColor={colors.textFaint}
              keyboardType="email-address"
              autoCapitalize="none"
              autoCorrect={false}
              returnKeyType="next"
            />

            <TextInput
              style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: error ? '#ef4444' : colors.borderFaint, color: colors.textPrimary }]}
              value={password}
              onChangeText={v => { setPassword(v); setError(''); }}
              placeholder={mode === 'register' ? 'Password (min 6 characters)' : 'Password'}
              placeholderTextColor={colors.textFaint}
              secureTextEntry
              autoCapitalize="none"
              autoCorrect={false}
              returnKeyType="done"
              onSubmitEditing={handleSubmit}
            />

            {!!error && (
              <Text style={styles.errorText}>{error}</Text>
            )}

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

      {/* Email verification modal */}
      <Modal
        visible={!!emailModal}
        transparent
        animationType="fade"
        statusBarTranslucent
        onRequestClose={() => setEmailModal('')}
      >
        <View style={styles.modalOverlay}>
          <View style={[styles.modalCard, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}>
            <View style={[styles.modalIconWrap, { backgroundColor: colors.purple600 + '33' }]}>
              <Text style={styles.modalIcon}>✉️</Text>
            </View>
            <Text style={[styles.modalTitle, { color: colors.textPrimary }]}>Check your email</Text>
            <Text style={[styles.modalBody, { color: colors.textSecondary }]}>
              We sent a verification link to
            </Text>
            <Text style={[styles.modalEmail, { color: colors.purple500 ?? '#a78bfa' }]} numberOfLines={1}>
              {emailModal}
            </Text>
            <Text style={[styles.modalBody, { color: colors.textSecondary }]}>
              Click the link to activate your account, then sign in here.
            </Text>
            <TouchableOpacity
              style={[styles.modalBtn, { backgroundColor: colors.purple600, shadowColor: colors.purple500 }]}
              onPress={() => setEmailModal('')}
              activeOpacity={0.85}
            >
              <Text style={styles.modalBtnText}>OK, got it</Text>
            </TouchableOpacity>
          </View>
        </View>
      </Modal>
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

  googleBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
    borderRadius: 14,
    borderWidth: 1,
    paddingVertical: 14,
    width: '100%',
  },
  googleBtnText: {
    fontSize: 15,
    fontFamily: 'Nunito_600SemiBold',
  },
  appleBtn: {
    width: '100%',
    height: 50,
    marginTop: 10,
  },
  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    width: '100%',
    gap: 10,
    marginVertical: 4,
  },
  dividerLine: {
    flex: 1,
    height: 1,
  },
  dividerText: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
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
  modalOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.7)',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 32,
  },
  modalCard: {
    width: '100%',
    borderRadius: 24,
    borderWidth: 1,
    padding: 28,
    alignItems: 'center',
    gap: 8,
  },
  modalIconWrap: {
    width: 64,
    height: 64,
    borderRadius: 32,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 4,
  },
  modalIcon: {
    fontSize: 30,
  },
  modalTitle: {
    fontSize: 22,
    fontFamily: 'CormorantGaramond_500Medium',
    marginBottom: 4,
  },
  modalBody: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
    lineHeight: 20,
  },
  modalEmail: {
    fontSize: 14,
    fontFamily: 'Nunito_700Bold',
    textAlign: 'center',
  },
  modalBtn: {
    marginTop: 12,
    width: '100%',
    paddingVertical: 16,
    borderRadius: 14,
    alignItems: 'center',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 6,
  },
  modalBtnText: {
    color: '#fff',
    fontSize: 16,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 0.5,
  },
  errorText: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    color: '#ef4444',
    lineHeight: 18,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 0.5,
  },
});
