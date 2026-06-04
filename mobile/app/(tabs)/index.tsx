/**
 * Home screen — the first thing the user sees.
 *
 * Features:
 * - Greeting based on time of day
 * - Streak badge (top right)
 * - Breathing orb (center) — tap to start recording
 * - Mini mood bar for the current week (bottom)
 * - Star field background
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  Animated,
  StyleSheet,
  StatusBar,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { DailyMood, StreakInfo } from '../../src/types';

// ── Star field ───────────────────────────────────────────────────────────────
const STARS = Array.from({ length: 28 }, (_, i) => ({
  id: i,
  left: Math.random() * 100,
  top: Math.random() * 100,
  size: Math.random() > 0.8 ? 2 : 1,
  delay: Math.random() * 4000,
}));

function StarField() {
  const { theme, colors } = useTheme();
  const anims = useRef(STARS.map(() => new Animated.Value(0.1))).current;

  useEffect(() => {
    const loops = anims.map((anim, i) => {
      const loop = Animated.loop(
        Animated.sequence([
          Animated.delay(STARS[i].delay),
          Animated.timing(anim, { toValue: 0.4, duration: 2000, useNativeDriver: true }),
          Animated.timing(anim, { toValue: 0.1, duration: 2000, useNativeDriver: true }),
        ]),
      );
      loop.start();
      return loop;
    });
    return () => loops.forEach((l) => l.stop());
  }, []);

  // Show star field only in curious (purple) theme
  if (theme !== 'curious') return null;

  return (
    <View style={StyleSheet.absoluteFill} pointerEvents="none">
      {STARS.map((star, i) => (
        <Animated.View
          key={star.id}
          style={{
            position: 'absolute',
            width: star.size,
            height: star.size,
            borderRadius: star.size / 2,
            backgroundColor: colors.purple300,
            left: `${star.left}%`,
            top: `${star.top}%`,
            opacity: anims[i],
          }}
        />
      ))}
    </View>
  );
}

// ── Breathing orb ─────────────────────────────────────────────────────────────
function BreathingOrb({ onPress }: { onPress: () => void }) {
  const { colors } = useTheme();
  const scaleAnim = useRef(new Animated.Value(1)).current;
  const glowAnim = useRef(new Animated.Value(0.3)).current;

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.parallel([
          Animated.timing(scaleAnim, { toValue: 1.08, duration: 4000, useNativeDriver: true }),
          Animated.timing(glowAnim, { toValue: 0.6, duration: 4000, useNativeDriver: true }),
        ]),
        Animated.parallel([
          Animated.timing(scaleAnim, { toValue: 1, duration: 4000, useNativeDriver: true }),
          Animated.timing(glowAnim, { toValue: 0.3, duration: 4000, useNativeDriver: true }),
        ]),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, []);

  return (
    <TouchableOpacity onPress={onPress} activeOpacity={0.9}>
      <View style={styles.orbContainer}>
        {/* Glow ring */}
        <Animated.View style={[styles.orbGlow, { backgroundColor: colors.purple600, opacity: glowAnim }]} />
        {/* Core orb */}
        <Animated.View
          style={[
            styles.orb,
            {
              backgroundColor: colors.purple700,
              shadowColor: colors.purple500,
              transform: [{ scale: scaleAnim }],
            },
          ]}
        >
          {/* Mic icon */}
          <View style={styles.micWrap}>
            <View style={styles.micBody} />
            <View style={styles.micBase} />
            <View style={styles.micLine} />
          </View>
        </Animated.View>
      </View>
    </TouchableOpacity>
  );
}

