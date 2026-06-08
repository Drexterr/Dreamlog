import { useState, useRef } from 'react';
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
  Animated,
  Easing,
} from 'react-native';
import { useRouter } from 'expo-router';
import { api } from '../src/api/client';
import { Fonts, THEMES } from '../src/theme';
import { useTheme } from '../src/context/ThemeContext';
import type { AgeRange, UserGoal } from '../src/types';

const { width: SW, height: SH } = Dimensions.get('window');

// Flood fill circle — starts at this diameter and scales up.
const FLOOD_D = 100;
const FLOOD_R = FLOOD_D / 2;

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
  { key: 'grief',         label: 'Processing grief',       description: "Loss, endings, things that can't be undone", emoji: '🕊️' },
  { key: 'depression',    label: 'Lifting low mood',      description: 'Sadness, low motivation, feeling flat', emoji: '☀️' },
  { key: 'relationships', label: 'Understanding relationships', description: 'Connection, conflict, how I show up for others', emoji: '❤️' },
  { key: 'career',        label: 'Career & purpose',      description: "Work, direction, what I'm building toward", emoji: '🌲' },
  { key: 'trauma',        label: 'Processing past / trauma', description: 'Difficult memories, healing, working through trauma', emoji: '🩹' },
  { key: 'curious',       label: 'Just exploring',         description: "No agenda — I'm curious about my inner life", emoji: '🌌' },
];

const COUNTRIES: { code: string; name: string; flag: string }[] = [
  { code: 'IN', name: 'India', flag: '🇮🇳' },
  { code: 'US', name: 'United States', flag: '🇺🇸' },
  { code: 'GB', name: 'United Kingdom', flag: '🇬🇧' },
  { code: 'CA', name: 'Canada', flag: '🇨🇦' },
  { code: 'AU', name: 'Australia', flag: '🇦🇺' },
  { code: 'DE', name: 'Germany', flag: '🇩🇪' },
  { code: 'FR', name: 'France', flag: '🇫🇷' },
  { code: 'NL', name: 'Netherlands', flag: '🇳🇱' },
  { code: 'SE', name: 'Sweden', flag: '🇸🇪' },
  { code: 'NO', name: 'Norway', flag: '🇳🇴' },
  { code: 'DK', name: 'Denmark', flag: '🇩🇰' },
  { code: 'CH', name: 'Switzerland', flag: '🇨🇭' },
  { code: 'AT', name: 'Austria', flag: '🇦🇹' },
  { code: 'ES', name: 'Spain', flag: '🇪🇸' },
  { code: 'IT', name: 'Italy', flag: '🇮🇹' },
  { code: 'PT', name: 'Portugal', flag: '🇵🇹' },
  { code: 'IE', name: 'Ireland', flag: '🇮🇪' },
  { code: 'BE', name: 'Belgium', flag: '🇧🇪' },
  { code: 'SG', name: 'Singapore', flag: '🇸🇬' },
  { code: 'NZ', name: 'New Zealand', flag: '🇳🇿' },
  { code: 'PK', name: 'Pakistan', flag: '🇵🇰' },
  { code: 'BD', name: 'Bangladesh', flag: '🇧🇩' },
  { code: 'NG', name: 'Nigeria', flag: '🇳🇬' },
  { code: 'ZA', name: 'South Africa', flag: '🇿🇦' },
  { code: 'BR', name: 'Brazil', flag: '🇧🇷' },
  { code: 'MX', name: 'Mexico', flag: '🇲🇽' },
  { code: 'JP', name: 'Japan', flag: '🇯🇵' },
  { code: 'KR', name: 'South Korea', flag: '🇰🇷' },
  { code: 'AE', name: 'UAE', flag: '🇦🇪' },
  { code: 'OTHER', name: 'Other', flag: '🌍' },
];

