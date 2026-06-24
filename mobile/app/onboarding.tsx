import { useEffect, useState, useRef, type ComponentProps } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  KeyboardAvoidingView,
  Platform,
  Dimensions,
  Animated,
  Easing,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useRouter } from 'expo-router';
import { Fonts, THEMES } from '../src/theme';
import { useTheme } from '../src/context/ThemeContext';
import type { ThemeColors } from '../src/theme';
import {
  saveGuestPreferences,
  markOnboardingDone,
  markTourPending,
} from '../src/services/guestStorage';
import { setRegionFromCountry } from '../src/services/region';
import type { AgeRange, UserGoal } from '../src/types';

const { width: SW, height: SH } = Dimensions.get('window');
const FLOOD_D = 100;
const FLOOD_R = FLOOD_D / 2;

// ── Intro slides data ─────────────────────────────────────────────────────────

const INTRO_SLIDES = [
  {
    icon: 'mic-outline' as const,
    title: 'Speak, don\'t type',
    body: 'Press record. Say what\'s true.\nYour voice carries everything.',
  },
  {
    icon: 'heart-outline' as const,
    title: 'Heard, not handled',
    body: 'An AI reads your words with care and\nwrites a reflection back — warmth, not advice.',
  },
  {
    icon: 'trending-up-outline' as const,
    title: 'See yourself clearly',
    body: 'Over time, your moods and emotions\nbecome a map. Patterns find you.',
  },
] as const;

// ── Breathing orb shown on each intro slide ────────────────────────────────────

function IntroOrb({ icon, colors }: { icon: ComponentProps<typeof Ionicons>['name']; colors: ThemeColors }) {
  const breathe = useRef(new Animated.Value(1)).current;
  const glow    = useRef(new Animated.Value(0.18)).current;

  useEffect(() => {
    const b = Animated.loop(
      Animated.sequence([
        Animated.timing(breathe, { toValue: 1.07, duration: 2600, easing: Easing.inOut(Easing.sin), useNativeDriver: true }),
        Animated.timing(breathe, { toValue: 1.00, duration: 2600, easing: Easing.inOut(Easing.sin), useNativeDriver: true }),
      ]),
    );
    const g = Animated.loop(
      Animated.sequence([
        Animated.timing(glow, { toValue: 0.32, duration: 2600, easing: Easing.inOut(Easing.sin), useNativeDriver: true }),
        Animated.timing(glow, { toValue: 0.18, duration: 2600, easing: Easing.inOut(Easing.sin), useNativeDriver: true }),
      ]),
    );
    b.start(); g.start();
    return () => { b.stop(); g.stop(); };
  }, []);

  return (
    <Animated.View style={[introOrbStyles.outer, { backgroundColor: colors.brandGlow, transform: [{ scale: breathe }] }]}>
      <Animated.View style={[introOrbStyles.inner, { backgroundColor: colors.brand, opacity: glow }]} />
      <Ionicons name={icon} size={52} color={colors.brand} />
    </Animated.View>
  );
}

const introOrbStyles = StyleSheet.create({
  outer: { width: 156, height: 156, borderRadius: 78, alignItems: 'center', justifyContent: 'center' },
  inner: { position: 'absolute', width: 110, height: 110, borderRadius: 55 },
});

const AGE_RANGES: { key: AgeRange; label: string }[] = [
  { key: 'under_18', label: 'Under 18' },
  { key: '18_24',    label: '18 – 24' },
  { key: '25_34',    label: '25 – 34' },
  { key: '35_44',    label: '35 – 44' },
  { key: '45_plus',  label: '45 or older' },
];