// ── Mini mood bar ─────────────────────────────────────────────────────────────
function MiniMoodBar({ days }: { days: DailyMood[] }) {
  const { colors, moodToColor } = useTheme();
  const dayLabels = ['M', 'T', 'W', 'T', 'F', 'S', 'S'];

  // Build a 7-slot array aligned to weekday
  const today = new Date();
  const todayDow = today.getDay(); // 0=Sun
  const slots = Array.from({ length: 7 }, (_, i) => {
    const offset = i - ((todayDow + 6) % 7); // Monday=0
    const d = new Date(today);
    d.setDate(today.getDate() + offset);
    const key = d.toISOString().slice(0, 10);
    return days.find((m) => m.day === key) ?? null;
  });

  return (
    <View style={styles.moodBarWrap}>
      <Text style={[styles.moodBarLabel, { color: colors.textSecondary }]}>THIS WEEK</Text>
      <View style={styles.moodBars}>
        {slots.map((slot, i) => (
          <View key={i} style={styles.moodBarCol}>
            {slot ? (
              <View
                style={[
                  styles.moodBarFill,
                  {
                    height: Math.max(8, (slot.avg_mood / 100) * 40),
                    backgroundColor: moodToColor(slot.avg_mood) + '55',
                    borderTopColor: moodToColor(slot.avg_mood),
                  },
                ]}
              />
            ) : (
              <View style={[styles.moodBarEmpty, { borderColor: colors.border }]} />
            )}
            <Text style={[styles.moodBarDay, { color: colors.textMuted }]}>{dayLabels[i]}</Text>
          </View>
        ))}
      </View>
    </View>
  );
}

// ── Home screen ───────────────────────────────────────────────────────────────
export default function HomeScreen() {
  const router = useRouter();
  const { colors } = useTheme();
  const [greeting, setGreeting] = useState('Good evening');
  const [streak, setStreak] = useState<StreakInfo | null>(null);
  const [weekMoods, setWeekMoods] = useState<DailyMood[]>([]);

  const fadeAnim = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    const h = new Date().getHours();
    if (h < 12) setGreeting('Good morning');
    else if (h < 17) setGreeting('Good afternoon');
    else if (h < 21) setGreeting('Good evening');
    else setGreeting('Hey there');

    Animated.timing(fadeAnim, { toValue: 1, duration: 800, useNativeDriver: true }).start();

    // Load streak + mood data.
    api.streak().then(setStreak).catch(() => {});
    api.weeklyMood().then((r) => setWeekMoods(r.days ?? [])).catch(() => {});
  }, []);

  const handleOrbPress = useCallback(() => {
    router.push('/record');
  }, [router]);

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <StarField />

      <SafeAreaView style={styles.safe}>
        {/* Streak badge — absolute in top-right, within safe area */}
        {streak && streak.current_streak > 0 && (
          <View style={[styles.streakBadge, { borderColor: colors.border, backgroundColor: colors.card }]}>
            <Text style={styles.streakEmoji}>🔥</Text>
            <Text style={[styles.streakText, { color: colors.purple300 }]}>{streak.current_streak} days</Text>
          </View>
        )}

        {/* Centered content — grows to fill available space */}
        <Animated.View style={[styles.centerContent, { opacity: fadeAnim }]}>
          <View style={styles.greetingWrap}>
            <Text style={[styles.greetingSub, { color: colors.textSecondary }]}>{greeting}</Text>
            <Text style={[styles.greetingMain, { color: colors.textPrimary }]}>How was your day?</Text>
          </View>
          <BreathingOrb onPress={handleOrbPress} />
          <Text style={[styles.tapHint, { color: colors.textMuted }]}>tap to start talking</Text>
          <View style={styles.quickLinks}>
            <TouchableOpacity
              style={[styles.quickBtn, { borderColor: colors.border, backgroundColor: colors.card }]}
              onPress={() => router.push('/journeys')}
              activeOpacity={0.8}
            >
              <Text style={[styles.quickBtnText, { color: colors.textSecondary }]}>Guided Journeys →</Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={[styles.quickBtn, { borderColor: colors.border, backgroundColor: colors.card }]}
              onPress={() => router.push('/therapy' as any)}
              activeOpacity={0.8}
            >
              <Text style={[styles.quickBtnText, { color: colors.textSecondary }]}>🌿 Reflection Session →</Text>
            </TouchableOpacity>
          </View>
        </Animated.View>

        {/* Mini mood bar — always below center content, never overlaps */}
        <Animated.View style={[styles.moodSection, { opacity: fadeAnim }]}>
          <MiniMoodBar days={weekMoods} />
        </Animated.View>
      </SafeAreaView>
    </View>
  );
}

