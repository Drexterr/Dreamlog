/**
 * Share with Therapist screen.
 *
 * Opens from the reflection screen via "Share with my therapist".
 * Creates a 72-hour passcode-protected read-only link and presents it
 * with a native Share sheet so the user can send it via WhatsApp / email / etc.
 *
 * Also lists any existing active links so the user can revoke them.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  
  StatusBar,
  Share,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { CreateShareLinkResult, ShareLink } from '../../src/types';
import { T } from '../../src/testIDs';
import { Palette } from '../../src/theme';

export default function ShareTherapistScreen() {
  const router = useRouter();
  const { colors } = useTheme();

  const [creating, setCreating] = useState(false);
  const [newLink, setNewLink] = useState<CreateShareLinkResult | null>(null);
  const [activeLinks, setActiveLinks] = useState<ShareLink[]>([]);
  const [loadingLinks, setLoadingLinks] = useState(true);
  const [linksError, setLinksError] = useState(false);
  const [revoking, setRevoking] = useState<string[]>([]);
  const mountedRef = useRef(false);
  const listRequestRef = useRef(0);
  const creatingRef = useRef(false);
  const revokingRef = useRef(new Set<string>());

  const loadLinks = useCallback(async () => {
    if (!mountedRef.current) return;
    const request = ++listRequestRef.current;
    setLoadingLinks(true);
    setLinksError(false);
    try {
      const result = await api.listShareLinks();
      if (mountedRef.current && request === listRequestRef.current) setActiveLinks(result.links ?? []);
    } catch {
      if (mountedRef.current && request === listRequestRef.current) setLinksError(true);
    } finally {
      if (mountedRef.current && request === listRequestRef.current) setLoadingLinks(false);
    }
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    void loadLinks();
    return () => { mountedRef.current = false; listRequestRef.current++; };
  }, [loadLinks]);

  const handleCreate = async () => {
    if (creatingRef.current || newLink) return;
    creatingRef.current = true;
    setCreating(true);
    try {
      const result = await api.createShareLink();
      if (!mountedRef.current) return;
      setNewLink(result);
      // Creation returns a public token, not the UUID required for revocation.
      await loadLinks();
    } catch {
      if (mountedRef.current) Alert.alert('Error', 'Failed to create share link. Please try again.');
    } finally {
      creatingRef.current = false;
      if (mountedRef.current) setCreating(false);
    }
  };

  const handleShare = async (link: CreateShareLinkResult) => {
    try {
      await Share.share({
        message: `Here's my Ode emotional summary for the past 30 days.\n\nLink: ${link.url}\nPasscode: ${link.passcode}\n\nThis link expires in 72 hours.`,
        title: 'My Ode Summary',
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
          if (revokingRef.current.has(linkId)) return;
          revokingRef.current.add(linkId);
          setRevoking((prev) => [...prev, linkId]);
          try {
            await api.revokeShareLink(linkId);
            if (!mountedRef.current) return;
            listRequestRef.current++;
            setLoadingLinks(false);
            const revokedToken = activeLinks.find((link) => link.id === linkId)?.token;
            setNewLink((current) => current?.token === revokedToken ? null : current);
            setActiveLinks((prev) => prev.filter((l) => l.id !== linkId));
          } catch {
            if (mountedRef.current) Alert.alert('Error', 'Failed to revoke link.');
          } finally {
            revokingRef.current.delete(linkId);
            if (mountedRef.current) setRevoking((prev) => prev.filter((id) => id !== linkId));
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
    <View testID={T.share.screen} style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

          {/* Header */}
          <TouchableOpacity testID={T.share.back} onPress={() => router.back()} style={styles.backBtn}>
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
            {['30-day mood arc', 'AI-generated entry summaries', 'Top emotions for the period', 'Your display name'].map((item) => (
              <Text key={item} style={[styles.infoItem, { color: colors.textSecondary }]}>✓  {item}</Text>
            ))}
            <Text style={[styles.infoExclude, { color: colors.textMuted }]}>
              Not included: raw transcripts, voice recordings, reflections
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
                testID={T.share.sendButton}
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
              testID={T.share.createButton}
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
          {loadingLinks && <ActivityIndicator testID={T.share.linksLoading} color={colors.brand} />}
          {linksError && (
            <View>
              <Text style={[styles.footer, { color: colors.textSecondary }]}>Could not load active links.</Text>
              <TouchableOpacity testID={T.share.retryButton} onPress={() => void loadLinks()} style={styles.backBtn}>
                <Text style={[styles.backText, { color: colors.brand }]}>Retry</Text>
              </TouchableOpacity>
            </View>
          )}
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
                      <TouchableOpacity testID={T.share.revokeButton} onPress={() => handleRevoke(link.id)} disabled={revoking.includes(link.id)} activeOpacity={0.7}>
                        <Text style={styles.revokeText}>{revoking.includes(link.id) ? 'Revoking...' : 'Revoke'}</Text>
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
  backText: { fontSize: 14, fontFamily: 'HankenGrotesk_400Regular' },

  title: {
    fontFamily: 'Erode_300Light',
    fontSize: 28,
    fontWeight: '300',
    marginBottom: 10,
  },
  subtitle: {
    fontFamily: 'HankenGrotesk_400Regular',
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
    fontFamily: 'HankenGrotesk_700Bold',
    fontSize: 12,
    letterSpacing: 1,
    marginBottom: 10,
  },
  infoItem: {
    fontFamily: 'HankenGrotesk_400Regular',
    fontSize: 13,
    marginBottom: 5,
    lineHeight: 20,
  },
  infoExclude: {
    fontFamily: 'HankenGrotesk_400Regular',
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
    fontFamily: 'HankenGrotesk_700Bold',
    fontSize: 14,
    marginBottom: 14,
  },
  newLinkLabel: {
    fontFamily: 'HankenGrotesk_600SemiBold',
    fontSize: 10,
    letterSpacing: 1.2,
    marginBottom: 4,
  },
  passcode: {
    fontFamily: 'HankenGrotesk_700Bold',
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
    fontFamily: 'HankenGrotesk_700Bold',
    fontSize: 15,
  },

  createBtn: {
    borderRadius: 16,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 32,
  },
  createBtnText: {
    fontFamily: 'HankenGrotesk_700Bold',
    fontSize: 15,
  },

  sectionLabel: {
    fontFamily: 'HankenGrotesk_600SemiBold',
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
  linkToken: { fontFamily: 'HankenGrotesk_400Regular', fontSize: 13, marginBottom: 2 },
  linkExpiry: { fontFamily: 'HankenGrotesk_400Regular', fontSize: 11 },
  revokeText: { color: Palette.error, fontFamily: 'HankenGrotesk_600SemiBold', fontSize: 13 },

  footer: {
    fontFamily: 'HankenGrotesk_400Regular',
    fontSize: 12,
    lineHeight: 18,
    textAlign: 'center',
    marginTop: 8,
  },
});
