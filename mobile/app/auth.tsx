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
  Linking,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { GoogleSignin, statusCodes, isErrorWithCode } from '@react-native-google-signin/google-signin';
import * as AppleAuthentication from 'expo-apple-authentication';
import Svg, { Path } from 'react-native-svg';
import { supabase } from '../src/lib/supabase';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';
import type { ThemeColors } from '../src/theme';
import { setAppRole, resolvePostAuthRoute, type AppRole } from '../src/services/postAuthRoute';
import { T } from '../src/testIDs';

type Mode = 'login' | 'register';

// dreamlog.app is not yet DNS-pointed at the Firebase Hosting site - use the
// working *.web.app URL until that's done (see docs/LAUNCH_CHECKLIST.md).
const TERMS_URL = 'https://dreamlog-48f94.web.app/terms';
const PRIVACY_URL = 'https://dreamlog-48f94.web.app/privacy';

// Official Google "G" mark (18x18 viewBox), four brand colors.
function GoogleLogo() {
  return (
    <Svg width={20} height={20} viewBox="0 0 18 18">
      <Path fill="#4285F4" d="M17.64 9.2045c0-.6381-.0573-1.2518-.1636-1.8409H9v3.4814h4.8436c-.2086 1.125-.8427 2.0782-1.7959 2.7164v2.2581h2.9087c1.7018-1.5668 2.6836-3.874 2.6836-6.615z" />
      <Path fill="#34A853" d="M9 18c2.43 0 4.4673-.806 5.9564-2.1805l-2.9087-2.2581c-.8059.5401-1.8368.8591-3.0477.8591-2.344 0-4.3282-1.5831-5.036-3.7104H.9573v2.3318C2.4382 15.9832 5.4818 18 9 18z" />
      <Path fill="#FBBC05" d="M3.964 10.71c-.18-.5401-.2822-1.1168-.2822-1.71s.1023-1.1699.2822-1.71V4.9582H.9573C.3477 6.1732 0 7.5477 0 9s.3477 2.8268.9573 4.0418L3.964 10.71z" />
      <Path fill="#EA4335" d="M9 3.5795c1.3214 0 2.5077.4541 3.4405 1.346l2.5813-2.5814C13.4632.8918 11.426 0 9 0 5.4818 0 2.4382 2.0168.9573 4.9582L3.964 7.29C4.6718 5.1627 6.656 3.5795 9 3.5795z" />
    </Svg>
  );
}

// The Breath Line brand mark (static) - matches src/components/BrandSplash.tsx.
function BreathLineLogo({ colors }: { colors: ThemeColors }) {
  return (
    <View style={[styles.logoWrap, { shadowColor: colors.brandGlow }]}>
      <Svg width={140} height={47} viewBox="0 0 192 64">
        <Path
          d="M 8 32 C 18 10, 28 54, 38 32 C 47 15, 56 49, 65 32 C 73 20, 81 44, 89 32 C 96 25, 103 39, 110 32 C 117 28.5, 124 35.5, 131 32 C 141 30, 153 34, 164 32"
          stroke={colors.brand}
          strokeWidth={5}
          strokeLinecap="round"
          fill="none"
        />
        <Path
          d={`M ${179 - 5} 32 a 5 5 0 1 0 10 0 a 5 5 0 1 0 -10 0`}
          fill={colors.purple400}
        />
      </Svg>
    </View>
  );
}

