/**
 * MoodChartScreen — 7-day mood trend + streak display.
 *
 * Uses a manual SVG-free bar chart (React Native View-based) to avoid
 * adding a charting dependency. Replace with Victory Native or Recharts
 * if richer charts are needed later.
 */

import React, { useEffect, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  SafeAreaView,
  ScrollView,
  ActivityIndicator,
} from 'react-native';
import { api } from '../api/client';
import { DailyMood, StreakInfo } from '../types';

export function MoodChartScreen() {
  const [moods, setMoods] = useState<DailyMood[]>([]);
  const [streak, setStreak] = useState<StreakInfo | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([api.weeklyMood(), api.streak()])
      .then(([m, s]) => {
        setMoods(m.days ?? []);
        setStreak(s);
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <SafeAreaView style={styles.container}>
        <ActivityIndicator color="#3b82f6" style={{ marginTop: 80 }} />
      </SafeAreaView>
    );
  }

  const maxMood = Math.max(...moods.map((m) => m.avg_mood), 1);

  return (
    <SafeAreaView style={styles.container}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <Text style={styles.title}>Your week</Text>

        {/* Streak cards */}
        {streak && (
          <View style={styles.streakRow}>
            <StreakCard
              value={streak.current_streak}
              label="day streak"
              highlight={streak.current_streak >= 3}
            />
            <StreakCard value={streak.longest_streak} label="best streak" />
            <StreakCard value={streak.total_days} label="days journaled" />
          </View>
        )}

        {/* Bar chart */}
        <View style={styles.chartCard}>
          <Text style={styles.chartLabel}>Mood over 7 days</Text>
          {moods.length === 0 ? (
            <Text style={styles.emptyChart}>No data yet — keep journaling.</Text>
          ) : (
            <View style={styles.bars}>
              {moods.map((day) => {
                const heightPct = (day.avg_mood / 100) * 100;
                const color = moodToColor(day.avg_mood);
                const label = formatDay(day.day);
                return (
                  <View key={day.day} style={styles.barColumn}>
                    <View style={styles.barTrack}>
                      <View
                        style={[
                          styles.bar,
                          { height: `${heightPct}%`, backgroundColor: color },
                        ]}
                      />
                    </View>
                    <Text style={styles.barLabel}>{label}</Text>
                    <Text style={styles.barCount}>{day.entry_count}</Text>
                  </View>
                );
              })}
            </View>
          )}
        </View>

        {/* Legend */}
        <View style={styles.legend}>
          {MOOD_LEGEND.map(({ color, label }) => (
            <View key={label} style={styles.legendItem}>
              <View style={[styles.legendDot, { backgroundColor: color }]} />
              <Text style={styles.legendText}>{label}</Text>
            </View>
          ))}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

function StreakCard({
  value,
  label,
  highlight = false,
}: {
  value: number;
  label: string;
  highlight?: boolean;
}) {
  return (
    <View style={[styles.streakCard, highlight && styles.streakCardHighlight]}>
      <Text style={[styles.streakValue, highlight && styles.streakValueHighlight]}>
        {value}
      </Text>
      <Text style={styles.streakLabel}>{label}</Text>
    </View>
  );
}

function moodToColor(score: number): string {
  if (score <= 20) return '#7f1d1d';
  if (score <= 40) return '#92400e';
  if (score <= 60) return '#1e3a8a';
  if (score <= 80) return '#14532d';
  return '#064e3b';
}

function formatDay(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00Z');
  return d.toLocaleDateString('en-US', { weekday: 'short', timeZone: 'UTC' });
}

const MOOD_LEGEND = [
  { color: '#7f1d1d', label: 'Heavy' },
  { color: '#1e3a8a', label: 'Neutral' },
  { color: '#064e3b', label: 'Uplifted' },
];

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0f0f1a' },
  scroll: { paddingHorizontal: 24, paddingTop: 32, paddingBottom: 48 },
  title: { fontSize: 24, fontWeight: '700', color: '#f3f4f6', marginBottom: 24 },

  streakRow: { flexDirection: 'row', gap: 10, marginBottom: 24 },
  streakCard: {
    flex: 1,
    backgroundColor: '#161625',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
  },
  streakCardHighlight: { backgroundColor: '#1e3a5f', borderWidth: 1, borderColor: '#1d4ed8' },
  streakValue: { fontSize: 28, fontWeight: '700', color: '#9ca3af' },
  streakValueHighlight: { color: '#93c5fd' },
  streakLabel: { fontSize: 11, color: '#4b5563', marginTop: 4, textAlign: 'center' },

  chartCard: {
    backgroundColor: '#161625',
    borderRadius: 16,
    padding: 20,
    marginBottom: 16,
  },
  chartLabel: {
    fontSize: 11,
    color: '#4b5563',
    letterSpacing: 1,
    textTransform: 'uppercase',
    marginBottom: 20,
  },
  emptyChart: { color: '#374151', textAlign: 'center', paddingVertical: 40 },
  bars: { flexDirection: 'row', alignItems: 'flex-end', height: 140, gap: 8 },
  barColumn: { flex: 1, alignItems: 'center', height: '100%' },
  barTrack: {
    flex: 1,
    width: '100%',
    backgroundColor: '#1f2937',
    borderRadius: 4,
    justifyContent: 'flex-end',
    overflow: 'hidden',
  },
  bar: { width: '100%', borderRadius: 4 },
  barLabel: { fontSize: 10, color: '#6b7280', marginTop: 6 },
  barCount: { fontSize: 9, color: '#374151' },

  legend: { flexDirection: 'row', gap: 16, flexWrap: 'wrap' },
  legendItem: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  legendDot: { width: 8, height: 8, borderRadius: 4 },
  legendText: { fontSize: 12, color: '#6b7280' },
});
