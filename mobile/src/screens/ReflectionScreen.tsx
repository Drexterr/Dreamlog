/**
 * ReflectionScreen — displayed after processing completes.
 *
 * Features:
 * - Dark calming UI
 * - Smooth fade-in animation (via Animated API — no reanimated dep required for this)
 * - Mood color indicator (no numeric score shown to user)
 * - "Goodnight" / "Tell me more" CTA pair
 */

import React, { useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  StyleSheet,
  Animated,
  TouchableOpacity,
  SafeAreaView,
  StatusBar,
} from 'react-native';
import { Entry, EntryAnalysis } from '../types';
import { api } from '../api/client';

interface Props {
  entry: Entry;
  onGoodnight: () => void;
  onTellMeMore: (entry: Entry, analysis: EntryAnalysis) => void;
}

export function ReflectionScreen({ entry, onGoodnight, onTellMeMore }: Props) {
  const [analysis, setAnalysis] = useState<EntryAnalysis | null>(null);
  const [loading, setLoading] = useState(true);

  const fadeAnim = useRef(new Animated.Value(0)).current;
  const slideAnim = useRef(new Animated.Value(24)).current;

  useEffect(() => {
    loadAnalysis();
  }, [entry.id]);

  const loadAnalysis = async () => {
    try {
      const data = await api.getAnalysis(entry.id);
      setAnalysis(data);
      // Trigger fade-in after data loads.
      Animated.parallel([
        Animated.timing(fadeAnim, {
          toValue: 1,
          duration: 800,
          useNativeDriver: true,
        }),
        Animated.timing(slideAnim, {
          toValue: 0,
          duration: 600,
          useNativeDriver: true,
        }),
      ]).start();
    } catch {
      // Entry may not have analysis yet — silently show minimal UI.
    } finally {
      setLoading(false);
    }
  };

  // Map mood_score to a calming color hue — never shown as a number.
  const moodColor = analysis ? moodToColor(analysis.mood_score) : '#4a5568';

  return (
    <SafeAreaView style={styles.container}>
      <StatusBar barStyle="light-content" />

      {/* Mood accent line at top */}
      <View style={[styles.moodBar, { backgroundColor: moodColor }]} />

      <ScrollView
        contentContainerStyle={styles.scroll}
        showsVerticalScrollIndicator={false}
      >
        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.dateLabel}>
            {new Date(entry.created_at).toLocaleDateString('en-US', {
              weekday: 'long',
              month: 'long',
              day: 'numeric',
            })}
          </Text>
          {analysis && (
            <View style={styles.topicsRow}>
              {analysis.topics.slice(0, 3).map((t) => (
                <View key={t} style={styles.topicBadge}>
                  <Text style={styles.topicText}>{t}</Text>
                </View>
              ))}
            </View>
          )}
        </View>

        {/* Reflection text — core content */}
        {analysis ? (
          <Animated.View
            style={[
              styles.reflectionCard,
              { opacity: fadeAnim, transform: [{ translateY: slideAnim }] },
            ]}
          >
            <Text style={styles.reflectionLabel}>Your reflection</Text>
            <Text style={styles.reflectionText}>{analysis.reflection}</Text>

            {/* Emotional tone dots */}
            {analysis.emotional_tone?.length > 0 && (
              <View style={styles.toneRow}>
                {analysis.emotional_tone.slice(0, 3).map((et) => (
                  <View key={et.emotion} style={styles.toneItem}>
                    <View
                      style={[
                        styles.toneDot,
                        {
                          backgroundColor: moodColor,
                          opacity: et.intensity,
                          transform: [{ scale: 0.6 + et.intensity * 0.6 }],
                        },
                      ]}
                    />
                    <Text style={styles.toneLabel}>{et.emotion}</Text>
                  </View>
                ))}
              </View>
            )}
          </Animated.View>
        ) : (
          <View style={styles.reflectionCard}>
            <Text style={styles.reflectionText}>
              {loading ? 'Reflecting…' : 'No reflection available yet.'}
            </Text>
          </View>
        )}

        {/* Key quote */}
        {analysis?.key_quotes?.[0] && (
          <Animated.View style={[styles.quoteCard, { opacity: fadeAnim }]}>
            <Text style={styles.quoteText}>"{analysis.key_quotes[0]}"</Text>
          </Animated.View>
        )}

        {/* CTAs */}
        <View style={styles.ctaRow}>
          <TouchableOpacity
            style={[styles.ctaButton, styles.goodnightButton]}
            onPress={onGoodnight}
            activeOpacity={0.8}
          >
            <Text style={styles.goodnightText}>Goodnight</Text>
          </TouchableOpacity>

          {analysis && !analysis.is_crisis && (
            <TouchableOpacity
              style={[styles.ctaButton, styles.tellMoreButton]}
              onPress={() => onTellMeMore(entry, analysis)}
              activeOpacity={0.8}
            >
              <Text style={styles.tellMoreText}>Tell me more</Text>
            </TouchableOpacity>
          )}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

// Map 1-100 mood score to a color without showing the number to the user.
function moodToColor(score: number): string {
  if (score <= 20) return '#7f1d1d'; // deep red — heavy
  if (score <= 40) return '#92400e'; // amber-brown — struggling
  if (score <= 60) return '#1e3a5f'; // deep blue — neutral/processing
  if (score <= 80) return '#1e4035'; // forest green — hopeful
  return '#1a3a2a';                  // deep emerald — uplifted
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0f0f1a' },
  moodBar: { height: 3, width: '100%' },
  scroll: { paddingHorizontal: 24, paddingTop: 32, paddingBottom: 48 },

  header: { marginBottom: 28 },
  dateLabel: { fontSize: 13, color: '#6b7280', letterSpacing: 0.5, marginBottom: 12 },
  topicsRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  topicBadge: {
    borderRadius: 20,
    paddingHorizontal: 10,
    paddingVertical: 4,
    backgroundColor: '#1f2937',
  },
  topicText: { fontSize: 12, color: '#9ca3af' },

  reflectionCard: {
    backgroundColor: '#161625',
    borderRadius: 16,
    padding: 24,
    marginBottom: 20,
  },
  reflectionLabel: {
    fontSize: 11,
    color: '#4b5563',
    letterSpacing: 1.2,
    textTransform: 'uppercase',
    marginBottom: 16,
  },
  reflectionText: {
    fontSize: 17,
    color: '#e5e7eb',
    lineHeight: 28,
    fontWeight: '300',
  },

  toneRow: { flexDirection: 'row', marginTop: 24, gap: 20 },
  toneItem: { alignItems: 'center', gap: 6 },
  toneDot: { width: 10, height: 10, borderRadius: 5 },
  toneLabel: { fontSize: 11, color: '#6b7280' },

  quoteCard: {
    borderLeftWidth: 2,
    borderLeftColor: '#374151',
    paddingLeft: 16,
    marginBottom: 32,
  },
  quoteText: { fontSize: 14, color: '#6b7280', fontStyle: 'italic', lineHeight: 22 },

  ctaRow: { flexDirection: 'row', gap: 12 },
  ctaButton: {
    flex: 1,
    paddingVertical: 16,
    borderRadius: 14,
    alignItems: 'center',
  },
  goodnightButton: { backgroundColor: '#1f2937' },
  goodnightText: { fontSize: 16, color: '#9ca3af', fontWeight: '500' },
  tellMoreButton: { backgroundColor: '#1e3a5f' },
  tellMoreText: { fontSize: 16, color: '#93c5fd', fontWeight: '600' },
});
