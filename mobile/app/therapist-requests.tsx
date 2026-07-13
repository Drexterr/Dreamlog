/**
 * Therapist Link Requests — client-facing consent screen.
 *
 * A therapist can request a link to a client (any authenticated user) by their
 * DreamLog user ID. That link stays in `pending` and grants no access until the
 * client approves it here. This screen lists those pending requests and lets the
 * client approve (grant the therapist read access to their journal summaries) or
 * decline (reject / revoke).
 *
 * Reached from Settings → Privacy → Therapist requests.
 */

import { useCallback, useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  StyleSheet,
  StatusBar,
  ActivityIndicator,
  TouchableOpacity,
  RefreshControl,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api, isApiError } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';
import type { ThemeColors } from '../src/theme';
import type { TherapistLinkRequest } from '../src/types';
import { T } from '../src/testIDs';

function relativeDate(iso: string): string {
  const then = new Date(iso);
  if (isNaN(then.getTime())) return '';
  const days = Math.floor((Date.now() - then.getTime()) / 86_400_000);
  if (days <= 0) return 'today';
  if (days === 1) return 'yesterday';
  if (days < 7) return `${days} days ago`;
  if (days < 14) return 'last week';
  if (days < 30) return `${Math.floor(days / 7)} weeks ago`;
  if (days < 60) return 'last month';
  if (days < 365) return `${Math.floor(days / 30)} months ago`;
  return then.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
}

function initial(name: string): string {
  return (name.trim().charAt(0) || '?').toUpperCase();
}

