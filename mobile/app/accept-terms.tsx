import { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
  Linking,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';
import { resolvePostAuthRoute } from '../src/services/postAuthRoute';
import { T } from '../src/testIDs';

const TERMS_URL = 'https://dreamlog.app/terms';
const PRIVACY_URL = 'https://dreamlog.app/privacy';

// One-time terms acceptance gate. Shown after sign-in when the user has not
// accepted the current Terms of Service version - covers Google/Apple sign-in
// (no form checkbox) and existing users after a terms update.
export default function AcceptTermsScreen() {
  const [agreed, setAgreed] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const router = useRouter();
  const { colors } = useTheme();

  const handleContinue = async () => {
    if (!agreed || loading) return;
    setLoading(true);
    setError('');
    try {
      await api.acceptTerms();
      const dest = await resolvePostAuthRoute({ skipTermsGate: true });
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      router.replace(dest as any);
    } catch {
      setError('Could not save your acceptance. Please check your connection and try again.');
      setLoading(false);
    }
  };

  return (
    <SafeAreaView testID={T.acceptTerms.screen} style={[styles.container, { backgroundColor: colors.bg }]}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={[styles.iconWrap, { backgroundColor: colors.purple600 + '33' }]}>
          <Text style={styles.icon}>📜</Text>
        </View>
        <Text style={[styles.title, { color: colors.textPrimary }]}>Before you continue</Text>
        <Text style={[styles.body, { color: colors.textSecondary }]}>
          Ode stores your voice-journal transcripts and reflections to give you personalized
          insights. Please review and accept our terms to use the app.
        </Text>

        <View style={[styles.card, { backgroundColor: colors.card, borderColor: colors.border }]}>
          <TouchableOpacity style={styles.linkRow} onPress={() => Linking.openURL(TERMS_URL)}>
            <Text style={[styles.linkText, { color: colors.textPrimary }]}>Terms of Service</Text>
            <Text style={[styles.chevron, { color: colors.textMuted }]}>›</Text>
          </TouchableOpacity>
          <View style={[styles.separator, { backgroundColor: colors.border }]} />
          <TouchableOpacity style={styles.linkRow} onPress={() => Linking.openURL(PRIVACY_URL)}>
            <Text style={[styles.linkText, { color: colors.textPrimary }]}>Privacy Policy</Text>
            <Text style={[styles.chevron, { color: colors.textMuted }]}>›</Text>
          </TouchableOpacity>
        </View>

        <TouchableOpacity testID={T.acceptTerms.checkbox} style={styles.consentRow} onPress={() => setAgreed(a => !a)} activeOpacity={0.7}>
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
            I have read and agree to the Terms of Service and Privacy Policy
          </Text>
        </TouchableOpacity>

        {!!error && <Text style={styles.errorText}>{error}</Text>}

        <TouchableOpacity
          testID={T.acceptTerms.continue}
          style={[
            styles.button,
            { backgroundColor: colors.purple600, shadowColor: colors.purple500 },
            (!agreed || loading) && styles.buttonDisabled,
          ]}
          onPress={handleContinue}
          disabled={!agreed || loading}
          activeOpacity={0.8}
        >
          {loading ? (
            <ActivityIndicator color="#fff" size="small" />
          ) : (
            <Text style={styles.buttonText}>Agree and continue</Text>
          )}
        </TouchableOpacity>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: {
    padding: 28,
    paddingTop: 72,
    alignItems: 'center',
  },
  iconWrap: {
    width: 72,
    height: 72,
    borderRadius: 36,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 20,
  },
  icon: { fontSize: 32 },
  title: {
    fontSize: 28,
    fontFamily: 'Erode_500Medium',
    marginBottom: 12,
    textAlign: 'center',
  },
  body: {
    fontSize: 14,
    fontFamily: 'HankenGrotesk_400Regular',
    lineHeight: 21,
    textAlign: 'center',
    marginBottom: 28,
  },
  card: {
    width: '100%',
    borderRadius: 16,
    borderWidth: 1,
    marginBottom: 24,
  },
  linkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: 16,
  },
  linkText: {
    fontSize: 15,
    fontFamily: 'HankenGrotesk_600SemiBold',
  },
  chevron: { fontSize: 22, lineHeight: 22 },
  separator: { height: 1, marginHorizontal: 16 },
  consentRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 10,
    width: '100%',
    marginBottom: 20,
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
    fontSize: 13.5,
    fontFamily: 'HankenGrotesk_400Regular',
    lineHeight: 19,
  },
  errorText: {
    fontSize: 13,
    fontFamily: 'HankenGrotesk_400Regular',
    color: '#ef4444',
    marginBottom: 12,
    textAlign: 'center',
  },
  button: {
    width: '100%',
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 6,
  },
  buttonDisabled: { opacity: 0.5 },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontFamily: 'HankenGrotesk_600SemiBold',
    letterSpacing: 0.5,
  },
});