const ORB_SIZE = 148;

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  safe: {
    flex: 1,
    paddingHorizontal: 24,
    paddingBottom: 20,
  },

  // Streak
  streakBadge: {
    position: 'absolute',
    top: 16,
    right: 0,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    borderRadius: 20,
    paddingHorizontal: 14,
    paddingVertical: 6,
    borderWidth: 1,
    zIndex: 1,
  },
  streakEmoji: { fontSize: 14 },
  streakText: {
    fontSize: 13,
    fontFamily: 'Nunito_600SemiBold',
  },

  // Center content — fills available vertical space, centres children
  centerContent: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },

  // Greeting
  greetingWrap: { alignItems: 'center', marginBottom: 44 },
  greetingSub: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 1,
    marginBottom: 8,
  },
  greetingMain: {
    fontSize: 28,
    fontFamily: 'CormorantGaramond_300Light',
    fontWeight: '300',
    letterSpacing: 0.5,
  },

  // Orb
  orbContainer: {
    width: ORB_SIZE + 40,
    height: ORB_SIZE + 40,
    alignItems: 'center',
    justifyContent: 'center',
  },
  orbGlow: {
    position: 'absolute',
    width: ORB_SIZE + 40,
    height: ORB_SIZE + 40,
    borderRadius: (ORB_SIZE + 40) / 2,
  },
  orb: {
    width: ORB_SIZE,
    height: ORB_SIZE,
    borderRadius: ORB_SIZE / 2,
    alignItems: 'center',
    justifyContent: 'center',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.5,
    shadowRadius: 24,
    elevation: 12,
  },
  // Simple mic shape
  micWrap: { alignItems: 'center', gap: 4 },
  micBody: {
    width: 20,
    height: 28,
    borderRadius: 10,
    borderWidth: 2.5,
    borderColor: 'rgba(255,255,255,0.85)',
  },
  micBase: {
    width: 28,
    height: 10,
    borderTopLeftRadius: 14,
    borderTopRightRadius: 14,
    borderWidth: 2,
    borderBottomWidth: 0,
    borderColor: 'rgba(255,255,255,0.85)',
  },
  micLine: {
    width: 2,
    height: 6,
    backgroundColor: 'rgba(255,255,255,0.85)',
    borderRadius: 1,
  },

  tapHint: {
    marginTop: 20,
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.5,
  },
  quickLinks: {
    marginTop: 16,
    gap: 8,
    alignItems: 'center',
  },
  quickBtn: {
    borderWidth: 1,
    borderRadius: 20,
    paddingHorizontal: 18,
    paddingVertical: 8,
  },
  quickBtnText: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.3,
  },

  // Mini mood bar — always below the orb section, no absolute positioning
  moodSection: {
    width: '100%',
    paddingTop: 16,
  },
  moodBarWrap: {},
  moodBarLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.5,
    marginBottom: 10,
  },
  moodBars: {
    flexDirection: 'row',
    gap: 6,
    alignItems: 'flex-end',
    height: 52,
  },
  moodBarCol: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: 4,
  },
  moodBarFill: {
    width: '100%',
    borderRadius: 4,
    borderTopWidth: 2,
  },
  moodBarEmpty: {
    width: '100%',
    height: 8,
    borderRadius: 4,
    borderWidth: 1,
    borderStyle: 'dashed',
  },
  moodBarDay: {
    fontSize: 9,
    fontFamily: 'Nunito_400Regular',
  },
});
