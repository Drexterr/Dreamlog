/**
 * Timeline screen — list of journal entries with analysis.
 * Matches the dark card aesthetic from DreamLog_UI.jsx TimelineScreen.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  FlatList,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  SafeAreaView,
  StatusBar,
  RefreshControl,
  ActivityIndicator,
} from 'react-native';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { TimelineEntry } from '../../src/types';

function MoodDot({ score, size = 8 }: { score: number; size?: number }) {
  const { moodToColor } = useTheme();
  return (
    <View
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        backgroundColor: moodToColor(score),
        flexShrink: 0,
      }}
    />
  );
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  if (diff < 86_400_000 && now.getDate() === d.getDate()) return 'Today';
  if (diff < 172_800_000) return 'Yesterday';
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return m > 0 ? `${m}:${String(s).padStart(2, '0')}` : `0:${String(s).padStart(2, '0')}`;
}

function EntryCard({ item, index, onPress }: { item: TimelineEntry; index: number; onPress: () => void }) {
  const { entry, analysis } = item;
  const moodScore = analysis?.mood_score ?? 0;
  const isRecent = index === 0;
  const { colors } = useTheme();

  return (
    <TouchableOpacity
      onPress={onPress}
      activeOpacity={0.8}
      style={[
        styles.card,
        isRecent && {
          backgroundColor: colors.card,
          borderColor: colors.border,
        },
      ]}
    >
      <View style={styles.cardHeader}>
        <View style={styles.cardMeta}>
          {analysis && <MoodDot score={moodScore} />}
          <Text style={[styles.cardDate, { color: colors.textSecondary }]}>{formatDate(entry.created_at)}</Text>
          <Text style={[styles.cardTime, { color: colors.textMuted }]}>{formatTime(entry.created_at)}</Text>
        </View>
        <Text style={[styles.cardDuration, { color: colors.textMuted }]}>{formatDuration(entry.duration_sec)}</Text>
      </View>

      {/* Summary or status */}
      {analysis?.summary ? (
        <Text style={[styles.cardSummary, { color: colors.textSecondary }]} numberOfLines={2}>
          {analysis.summary}
        </Text>
      ) : (
        <Text style={[styles.cardPending, { color: colors.textFaint }]}>
          {entry.status === 'failed'
            ? entry.error_msg ?? 'Transcription failed'
            : entry.status === 'completed'
            ? 'No analysis yet'
            : 'Processing…'}
        </Text>
      )}

      {/* Topic badges */}
      {analysis?.topics?.length > 0 && (
        <View style={styles.topicRow}>
          {analysis.topics.slice(0, 3).map((t) => (
            <View key={t} style={[styles.topicBadge, { backgroundColor: colors.brandGlow }]}>
              <Text style={[styles.topicText, { color: colors.textMuted }]}>{t}</Text>
            </View>
          ))}
        </View>
      )}
    </TouchableOpacity>
  );
}

