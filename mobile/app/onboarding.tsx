import { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Dimensions,
} from 'react-native';
import { useRouter } from 'expo-router';
import { api } from '../src/api/client';
import { Fonts } from '../src/theme';
import { useTheme } from '../src/context/ThemeContext';
import type { AgeRange, UserGoal } from '../src/types';

const { width: SW, height: SH } = Dimensions.get('window');

const AGE_RANGES: { key: AgeRange; label: string }[] = [
  { key: 'under_18', label: 'Under 18' },
  { key: '18_24',    label: '18 – 24' },
  { key: '25_34',    label: '25 – 34' },
  { key: '35_44',    label: '35 – 44' },
  { key: '45_plus',  label: '45 or older' },
];

const GOALS: { key: UserGoal; label: string; description: string; emoji: string }[] = [
  { key: 'anxiety',       label: 'Working through anxiety', description: 'Worry, uncertainty, restless thoughts', emoji: '🌱' },
  { key: 'stress',        label: 'Managing stress',       description: 'Overwhelm, pressure, too much on my plate', emoji: '🌊' },
  { key: 'grief',         label: 'Processing grief',       description: 'Loss, endings, things that can\'t be undone', emoji: '🕊️' },
  { key: 'depression',    label: 'Lifting low mood',      description: 'Sadness, low motivation, feeling flat', emoji: '☀️' },
  { key: 'relationships', label: 'Understanding relationships', description: 'Connection, conflict, how I show up for others', emoji: '❤️' },
  { key: 'career',        label: 'Career & purpose',      description: 'Work, direction, what I\'m building toward', emoji: '🌲' },
  { key: 'trauma',        label: 'Processing past / trauma', description: 'Difficult memories, healing, working through trauma', emoji: '🩹' },
  { key: 'curious',       label: 'Just exploring',         description: 'No agenda — I\'m curious about my inner life', emoji: '🌌' },
];

