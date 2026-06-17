import { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  Modal,
  Platform,
  KeyboardAvoidingView,
  ScrollView,
} from 'react-native';
import * as AppleAuthentication from 'expo-apple-authentication';
import { GoogleSignin, statusCodes, isErrorWithCode } from '@react-native-google-signin/google-signin';
import { supabase } from '../lib/supabase';
import { useTheme } from '../context/ThemeContext';

type Mode = 'social' | 'email';

interface AuthSheetProps {
  visible: boolean;
  prompt?: string;
  onClose: () => void;
}

function GoogleLogo() {
  return (
    <View style={logo.wrap}>
      <View style={logo.ring} />
      <View style={logo.bar} />
      <Text style={logo.letter}>G</Text>
    </View>
  );
}

const logo = StyleSheet.create({
  wrap:   { width: 20, height: 20, borderRadius: 10, backgroundColor: '#fff', alignItems: 'center', justifyContent: 'center' },
  ring:   { position: 'absolute', width: 20, height: 20, borderRadius: 10, borderWidth: 3, borderColor: '#4285F4', borderRightColor: '#34A853', borderBottomColor: '#FBBC05', transform: [{ rotate: '-45deg' }] },
  bar:    { position: 'absolute', right: 0, top: '50%', width: 8, height: 3, backgroundColor: '#4285F4', marginTop: -1.5 },
  letter: { fontSize: 11, fontWeight: '700', color: '#4285F4', lineHeight: 20 },
});