export default function TimelineScreen() {
  const router = useRouter();
  const { colors } = useTheme();
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [searching, setSearching] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const loadTimeline = useCallback(async () => {
    try {
      const resp = await api.getTimeline(1, 30);
      setEntries(resp.entries);
    } catch (err) {
      console.error('timeline load', err);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadTimeline();
  }, []);

  const onRefresh = useCallback(() => {
    setRefreshing(true);
    loadTimeline();
  }, [loadTimeline]);

  // Debounced full-text search
  const handleSearch = useCallback((q: string) => {
    setSearchQuery(q);
    if (searchTimer.current) clearTimeout(searchTimer.current);
    if (!q.trim()) {
      loadTimeline();
      return;
    }
    searchTimer.current = setTimeout(async () => {
      setSearching(true);
      try {
        const res = await api.searchEntries(q.trim(), 20);
        // Wrap plain entries in TimelineEntry shape
        setEntries(res.entries.map((e) => ({ entry: e })));
      } catch {
        // keep current results
      } finally {
        setSearching(false);
      }
    }, 400);
  }, [loadTimeline]);

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor: colors.bg }]}>
        <SafeAreaView style={styles.center}>
          <ActivityIndicator color={colors.purple400} />
        </SafeAreaView>
      </View>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={styles.safe}>
        {/* Header */}
        <View style={styles.header}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Your entries</Text>
          <Text style={[styles.subtitle, { color: colors.textMuted }]}>
            {entries.length} {entries.length === 1 ? 'entry' : 'entries'}
          </Text>
        </View>

        {/* Search */}
        <View style={[styles.searchBar, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
          <View style={styles.searchIcon}>
            <View style={[styles.searchCircle, { borderColor: colors.textMuted }]} />
            <View style={[styles.searchHandle, { backgroundColor: colors.textMuted }]} />
          </View>
          <TextInput
            style={[styles.searchInput, { color: colors.textPrimary }]}
            value={searchQuery}
            onChangeText={handleSearch}
            placeholder="Search your entries..."
            placeholderTextColor={colors.textFaint}
            returnKeyType="search"
            autoCapitalize="none"
            autoCorrect={false}
          />
          {searching && <ActivityIndicator size="small" color={colors.purple400} />}
        </View>

        {/* List */}
        <FlatList
          data={entries}
          keyExtractor={(item) => item.entry.id}
          contentContainerStyle={styles.list}
          removeClippedSubviews
          maxToRenderPerBatch={10}
          updateCellsBatchingPeriod={50}
          windowSize={10}
          refreshControl={
            <RefreshControl
              refreshing={refreshing}
              onRefresh={onRefresh}
              tintColor={colors.purple400}
            />
          }
          ListEmptyComponent={
            <View style={styles.emptyWrap}>
              <Text style={[styles.emptyText, { color: colors.textSecondary }]}>
                {searchQuery ? 'No matching entries.' : 'No entries yet.\nRecord your first journal.'}
              </Text>
            </View>
          }
          renderItem={({ item, index }) => (
            <EntryCard
              item={item}
              index={index}
              onPress={() => {
                if (item.analysis) {
                  router.push(`/reflection/${item.entry.id}`);
                }
              }}
            />
          )}
        />
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  safe: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },

  header: { paddingHorizontal: 20, paddingTop: 20, paddingBottom: 8 },
  title: {
    fontSize: 26,
    fontFamily: 'CormorantGaramond_300Light',
    marginBottom: 2,
  },
  subtitle: {
    fontSize: 11,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.5,
  },

  searchBar: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 14,
    borderWidth: 1,
    marginHorizontal: 20,
    marginBottom: 12,
    paddingHorizontal: 14,
    paddingVertical: 10,
    gap: 10,
  },
  searchIcon: { width: 16, height: 16, position: 'relative' },
  searchCircle: {
    width: 10, height: 10, borderRadius: 5,
    borderWidth: 1.5,
  },
  searchHandle: {
    position: 'absolute', bottom: 0, right: 0,
    width: 5, height: 1.5,
    transform: [{ rotate: '45deg' }],
  },
  searchInput: {
    flex: 1,
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    padding: 0,
  },

  list: { paddingHorizontal: 16, paddingBottom: 40 },

  card: {
    padding: 18,
    borderRadius: 16,
    marginBottom: 4,
    backgroundColor: 'transparent',
    borderWidth: 1,
    borderColor: 'transparent',
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  cardMeta: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  cardDate: {
    fontSize: 13,
    fontFamily: 'Nunito_600SemiBold',
  },
  cardTime: {
    fontSize: 11,
    fontFamily: 'Nunito_400Regular',
  },
  cardDuration: {
    fontSize: 11,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.5,
  },
  cardSummary: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 22,
    marginBottom: 10,
  },
  cardPending: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    fontStyle: 'italic',
    marginBottom: 8,
  },
  topicRow: { flexDirection: 'row', gap: 6, flexWrap: 'wrap' },
  topicBadge: {
    borderRadius: 8,
    paddingHorizontal: 8,
    paddingVertical: 2,
  },
  topicText: {
    fontSize: 10,
    fontFamily: 'Nunito_400Regular',
  },

  emptyWrap: { paddingTop: 60, alignItems: 'center' },
  emptyText: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    textAlign: 'center',
    lineHeight: 22,
  },
});
