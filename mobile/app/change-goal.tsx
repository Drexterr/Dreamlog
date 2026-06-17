import { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  ActivityIndicator,
  Dimensions,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { api } from '../src/api/client';
import { Fonts } from '../src/theme';
import { useTheme } from '../src/context/ThemeContext';
import type { UserGoal } from '../src/types';

const { width: SW, height: SH } = Dimensions.get('window');

const GOALS: { key: UserGoal; label: string; description: string; emoji: string }[] = [
  { key: 'anxiety',       label: 'Working through anxiety',     description: 'Worry, uncertainty, restless thoughts',                    emoji: '🌱' },
  { key: 'stress',        label: 'Managing stress',             description: 'Overwhelm, pressure, too much on my plate',                emoji: '🌊' },
  { key: 'grief',         label: 'Processing grief',            description: 'Loss, endings, things that can\'t be undone',              emoji: '🕊️' },
  { key: 'depression',    label: 'Lifting low mood',            description: 'Sadness, low motivation, feeling flat',                    emoji: '☀️' },
  { key: 'relationships', label: 'Understanding relationships',  description: 'Connection, conflict, how I show up for others',           emoji: '❤️' },
  { key: 'career',        label: 'Career & purpose',            description: 'Work, direction, what I\'m building toward',               emoji: '🌲' },
  { key: 'trauma',        label: 'Processing past / trauma',    description: 'Difficult memories, healing, working through trauma',      emoji: '🩹' },
  { key: 'curious',       label: 'Just exploring',              description: 'No agenda - I\'m curious about my inner life',             emoji: '🌌' },
];

export default function ChangeGoalScreen() {
  const router = useRouter();
  const { theme, colors, setThemeWithBubble } = useTheme();
  const [saving, setSaving] = useState<UserGoal | null>(null);

  const handleSelect = async (goal: UserGoal, pageX: number, pageY: number) => {
    if (saving || goal === theme) {
      router.back();
      return;
    }

    setSaving(goal);
    setThemeWithBubble(goal, pageX || SW / 2, pageY || SH / 2);

    try {
      await api.updateMe({ goal });
    } catch {
      // theme already changed optimistically - don't revert the visual
    } finally {
      setSaving(null);
      router.back();
    }
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        {/* Header */}
        <View style={[styles.header, { borderBottomColor: colors.borderFaint }]}>
          <TouchableOpacity onPress={() => router.back()} style={styles.backBtn}>
            <Text style={[styles.backText, { color: colors.textMuted }]}>← Back</Text>
          </TouchableOpacity>
          <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Emotional Goal</Text>
          <View style={styles.backBtn} />
        </View>

        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
          <Text style={[styles.heading, { color: colors.textPrimary }]}>What are you working on?</Text>
          <Text style={[styles.sub, { color: colors.textSecondary }]}>
            This shapes how your reflections are written. You can change it anytime.
          </Text>

          <View style={styles.goalList}>
            {GOALS.map((g) => {
              const isSelected = theme === g.key;
              const isSaving = saving === g.key;

              return (
                <TouchableOpacity
                  key={g.key}
                  style={[
                    styles.goalCard,
                    {
                      backgroundColor: isSelected ? colors.brandGlow : colors.card,
                      borderColor: isSelected ? colors.brand : colors.border,
                    },
                  ]}
                  onPress={(e) => handleSelect(g.key, e.nativeEvent.pageX, e.nativeEvent.pageY)}
                  disabled={!!saving}
                  activeOpacity={0.7}
                >
                  <View style={styles.goalHeaderRow}>
                    <Text style={[styles.goalLabel, { color: isSelected ? colors.purple300 : colors.textPrimary }]}>
                      {g.label}
                    </Text>
                    {isSaving ? (
                      <ActivityIndicator size="small" color={colors.brand} />
                    ) : isSelected ? (
                      <Text style={[styles.checkmark, { color: colors.brand }]}>✓</Text>
                    ) : null}
                  </View>
                  <Text style={[styles.goalDesc, { color: colors.textMuted }]}>{g.description}</Text>
                </TouchableOpacity>
              );
            })}
          </View>
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },

  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderBottomWidth: 1,
  },
  backBtn: { width: 60 },
  backText: {
    fontFamily: Fonts.sans,
    fontSize: 15,
  },
  headerTitle: {
    fontFamily: Fonts.sansSB,
    fontSize: 17,
  },

  scroll: {
    paddingHorizontal: 20,
    paddingTop: 24,
    paddingBottom: 48,
  },

  heading: {
    fontFamily: Fonts.serif,
    fontSize: 26,
    marginBottom: 8,
    lineHeight: 34,
  },
  sub: {
    fontFamily: Fonts.sans,
    fontSize: 14,
    lineHeight: 20,
    marginBottom: 24,
  },

  goalList: { gap: 10 },

  goalCard: {
    borderWidth: 1,
    borderRadius: 14,
    padding: 16,
  },
  goalHeaderRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 4,
  },
  goalLabel: {
    fontFamily: Fonts.sansSB,
    fontSize: 15,
    flex: 1,
  },
  checkmark: {
    fontSize: 16,
    fontFamily: Fonts.sansSB,
  },
  goalDesc: {
    fontFamily: Fonts.sans,
    fontSize: 13,
    lineHeight: 18,
  },
});