export default function AuthScreen() {
  const [mode, setMode] = useState<Mode>('login');
  const [role, setRole] = useState<AppRole>('user');
  const [agreed, setAgreed] = useState(false);
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
    setAgreed(false);
  };

  const pickRole = (next: AppRole) => {
    setRole(next);
    setAppRole(next).catch(() => {});
  };

  // Route to the right destination after any successful sign-in:
  // terms gate → therapist dashboard/registration → onboarding/tabs.
  const routeAfterAuth = async () => {
    const dest = await resolvePostAuthRoute();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    router.replace(dest as any);
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

      await routeAfterAuth();
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

      await routeAfterAuth();
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
    if (mode === 'register' && !agreed) {
      setError('Please accept the Terms of Service and Privacy Policy to continue.');
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
            emailRedirectTo: 'ode://auth/callback',
          },
        });

        if (signUpError) throw signUpError;

        if (!data.session) {
          setEmailModal(emailTrimmed);
          reset('login');
          return;
        }

        // The checkbox was ticked - record acceptance server-side.
        api.acceptTerms().catch(() => {});
        await routeAfterAuth();
      } else {
        const { error: signInError } = await supabase.auth.signInWithPassword({
          email: emailTrimmed,
          password: passwordTrimmed,
        });

        if (signInError) throw signInError;

        await routeAfterAuth();
      }
    } catch (err: any) {
      setError(err?.message ?? (mode === 'login' ? 'Invalid email or password.' : 'Registration failed.'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]}>
      {/* Role toggle: user vs therapist - small link, top right */}
      <View style={styles.topBar}>
        {role === 'user' ? (
          <TouchableOpacity
            testID={T.authRole.therapist}
            style={styles.roleLink}
            onPress={() => pickRole('therapist')}
            hitSlop={{ top: 10, bottom: 10, left: 10, right: 10 }}
            activeOpacity={0.7}
          >
            <Text style={[styles.roleLinkText, { color: colors.textMuted }]}>I'm a therapist</Text>
            <Text style={[styles.roleLinkArrow, { color: colors.textMuted }]}>›</Text>
          </TouchableOpacity>
        ) : (
          <TouchableOpacity
            testID={T.authRole.me}
            style={styles.roleLink}
            onPress={() => pickRole('user')}
            hitSlop={{ top: 10, bottom: 10, left: 10, right: 10 }}
            activeOpacity={0.7}
          >
            <Text style={[styles.roleLinkArrow, { color: colors.textMuted }]}>‹</Text>
            <Text style={[styles.roleLinkText, { color: colors.textMuted }]}>For me</Text>
          </TouchableOpacity>
        )}
      </View>

      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.kav}
      >
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <BreathLineLogo colors={colors} />
          <Text style={[styles.title, { color: colors.textPrimary }]}>Ode</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>Your AI listener that remembers</Text>

          {role === 'therapist' && (
            <Text testID={T.authRole.therapistHint} style={[styles.roleHint, { color: colors.textMuted }]}>
              Manage your clients' session notes, get AI summaries, and use the journal yourself.
            </Text>
          )}

          {/* Tab switcher */}
          <View style={[styles.tabs, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <TouchableOpacity
              testID={T.auth.tabLogin}
              style={[styles.tab, mode === 'login' && { backgroundColor: colors.purple600 }]}
              onPress={() => reset('login')}
            >
              <Text style={[styles.tabText, { color: colors.textMuted }, mode === 'login' && styles.tabTextActive]}>Sign in</Text>
            </TouchableOpacity>
            <TouchableOpacity
              testID={T.auth.tabRegister}
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
              testID={T.auth.emailInput}
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
              testID={T.auth.passwordInput}
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

            {mode === 'register' && (
              <TouchableOpacity
                style={styles.consentRow}
                onPress={() => { setAgreed(a => !a); setError(''); }}
                activeOpacity={0.7}
              >
                <View
                  style={[
                    styles.checkbox,
                    { borderColor: colors.border },
                    agreed && { backgroundColor: colors.purple600, borderColor: colors.purple600 },
                  ]}
                >
                  {agreed && <Text style={styles.checkboxTick}>✓</Text>}
                </View>
                <Text style={[styles.consentText, { color: colors.textSecondary }]}>
                  I have read and agree to the{' '}
                  <Text style={[styles.consentLink, { color: colors.purple500 ?? '#a78bfa' }]} onPress={() => Linking.openURL(TERMS_URL)}>
                    Terms of Service
                  </Text>{' '}
                  and{' '}
                  <Text style={[styles.consentLink, { color: colors.purple500 ?? '#a78bfa' }]} onPress={() => Linking.openURL(PRIVACY_URL)}>
                    Privacy Policy
                  </Text>
                </Text>
              </TouchableOpacity>
            )}

            {!!error && (
              <Text style={styles.errorText}>{error}</Text>
            )}

            <TouchableOpacity
              testID={T.auth.submit}
              style={[
                styles.button,
                { backgroundColor: colors.purple600, shadowColor: colors.purple500 },
                (loading || (mode === 'register' && !agreed)) && styles.buttonLoading,
              ]}
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
    paddingTop: 24,
  },

  logoWrap: {
    marginBottom: 20,
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 0.7,
    shadowRadius: 20,
    elevation: 6,
  },

  topBar: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    paddingHorizontal: 20,
    paddingTop: 8,
  },
  roleLink: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 3,
    paddingVertical: 8,
    paddingHorizontal: 4,
  },
  roleLinkText: {
    fontSize: 12,
    fontFamily: 'HankenGrotesk_600SemiBold',
    letterSpacing: 0.2,
  },
  roleLinkArrow: {
    fontSize: 15,
    fontFamily: 'HankenGrotesk_600SemiBold',
    lineHeight: 15,
  },

  title: {
    fontSize: 32,
    fontFamily: 'Erode_300Light',
    letterSpacing: 1,
    marginBottom: 6,
  },
  subtitle: {
    fontSize: 14,
    fontFamily: 'HankenGrotesk_400Regular',
    letterSpacing: 0.5,
    marginBottom: 32,
  },

  roleHint: {
    fontSize: 12,
    fontFamily: 'HankenGrotesk_400Regular',
    textAlign: 'center',
    marginBottom: 20,
    marginTop: -8,
    paddingHorizontal: 12,
    lineHeight: 17,
  },
  consentRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 10,
    marginTop: 2,
  },
  checkbox: {
    width: 22,
    height: 22,
    borderRadius: 6,
    borderWidth: 1.5,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 1,
  },
  checkboxTick: {
    color: '#fff',
    fontSize: 13,
    fontWeight: '700',
    lineHeight: 16,
  },
  consentText: {
    flex: 1,
    fontSize: 12.5,
    fontFamily: 'HankenGrotesk_400Regular',
    lineHeight: 18,
  },
  consentLink: {
    fontFamily: 'HankenGrotesk_700Bold',
    textDecorationLine: 'underline',
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
    fontFamily: 'HankenGrotesk_600SemiBold',
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
    fontFamily: 'HankenGrotesk_400Regular',
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
    fontFamily: 'HankenGrotesk_600SemiBold',
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
    fontFamily: 'HankenGrotesk_400Regular',
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
    fontFamily: 'Erode_500Medium',
    marginBottom: 4,
  },
  modalBody: {
    fontSize: 14,
    fontFamily: 'HankenGrotesk_400Regular',
    textAlign: 'center',
    lineHeight: 20,
  },
  modalEmail: {
    fontSize: 14,
    fontFamily: 'HankenGrotesk_700Bold',
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
    fontFamily: 'HankenGrotesk_600SemiBold',
    letterSpacing: 0.5,
  },
  errorText: {
    fontSize: 13,
    fontFamily: 'HankenGrotesk_400Regular',
    color: '#ef4444',
    lineHeight: 18,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontFamily: 'HankenGrotesk_600SemiBold',
    letterSpacing: 0.5,
  },
});
