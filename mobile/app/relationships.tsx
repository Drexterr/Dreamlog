/**
 * Relationship Map screen.
 *
 * Surfaces the people Claude extracts from journal entries (Phase 7e). Lists
 * everyone mentioned, sorted by how often they come up, with a sentiment
 * breakdown per person. Tapping a person expands their recent mentions inline
 * (GET /relationships/:id).
 *
 * Reached from the Mood Map screen.
 */

import { useEffect, useState, useCallback } from 'react';
import {
  View,
  Text,
  ScrollView,
  StyleSheet,
  StatusBar,
  ActivityIndicator,
  TouchableOpacity,
  Modal,
  TextInput,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';
import type { Person, PersonDetail, PersonMention, PersonRole, PersonSentiment } from '../src/types';

// The avatar shows the person's initial (identifies the person); the role is
// carried by the text chip beside the name.
const ROLE_LABEL: Record<PersonRole, string> = {
  family:    'Family',
  friend:    'Friend',
  colleague: 'Colleague',
  romantic:  'Romantic',
  other:     'Other',
};

// A person is "drifting" if they mattered (≥3 mentions) but haven't come up in
// the last 3 weeks — a gentle prompt, surfaced as a badge.
const DRIFT_DAYS = 21;
const DRIFT_MIN_MENTIONS = 3;

function daysAgo(iso: string): number {
  const then = new Date(iso).getTime();
  if (isNaN(then)) return 0;
  return Math.floor((Date.now() - then) / 86_400_000);
}

// ── Sentiment trend (computed client-side from mention history) ──────────────
type Trend = 'warming' | 'cooling' | 'steady' | 'unknown';

function sentimentScore(s: PersonSentiment): number {
  return s === 'positive' ? 1 : s === 'negative' ? -1 : 0;
}

// Compares the average sentiment of the newer half of mentions against the
// older half. Needs ≥4 mentions to mean anything.
function computeTrend(mentions: PersonMention[]): Trend {
  if (mentions.length < 4) return 'unknown';
  const sorted = [...mentions].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  );
  const mid = Math.floor(sorted.length / 2);
  const avg = (arr: PersonMention[]) =>
    arr.reduce((s, m) => s + sentimentScore(m.sentiment), 0) / arr.length;
  const delta = avg(sorted.slice(mid)) - avg(sorted.slice(0, mid));
  if (delta > 0.25) return 'warming';
  if (delta < -0.25) return 'cooling';
  return 'steady';
}

function trendMeta(
  trend: Trend,
  colors: any,
): { arrow: string; label: string; color: string } | null {
  switch (trend) {
    case 'warming': return { arrow: '↗', label: 'Warming over time', color: colors.moodGreen };
    case 'cooling': return { arrow: '↘', label: 'Cooling over time', color: colors.moodRed };
    case 'steady':  return { arrow: '→', label: 'Steady', color: colors.textMuted };
    default:        return null;
  }
}

function relativeDate(iso: string): string {
  const then = new Date(iso);
  if (isNaN(then.getTime())) return '';
  const now = new Date();
  const days = Math.floor((now.getTime() - then.getTime()) / 86_400_000);
  if (days <= 0) return 'today';
  if (days === 1) return 'yesterday';
  if (days < 7) return `${days} days ago`;
  if (days < 14) return 'last week';
  if (days < 30) return `${Math.floor(days / 7)} weeks ago`;
  if (days < 60) return 'last month';
  if (days < 365) return `${Math.floor(days / 30)} months ago`;
  return then.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
}