export default function TherapistRequestsScreen() {
  const router = useRouter();
  const { colors } = useTheme();
  const styles = getStyles(colors);

  const [requests, setRequests] = useState<TherapistLinkRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState(false);
  // therapist_id currently being approved/declined — disables both its buttons.
  const [pendingId, setPendingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(false);
    try {
      const res = await api.listLinkRequests();
      setRequests(res.requests ?? []);
    } catch {
      setError(true);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const onRefresh = useCallback(() => {
    setRefreshing(true);
    load();
  }, [load]);

  const removeRow = useCallback((therapistID: string) => {
    setRequests((prev) => prev.filter((r) => r.therapist_id !== therapistID));
  }, []);

  const runAction = useCallback(
    async (req: TherapistLinkRequest, action: 'approve' | 'decline') => {
      if (pendingId) return; // an action is already in flight
      setPendingId(req.therapist_id);
      try {
        if (action === 'approve') {
          await api.approveLink(req.therapist_id);
        } else {
          await api.declineLink(req.therapist_id);
        }
        removeRow(req.therapist_id);
      } catch (e) {
        // 404 = the request no longer exists (already actioned elsewhere).
        // Drop the stale row and re-sync the list rather than erroring.
        if (isApiError(e) && e.response?.status === 404) {
          removeRow(req.therapist_id);
          load();
        } else {
          Alert.alert('Something went wrong', 'Could not update this request. Please try again.');
        }
      } finally {
        setPendingId(null);
      }
    },
    [pendingId, removeRow, load],
  );

  const confirmApprove = useCallback(
    (req: TherapistLinkRequest) => {
      Alert.alert(
        `Approve ${req.therapist_name}?`,
        'They will be able to see summaries of your journal — mood trends, entry summaries, and top emotions. Your transcripts and reflections are never shared. You can revoke access at any time.',
        [
          { text: 'Cancel', style: 'cancel' },
          { text: 'Approve', onPress: () => runAction(req, 'approve') },
        ],
      );
    },
    [runAction],
  );

  const confirmDecline = useCallback(
    (req: TherapistLinkRequest) => {
      Alert.alert(
        `Decline ${req.therapist_name}?`,
        "They won't be able to access any of your journal data.",
        [
          { text: 'Cancel', style: 'cancel' },
          { text: 'Decline', style: 'destructive', onPress: () => runAction(req, 'decline') },
        ],
      );
    },
    [runAction],
  );

  return (
    <View testID={T.therapistRequests.screen} style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView
          contentContainerStyle={styles.scroll}
          showsVerticalScrollIndicator={false}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={colors.brand} />
          }
        >
          <TouchableOpacity testID={T.therapistRequests.back} onPress={() => router.back()} style={styles.backBtn}>
            <Text style={styles.backText}>← Back</Text>
          </TouchableOpacity>

          <Text style={styles.title}>Therapist Requests</Text>
          <Text style={styles.subtitle}>
            Therapists who&apos;ve asked to view your journal summaries. Approve only people you trust.
          </Text>

          {loading ? (
            <View style={styles.center}>
              <ActivityIndicator color={colors.brand} />
            </View>
          ) : error ? (
            <View style={styles.emptyBox}>
              <Text style={styles.emptyText}>
                Couldn&apos;t load your requests. Pull down to try again.
              </Text>
            </View>
          ) : requests.length === 0 ? (
            <View style={styles.emptyBox}>
              <View style={styles.emptyIcon}>
                <View style={styles.emptyRing} />
                <View style={[styles.emptyRing, styles.emptyRingOverlap]} />
              </View>
              <Text style={styles.emptyText}>
                No pending requests. When a therapist asks to link with you, it&apos;ll appear here for
                your approval.
              </Text>
            </View>
          ) : (
            <>
              <Text style={styles.countLine}>
                {requests.length} pending {requests.length === 1 ? 'request' : 'requests'}
              </Text>

              {requests.map((req) => {
                const busy = pendingId === req.therapist_id;
                return (
                  <View key={req.therapist_id} style={styles.card}>
                    <View style={styles.cardHead}>
                      <View style={styles.avatar}>
                        <Text style={styles.avatarInitial}>{initial(req.therapist_name)}</Text>
                      </View>
                      <View style={{ flex: 1 }}>
                        <Text style={styles.name}>{req.therapist_name}</Text>
                        {req.credentials ? (
                          <Text style={styles.credentials}>{req.credentials}</Text>
                        ) : null}
                        <Text style={styles.meta}>Requested {relativeDate(req.requested_at)}</Text>
                      </View>
                    </View>

                    <View style={styles.actionsRow}>
                      <TouchableOpacity
                        testID={T.therapistRequests.declineButton}
                        style={[styles.declineBtn, busy && styles.btnDisabled]}
                        onPress={() => confirmDecline(req)}
                        disabled={busy}
                        activeOpacity={0.8}
                      >
                        <Text style={styles.declineText}>Decline</Text>
                      </TouchableOpacity>
                      <TouchableOpacity
                        testID={T.therapistRequests.approveButton}
                        style={[styles.approveBtn, busy && styles.btnDisabled]}
                        onPress={() => confirmApprove(req)}
                        disabled={busy}
                        activeOpacity={0.8}
                      >
                        {busy ? (
                          <ActivityIndicator color={colors.bg} size="small" />
                        ) : (
                          <Text style={styles.approveText}>Approve</Text>
                        )}
                      </TouchableOpacity>
                    </View>
                  </View>
                );
              })}

              <Text style={styles.footer}>
                Approving shares only your journal summaries — never your recordings, transcripts, or
                reflections. You can revoke a therapist&apos;s access at any time.
              </Text>
            </>
          )}
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const getStyles = (colors: ThemeColors) =>
  StyleSheet.create({
    container: { flex: 1 },
    scroll: { padding: 20, paddingBottom: 60 },
    center: { paddingVertical: 60, alignItems: 'center', justifyContent: 'center' },

    backBtn: { marginBottom: 16 },
    backText: { fontSize: 14, color: colors.textMuted, fontFamily: 'Nunito_400Regular' },

    title: {
      fontSize: 28,
      color: colors.textPrimary,
      fontFamily: 'CormorantGaramond_300Light',
      marginBottom: 4,
    },
    subtitle: {
      fontSize: 14,
      color: colors.textSecondary,
      fontFamily: 'Nunito_400Regular',
      lineHeight: 22,
      marginBottom: 24,
    },
    countLine: {
      fontSize: 10,
      color: colors.textMuted,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 1.2,
      textTransform: 'uppercase',
      marginBottom: 14,
    },

    emptyBox: { alignItems: 'center', paddingVertical: 40, paddingHorizontal: 12 },
    emptyIcon: { flexDirection: 'row', marginBottom: 18 },
    emptyRing: {
      width: 30,
      height: 30,
      borderRadius: 15,
      borderWidth: 1.5,
      borderColor: colors.textFaint,
    },
    emptyRingOverlap: { marginLeft: -10 },
    emptyText: {
      fontSize: 14,
      color: colors.textSecondary,
      fontFamily: 'Nunito_400Regular',
      lineHeight: 22,
      textAlign: 'center',
    },

    card: {
      backgroundColor: colors.card,
      borderRadius: 18,
      borderWidth: 1,
      borderColor: colors.borderFaint,
      padding: 16,
      marginBottom: 12,
    },
    cardHead: { flexDirection: 'row', alignItems: 'center', gap: 12 },
    avatar: {
      width: 38,
      height: 38,
      borderRadius: 19,
      backgroundColor: colors.brandGlow,
      borderWidth: 1,
      borderColor: colors.borderFaint,
      alignItems: 'center',
      justifyContent: 'center',
    },
    avatarInitial: {
      fontSize: 18,
      color: colors.brand,
      fontFamily: 'CormorantGaramond_500Medium',
      lineHeight: 22,
    },
    name: {
      fontSize: 19,
      color: colors.textPrimary,
      fontFamily: 'CormorantGaramond_400Regular',
      marginBottom: 2,
    },
    credentials: {
      fontSize: 12,
      color: colors.purple300,
      fontFamily: 'Nunito_600SemiBold',
      marginBottom: 2,
    },
    meta: { fontSize: 11, color: colors.textMuted, fontFamily: 'Nunito_400Regular' },

    actionsRow: {
      flexDirection: 'row',
      gap: 12,
      marginTop: 16,
      paddingTop: 14,
      borderTopWidth: 1,
      borderTopColor: colors.borderFaint,
    },
    declineBtn: {
      flex: 1,
      backgroundColor: colors.card,
      borderWidth: 1,
      borderColor: colors.border,
      borderRadius: 12,
      paddingVertical: 12,
      alignItems: 'center',
    },
    declineText: {
      fontSize: 14,
      color: colors.textSecondary,
      fontFamily: 'Nunito_600SemiBold',
    },
    approveBtn: {
      flex: 1,
      backgroundColor: colors.brand,
      borderRadius: 12,
      paddingVertical: 12,
      alignItems: 'center',
      justifyContent: 'center',
    },
    approveText: {
      fontSize: 14,
      color: colors.bg,
      fontFamily: 'Nunito_600SemiBold',
    },
    btnDisabled: { opacity: 0.5 },

    footer: {
      fontSize: 11,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      lineHeight: 18,
      textAlign: 'center',
      marginTop: 16,
    },
  });