export default function AuthSheet({ visible, prompt, onClose }: AuthSheetProps) {
  const { colors } = useTheme();
  const [mode, setMode] = useState<Mode>('social');
  const [emailMode, setEmailMode] = useState<'login' | 'register'>('login');
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  function reset() {
    setMode('social');
    setEmailMode('login');
    setEmail('');
    setName('');
    setPassword('');
    setError('');
    setLoading(false);
  }

  function handleClose() {
    reset();
    onClose();
  }

  async function handleGoogle() {
    setError('');
    setLoading(true);
    try {
      await GoogleSignin.hasPlayServices();
      const response = await GoogleSignin.signIn();
      if (response.type === 'cancelled') { setLoading(false); return; }
      const idToken = response.data?.idToken;
      if (!idToken) throw new Error('No ID token from Google');
      const { error: err } = await supabase.auth.signInWithIdToken({ provider: 'google', token: idToken });
      if (err) throw err;
      // onAuthStateChange in _layout.tsx handles the rest
    } catch (err: unknown) {
      if (isErrorWithCode(err) && (err as any).code === statusCodes.SIGN_IN_CANCELLED) {
        setLoading(false);
        return;
      }
      setError((err as any)?.message ?? 'Google sign-in failed');
      setLoading(false);
    }
  }

  async function handleApple() {
    setError('');
    setLoading(true);
    try {
      const cred = await AppleAuthentication.signInAsync({
        requestedScopes: [
          AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
          AppleAuthentication.AppleAuthenticationScope.EMAIL,
        ],
      });
      if (!cred.identityToken) throw new Error('No identity token from Apple');
      const { error: err } = await supabase.auth.signInWithIdToken({ provider: 'apple', token: cred.identityToken });
      if (err) throw err;
      // onAuthStateChange in _layout.tsx handles the rest
    } catch (err: unknown) {
      if ((err as any)?.code === 'ERR_REQUEST_CANCELED') { setLoading(false); return; }
      setError((err as any)?.message ?? 'Apple sign-in failed');
      setLoading(false);
    }
  }

  async function handleEmailSubmit() {
    const e = email.trim();
    const p = password.trim();
    const n = name.trim();
    if (!e || !p) { setError('Email and password are required.'); return; }
    if (emailMode === 'register' && !n) { setError('Name is required.'); return; }
    if (emailMode === 'register' && p.length < 6) { setError('Password must be at least 6 characters.'); return; }
    setError('');
    setLoading(true);
    try {
      if (emailMode === 'register') {
        const { data, error: err } = await supabase.auth.signUp({
          email: e, password: p,
          options: { data: { full_name: n }, emailRedirectTo: 'dreamlog://auth/callback' },
        });
        if (err) throw err;
        if (!data.session) {
          // Email verification required - show a message and close
          handleClose();
          return;
        }
        // onAuthStateChange handles the rest
      } else {
        const { error: err } = await supabase.auth.signInWithPassword({ email: e, password: p });
        if (err) throw err;
        // onAuthStateChange handles the rest
      }
    } catch (err: unknown) {
      setError((err as any)?.message ?? (emailMode === 'login' ? 'Invalid email or password.' : 'Registration failed.'));
      setLoading(false);
    }
  }

  return (
    <Modal
      visible={visible}
      transparent
      animationType="slide"
      statusBarTranslucent
      onRequestClose={handleClose}
    >
      <TouchableOpacity style={s.backdrop} onPress={handleClose} activeOpacity={1} />

      <KeyboardAvoidingView
        style={s.sheetWrap}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        <View style={[s.sheet, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}>
          {/* Handle */}
          <View style={[s.handle, { backgroundColor: colors.borderFaint }]} />

          <ScrollView
            contentContainerStyle={s.content}
            keyboardShouldPersistTaps="handled"
            showsVerticalScrollIndicator={false}
          >
            {/* Header */}
            <Text style={[s.title, { color: colors.textPrimary }]}>
              {prompt ?? 'Sign in to continue'}
            </Text>
            <Text style={[s.sub, { color: colors.textSecondary }]}>
              Your data stays private. No ads, no sharing.
            </Text>

            {/* Social buttons */}
            <TouchableOpacity
              style={[s.socialBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
              onPress={handleGoogle}
              disabled={loading}
              activeOpacity={0.8}
            >
              {loading ? <ActivityIndicator color={colors.textMuted} size="small" /> : (
                <>
                  <GoogleLogo />
                  <Text style={[s.socialBtnText, { color: colors.textPrimary }]}>Continue with Google</Text>
                </>
              )}
            </TouchableOpacity>

            {Platform.OS === 'ios' && (
              <AppleAuthentication.AppleAuthenticationButton
                buttonType={AppleAuthentication.AppleAuthenticationButtonType.SIGN_IN}
                buttonStyle={AppleAuthentication.AppleAuthenticationButtonStyle.WHITE}
                cornerRadius={12}
                style={s.appleBtn}
                onPress={handleApple}
              />
            )}

            {/* Email toggle */}
            {mode === 'social' ? (
              <TouchableOpacity onPress={() => setMode('email')} activeOpacity={0.7} style={s.emailToggle}>
                <Text style={[s.emailToggleText, { color: colors.textMuted, borderBottomColor: colors.borderFaint }]}>
                  Use email instead
                </Text>
              </TouchableOpacity>
            ) : (
              <View style={s.emailSection}>
                {/* Login/Register tabs */}
                <View style={[s.tabs, { backgroundColor: colors.card, borderColor: colors.border }]}>
                  {(['login', 'register'] as const).map((m) => (
                    <TouchableOpacity
                      key={m}
                      style={[s.tab, emailMode === m && { backgroundColor: colors.purple600 }]}
                      onPress={() => { setEmailMode(m); setError(''); }}
                    >
                      <Text style={[s.tabText, { color: colors.textMuted }, emailMode === m && { color: '#fff' }]}>
                        {m === 'login' ? 'Sign in' : 'Create account'}
                      </Text>
                    </TouchableOpacity>
                  ))}
                </View>

                {emailMode === 'register' && (
                  <TextInput
                    style={[s.input, { backgroundColor: colors.card, borderColor: colors.borderFaint, color: colors.textPrimary }]}
                    value={name}
                    onChangeText={(v) => { setName(v); setError(''); }}
                    placeholder="Your name"
                    placeholderTextColor={colors.textFaint}
                    autoCapitalize="words"
                    autoCorrect={false}
                  />
                )}

                <TextInput
                  style={[s.input, { backgroundColor: colors.card, borderColor: colors.borderFaint, color: colors.textPrimary }]}
                  value={email}
                  onChangeText={(v) => { setEmail(v); setError(''); }}
                  placeholder="Email"
                  placeholderTextColor={colors.textFaint}
                  keyboardType="email-address"
                  autoCapitalize="none"
                  autoCorrect={false}
                />

                <TextInput
                  style={[s.input, { backgroundColor: colors.card, borderColor: colors.borderFaint, color: colors.textPrimary }]}
                  value={password}
                  onChangeText={(v) => { setPassword(v); setError(''); }}
                  placeholder={emailMode === 'register' ? 'Password (min 6 chars)' : 'Password'}
                  placeholderTextColor={colors.textFaint}
                  secureTextEntry
                  autoCapitalize="none"
                  autoCorrect={false}
                  returnKeyType="done"
                  onSubmitEditing={handleEmailSubmit}
                />

                {!!error && <Text style={s.error}>{error}</Text>}

                <TouchableOpacity
                  style={[s.submitBtn, { backgroundColor: colors.purple600 }, loading && { opacity: 0.6 }]}
                  onPress={handleEmailSubmit}
                  disabled={loading}
                  activeOpacity={0.8}
                >
                  {loading
                    ? <ActivityIndicator color="#fff" size="small" />
                    : <Text style={s.submitBtnText}>{emailMode === 'login' ? 'Sign in' : 'Create account'}</Text>
                  }
                </TouchableOpacity>

                <TouchableOpacity onPress={() => { setMode('social'); setError(''); }} activeOpacity={0.7} style={s.backLink}>
                  <Text style={[s.backLinkText, { color: colors.textMuted }]}>← Back to social login</Text>
                </TouchableOpacity>
              </View>
            )}

            {/* Dismiss */}
            <TouchableOpacity onPress={handleClose} activeOpacity={0.7} style={s.laterBtn}>
              <Text style={[s.laterText, { color: colors.textFaint }]}>Later</Text>
            </TouchableOpacity>
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const s = StyleSheet.create({
  backdrop: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0,0,0,0.55)',
  },
  sheetWrap: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
  },
  sheet: {
    borderTopLeftRadius: 28,
    borderTopRightRadius: 28,
    borderWidth: 1,
    borderBottomWidth: 0,
    paddingBottom: 32,
  },
  handle: {
    width: 40,
    height: 4,
    borderRadius: 2,
    alignSelf: 'center',
    marginTop: 12,
    marginBottom: 4,
  },
  content: {
    padding: 24,
    paddingTop: 12,
    gap: 12,
  },
  title: {
    fontFamily: 'CormorantGaramond_300Light',
    fontSize: 26,
    marginBottom: 2,
  },
  sub: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 13,
    lineHeight: 18,
    marginBottom: 4,
  },
  socialBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
    borderRadius: 12,
    borderWidth: 1,
    paddingVertical: 14,
  },
  socialBtnText: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 15,
  },
  appleBtn: {
    width: '100%',
    height: 50,
  },
  emailToggle: {
    alignSelf: 'center',
    paddingVertical: 4,
  },
  emailToggleText: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 13,
    borderBottomWidth: 1,
    paddingBottom: 1,
  },
  emailSection: {
    gap: 10,
  },
  tabs: {
    flexDirection: 'row',
    borderRadius: 12,
    borderWidth: 1,
    padding: 4,
  },
  tab: {
    flex: 1,
    paddingVertical: 8,
    alignItems: 'center',
    borderRadius: 8,
  },
  tabText: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 13,
  },
  input: {
    borderRadius: 10,
    borderWidth: 1,
    padding: 12,
    fontFamily: 'Nunito_400Regular',
    fontSize: 14,
  },
  error: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 12,
    color: '#ef4444',
  },
  submitBtn: {
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: 'center',
  },
  submitBtnText: {
    color: '#fff',
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 15,
  },
  backLink: {
    alignSelf: 'flex-start',
  },
  backLinkText: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 13,
  },
  laterBtn: {
    alignSelf: 'center',
    paddingVertical: 8,
    marginTop: 4,
  },
  laterText: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 13,
  },
});