export default function RelationshipsScreen() {
  const router = useRouter();
  const { colors } = useTheme();
  const styles = getStyles(colors);

  const [people, setPeople] = useState<Person[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [details, setDetails] = useState<Record<string, PersonDetail>>({});
  const [detailLoadingId, setDetailLoadingId] = useState<string | null>(null);

  // Manage (rename / merge / hide)
  const [managePerson, setManagePerson] = useState<Person | null>(null);
  const [mode, setMode] = useState<'rename' | 'merge' | null>(null);
  const [renameText, setRenameText] = useState('');
  const [busy, setBusy] = useState(false);

  const closeManage = useCallback(() => {
    setManagePerson(null);
    setMode(null);
    setRenameText('');
  }, []);

  const handleHide = useCallback(
    (p: Person) => {
      Alert.alert(
        `Hide ${p.name}?`,
        "They'll be removed from your map. This doesn't delete any journal entries.",
        [
          { text: 'Cancel', style: 'cancel' },
          {
            text: 'Hide',
            style: 'destructive',
            onPress: async () => {
              try {
                await api.updatePerson(p.id, { hidden: true });
                setPeople((prev) => prev.filter((x) => x.id !== p.id));
                setExpandedId((cur) => (cur === p.id ? null : cur));
              } catch {
                Alert.alert('Error', 'Could not hide. Please try again.');
              }
            },
          },
        ],
      );
    },
    [],
  );

  const handleRename = useCallback(async () => {
    if (!managePerson) return;
    const name = renameText.trim();
    if (!name) return;
    setBusy(true);
    try {
      const updated = await api.updatePerson(managePerson.id, { name });
      setPeople((prev) => prev.map((x) => (x.id === updated.id ? updated : x)));
      closeManage();
    } catch (e) {
      const status = (e as { response?: { status?: number } })?.response?.status;
      if (status === 409) {
        Alert.alert(
          'Name already in use',
          'Someone with that name is already on your map. Use Merge to combine them instead.',
        );
      } else {
        Alert.alert('Error', 'Could not rename. Please try again.');
      }
    } finally {
      setBusy(false);
    }
  }, [managePerson, renameText, closeManage]);

  const handleMerge = useCallback(
    async (source: Person) => {
      if (!managePerson) return;
      setBusy(true);
      try {
        const updated = await api.mergePerson(managePerson.id, source.id);
        setPeople((prev) =>
          prev.filter((x) => x.id !== source.id).map((x) => (x.id === updated.id ? updated : x)),
        );
        // Drop cached detail so the merged person refetches its mentions.
        setDetails((prev) => {
          const next = { ...prev };
          delete next[updated.id];
          return next;
        });
        closeManage();
      } catch {
        Alert.alert('Error', 'Could not merge. Please try again.');
      } finally {
        setBusy(false);
      }
    },
    [managePerson, closeManage],
  );

  useEffect(() => {
    let active = true;
    api
      .getRelationships()
      .then((res) => {
        if (!active) return;
        const sorted = [...res.people].sort((a, b) => b.mention_count - a.mention_count);
        setPeople(sorted);
      })
      .catch(() => active && setError(true))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, []);

  const toggle = useCallback(
    (person: Person) => {
      if (expandedId === person.id) {
        setExpandedId(null);
        return;
      }
      setExpandedId(person.id);
      if (!details[person.id]) {
        setDetailLoadingId(person.id);
        api
          .getPersonDetail(person.id)
          .then((d) => setDetails((prev) => ({ ...prev, [person.id]: d })))
          .catch(() => {})
          .finally(() => setDetailLoadingId((cur) => (cur === person.id ? null : cur)));
      }
    },
    [expandedId, details],
  );

  const sentimentColor = (s: PersonSentiment) =>
    s === 'positive' ? colors.moodGreen : s === 'negative' ? colors.moodRed : colors.textMuted;

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
          <TouchableOpacity onPress={() => router.back()} style={styles.backBtn}>
            <Text style={styles.backText}>← Back</Text>
          </TouchableOpacity>

          <Text style={styles.title}>Relationship Map</Text>
          <Text style={styles.subtitle}>The people woven through your reflections.</Text>

          {loading ? (
            <View style={styles.center}>
              <ActivityIndicator color={colors.brand} />
            </View>
          ) : error ? (
            <Text style={styles.emptyText}>Couldn&apos;t load your relationship map. Please try again.</Text>
          ) : people.length === 0 ? (
            <View style={styles.emptyBox}>
              <View style={styles.emptyIcon}>
                <View style={styles.emptyRing} />
                <View style={[styles.emptyRing, styles.emptyRingOverlap]} />
              </View>
              <Text style={styles.emptyText}>
                As you journal, the people you mention will appear here — along with how often they come up and the
                feeling around each one.
              </Text>
            </View>
          ) : (
            <>
              <Text style={styles.countLine}>
                {people.length} {people.length === 1 ? 'person' : 'people'} across your entries
              </Text>

              {people.map((p) => {
                const roleLabel = ROLE_LABEL[p.role] ?? 'Other';
                const initial = (p.name.trim().charAt(0) || '?').toUpperCase();
                const neutral = Math.max(0, p.mention_count - p.positive_count - p.negative_count);
                const total = Math.max(1, p.positive_count + neutral + p.negative_count);
                const isOpen = expandedId === p.id;
                const detail = details[p.id];
                const drifting =
                  p.mention_count >= DRIFT_MIN_MENTIONS && daysAgo(p.last_mentioned_at) >= DRIFT_DAYS;

                return (
                  <TouchableOpacity
                    key={p.id}
                    style={styles.card}
                    activeOpacity={0.85}
                    onPress={() => toggle(p)}
                  >
                    <View style={styles.cardHead}>
                      <View style={styles.avatar}>
                        <Text style={styles.avatarInitial}>{initial}</Text>
                      </View>
                      <View style={{ flex: 1 }}>
                        <Text style={styles.name}>{p.name}</Text>
                        <Text style={styles.meta}>
                          {p.mention_count} {p.mention_count === 1 ? 'mention' : 'mentions'} · last {relativeDate(p.last_mentioned_at)}
                        </Text>
                      </View>
                      <View style={styles.headRight}>
                        <View style={styles.roleChip}>
                          <Text style={styles.roleChipText}>{roleLabel}</Text>
                        </View>
                        {drifting && (
                          <View style={styles.driftBadge}>
                            <Text style={styles.driftText}>Drifting</Text>
                          </View>
                        )}
                      </View>
                    </View>

                    {/* Sentiment bar */}
                    <View style={styles.sentBar}>
                      {p.positive_count > 0 && (
                        <View style={{ flex: p.positive_count / total, backgroundColor: colors.moodGreen }} />
                      )}
                      {neutral > 0 && (
                        <View style={{ flex: neutral / total, backgroundColor: colors.textFaint }} />
                      )}
                      {p.negative_count > 0 && (
                        <View style={{ flex: p.negative_count / total, backgroundColor: colors.moodRed }} />
                      )}
                    </View>
                    <View style={styles.sentLegend}>
                      <Text style={[styles.sentCount, { color: colors.moodGreen }]}>{p.positive_count} warm</Text>
                      <Text style={[styles.sentCount, { color: colors.textMuted }]}>{neutral} neutral</Text>
                      <Text style={[styles.sentCount, { color: colors.moodRed }]}>{p.negative_count} hard</Text>
                    </View>

                    {/* Expanded: trend + recent mentions */}
                    {isOpen && (
                      <View style={styles.detailWrap}>
                        {detailLoadingId === p.id && !detail ? (
                          <ActivityIndicator color={colors.brand} style={{ marginVertical: 12 }} />
                        ) : detail && detail.mentions.length > 0 ? (
                          <>
                            {(() => {
                              const tm = trendMeta(computeTrend(detail.mentions), colors);
                              return tm ? (
                                <View style={[styles.trendChip, { borderColor: tm.color + '55' }]}>
                                  <Text style={[styles.trendText, { color: tm.color }]}>
                                    {tm.arrow}  {tm.label}
                                  </Text>
                                </View>
                              ) : null;
                            })()}
                            {detail.mentions.map((m) => (
                              <TouchableOpacity
                                key={m.id}
                                style={styles.mentionRow}
                                activeOpacity={0.7}
                                onPress={() => router.push(`/reflection/${m.entry_id}` as never)}
                              >
                                <View style={[styles.mentionDot, { backgroundColor: sentimentColor(m.sentiment) }]} />
                                <View style={{ flex: 1 }}>
                                  <Text style={styles.mentionContext}>&ldquo;{m.context}&rdquo;</Text>
                                  <View style={styles.mentionFootRow}>
                                    <Text style={styles.mentionDate}>{relativeDate(m.created_at)}</Text>
                                    <Text style={styles.mentionOpen}>View entry →</Text>
                                  </View>
                                </View>
                              </TouchableOpacity>
                            ))}
                          </>
                        ) : (
                          <Text style={styles.mentionEmpty}>No recent mentions to show.</Text>
                        )}

                        {/* Manage actions */}
                        <View style={styles.actionsRow}>
                          <TouchableOpacity
                            onPress={() => {
                              setManagePerson(p);
                              setRenameText(p.name);
                              setMode('rename');
                            }}
                          >
                            <Text style={styles.actionBtn}>Rename</Text>
                          </TouchableOpacity>
                          <TouchableOpacity
                            onPress={() => {
                              setManagePerson(p);
                              setMode('merge');
                            }}
                          >
                            <Text style={styles.actionBtn}>Merge</Text>
                          </TouchableOpacity>
                          <TouchableOpacity onPress={() => handleHide(p)}>
                            <Text style={[styles.actionBtn, { color: colors.moodRed }]}>Hide</Text>
                          </TouchableOpacity>
                        </View>
                      </View>
                    )}
                  </TouchableOpacity>
                );
              })}

              <Text style={styles.footer}>
                People are detected automatically from what you say. Names never leave your private journal.
              </Text>
            </>
          )}
        </ScrollView>
      </SafeAreaView>

      {/* Rename modal */}
      <Modal visible={mode === 'rename'} transparent animationType="fade" onRequestClose={closeManage}>
        <View style={styles.modalOverlay}>
          <View style={styles.modalSheet}>
            <Text style={styles.modalTitle}>Rename</Text>
            <Text style={styles.modalSub}>Tidy up how this person appears on your map.</Text>
            <TextInput
              style={styles.input}
              value={renameText}
              onChangeText={setRenameText}
              placeholder="Name"
              placeholderTextColor={colors.textMuted}
              autoFocus
            />
            <View style={styles.modalActions}>
              <TouchableOpacity style={styles.modalCancel} onPress={closeManage}>
                <Text style={styles.modalCancelText}>Cancel</Text>
              </TouchableOpacity>
              <TouchableOpacity
                style={[styles.modalSave, (busy || !renameText.trim()) && { opacity: 0.5 }]}
                onPress={handleRename}
                disabled={busy || !renameText.trim()}
              >
                {busy ? (
                  <ActivityIndicator color={colors.bg} />
                ) : (
                  <Text style={styles.modalSaveText}>Save</Text>
                )}
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </Modal>

      {/* Merge modal */}
      <Modal visible={mode === 'merge'} transparent animationType="fade" onRequestClose={closeManage}>
        <View style={styles.modalOverlay}>
          <View style={styles.modalSheet}>
            <Text style={styles.modalTitle}>Merge into {managePerson?.name}</Text>
            <Text style={styles.modalSub}>
              Pick a duplicate to fold into {managePerson?.name}. Their mentions move over and the duplicate is removed.
            </Text>
            <ScrollView style={{ maxHeight: 300 }}>
              {people
                .filter((x) => x.id !== managePerson?.id)
                .map((x) => (
                  <TouchableOpacity
                    key={x.id}
                    style={styles.mergeRow}
                    disabled={busy}
                    onPress={() =>
                      Alert.alert(
                        `Merge ${x.name} into ${managePerson?.name}?`,
                        "This can't be undone.",
                        [
                          { text: 'Cancel', style: 'cancel' },
                          { text: 'Merge', onPress: () => handleMerge(x) },
                        ],
                      )
                    }
                  >
                    <Text style={styles.mergeName}>{x.name}</Text>
                    <Text style={styles.mergeMeta}>{x.mention_count} mentions</Text>
                  </TouchableOpacity>
                ))}
              {people.filter((x) => x.id !== managePerson?.id).length === 0 && (
                <Text style={styles.mentionEmpty}>No other people to merge.</Text>
              )}
            </ScrollView>
            <TouchableOpacity style={[styles.modalCancel, { marginTop: 12 }]} onPress={closeManage}>
              <Text style={styles.modalCancelText}>Cancel</Text>
            </TouchableOpacity>
          </View>
        </View>
      </Modal>
    </View>
  );
}

const getStyles = (colors: any) =>
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
    meta: { fontSize: 11, color: colors.textMuted, fontFamily: 'Nunito_400Regular' },
    headRight: { alignItems: 'flex-end', gap: 5 },
    driftBadge: {
      borderWidth: 1,
      borderColor: colors.borderFaint,
      borderRadius: 8,
      paddingHorizontal: 8,
      paddingVertical: 2,
    },
    driftText: {
      fontSize: 9,
      color: colors.textMuted,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 0.4,
    },
    roleChip: {
      backgroundColor: colors.brandGlow,
      borderRadius: 8,
      paddingHorizontal: 9,
      paddingVertical: 3,
    },
    roleChipText: {
      fontSize: 10,
      color: colors.purple300,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 0.3,
    },

    sentBar: {
      flexDirection: 'row',
      height: 5,
      borderRadius: 3,
      overflow: 'hidden',
      backgroundColor: colors.cardSolid,
      marginTop: 14,
      marginBottom: 7,
    },
    sentLegend: { flexDirection: 'row', gap: 12 },
    sentCount: { fontSize: 10, fontFamily: 'Nunito_400Regular' },

    detailWrap: {
      marginTop: 14,
      paddingTop: 14,
      borderTopWidth: 1,
      borderTopColor: colors.borderFaint,
      gap: 12,
    },
    trendChip: {
      alignSelf: 'flex-start',
      borderWidth: 1,
      borderRadius: 100,
      paddingHorizontal: 12,
      paddingVertical: 5,
    },
    trendText: { fontSize: 11, fontFamily: 'Nunito_600SemiBold', letterSpacing: 0.3 },
    mentionRow: { flexDirection: 'row', gap: 10, alignItems: 'flex-start' },
    mentionDot: { width: 7, height: 7, borderRadius: 3.5, marginTop: 6 },
    mentionContext: {
      fontSize: 14,
      color: colors.textSecondary,
      fontFamily: 'CormorantGaramond_400Regular',
      fontStyle: 'italic',
      lineHeight: 21,
    },
    mentionFootRow: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'space-between',
      marginTop: 4,
    },
    mentionDate: {
      fontSize: 10,
      color: colors.textFaint,
      fontFamily: 'Nunito_400Regular',
    },
    mentionOpen: {
      fontSize: 10,
      color: colors.brand,
      fontFamily: 'Nunito_600SemiBold',
    },
    mentionEmpty: {
      fontSize: 12,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      paddingVertical: 6,
    },

    actionsRow: {
      flexDirection: 'row',
      gap: 20,
      marginTop: 4,
      paddingTop: 12,
      borderTopWidth: 1,
      borderTopColor: colors.borderFaint,
    },
    actionBtn: {
      fontSize: 12,
      color: colors.purple300,
      fontFamily: 'Nunito_600SemiBold',
      letterSpacing: 0.3,
    },

    // Manage modals
    modalOverlay: {
      flex: 1,
      backgroundColor: 'rgba(10,5,20,0.8)',
      justifyContent: 'center',
      padding: 24,
    },
    modalSheet: {
      backgroundColor: colors.cardSolid,
      borderRadius: 20,
      borderWidth: 1,
      borderColor: colors.border,
      padding: 22,
    },
    modalTitle: {
      fontSize: 20,
      color: colors.textPrimary,
      fontFamily: 'CormorantGaramond_400Regular',
      marginBottom: 6,
    },
    modalSub: {
      fontSize: 13,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      lineHeight: 19,
      marginBottom: 16,
    },
    input: {
      backgroundColor: colors.card,
      borderRadius: 12,
      borderWidth: 1,
      borderColor: colors.borderFaint,
      paddingHorizontal: 14,
      paddingVertical: 12,
      color: colors.textPrimary,
      fontFamily: 'Nunito_400Regular',
      fontSize: 15,
      marginBottom: 16,
    },
    modalActions: { flexDirection: 'row', gap: 12 },
    modalCancel: {
      flex: 1,
      backgroundColor: colors.card,
      borderRadius: 12,
      paddingVertical: 12,
      alignItems: 'center',
    },
    modalCancelText: {
      color: colors.textSecondary,
      fontFamily: 'Nunito_400Regular',
      fontSize: 14,
    },
    modalSave: {
      flex: 1,
      backgroundColor: colors.brand,
      borderRadius: 12,
      paddingVertical: 12,
      alignItems: 'center',
    },
    modalSaveText: {
      color: colors.bg,
      fontFamily: 'Nunito_600SemiBold',
      fontSize: 14,
    },
    mergeRow: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      alignItems: 'center',
      paddingVertical: 12,
      borderBottomWidth: 1,
      borderBottomColor: colors.borderFaint,
    },
    mergeName: {
      fontSize: 16,
      color: colors.textPrimary,
      fontFamily: 'CormorantGaramond_400Regular',
    },
    mergeMeta: { fontSize: 11, color: colors.textMuted, fontFamily: 'Nunito_400Regular' },

    footer: {
      fontSize: 11,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      lineHeight: 18,
      textAlign: 'center',
      marginTop: 16,
    },
  });