export default function OnboardingScreen() {
  const router = useRouter();
  const { colors, setTheme } = useTheme();

  const [step, setStep] = useState<1 | 2 | 3 | 4 | 5>(1);
  const [selectedGoal, setSelectedGoal] = useState<UserGoal | null>(null);
  const [preferredName, setPreferredName] = useState('');
  const [selectedAgeRange, setSelectedAgeRange] = useState<AgeRange | null>(null);
  const [selectedCountry, setSelectedCountry] = useState<string | null>(null);
  const [countrySearch, setCountrySearch] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // ── Flood fill animation state ──────────────────────────────────────────────
  const floodAnim    = useRef(new Animated.Value(0)).current;
  const contentOpacity = useRef(new Animated.Value(1)).current;
  const floodActive  = useRef(false);
  const floodColor   = useRef(colors.bg);
  const floodOriginX = useRef(SW / 2);
  const floodOriginY = useRef(SH / 2);

  const filteredCountries = countrySearch.trim()
    ? COUNTRIES.filter(c =>
        c.name.toLowerCase().includes(countrySearch.toLowerCase()) ||
        c.code.toLowerCase().includes(countrySearch.toLowerCase())
      )
    : COUNTRIES;

  // ── Goal tap: start flood fill, advance step after animation ────────────────
  function handleGoalSelect(goal: UserGoal, pageX: number, pageY: number) {
    if (floodActive.current) return;
    floodActive.current = true;

    setSelectedGoal(goal);

    const targetBg = THEMES[goal].bg;
    floodColor.current  = targetBg;
    floodOriginX.current = pageX > 0 ? pageX : SW / 2;
    floodOriginY.current = pageY > 0 ? pageY : SH / 2;

    floodAnim.setValue(0);
    contentOpacity.setValue(1);

    // Scale needed so the circle fully covers the screen from the tap origin.
    const dx = Math.max(floodOriginX.current, SW - floodOriginX.current);
    const dy = Math.max(floodOriginY.current, SH - floodOriginY.current);
    const maxDist = Math.sqrt(dx * dx + dy * dy);
    const targetScale = (maxDist / FLOOD_R) * 1.15;

    Animated.parallel([
      // Flood circle expands — starts gently, builds momentum, settles softly.
      Animated.timing(floodAnim, {
        toValue: targetScale,
        duration: 1250,
        easing: Easing.bezier(0.35, 0.01, 0.08, 1),
        useNativeDriver: true,
      }),
      // Cards hold full opacity for 800 ms, then fade out over 450 ms.
      Animated.sequence([
        Animated.delay(800),
        Animated.timing(contentOpacity, {
          toValue: 0,
          duration: 450,
          easing: Easing.out(Easing.quad),
          useNativeDriver: true,
        }),
      ]),
    ]).start(() => {
      // Apply theme now — bg already matches floodColor, seamless swap.
      setTheme(goal);
      api.updateMe({ goal }).catch(() => {});
      contentOpacity.setValue(1);
      floodActive.current = false;
      setStep(2);
    });
  }

  // ── Save profile (step 4 → 5) ───────────────────────────────────────────────
  async function handleSaveAndContinue() {
    if (!selectedGoal) return;
    setLoading(true);
    setError('');
    try {
      await api.updateMe({
        goal: selectedGoal,
        ...(preferredName.trim() ? { preferred_name: preferredName.trim() } : {}),
        ...(selectedAgeRange ? { age_range: selectedAgeRange } : {}),
        ...(selectedCountry && selectedCountry !== 'OTHER' ? { country: selectedCountry } : {}),
      });
      setStep(5);
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
    router.replace('/therapy/persona-picker' as any);
  }

  return (
    <KeyboardAvoidingView
      style={[styles.root, { backgroundColor: colors.bg }]}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      {/* ── Step 1: Goal (with flood fill animation) ── */}
      {step === 1 && (
        <View style={styles.floodContainer}>
          {/*
            Flood fill circle — rendered BEFORE the ScrollView so it sits
            behind the card content (later siblings paint on top in RN).
            pointerEvents="none" so taps pass through to the cards.
          */}
          <Animated.View
            pointerEvents="none"
            style={[
              styles.floodCircle,
              {
                backgroundColor: floodColor.current,
                left: floodOriginX.current - FLOOD_R,
                top:  floodOriginY.current - FLOOD_R,
                transform: [{ scale: floodAnim }],
              },
            ]}
          />

          {/* Cards layer — higher zIndex so they always render above the flood */}
          <Animated.ScrollView
            contentContainerStyle={styles.scroll}
            keyboardShouldPersistTaps="handled"
            style={[styles.scrollLayer, { opacity: contentOpacity }]}
          >
            <Text style={[styles.wordmark, { color: colors.purple300 }]}>DreamLog</Text>
            <Text style={[styles.heading, { color: colors.textPrimary }]}>What brings you here?</Text>
            <Text style={[styles.sub, { color: colors.textSecondary }]}>
              Tap any goal — it shapes how we reflect your words back to you.
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
                      handleGoalSelect(g.key, pageX, pageY);
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
          </Animated.ScrollView>
        </View>
      )}

      {/* ── Steps 2-5: plain scroll, no flood animation needed ── */}
      {step !== 1 && (
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <Text style={[styles.wordmark, { color: colors.purple300 }]}>DreamLog</Text>

          {/* ── Step 2: Preferred Name ── */}
          {step === 2 && (
            <>
              <TouchableOpacity style={styles.backTop} onPress={() => setStep(1)}>
                <Text style={[styles.backTopText, { color: colors.textMuted }]}>← Back</Text>
              </TouchableOpacity>

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
            </>
          )}

          {/* ── Step 3: Age Range ── */}
          {step === 3 && (
            <>
              <TouchableOpacity style={styles.backTop} onPress={() => setStep(2)}>
                <Text style={[styles.backTopText, { color: colors.textMuted }]}>← Back</Text>
              </TouchableOpacity>

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

              <TouchableOpacity
                style={[styles.btn, { backgroundColor: colors.brand }]}
                onPress={() => setStep(4)}
                activeOpacity={0.8}
              >
                <Text style={[styles.btnText, { color: colors.textPrimary }]}>
                  {selectedAgeRange ? 'Continue' : 'Skip'}
                </Text>
              </TouchableOpacity>
            </>
          )}

          {/* ── Step 4: Country ── */}
          {step === 4 && (
            <>
              <TouchableOpacity style={styles.backTop} onPress={() => setStep(3)}>
                <Text style={[styles.backTopText, { color: colors.textMuted }]}>← Back</Text>
              </TouchableOpacity>

              <Text style={[styles.heading, { color: colors.textPrimary }]}>Where are you based?</Text>
              <Text style={[styles.sub, { color: colors.textSecondary }]}>
                Used to show local support resources and pricing in your currency.
              </Text>

              <TextInput
                style={[
                  styles.searchInput,
                  {
                    backgroundColor: colors.card,
                    borderColor: colors.border,
                    color: colors.textPrimary,
                  },
                ]}
                placeholder="Search country…"
                placeholderTextColor={colors.textMuted}
                value={countrySearch}
                onChangeText={setCountrySearch}
                returnKeyType="search"
              />

              <View style={styles.countryList}>
                {filteredCountries.map((c) => {
                  const isSelected = selectedCountry === c.code;
                  return (
                    <TouchableOpacity
                      key={c.code}
                      style={[
                        styles.countryCard,
                        {
                          backgroundColor: colors.card,
                          borderColor: isSelected ? colors.brand : colors.border,
                        },
                        isSelected && { backgroundColor: colors.brandGlow },
                      ]}
                      onPress={() => setSelectedCountry(isSelected ? null : c.code)}
                      activeOpacity={0.7}
                    >
                      <Text style={styles.countryFlag}>{c.flag}</Text>
                      <Text style={[styles.countryName, { color: isSelected ? colors.purple300 : colors.textPrimary }]}>
                        {c.name}
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
                    {selectedCountry ? 'Continue' : 'Skip'}
                  </Text>
                )}
              </TouchableOpacity>
            </>
          )}

          {/* ── Step 5: Journal vs Therapy gate ── */}
          {step === 5 && (
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
      )}
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
  },
  // ── Flood fill container (wraps step 1) ──────────────────────────────────────
  floodContainer: {
    flex: 1,
    position: 'relative',
  },
  floodCircle: {
    position: 'absolute',
    width: FLOOD_D,
    height: FLOOD_D,
    borderRadius: FLOOD_R,
    zIndex: 1,
  },
  scrollLayer: {
    flex: 1,
    zIndex: 2,
  },
  // ─────────────────────────────────────────────────────────────────────────────
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
  backTop: {
    marginBottom: 20,
    alignSelf: 'flex-start',
  },
  backTopText: {
    fontFamily: Fonts.sans,
    fontSize: 14,
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
  searchInput: {
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 12,
    fontFamily: Fonts.sans,
    fontSize: 15,
    marginBottom: 16,
  },
  error: {
    fontFamily: Fonts.sans,
    fontSize: 13,
    marginBottom: 16,
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
  countryList: {
    gap: 8,
    marginBottom: 24,
  },
  countryCard: {
    borderWidth: 1,
    borderRadius: 12,
    paddingVertical: 12,
    paddingHorizontal: 16,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  countryFlag: {
    fontSize: 22,
  },
  countryName: {
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
