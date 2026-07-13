import { useEffect, useState } from 'react';
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
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { T } from '../../src/testIDs';

// Therapist registration: creates the therapist profile and records the
// client-data responsibility consent (required before any client can be added).
export default function TherapistRegisterScreen() {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [credentials, setCredentials] = useState('');
  const [consented, setConsented] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const router = useRouter();
  const { colors } = useTheme();

  // Prefill from the signed-in account.
  useEffect(() => {
    api.me()
      .then(u => {
        setName(prev => prev || u.name || '');
        setEmail(prev => prev || u.email || '');
      })
      .catch(() => {});
  }, []);

  const handleSubmit = async () => {
    const nameTrimmed = name.trim();
    const emailTrimmed = email.trim();
    if (!nameTrimmed || !emailTrimmed) {
      setError('Name and email are required.');
      return;
    }
    if (!consented) {
      setError('You must accept the client-data responsibility terms.');
      return;
    }
    setError('');
    setLoading(true);
    try {
      await api.registerTherapist(nameTrimmed, emailTrimmed, credentials.trim());
      await api.acceptTherapistConsent();
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      router.replace('/therapist' as any);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Registration failed. Please try again.';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]}>
      <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <Text style={[styles.title, { color: colors.textPrimary }]}>Therapist profile</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
            Set up your practice workspace. You can manage your own clients' session notes and also
            see Ode users who share their journal with you.
          </Text>

          <View style={[styles.card, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <TextInput
              testID={T.therapistPortal.registerNameInput}
              style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint, color: colors.textPrimary }]}
              value={name}
              onChangeText={v => { setName(v); setError(''); }}
              placeholder="Full name"
              placeholderTextColor={colors.textFaint}
              autoCapitalize="words"
            />
            <TextInput
              testID={T.therapistPortal.registerEmailInput}
              style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint, color: colors.textPrimary }]}
              value={email}
              onChangeText={v => { setEmail(v); setError(''); }}
              placeholder="Professional email"
              placeholderTextColor={colors.textFaint}
              keyboardType="email-address"
              autoCapitalize="none"
            />
            <TextInput
              testID={T.therapistPortal.registerCredentialsInput}
              style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint, color: colors.textPrimary }]}
              value={credentials}
              onChangeText={setCredentials}
              placeholder="Credentials (e.g. M.Phil Clinical Psychology)"
              placeholderTextColor={colors.textFaint}
            />
          </View>

          <View style={[styles.consentCard, { backgroundColor: colors.card, borderColor: colors.border }]}>
            <Text style={[styles.consentTitle, { color: colors.textPrimary }]}>Client data responsibility</Text>
            <Text style={[styles.consentBody, { color: colors.textSecondary }]}>
              • I confirm I have my clients' consent to store notes about our sessions.{'\n'}
              • I am responsible for the client information I upload.{'\n'}
              • I will use identifiers my clients are comfortable with (first name or initials are
              recommended).{'\n'}
              • Notes are encrypted at rest; note photos are deleted right after text extraction.
            </Text>
            <TouchableOpacity testID={T.therapistPortal.registerConsentCheckbox} style={styles.consentRow} onPress={() => { setConsented(c => !c); setError(''); }} activeOpacity={0.7}>
              <View
                style={[
                  styles.checkbox,
                  { borderColor: colors.border },
                  consented && { backgroundColor: colors.purple600, borderColor: colors.purple600 },
                ]}
              >
                {consented && <Text style={styles.checkboxTick}>✓</Text>}
              </View>
              <Text style={[styles.consentText, { color: colors.textSecondary }]}>
                I understand and accept these terms
              </Text>
            </TouchableOpacity>
          </View>

          {!!error && <Text style={styles.errorText}>{error}</Text>}

          <TouchableOpacity
            testID={T.therapistPortal.registerSubmit}
            style={[
              styles.button,
              { backgroundColor: colors.purple600, shadowColor: colors.purple500 },
              (loading || !consented) && styles.buttonDisabled,
            ]}
            onPress={handleSubmit}
            disabled={loading}
            activeOpacity={0.8}
          >
            {loading ? <ActivityIndicator color="#fff" size="small" /> : <Text style={styles.buttonText}>Create workspace</Text>}
          </TouchableOpacity>

          <TouchableOpacity testID={T.therapistPortal.registerGoToJournal} onPress={() => router.replace('/(tabs)')} style={styles.skipBtn}>
            <Text style={[styles.skipText, { color: colors.textMuted }]}>Not a therapist? Go to my journal</Text>
          </TouchableOpacity>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 24, paddingTop: 48 },
  title: {
    fontSize: 30,
    fontFamily: 'CormorantGaramond_500Medium',
    marginBottom: 10,
  },
  subtitle: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 20,
    marginBottom: 24,
  },
  card: {
    borderRadius: 18,
    borderWidth: 1,
    padding: 18,
    gap: 12,
    marginBottom: 16,
  },
  input: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 14,
    fontFamily: 'Nunito_400Regular',
    fontSize: 15,
  },
  consentCard: {
    borderRadius: 18,
    borderWidth: 1,
    padding: 18,
    marginBottom: 16,
  },
  consentTitle: {
    fontSize: 16,
    fontFamily: 'Nunito_700Bold',
    marginBottom: 10,
  },
  consentBody: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 21,
    marginBottom: 14,
  },
  consentRow: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  checkbox: {
    width: 22,
    height: 22,
    borderRadius: 6,
    borderWidth: 1.5,
    alignItems: 'center',
    justifyContent: 'center',
  },
  checkboxTick: { color: '#fff', fontSize: 13, fontWeight: '700', lineHeight: 16 },
  consentText: { flex: 1, fontSize: 13.5, fontFamily: 'Nunito_600SemiBold' },
  errorText: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    color: '#ef4444',
    marginBottom: 12,
  },
  button: {
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 6,
  },
  buttonDisabled: { opacity: 0.5 },
  buttonText: { color: '#fff', fontSize: 16, fontFamily: 'Nunito_600SemiBold', letterSpacing: 0.5 },
  skipBtn: { alignItems: 'center', paddingVertical: 18 },
  skipText: { fontSize: 13, fontFamily: 'Nunito_400Regular', textDecorationLine: 'underline' },
});