const GOALS: {
  key: UserGoal;
  label: string;
  description: string;
  revealMessage: string;
  revealSub: string;
}[] = [
  {
    key: 'anxiety',
    label: 'Working through anxiety',
    description: 'Worry, uncertainty, restless thoughts',
    revealMessage: 'A place to exhale.',
    revealSub: "You don't have to solve everything tonight. We'll listen.",
  },
  {
    key: 'stress',
    label: 'Managing stress',
    description: 'Overwhelm, pressure, too much on my plate',
    revealMessage: 'A place to slow down.',
    revealSub: 'When everything is loud, your voice still matters.',
  },
  {
    key: 'grief',
    label: 'Processing grief',
    description: "Loss, endings, things that can't be undone",
    revealMessage: 'A place to sit with it.',
    revealSub: 'There is no timeline for this. We hold space, not clocks.',
  },
  {
    key: 'depression',
    label: 'Lifting low mood',
    description: 'Sadness, low motivation, feeling flat',
    revealMessage: 'A place to find light.',
    revealSub: "Even small words matter. We'll help you find yours.",
  },
  {
    key: 'relationships',
    label: 'Understanding relationships',
    description: 'Connection, conflict, how I show up for others',
    revealMessage: 'A place to understand.',
    revealSub: "The patterns are there. We'll help you see them.",
  },
  {
    key: 'career',
    label: 'Career & purpose',
    description: "Work, direction, what I'm building toward",
    revealMessage: 'A place to think out loud.',
    revealSub: 'Your thoughts have more clarity than you think.',
  },
  {
    key: 'trauma',
    label: 'Processing past / trauma',
    description: 'Difficult memories, healing, working through it',
    revealMessage: 'A place to be honest.',
    revealSub: 'At your pace, in your words. Always.',
  },
  {
    key: 'curious',
    label: 'Just exploring',
    description: "No agenda - I'm curious about my inner life",
    revealMessage: 'A place to explore.',
    revealSub: "Curiosity is its own kind of wisdom. We're here for it.",
  },
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

// Step map:
//   0       = welcome
//   'intro' = feature intro slides (new)
//   1       = goal (flood fill)
//   2       = reveal
//   3       = name
//   4       = age range
//   5       = country
//   6       = journal / therapy gate
type Step = 0 | 'intro' | 1 | 2 | 3 | 4 | 5 | 6;

export default function OnboardingScreen() {
  const router = useRouter();
  const { colors, setTheme } = useTheme();

  const [step, setStep] = useState<Step>(0);
  const [selectedGoal, setSelectedGoal] = useState<UserGoal | null>(null);
  const [preferredName, setPreferredName] = useState('');
  const [selectedAgeRange, setSelectedAgeRange] = useState<AgeRange | null>(null);
  const [selectedCountry, setSelectedCountry] = useState<string | null>(null);
  const [countrySearch, setCountrySearch] = useState('');

  // Intro slides state
  const [introSlide, setIntroSlide] = useState(0);
  const introAnim    = useRef(new Animated.Value(0)).current;
  const introStarted = useRef(false);

  // Reveal screen entrance animations (step 2)
  const revealLineAnim  = useRef(new Animated.Value(0)).current;
  const revealLabelAnim = useRef(new Animated.Value(0)).current;
  const revealMsgAnim   = useRef(new Animated.Value(0)).current;
  const revealSubAnim   = useRef(new Animated.Value(0)).current;
  const revealBtnAnim   = useRef(new Animated.Value(0)).current;

  // Flood fill state (step 1 → 2)
  const floodAnim      = useRef(new Animated.Value(0)).current;
  const contentOpacity = useRef(new Animated.Value(1)).current;
  const floodActive    = useRef(false);
  const floodColor     = useRef(colors.bg);
  const floodOriginX   = useRef(SW / 2);
  const floodOriginY   = useRef(SH / 2);

  // Welcome screen entrance
  const welcomeAnim    = useRef(new Animated.Value(0)).current;
  const welcomeStarted = useRef(false);
  if (step === 0 && !welcomeStarted.current) {
    welcomeStarted.current = true;
    Animated.timing(welcomeAnim, {
      toValue: 1,
      duration: 900,
      easing: Easing.out(Easing.cubic),
      useNativeDriver: true,
    }).start();
  }

  // Intro slides entrance — fade + slide-up on first render of this step
  if (step === 'intro' && !introStarted.current) {
    introStarted.current = true;
    introAnim.setValue(0);
    Animated.timing(introAnim, {
      toValue: 1,
      duration: 550,
      easing: Easing.out(Easing.cubic),
      useNativeDriver: true,
    }).start();
  }

  const goalMeta = selectedGoal ? GOALS.find((g) => g.key === selectedGoal) : null;

  const filteredCountries = countrySearch.trim()
    ? COUNTRIES.filter((c) =>
        c.name.toLowerCase().includes(countrySearch.toLowerCase()) ||
        c.code.toLowerCase().includes(countrySearch.toLowerCase()),
      )
    : COUNTRIES;

  // ── Goal tap: flood fill then advance ──────────────────────────────────────
  function handleGoalSelect(goal: UserGoal, pageX: number, pageY: number) {
    if (floodActive.current) return;
    floodActive.current = true;
    setSelectedGoal(goal);

    const targetBg       = THEMES[goal].bg;
    floodColor.current   = targetBg;
    floodOriginX.current = pageX > 0 ? pageX : SW / 2;
    floodOriginY.current = pageY > 0 ? pageY : SH / 2;
    floodAnim.setValue(0);
    contentOpacity.setValue(1);

    const dx = Math.max(floodOriginX.current, SW - floodOriginX.current);
    const dy = Math.max(floodOriginY.current, SH - floodOriginY.current);
    const targetScale = (Math.sqrt(dx * dx + dy * dy) / FLOOD_R) * 1.15;

    Animated.parallel([
      Animated.timing(floodAnim, {
        toValue: targetScale,
        duration: 1250,
        easing: Easing.bezier(0.35, 0.01, 0.08, 1),
        useNativeDriver: true,
      }),
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
      setTheme(goal);
      contentOpacity.setValue(1);
      floodActive.current = false;
      // Reset & play reveal entrance
      [revealLineAnim, revealLabelAnim, revealMsgAnim, revealSubAnim, revealBtnAnim].forEach((a) => a.setValue(0));
      setStep(2);
      Animated.stagger(90, [
        Animated.timing(revealLineAnim,  { toValue: 1, duration: 400, useNativeDriver: false }),
        Animated.timing(revealLabelAnim, { toValue: 1, duration: 320, useNativeDriver: true }),
        Animated.timing(revealMsgAnim,   { toValue: 1, duration: 380, useNativeDriver: true }),
        Animated.timing(revealSubAnim,   { toValue: 1, duration: 340, useNativeDriver: true }),
        Animated.timing(revealBtnAnim,   { toValue: 1, duration: 300, useNativeDriver: true }),
      ]).start();
    });
  }

  // ── Advance intro slides ────────────────────────────────────────────────────
  function advanceIntro() {
    const isLast = introSlide >= INTRO_SLIDES.length - 1;
    Animated.timing(introAnim, {
      toValue: 0,
      duration: 220,
      easing: Easing.in(Easing.quad),
      useNativeDriver: true,
    }).start(() => {
      if (isLast) {
        setStep(1);
      } else {
        setIntroSlide((s) => s + 1);
        Animated.timing(introAnim, {
          toValue: 1,
          duration: 400,
          easing: Easing.out(Easing.cubic),
          useNativeDriver: true,
        }).start();
      }
    });
  }

  // ── Finish: save locally, navigate to tabs as guest ────────────────────────
  async function handleFinish() {
    await saveGuestPreferences({
      ...(selectedGoal                                   ? { goal: selectedGoal }           : {}),
      ...(preferredName.trim()                           ? { name: preferredName.trim() }    : {}),
      ...(selectedAgeRange                               ? { ageRange: selectedAgeRange }    : {}),
      ...(selectedCountry && selectedCountry !== 'OTHER' ? { country: selectedCountry }      : {}),
    });
    if (selectedCountry) {
      setRegionFromCountry(selectedCountry === 'OTHER' ? null : selectedCountry).catch(() => {});
    }
    await markOnboardingDone();
    await markTourPending(); // triggers the in-app guided tour after landing on home
    setStep(6);
  }

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <KeyboardAvoidingView
      style={[styles.root, { backgroundColor: colors.bg }]}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >

      {/* ── Step 0: Welcome ── */}
      {step === 0 && (
        <View style={styles.welcomeWrap}>
          <Animated.View
            style={[
              styles.welcomeInner,
              {
                opacity: welcomeAnim,
                transform: [{ translateY: welcomeAnim.interpolate({ inputRange: [0, 1], outputRange: [24, 0] }) }],
              },
            ]}
          >
            <Text style={[styles.wordmark, { color: colors.purple300 }]}>DreamLog</Text>
            <Text style={[styles.welcomeHeading, { color: colors.textPrimary }]}>
              A private space{'\n'}for honest thought.
            </Text>
            <Text style={[styles.welcomeSub, { color: colors.textSecondary }]}>
              No tracking. No judgment.{'\n'}Just you and the page.
            </Text>
            <TouchableOpacity
              style={[styles.beginBtn, { borderColor: colors.brand }]}
              onPress={() => { introStarted.current = false; setStep('intro'); }}
              activeOpacity={0.8}
            >
              <Text style={[styles.beginBtnText, { color: colors.brand }]}>Begin</Text>
            </TouchableOpacity>
          </Animated.View>
        </View>
      )}

      {/* ── Intro slides ── */}
      {step === 'intro' && (
        <Animated.View
          style={[
            styles.introWrap,
            {
              opacity: introAnim,
              transform: [{ translateY: introAnim.interpolate({ inputRange: [0, 1], outputRange: [28, 0] }) }],
            },
          ]}
        >
          {/* Skip — top right */}
          <TouchableOpacity
            style={styles.introSkip}
            onPress={() => { introAnim.setValue(0); setStep(1); }}
            hitSlop={{ top: 12, bottom: 12, left: 12, right: 12 }}
          >
            <Text style={[styles.introSkipText, { color: colors.textMuted }]}>Skip</Text>
          </TouchableOpacity>

          {/* Centered content */}
          <View style={styles.introContent}>
            {/* Breathing orb — keyed so it remounts (resets animation) per slide */}
            <IntroOrb key={introSlide} icon={INTRO_SLIDES[introSlide].icon} colors={colors} />

            {/* Progress dots */}
            <View style={styles.introDots}>
              {INTRO_SLIDES.map((_, i) => (
                <View
                  key={i}
                  style={[
                    styles.introDot,
                    { backgroundColor: i === introSlide ? colors.brand : colors.border },
                    i === introSlide && styles.introDotActive,
                  ]}
                />
              ))}
            </View>

            {/* Text */}
            <Text style={[styles.introTitle, { color: colors.textPrimary }]}>
              {INTRO_SLIDES[introSlide].title}
            </Text>
            <Text style={[styles.introBody, { color: colors.textSecondary }]}>
              {INTRO_SLIDES[introSlide].body}
            </Text>
          </View>

          {/* Button pinned to bottom */}
          <View style={styles.introBtnWrap}>
            <TouchableOpacity
              style={[styles.introBtn, { backgroundColor: colors.brand }]}
              onPress={advanceIntro}
              activeOpacity={0.8}
            >
              <Text style={[styles.introBtnText, { color: colors.textPrimary }]}>
                {introSlide < INTRO_SLIDES.length - 1 ? 'Continue' : 'Get started'}
              </Text>
            </TouchableOpacity>
          </View>
        </Animated.View>
      )}

      {/* ── Step 1: Goal selection (flood fill) ── */}
      {step === 1 && (
        <View style={styles.floodContainer}>
          <Animated.View
            pointerEvents="none"
            style={[
              styles.floodCircle,
              {
                backgroundColor: floodColor.current,
                left: floodOriginX.current - FLOOD_R,
                top: floodOriginY.current - FLOOD_R,
                transform: [{ scale: floodAnim }],
              },
            ]}
          />
          <Animated.ScrollView
            contentContainerStyle={styles.scroll}
            keyboardShouldPersistTaps="handled"
            style={[styles.scrollLayer, { opacity: contentOpacity }]}
          >
            <Text style={[styles.wordmark, { color: colors.purple300 }]}>DreamLog</Text>
            <TouchableOpacity style={styles.backTop} onPress={() => setStep(0)}>
              <Text style={[styles.backTopText, { color: colors.textMuted }]}>← Back</Text>
            </TouchableOpacity>
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
                      { backgroundColor: colors.card, borderColor: isSelected ? colors.brand : colors.border },
                      isSelected && { backgroundColor: colors.brandGlow },
                    ]}
                    onPress={(e) => handleGoalSelect(g.key, e.nativeEvent.pageX, e.nativeEvent.pageY)}
                    activeOpacity={0.7}
                  >
                    <Text style={[styles.goalLabel, { color: isSelected ? colors.purple300 : colors.textPrimary }]}>
                      {g.label}
                    </Text>
                    <Text style={[styles.goalDesc, { color: colors.textMuted }]}>{g.description}</Text>
                  </TouchableOpacity>
                );
              })}
            </View>
          </Animated.ScrollView>
        </View>
      )}

      {/* ── Step 2: Reveal ── */}
      {step === 2 && goalMeta && (
        <View style={[styles.revealWrap, { backgroundColor: colors.bg }]}>
          <View style={styles.revealContent}>

            {/* Animated top line expanding left → right */}
            <Animated.View
              style={[
                styles.revealTopLine,
                {
                  backgroundColor: colors.brand,
                  width: revealLineAnim.interpolate({ inputRange: [0, 1], outputRange: ['0%', '40%'] }),
                },
              ]}
            />

            {/* Goal label */}
            <Animated.Text
              style={[
                styles.revealLabel,
                {
                  color: colors.textMuted,
                  opacity: revealLabelAnim,
                  transform: [{ translateY: revealLabelAnim.interpolate({ inputRange: [0, 1], outputRange: [6, 0] }) }],
                },
              ]}
            >
              {goalMeta.label.toLowerCase()}
            </Animated.Text>

            {/* Main message */}
            <Animated.Text
              style={[
                styles.revealMessage,
                {
                  color: colors.textPrimary,
                  opacity: revealMsgAnim,
                  transform: [{ translateY: revealMsgAnim.interpolate({ inputRange: [0, 1], outputRange: [10, 0] }) }],
                },
              ]}
            >
              {goalMeta.revealMessage}
            </Animated.Text>

            {/* Thin mid-rule */}
            <View style={[styles.revealMidRule, { backgroundColor: colors.border }]} />

            {/* Sub-text */}
            <Animated.Text
              style={[
                styles.revealSub,
                {
                  color: colors.textSecondary,
                  opacity: revealSubAnim,
                  transform: [{ translateY: revealSubAnim.interpolate({ inputRange: [0, 1], outputRange: [6, 0] }) }],
                },
              ]}
            >
              {goalMeta.revealSub}
            </Animated.Text>

            {/* Continue — right-aligned text link */}
            <Animated.View
              style={[
                styles.revealBtnRow,
                {
                  opacity: revealBtnAnim,
                  transform: [{ translateY: revealBtnAnim.interpolate({ inputRange: [0, 1], outputRange: [4, 0] }) }],
                },
              ]}
            >
              <TouchableOpacity
                style={[styles.revealBtn, { borderColor: colors.brand }]}
                onPress={() => setStep(3)}
                activeOpacity={0.7}
              >
                <Text style={[styles.revealBtnText, { color: colors.brand }]}>Continue →</Text>
              </TouchableOpacity>
            </Animated.View>

          </View>
        </View>
      )}

      {/* ── Steps 3 + 4 + 5 via ScrollView ── */}
      {(step === 3 || step === 4 || step === 5) && (
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <Text style={[styles.wordmark, { color: colors.purple300 }]}>DreamLog</Text>

          {/* ── Step 3: Name ── */}
          {step === 3 && (
            <>
              <TouchableOpacity style={styles.backTop} onPress={() => setStep(2)}>
                <Text style={[styles.backTopText, { color: colors.textMuted }]}>← Back</Text>
              </TouchableOpacity>
              <Text style={[styles.heading, { color: colors.textPrimary }]}>What should I call you?</Text>
              <Text style={[styles.sub, { color: colors.textSecondary }]}>
                Optional — leave blank to use your account name.
              </Text>
              <TextInput
                style={[styles.input, { backgroundColor: colors.card, borderColor: colors.border, color: colors.textPrimary }]}
                placeholder="e.g. Alex, or skip this"
                placeholderTextColor={colors.textMuted}
                value={preferredName}
                onChangeText={setPreferredName}
                maxLength={100}
                autoFocus
                returnKeyType="done"
                onSubmitEditing={() => setStep(4)}
              />
              <TouchableOpacity
                style={[styles.btn, { backgroundColor: colors.brand }]}
                onPress={() => setStep(4)}
                activeOpacity={0.8}
              >
                <Text style={[styles.btnText, { color: colors.textPrimary }]}>Continue</Text>
              </TouchableOpacity>
            </>
          )}

          {/* ── Step 4: Age Range ── */}
          {step === 4 && (
            <>
              <TouchableOpacity style={styles.backTop} onPress={() => setStep(3)}>
                <Text style={[styles.backTopText, { color: colors.textMuted }]}>← Back</Text>
              </TouchableOpacity>
              <Text style={[styles.heading, { color: colors.textPrimary }]}>How old are you?</Text>
              <Text style={[styles.sub, { color: colors.textSecondary }]}>
                Optional — helps us understand who uses DreamLog. Never shared.
              </Text>
              <View style={styles.ageList}>
                {AGE_RANGES.map((a) => {
                  const isSelected = selectedAgeRange === a.key;
                  return (
                    <TouchableOpacity
                      key={a.key}
                      style={[
                        styles.ageCard,
                        { backgroundColor: colors.card, borderColor: isSelected ? colors.brand : colors.border },
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
                onPress={() => setStep(5)}
                activeOpacity={0.8}
              >
                <Text style={[styles.btnText, { color: colors.textPrimary }]}>
                  {selectedAgeRange ? 'Continue' : 'Skip'}
                </Text>
              </TouchableOpacity>
            </>
          )}

          {/* ── Step 5: Country (inline, no separate layout) ── */}
          {step === 5 && (
            <>
              <TouchableOpacity style={styles.backTop} onPress={() => setStep(4)}>
                <Text style={[styles.backTopText, { color: colors.textMuted }]}>← Back</Text>
              </TouchableOpacity>
              <Text style={[styles.heading, { color: colors.textPrimary }]}>Where are you based?</Text>
              <Text style={[styles.sub, { color: colors.textSecondary }]}>
                Used to show local support resources and pricing in your currency.
              </Text>
              <View style={styles.countrySearchWrap}>
                <View style={styles.countryInputRow}>
                  <TextInput
                    style={[
                      styles.countryInput,
                      { backgroundColor: colors.card, borderColor: selectedCountry ? colors.brand : colors.border, color: colors.textPrimary },
                    ]}
                    placeholder="Search country…"
                    placeholderTextColor={colors.textMuted}
                    value={countrySearch}
                    onChangeText={(t) => { setCountrySearch(t); if (selectedCountry) setSelectedCountry(null); }}
                    autoFocus
                    returnKeyType="search"
                  />
                  {selectedCountry ? (
                    <TouchableOpacity style={styles.inputAccessory} onPress={() => { setSelectedCountry(null); setCountrySearch(''); }}>
                      <Text style={[styles.inputAccessoryText, { color: colors.brand }]}>✓</Text>
                    </TouchableOpacity>
                  ) : countrySearch.length > 0 ? (
                    <TouchableOpacity style={styles.inputAccessory} onPress={() => setCountrySearch('')}>
                      <Text style={[styles.inputAccessoryText, { color: colors.textMuted }]}>×</Text>
                    </TouchableOpacity>
                  ) : null}
                </View>
                {!selectedCountry && countrySearch.trim().length > 0 && filteredCountries.length > 0 && (
                  <View style={[styles.dropdown, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}>
                    <ScrollView keyboardShouldPersistTaps="handled" showsVerticalScrollIndicator={false} style={{ maxHeight: 220 }}>
                      {filteredCountries.map((c, i) => (
                        <TouchableOpacity
                          key={c.code}
                          style={[styles.dropdownItem, { borderBottomColor: colors.borderFaint }, i === filteredCountries.length - 1 && { borderBottomWidth: 0 }]}
                          onPress={() => { setSelectedCountry(c.code); setCountrySearch(c.name); }}
                          activeOpacity={0.7}
                        >
                          <Text style={styles.dropdownFlag}>{c.flag}</Text>
                          <Text style={[styles.dropdownName, { color: colors.textPrimary }]}>{c.name}</Text>
                        </TouchableOpacity>
                      ))}
                    </ScrollView>
                  </View>
                )}
                {!selectedCountry && countrySearch.trim().length > 0 && filteredCountries.length === 0 && (
                  <View style={[styles.dropdown, styles.dropdownEmpty, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}>
                    <Text style={[styles.dropdownEmptyText, { color: colors.textMuted }]}>No match — you can skip this step</Text>
                  </View>
                )}
              </View>
              <TouchableOpacity
                style={[styles.btn, { backgroundColor: colors.brand, marginTop: 24 }]}
                onPress={handleFinish}
                activeOpacity={0.8}
              >
                <Text style={[styles.btnText, { color: colors.textPrimary }]}>
                  {selectedCountry ? 'Continue' : 'Skip'}
                </Text>
              </TouchableOpacity>
            </>
          )}
        </ScrollView>
      )}

      {/* ── Step 6: Journal vs Therapy gate ── */}
      {step === 6 && (
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={[styles.wordmark, { color: colors.purple300 }]}>DreamLog</Text>
          <Text style={[styles.heading, { color: colors.textPrimary }]}>How would you like to begin?</Text>
          <Text style={[styles.sub, { color: colors.textSecondary }]}>
            You can always switch later — both live in the app.
          </Text>
          <View style={styles.modeCards}>
            <TouchableOpacity
              style={[styles.modeCard, { backgroundColor: colors.card, borderColor: colors.border }]}
              onPress={() => router.replace('/(tabs)')}
              activeOpacity={0.75}
            >
              <Text style={styles.modeEmoji}>📓</Text>
              <Text style={[styles.modeLabel, { color: colors.textPrimary }]}>Journal</Text>
              <Text style={[styles.modeDesc, { color: colors.textSecondary }]}>
                Record a voice entry and get a personalised AI reflection. At your own pace.
              </Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={[styles.modeCard, { backgroundColor: colors.card, borderColor: colors.border }]}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              onPress={() => router.push('/therapy/persona-picker' as any)}
              activeOpacity={0.75}
            >
              <Text style={[styles.modeLabel, { color: colors.textPrimary }]}>Reflection Session</Text>
              <Text style={[styles.modeDesc, { color: colors.textSecondary }]}>
                A live conversation with an AI companion grounded in your journal history.
              </Text>
              <View style={[styles.modeBadge, { backgroundColor: colors.brandGlow }]}>
                <Text style={[styles.modeBadgeText, { color: colors.brand }]}>First session free</Text>
              </View>
            </TouchableOpacity>
          </View>
        </ScrollView>
      )}

    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },

  // Intro slides
  introWrap: {
    flex: 1,
    paddingHorizontal: 32,
    paddingTop: 56,
    paddingBottom: 40,
  },
  introSkip: {
    position: 'absolute',
    top: 56,
    right: 28,
  },
  introSkipText: {
    fontFamily: Fonts.sans,
    fontSize: 14,
    letterSpacing: 0.2,
  },
  introContent: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'flex-start',
    paddingBottom: 32,
  },
  introDots: {
    flexDirection: 'row',
    gap: 7,
    marginTop: 44,
    marginBottom: 28,
  },
  introDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
  },
  introDotActive: {
    width: 20,
    borderRadius: 3,
  },
  introTitle: {
    fontFamily: Fonts.serif,
    fontSize: 34,
    lineHeight: 42,
    letterSpacing: 0.4,
    marginBottom: 14,
  },
  introBody: {
    fontFamily: Fonts.sans,
    fontSize: 16,
    lineHeight: 26,
  },
  introBtnWrap: {
    paddingBottom: 8,
  },
  introBtn: {
    borderRadius: 14,
    paddingVertical: 17,
    alignItems: 'center',
  },
  introBtnText: {
    fontFamily: Fonts.sansSB,
    fontSize: 16,
    letterSpacing: 0.4,
  },

  // Welcome
  welcomeWrap: {
    flex: 1,
    justifyContent: 'center',
    paddingHorizontal: 32,
    paddingBottom: 64,
  },
  welcomeInner: { alignItems: 'flex-start' },
  welcomeHeading: {
    fontFamily: Fonts.serif,
    fontSize: 36,
    lineHeight: 44,
    marginBottom: 16,
    letterSpacing: 0.5,
  },
  welcomeSub: {
    fontFamily: Fonts.sans,
    fontSize: 16,
    lineHeight: 24,
    marginBottom: 48,
  },
  beginBtn: {
    borderWidth: 1.5,
    borderRadius: 12,
    paddingVertical: 14,
    paddingHorizontal: 40,
  },
  beginBtnText: {
    fontFamily: Fonts.sansSB,
    fontSize: 16,
    letterSpacing: 0.5,
  },

  // Flood fill
  floodContainer: { flex: 1, position: 'relative' },
  floodCircle: {
    position: 'absolute',
    width: FLOOD_D,
    height: FLOOD_D,
    borderRadius: FLOOD_R,
    zIndex: 1,
  },
  scrollLayer: { flex: 1, zIndex: 2 },

  // Reveal — editorial, no glows
  revealWrap: { flex: 1, justifyContent: 'center' },
  revealContent: {
    paddingHorizontal: 32,
    paddingBottom: 64,
  },
  revealTopLine: {
    height: 1.5,
    borderRadius: 1,
    marginBottom: 22,
  },
  revealLabel: {
    fontFamily: Fonts.sans,
    fontSize: 12,
    fontStyle: 'italic',
    marginBottom: 20,
    letterSpacing: 0.2,
  },
  revealMessage: {
    fontFamily: Fonts.serif,
    fontSize: 36,
    lineHeight: 44,
    marginBottom: 24,
    letterSpacing: 0.3,
  },
  revealMidRule: {
    height: 1,
    width: 36,
    borderRadius: 1,
    marginBottom: 20,
  },
  revealSub: {
    fontFamily: Fonts.sans,
    fontSize: 15,
    lineHeight: 24,
    marginBottom: 48,
  },
  revealBtnRow: {
    alignItems: 'flex-end',
  },
  revealBtn: {
    borderWidth: 1,
    borderRadius: 10,
    paddingVertical: 11,
    paddingHorizontal: 24,
  },
  revealBtnText: {
    fontFamily: Fonts.sansSB,
    fontSize: 14,
    letterSpacing: 0.4,
  },

  // Shared scroll layout
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
  backTop: { marginBottom: 20, alignSelf: 'flex-start' },
  backTopText: { fontFamily: Fonts.sans, fontSize: 14 },

  // Goal list
  goalList: { gap: 10, marginBottom: 32 },
  goalCard: { borderWidth: 1, borderRadius: 12, padding: 16 },
  goalLabel: { fontFamily: Fonts.sansSB, fontSize: 15, marginBottom: 3 },
  goalDesc: { fontFamily: Fonts.sans, fontSize: 13, lineHeight: 18 },

  // Button
  btn: { borderRadius: 12, paddingVertical: 16, alignItems: 'center' },
  btnText: { fontFamily: Fonts.sansSB, fontSize: 16 },

  // Input
  input: {
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 14,
    fontFamily: Fonts.sans,
    fontSize: 16,
    marginBottom: 24,
  },

  // Age range
  ageList: { gap: 10, marginBottom: 32 },
  ageCard: { borderWidth: 1, borderRadius: 12, paddingVertical: 14, paddingHorizontal: 16, alignItems: 'center' },
  ageLabel: { fontFamily: Fonts.sansSB, fontSize: 15 },

  // Country
  countrySearchWrap: { position: 'relative', zIndex: 10 },
  countryInputRow: { flexDirection: 'row', alignItems: 'center', position: 'relative' },
  countryInput: {
    flex: 1,
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 14,
    paddingRight: 44,
    fontFamily: Fonts.sans,
    fontSize: 15,
  },
  inputAccessory: { position: 'absolute', right: 14, padding: 4 },
  inputAccessoryText: { fontSize: 18, fontFamily: Fonts.sansSB },
  dropdown: {
    position: 'absolute',
    top: 54,
    left: 0,
    right: 0,
    borderWidth: 1,
    borderRadius: 12,
    overflow: 'hidden',
    zIndex: 20,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 6 },
    shadowOpacity: 0.35,
    shadowRadius: 12,
    elevation: 10,
  },
  dropdownItem: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 12,
    paddingHorizontal: 16,
    borderBottomWidth: 1,
  },
  dropdownFlag: { fontSize: 20 },
  dropdownName: { fontFamily: Fonts.sansSB, fontSize: 14 },
  dropdownEmpty: { paddingVertical: 14, paddingHorizontal: 16 },
  dropdownEmptyText: { fontFamily: Fonts.sans, fontSize: 13 },

  // Journal/Therapy gate
  modeCards: { gap: 14, marginBottom: 16 },
  modeCard: { borderWidth: 1, borderRadius: 16, padding: 20, gap: 6 },
  modeEmoji: { fontSize: 32, marginBottom: 4 },
  modeLabel: { fontFamily: Fonts.sansSB, fontSize: 18, marginBottom: 2 },
  modeDesc: { fontFamily: Fonts.sans, fontSize: 14, lineHeight: 20 },
  modeBadge: { alignSelf: 'flex-start', borderRadius: 6, paddingHorizontal: 8, paddingVertical: 3, marginTop: 8 },
  modeBadgeText: { fontFamily: Fonts.sansSB, fontSize: 12 },
});
