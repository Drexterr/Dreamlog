/**
 * Share with Therapist screen.
 *
 * Opens from the reflection screen via "Share with my therapist".
 * Creates a 72-hour passcode-protected read-only link and presents it
 * with a native Share sheet so the user can send it via WhatsApp / email / etc.
 *
 * Also lists any existing active links so the user can revoke them.
 */

import { useEffect, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  SafeAreaView,
  StatusBar,
  Share,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { CreateShareLinkResult, ShareLink } from '../../src/types';

export default function ShareTherapistScreen() {
  const router = useRouter();
  const { colors } = useTheme();

  const [creating, setCreating] = useState(false);
  const [newLink, setNewLink] = useState<CreateShareLinkResult | null>(null);
  const [activeLinks, setActiveLinks] = useState<ShareLink[]>([]);
  const [loadingLinks, setLoadingLinks] = useState(true);

  useEffect(() => {
    api.listShareLinks()
      .then((r) => setActiveLinks(r.links ?? []))
      .catch(() => {})
      .finally(() => setLoadingLinks(false));
  }, []);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const result = await api.createShareLink();
      setNewLink(result);
      setActiveLinks((prev) => [
        { id: result.token, token: result.token, url: result.url, expires_at: result.expires_at },
        ...prev,
      ]);
    } catch {
      Alert.alert('Error', 'Failed to create share link. Please try again.');
    } finally {
      setCreating(false);
    }
  };

  const handleShare = async (link: CreateShareLinkResult) => {
    try {
      await Share.share({
        message: `Here's my DreamLog emotional summary for the past 30 days.\n\nLink: ${link.url}\nPasscode: ${link.passcode}\n\nThis link expires in 72 hours.`,
        title: 'My DreamLog Summary',
      });
    } catch {}
  };

  const handleRevoke = (linkId: string) => {
    Alert.alert('Revoke link', 'This will immediately invalidate the share link. The therapist will no longer be able to view your data.', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Revoke',
        style: 'destructive',
        onPress: async () => {
          try {
            await api.revokeShareLink(linkId);
            setActiveLinks((prev) => prev.filter((l) => l.id !== linkId));
          } catch {
            Alert.alert('Error', 'Failed to revoke link.');
          }
        },
      },
    ]);
  };

  const formatExpiry = (iso: string) => {
    const d = new Date(iso);
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

          {/* Header */}
          <TouchableOpacity onPress={() => router.back()} style={styles.backBtn}>
            <Text style={[styles.backText, { color: colors.textMuted }]}>← Back</Text>
          </TouchableOpacity>

          <Text style={[styles.title, { color: colors.textPrimary }]}>Share with Therapist</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
            Creates a secure, read-only link showing your 30-day mood trend and AI entry summaries.
            Your therapist never sees raw transcripts or recordings.
          </Text>

          {/* What's shared info box */}
          <View style={[styles.infoBox, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
            <Text style={[styles.infoTitle, { color: colors.brand }]}>What's included</Text>
            {['30-day mood arc (anonymized)', 'AI-generated entry summaries', 'Top emotions for the period', 'Entry count'].map((item) => (
              <Text key={item} style={[styles.infoItem, { color: colors.textSecondary }]}>✓  {item}</Text>
            ))}
            <Text style={[styles.infoExclude, { color: colors.textMuted }]}>
              Not included: raw transcripts, voice recordings, reflections, personal details
            </Text>
          </View>

          {/* New link result */}
          {newLink && (
            <View style={[styles.newLinkCard, { backgroundColor: `${colors.brand}15`, borderColor: `${colors.brand}40` }]}>
              <Text style={[styles.newLinkTitle, { color: colors.brand }]}>Link created ✓</Text>
              <Text style={[styles.newLinkLabel, { color: colors.textMuted }]}>PASSCODE (shown once)</Text>
              <Text style={[styles.passcode, { color: colors.textPrimary }]}>{newLink.passcode}</Text>
              <Text style={[styles.newLinkLabel, { color: colors.textMuted }]}>Expires {formatExpiry(newLink.expires_at)}</Text>
              <TouchableOpacity
                style={[styles.shareBtn, { backgroundColor: colors.brand }]}
                onPress={() => handleShare(newLink)}
                activeOpacity={0.85}
              >
                <Text style={[styles.shareBtnText, { color: colors.bg }]}>Send to therapist</Text>
              </TouchableOpacity>
            </View>
          )}

          {/* Create button */}
          {!newLink && (
            <TouchableOpacity
              style={[styles.createBtn, { backgroundColor: colors.brand }, creating && { opacity: 0.6 }]}
              onPress={handleCreate}
              disabled={creating}
              activeOpacity={0.85}
            >
              {creating ? (
                <ActivityIndicator color={colors.bg} />
              ) : (
                <Text style={[styles.createBtnText, { color: colors.bg }]}>Generate secure link</Text>
              )}
            </TouchableOpacity>
          )}

          {/* Active links */}
          {!loadingLinks && activeLinks.length > 0 && (
            <>
              <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>ACTIVE LINKS</Text>
              <View style={[styles.linksCard, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
                {activeLinks.map((link, idx) => (
                  <View key={link.id}>
                    {idx > 0 && <View style={[styles.divider, { backgroundColor: colors.borderFaint }]} />}
                    <View style={styles.linkRow}>
                      <View style={{ flex: 1 }}>
                        <Text style={[styles.linkToken, { color: colors.textSecondary }]}>
                          {link.token.slice(0, 12)}…
                        </Text>
                        <Text style={[styles.linkExpiry, { color: colors.textMuted }]}>
                          Expires {formatExpiry(link.expires_at)}
                        </Text>
                      </View>
                      <TouchableOpacity onPress={() => handleRevoke(link.id)} activeOpacity={0.7}>
                        <Text style={styles.revokeText}>Revoke</Text>
                      </TouchableOpacity>
                    </View>
                  </View>
                ))}
              </View>
            </>
          )}

          <Text style={[styles.footer, { color: colors.textMuted }]}>
            Links expire automatically after 72 hours. You can revoke any link at any time.
          </Text>
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 24, paddingBottom: 60 },

  backBtn: { marginBottom: 20 },
  backText: { fontSize: 14, fontFamily: 'Nunito_400Regular' },

  title: {
    fontFamily: 'CormorantGaramond_300Light',
    fontSize: 28,
    fontWeight: '300',
    marginBottom: 10,
  },
  subtitle: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 14,
    lineHeight: 22,
    marginBottom: 24,
  },

  infoBox: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 18,
    marginBottom: 24,
  },
  infoTitle: {
    fontFamily: 'Nunito_700Bold',
    fontSize: 12,
    letterSpacing: 1,
    marginBottom: 10,
  },
  infoItem: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 13,
    marginBottom: 5,
    lineHeight: 20,
  },
  infoExclude: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 11,
    marginTop: 8,
    lineHeight: 18,
  },

  newLinkCard: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 20,
    marginBottom: 20,
  },
  newLinkTitle: {
    fontFamily: 'Nunito_700Bold',
    fontSize: 14,
    marginBottom: 14,
  },
  newLinkLabel: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 10,
    letterSpacing: 1.2,
    marginBottom: 4,
  },
  passcode: {
    fontFamily: 'Nunito_700Bold',
    fontSize: 32,
    letterSpacing: 8,
    marginBottom: 12,
  },
  shareBtn: {
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: 'center',
    marginTop: 8,
  },
  shareBtnText: {
    fontFamily: 'Nunito_700Bold',
    fontSize: 15,
  },

  createBtn: {
    borderRadius: 16,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 32,
  },
  createBtnText: {
    fontFamily: 'Nunito_700Bold',
    fontSize: 15,
  },

  sectionLabel: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 10,
    letterSpacing: 1.5,
    marginBottom: 10,
  },
  linksCard: {
    borderRadius: 16,
    borderWidth: 1,
    overflow: 'hidden',
    marginBottom: 24,
  },
  linkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 14,
    gap: 12,
  },
  divider: { height: 1, marginLeft: 14 },
  linkToken: { fontFamily: 'Nunito_400Regular', fontSize: 13, marginBottom: 2 },
  linkExpiry: { fontFamily: 'Nunito_400Regular', fontSize: 11 },
  revokeText: { color: '#ef4444', fontFamily: 'Nunito_600SemiBold', fontSize: 13 },

  footer: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 12,
    lineHeight: 18,
    textAlign: 'center',
    marginTop: 8,
  },
});
