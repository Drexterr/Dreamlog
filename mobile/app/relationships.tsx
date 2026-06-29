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
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';
import type { Person, PersonDetail, PersonRole, PersonSentiment } from '../src/types';

// Monochrome glyph icons in the app's accent style (same language as the
// Explore feature-card icons: ◎ ◐ ⊹). The role label chip carries the meaning.
const ROLE_META: Record<PersonRole, { glyph: string; label: string }> = {
  family:    { glyph: '⌂', label: 'Family' },
  friend:    { glyph: '◇', label: 'Friend' },
  colleague: { glyph: '◻', label: 'Colleague' },
  romantic:  { glyph: '♡', label: 'Romantic' },
  other:     { glyph: '◦', label: 'Other' },
};

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
              <Text style={styles.emptyGlyph}>❖</Text>
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
                const role = ROLE_META[p.role] ?? ROLE_META.other;
                const neutral = Math.max(0, p.mention_count - p.positive_count - p.negative_count);
                const total = Math.max(1, p.positive_count + neutral + p.negative_count);
                const isOpen = expandedId === p.id;
                const detail = details[p.id];

                return (
                  <TouchableOpacity
                    key={p.id}
                    style={styles.card}
                    activeOpacity={0.85}
                    onPress={() => toggle(p)}
                  >
                    <View style={styles.cardHead}>
                      <View style={styles.avatar}>
                        <Text style={styles.avatarGlyph}>{role.glyph}</Text>
                      </View>
                      <View style={{ flex: 1 }}>
                        <Text style={styles.name}>{p.name}</Text>
                        <Text style={styles.meta}>
                          {p.mention_count} {p.mention_count === 1 ? 'mention' : 'mentions'} · last {relativeDate(p.last_mentioned_at)}
                        </Text>
                      </View>
                      <View style={styles.roleChip}>
                        <Text style={styles.roleChipText}>{role.label}</Text>
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

                    {/* Expanded: recent mentions */}
                    {isOpen && (
                      <View style={styles.detailWrap}>
                        {detailLoadingId === p.id && !detail ? (
                          <ActivityIndicator color={colors.brand} style={{ marginVertical: 12 }} />
                        ) : detail && detail.mentions.length > 0 ? (
                          detail.mentions.map((m) => (
                            <View key={m.id} style={styles.mentionRow}>
                              <View style={[styles.mentionDot, { backgroundColor: sentimentColor(m.sentiment) }]} />
                              <View style={{ flex: 1 }}>
                                <Text style={styles.mentionContext}>&ldquo;{m.context}&rdquo;</Text>
                                <Text style={styles.mentionDate}>{relativeDate(m.created_at)}</Text>
                              </View>
                            </View>
                          ))
                        ) : (
                          <Text style={styles.mentionEmpty}>No recent mentions to show.</Text>
                        )}
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
    emptyGlyph: { fontSize: 38, color: colors.textFaint, marginBottom: 16 },
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
    avatarGlyph: {
      fontSize: 18,
      color: colors.brand,
      fontFamily: 'Nunito_400Regular',
      lineHeight: 22,
    },
    name: {
      fontSize: 19,
      color: colors.textPrimary,
      fontFamily: 'CormorantGaramond_400Regular',
      marginBottom: 2,
    },
    meta: { fontSize: 11, color: colors.textMuted, fontFamily: 'Nunito_400Regular' },
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
    mentionRow: { flexDirection: 'row', gap: 10, alignItems: 'flex-start' },
    mentionDot: { width: 7, height: 7, borderRadius: 3.5, marginTop: 6 },
    mentionContext: {
      fontSize: 14,
      color: colors.textSecondary,
      fontFamily: 'CormorantGaramond_400Regular',
      fontStyle: 'italic',
      lineHeight: 21,
    },
    mentionDate: {
      fontSize: 10,
      color: colors.textFaint,
      fontFamily: 'Nunito_400Regular',
      marginTop: 3,
    },
    mentionEmpty: {
      fontSize: 12,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      paddingVertical: 6,
    },

    footer: {
      fontSize: 11,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      lineHeight: 18,
      textAlign: 'center',
      marginTop: 16,
    },
  });