export default function OnboardingScreen() {
  const router = useRouter();
  const { colors, setThemeWithBubble } = useTheme();
  const [step, setStep] = useState<1 | 2 | 3 | 4>(1);
  const [selectedGoal, setSelectedGoal] = useState<UserGoal | null>(null);
  const [preferredName, setPreferredName] = useState('');
  const [selectedAgeRange, setSelectedAgeRange] = useState<AgeRange | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Called at end of step 3 — saves profile, then shows the mode gate (step 4).
  async function handleSaveAndContinue() {
    if (!selectedGoal) return;
    setLoading(true);
    setError('');
    try {
      await api.updateMe({
        goal: selectedGoal,
        ...(preferredName.trim() ? { preferred_name: preferredName.trim() } : {}),
        ...(selectedAgeRange ? { age_range: selectedAgeRange } : {}),
      });
      setStep(4);
    } catch {
      setError('Something went wrong. Please try again.');
    } finally {
      setLoading(false);
    }
  }

  function handleChooseJournal() {
    router.replace('/(tabs)');
  }

  function handleChooseTherapy() {
    // Replace so tapping back from the persona picker doesn't return here.
    router.replace('/therapy/persona-picker' as any);
  }

  return (
    <KeyboardAvoidingView
      style={[styles.root, { backgroundColor: colors.bg }]}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
        <Text style={[styles.wordmark, { color: colors.purple300 }]}>DreamLog</Text>

        {step === 1 ? (
          <>
            <Text style={[styles.heading, { color: colors.textPrimary }]}>What brings you here?</Text>
            <Text style={[styles.sub, { color: colors.textSecondary }]}>
              Your answer helps us shape how we reflect your words back to you.
            </Text>

            <View style={styles.goalList}>
              {GOALS.map((g) => {
                const isSelected = selectedGoal === g.key;
                return (
                  <TouchableOpacity
                    key={g.key}
                    style={[
                      styles.goalCard,
                      {
                        backgroundColor: colors.card,
                        borderColor: isSelected ? colors.brand : colors.border,
                      },
                      isSelected && { backgroundColor: colors.brandGlow },
                    ]}
                    onPress={(event) => {
                      const { pageX, pageY } = event.nativeEvent;
                      setSelectedGoal(g.key);
                      setThemeWithBubble(g.key, pageX || SW / 2, pageY || SH / 2);
                    }}
                    activeOpacity={0.7}
                  >
                    <View style={styles.goalHeaderRow}>
                      <Text style={[styles.goalLabel, { color: isSelected ? colors.purple300 : colors.textPrimary }]}>
                        {g.label}
                      </Text>
                      <Text style={{ fontSize: 16 }}>{g.emoji}</Text>
                    </View>
                    <Text style={[styles.goalDesc, { color: colors.textMuted }]}>{g.description}</Text>
                  </TouchableOpacity>
                );
              })}
            </View>

            <TouchableOpacity
              style={[styles.btn, { backgroundColor: colors.brand }, !selectedGoal && styles.btnDisabled]}
              onPress={() => selectedGoal && setStep(2)}
              disabled={!selectedGoal}
              activeOpacity={0.8}
            >
              <Text style={[styles.btnText, { color: colors.textPrimary }]}>Continue</Text>
            </TouchableOpacity>
          </>
        ) : step === 2 ? (
          <>
            <Text style={[styles.heading, { color: colors.textPrimary }]}>What name should I call you?</Text>
            <Text style={[styles.sub, { color: colors.textSecondary }]}>
              Optional — leave it blank and I'll use whatever's on your account.
            </Text>

            <TextInput
              style={[
                styles.input,
                {
                  backgroundColor: colors.card,
                  borderColor: colors.border,
                  color: colors.textPrimary,
                },
              ]}
              placeholder="e.g. Alex, or just skip this"
              placeholderTextColor={colors.textMuted}
              value={preferredName}
              onChangeText={setPreferredName}
              maxLength={100}
              autoFocus
              returnKeyType="done"
              onSubmitEditing={() => setStep(3)}
            />

            <TouchableOpacity
              style={[styles.btn, { backgroundColor: colors.brand }]}
              onPress={() => setStep(3)}
              activeOpacity={0.8}
            >
              <Text style={[styles.btnText, { color: colors.textPrimary }]}>Continue</Text>
            </TouchableOpacity>

            <TouchableOpacity style={styles.back} onPress={() => setStep(1)}>
              <Text style={[styles.backText, { color: colors.textMuted }]}>← Back</Text>
            </TouchableOpacity>
          </>
        ) : step === 3 ? (
          <>
            <Text style={[styles.heading, { color: colors.textPrimary }]}>How old are you?</Text>
            <Text style={[styles.sub, { color: colors.textSecondary }]}>
              Optional — helps us understand who uses DreamLog. We never share individual data.
            </Text>

            <View style={styles.ageList}>
              {AGE_RANGES.map((a) => {
                const isSelected = selectedAgeRange === a.key;
                return (
                  <TouchableOpacity
                    key={a.key}
                    style={[
                      styles.ageCard,
                      {
                        backgroundColor: colors.card,
                        borderColor: isSelected ? colors.brand : colors.border,
                      },
                      isSelected && { backgroundColor: colors.brandGlow },
                    ]}
                    onPress={() => setSelectedAgeRange(isSelected ? null : a.key)}
                    activeOpacity={0.7}
                  >
                    <Text style={[styles.ageLabel, { color: isSelected ? colors.purple300 : colors.textPrimary }]}>
                      {a.label}
                    </Text>
                  </TouchableOpacity>
                );
              })}
            </View>

            {error ? <Text style={[styles.error, { color: colors.danger }]}>{error}</Text> : null}

            <TouchableOpacity
              style={[styles.btn, { backgroundColor: colors.brand }, loading && styles.btnDisabled]}
              onPress={handleSaveAndContinue}
              disabled={loading}
              activeOpacity={0.8}
            >
              {loading ? (
                <ActivityIndicator color={colors.textPrimary} />
              ) : (
                <Text style={[styles.btnText, { color: colors.textPrimary }]}>
                  {selectedAgeRange ? 'Continue' : 'Skip'}
                </Text>
              )}
            </TouchableOpacity>

            <TouchableOpacity style={styles.back} onPress={() => setStep(2)}>
              <Text style={[styles.backText, { color: colors.textMuted }]}>← Back</Text>
            </TouchableOpacity>
          </>
        ) : (
          /* Step 4 — Journal vs Therapy gate */
          <>
            <Text style={[styles.heading, { color: colors.textPrimary }]}>How would you like to begin?</Text>
            <Text style={[styles.sub, { color: colors.textSecondary }]}>
              You can always switch later — both live in the app.
            </Text>

            <View style={styles.modeCards}>
              <TouchableOpacity
                style={[styles.modeCard, { backgroundColor: colors.card, borderColor: colors.border }]}
                onPress={handleChooseJournal}
                activeOpacity={0.75}
              >
                <Text style={styles.modeEmoji}>📓</Text>
                <Text style={[styles.modeLabel, { color: colors.textPrimary }]}>Journal</Text>
                <Text style={[styles.modeDesc, { color: colors.textSecondary }]}>
                  Record a voice entry and get a personalised AI reflection. Async — do it at your own pace.
                </Text>
              </TouchableOpacity>

              <TouchableOpacity
                style={[styles.modeCard, { backgroundColor: colors.card, borderColor: colors.border }]}
                onPress={handleChooseTherapy}
                activeOpacity={0.75}
              >
                <Text style={styles.modeEmoji}>🌿</Text>
                <Text style={[styles.modeLabel, { color: colors.textPrimary }]}>Reflection Session</Text>
                <Text style={[styles.modeDesc, { color: colors.textSecondary }]}>
                  A live back-and-forth conversation with an AI companion grounded in your journal history.
                </Text>
                <View style={[styles.modeBadge, { backgroundColor: colors.brandGlow }]}>
                  <Text style={[styles.modeBadgeText, { color: colors.brand }]}>First session free</Text>
                </View>
              </TouchableOpacity>
            </View>
          </>
        )}
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
  },
  scroll: {
    flexGrow: 1,
    paddingHorizontal: 24,
    paddingTop: 64,
    paddingBottom: 48,
  },
  wordmark: {
    fontFamily: Fonts.serif,
    fontSize: 22,
    marginBottom: 40,
    letterSpacing: 1,
  },
  heading: {
    fontFamily: Fonts.serif,
    fontSize: 30,
    marginBottom: 10,
    lineHeight: 38,
  },
  sub: {
    fontFamily: Fonts.sans,
    fontSize: 15,
    marginBottom: 32,
    lineHeight: 22,
  },
  goalList: {
    gap: 10,
    marginBottom: 32,
  },
  goalCard: {
    borderWidth: 1,
    borderRadius: 12,
    padding: 16,
  },
  goalHeaderRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 3,
  },
  goalLabel: {
    fontFamily: Fonts.sansSB,
    fontSize: 15,
  },
  goalDesc: {
    fontFamily: Fonts.sans,
    fontSize: 13,
    lineHeight: 18,
  },
  btn: {
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
  },
  btnDisabled: {
    opacity: 0.4,
  },
  btnText: {
    fontFamily: Fonts.sansSB,
    fontSize: 16,
  },
  input: {
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 14,
    fontFamily: Fonts.sans,
    fontSize: 16,
    marginBottom: 24,
  },
  error: {
    fontFamily: Fonts.sans,
    fontSize: 13,
    marginBottom: 16,
  },
  back: {
    alignItems: 'center',
    marginTop: 20,
  },
  backText: {
    fontFamily: Fonts.sans,
    fontSize: 14,
  },
  ageList: {
    gap: 10,
    marginBottom: 32,
  },
  ageCard: {
    borderWidth: 1,
    borderRadius: 12,
    paddingVertical: 14,
    paddingHorizontal: 16,
    alignItems: 'center',
  },
  ageLabel: {
    fontFamily: Fonts.sansSB,
    fontSize: 15,
  },
  modeCards: {
    gap: 14,
    marginBottom: 16,
  },
  modeCard: {
    borderWidth: 1,
    borderRadius: 16,
    padding: 20,
    gap: 6,
  },
  modeEmoji: {
    fontSize: 32,
    marginBottom: 4,
  },
  modeLabel: {
    fontFamily: Fonts.sansSB,
    fontSize: 18,
    marginBottom: 2,
  },
  modeDesc: {
    fontFamily: Fonts.sans,
    fontSize: 14,
    lineHeight: 20,
  },
  modeBadge: {
    alignSelf: 'flex-start',
    borderRadius: 6,
    paddingHorizontal: 8,
    paddingVertical: 3,
    marginTop: 8,
  },
  modeBadgeText: {
    fontFamily: Fonts.sansSB,
    fontSize: 12,
  },
});
