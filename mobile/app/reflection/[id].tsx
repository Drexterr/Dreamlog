import { useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  Animated,
  StyleSheet,
  
  StatusBar,
  Linking,
  Share,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { Entry, EntryAnalysis } from '../../src/types';

// ── Crisis hotlines shown in the care card ────────────────────────────────────
const CRISIS_HOTLINES = [
  { name: 'iCall', number: '9152987821', hours: 'Mon–Sat, 8 AM–10 PM', country: 'India', tel: 'tel:9152987821' },
  { name: 'Vandrevala Foundation', number: '1860-2662-345', hours: '24 / 7', country: 'India', tel: 'tel:18602662345' },
  { name: '988 Lifeline', number: '988', hours: '24 / 7', country: 'US', tel: 'tel:988' },
];

// ── Crisis Care Card ──────────────────────────────────────────────────────────
function CrisisCareView({
  fadeAnim,
  slideAnim,
  colors,
  onDone,
}: {
  fadeAnim: Animated.Value;
  slideAnim: Animated.Value;
  colors: any;
  onDone: () => void;
}) {
  const openURL = (url: string) => {
    Linking.canOpenURL(url).then((ok) => {
      if (ok) Linking.openURL(url);
    });
  };

  return (
    <Animated.View style={{ opacity: fadeAnim, transform: [{ translateY: slideAnim }] }}>
      {/* Header */}
      <Text style={[styles.crisisHeader, { color: colors.textPrimary }]}>
        You're not alone.
      </Text>
      <Text style={[styles.crisisSubtext, { color: colors.textSecondary }]}>
        What you're feeling is real - and support is available right now.
        Please reach out to someone who can help.
      </Text>

      {/* Hotline cards */}
      <Text style={[styles.resourceSectionLabel, { color: colors.textMuted }]}>CRISIS HELPLINES</Text>
      <View style={styles.hotlineList}>
        {CRISIS_HOTLINES.map((h) => (
          <TouchableOpacity
            key={h.name}
            style={[styles.hotlineCard, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}
            onPress={() => openURL(h.tel)}
            activeOpacity={0.75}
          >
            <View style={{ flex: 1 }}>
              <Text style={[styles.hotlineName, { color: colors.textPrimary }]}>{h.name}</Text>
              <Text style={[styles.hotlineHours, { color: colors.textMuted }]}>{h.country} · {h.hours}</Text>
            </View>
            <View style={[styles.callPill, { backgroundColor: `${colors.brand}22`, borderColor: `${colors.brand}55` }]}>
              <Text style={[styles.callPillText, { color: colors.brand }]}>📞 {h.number}</Text>
            </View>
          </TouchableOpacity>
        ))}
      </View>

      {/* Professional help */}
      <Text style={[styles.resourceSectionLabel, { color: colors.textMuted }]}>TALK TO A PROFESSIONAL</Text>

      <TouchableOpacity
        style={[styles.therapistBtn, { backgroundColor: colors.brand }]}
        onPress={() => openURL('https://www.practo.com/therapist?utm_source=dreamlog&utm_medium=crisis&utm_campaign=care-bridge')}
        activeOpacity={0.85}
      >
        <Text style={[styles.therapistBtnText, { color: colors.bg }]}>Find a therapist near you</Text>
        <Text style={[styles.therapistBtnSub, { color: colors.bg, opacity: 0.7 }]}>via Practo</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.onlineBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
        onPress={() => openURL('https://yourdost.com/?utm_source=dreamlog&utm_medium=crisis')}
        activeOpacity={0.8}
      >
        <Text style={[styles.onlineBtnText, { color: colors.textPrimary }]}>Talk online - YourDOST</Text>
        <Text style={[styles.onlineBtnSub, { color: colors.textMuted }]}>Anonymous · Available now</Text>
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.onlineBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
        onPress={() => openURL('https://findahelpline.com/?utm_source=dreamlog')}
        activeOpacity={0.8}
      >
        <Text style={[styles.onlineBtnText, { color: colors.textPrimary }]}>Find resources by country</Text>
        <Text style={[styles.onlineBtnSub, { color: colors.textMuted }]}>findahelpline.com · 200+ countries</Text>
      </TouchableOpacity>

      {/* Closing */}
      <Text style={[styles.crisisClosing, { color: colors.textMuted }]}>
        DreamLog will be here when you're ready. Tonight, please reach out.
      </Text>

      {/* Done */}
      <TouchableOpacity
        style={[styles.doneBtn, { borderColor: colors.borderFaint }]}
        onPress={onDone}
        activeOpacity={0.7}
      >
        <Text style={[styles.doneText, { color: colors.textMuted }]}>I'm okay for now</Text>
      </TouchableOpacity>
    </Animated.View>
  );
}

// ── Main Screen ───────────────────────────────────────────────────────────────
function MoodDot({ score, size = 10, colors }: { score: number; size?: number; colors: any }) {
  const { moodToColor } = useTheme();
  return (
    <View style={{ width: size, height: size, borderRadius: size / 2, backgroundColor: moodToColor(score) }} />
  );
}

export default function ReflectionScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { colors, moodToColor } = useTheme();

  const [entry, setEntry] = useState<Entry | null>(null);
  const [analysis, setAnalysis] = useState<EntryAnalysis | null>(null);
  const [loading, setLoading] = useState(true);
  const [reloadKey, setReloadKey] = useState(0);

  const fadeAnim = useRef(new Animated.Value(0)).current;
  const slideAnim = useRef(new Animated.Value(20)).current;

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    // The entry can exist while its analysis isn't available yet (still
    // processing, or processing failed) - fetch both independently so a
    // missing analysis still renders a useful state instead of a blank screen.
    Promise.allSettled([api.getEntry(id), api.getAnalysis(id)])
      .then(([e, a]) => {
        setEntry(e.status === 'fulfilled' ? e.value : null);
        setAnalysis(a.status === 'fulfilled' ? a.value : null);
      })
      .finally(() => {
        setLoading(false);
        fadeAnim.setValue(0);
        slideAnim.setValue(20);
        Animated.parallel([
          Animated.timing(fadeAnim, { toValue: 1, duration: 800, useNativeDriver: true }),
          Animated.timing(slideAnim, { toValue: 0, duration: 600, useNativeDriver: true }),
        ]).start();
      });
  }, [id, reloadKey]);

  const handleDone = () => router.replace('/(tabs)');

  const handleTellMeMore = () => {
    if (id) router.push(`/followup/${id}`);
  };

  const handleShareTherapist = () => {
    if (id) router.push(`/share/${id}` as any);
  };

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor: colors.bg }]}>
        <SafeAreaView style={styles.loadingCenter}>
          <View style={[styles.loadingOrb, { backgroundColor: colors.brand }]} />
          <Text style={[styles.loadingText, { color: colors.textSecondary }]}>
            reflecting on what you said…
          </Text>
        </SafeAreaView>
      </View>
    );
  }

  // No analysis to show - the entry may still be processing, may have failed,
  // or the fetch itself failed. Show a friendly state instead of a blank screen.
  if (!analysis) {
    const failed = entry?.status === 'failed';
    const processing = entry?.status === 'pending' || entry?.status === 'processing';
    return (
      <View style={[styles.container, { backgroundColor: colors.bg }]}>
        <StatusBar barStyle="light-content" />
        <SafeAreaView style={{ flex: 1 }}>
          <Animated.View
            style={[styles.fallbackCenter, { opacity: fadeAnim, transform: [{ translateY: slideAnim }] }]}
          >
            <Text style={[styles.fallbackTitle, { color: colors.textPrimary }]}>
              {failed
                ? "This entry couldn't be processed"
                : processing
                  ? 'Still reflecting on this one…'
                  : 'Reflection unavailable'}
            </Text>
            <Text style={[styles.fallbackText, { color: colors.textSecondary }]}>
              {failed
                ? 'Something went wrong while processing this entry. Your recording was received, but we couldn’t generate a reflection for it.'
                : processing
                  ? 'Your entry is still being transcribed and analyzed. Check back in a moment.'
                  : 'We couldn’t load this reflection. Check your connection and try again.'}
            </Text>
            {!failed && (
              <TouchableOpacity
                style={[styles.tellMoreBtn, { backgroundColor: colors.card, borderColor: colors.border, alignSelf: 'stretch' }]}
                onPress={() => setReloadKey((k) => k + 1)}
                activeOpacity={0.8}
              >
                <Text style={[styles.tellMoreText, { color: colors.brand }]}>Try again</Text>
              </TouchableOpacity>
            )}
            <TouchableOpacity
              style={[styles.goodnightBtn, { borderColor: colors.borderFaint, alignSelf: 'stretch' }]}
              onPress={() => router.back()}
              activeOpacity={0.8}
            >
              <Text style={[styles.goodnightText, { color: colors.textMuted }]}>Go back</Text>
            </TouchableOpacity>
          </Animated.View>
        </SafeAreaView>
      </View>
    );
  }

  const isCrisis = analysis.is_crisis;
  const moodScore = analysis.mood_score ?? 50;
  const paragraphs = analysis.reflection?.split('\n\n').filter(Boolean) ?? ['No reflection available yet.'];

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

          {/* ── Crisis path ─────────────────────────────────────────────── */}
          {isCrisis ? (
            <CrisisCareView
              fadeAnim={fadeAnim}
              slideAnim={slideAnim}
              colors={colors}
              onDone={handleDone}
            />
          ) : (
            /* ── Normal reflection path ─────────────────────────────────── */
            <>
              {/* Header */}
              <Animated.View
                style={[styles.header, { opacity: fadeAnim, transform: [{ translateY: slideAnim }] }]}
              >
                <View style={styles.moodRow}>
                  <MoodDot score={moodScore} colors={colors} />
                  <Text style={[styles.reflectionLabel, { color: colors.textMuted }]}>
                    TONIGHT'S REFLECTION
                  </Text>
                </View>
                {entry && (
                  <Text style={[styles.dateLabel, { color: colors.textMuted }]}>
                    {new Date(entry.created_at).toLocaleDateString('en-US', {
                      weekday: 'long', month: 'long', day: 'numeric',
                    })}
                  </Text>
                )}
              </Animated.View>

              {/* Topic badges */}
              {(analysis?.topics?.length ?? 0) > 0 && (
                <Animated.View style={[styles.topicRow, { opacity: fadeAnim }]}>
                  {analysis!.topics.slice(0, 4).map((t) => (
                    <View key={t} style={[styles.topicBadge, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
                      <Text style={[styles.topicText, { color: colors.textSecondary }]}>{t}</Text>
                    </View>
                  ))}
                </Animated.View>
              )}

              {/* Reflection text */}
              <View style={styles.reflectionCard}>
                {paragraphs.map((para, i) => (
                  <Animated.Text
                    key={i}
                    style={[
                      styles.reflectionText,
                      { color: colors.textPrimary, opacity: fadeAnim, transform: [{ translateY: slideAnim }] },
                    ]}
                  >
                    {para}
                  </Animated.Text>
                ))}

                {/* Emotional tone */}
                {(analysis?.emotional_tone?.length ?? 0) > 0 && (
                  <Animated.View style={[styles.toneRow, { opacity: fadeAnim }]}>
                    {analysis!.emotional_tone.slice(0, 3).map((et) => (
                      <View key={et.emotion} style={styles.toneItem}>
                        <View
                          style={[
                            styles.toneDot,
                            {
                              backgroundColor: moodToColor(moodScore),
                              opacity: et.intensity,
                              transform: [{ scale: 0.6 + et.intensity * 0.6 }],
                            },
                          ]}
                        />
                        <Text style={[styles.toneLabel, { color: colors.textMuted }]}>{et.emotion}</Text>
                      </View>
                    ))}
                  </Animated.View>
                )}
              </View>

              {/* Key quote */}
              {analysis?.key_quotes?.[0] && (
                <Animated.View style={[styles.quoteCard, { borderLeftColor: colors.border, opacity: fadeAnim }]}>
                  <Text style={[styles.quoteText, { color: colors.textSecondary }]}>
                    "{analysis.key_quotes[0]}"
                  </Text>
                </Animated.View>
              )}

              {/* CTAs */}
              <Animated.View style={[styles.ctaWrap, { opacity: fadeAnim }]}>
                <TouchableOpacity
                  style={[styles.tellMoreBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
                  onPress={handleTellMeMore}
                  activeOpacity={0.8}
                >
                  <Text style={[styles.tellMoreText, { color: colors.brand }]}>Tell me more</Text>
                </TouchableOpacity>

                <TouchableOpacity
                  style={[styles.shareTherapistBtn, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}
                  onPress={handleShareTherapist}
                  activeOpacity={0.8}
                >
                  <Text style={[styles.shareTherapistText, { color: colors.textSecondary }]}>
                    Share with my therapist
                  </Text>
                </TouchableOpacity>

                <TouchableOpacity
                  style={[styles.goodnightBtn, { borderColor: colors.borderFaint }]}
                  onPress={handleDone}
                  activeOpacity={0.8}
                >
                  <Text style={[styles.goodnightText, { color: colors.textMuted }]}>I'm here whenever you need me.</Text>
                </TouchableOpacity>
              </Animated.View>
            </>
          )}
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  loadingCenter: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 24 },
  loadingOrb: { width: 48, height: 48, borderRadius: 24, opacity: 0.7 },
  loadingText: { fontSize: 14, fontFamily: 'Nunito_400Regular', letterSpacing: 0.5 },

  scroll: { paddingHorizontal: 24, paddingTop: 40, paddingBottom: 60 },

  fallbackCenter: {
    flex: 1,
    justifyContent: 'center',
    paddingHorizontal: 24,
    gap: 12,
  },
  fallbackTitle: {
    fontFamily: 'CormorantGaramond_300Light',
    fontSize: 28,
    lineHeight: 36,
  },
  fallbackText: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 14,
    lineHeight: 22,
    marginBottom: 12,
  },

  // ── Crisis styles ──────────────────────────────────────────────────────────
  crisisHeader: {
    fontFamily: 'CormorantGaramond_300Light',
    fontSize: 32,
    fontWeight: '300',
    marginBottom: 14,
    lineHeight: 40,
  },
  crisisSubtext: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 15,
    lineHeight: 24,
    marginBottom: 32,
  },
  resourceSectionLabel: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 10,
    letterSpacing: 1.5,
    marginBottom: 10,
  },
  hotlineList: { gap: 8, marginBottom: 28 },
  hotlineCard: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 16,
    borderWidth: 1,
    padding: 16,
    gap: 12,
  },
  hotlineName: { fontFamily: 'Nunito_600SemiBold', fontSize: 14, marginBottom: 2 },
  hotlineHours: { fontFamily: 'Nunito_400Regular', fontSize: 11 },
  callPill: {
    borderRadius: 10,
    borderWidth: 1,
    paddingHorizontal: 10,
    paddingVertical: 6,
  },
  callPillText: { fontFamily: 'Nunito_700Bold', fontSize: 12 },

  therapistBtn: {
    borderRadius: 16,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 10,
  },
  therapistBtnText: { fontFamily: 'Nunito_700Bold', fontSize: 15 },
  therapistBtnSub: { fontFamily: 'Nunito_400Regular', fontSize: 11, marginTop: 2 },

  onlineBtn: {
    borderRadius: 16,
    borderWidth: 1,
    paddingVertical: 14,
    paddingHorizontal: 18,
    marginBottom: 10,
  },
  onlineBtnText: { fontFamily: 'Nunito_600SemiBold', fontSize: 14, marginBottom: 2 },
  onlineBtnSub: { fontFamily: 'Nunito_400Regular', fontSize: 11 },

  crisisClosing: {
    fontFamily: 'CormorantGaramond_400Regular',
    fontStyle: 'italic',
    fontSize: 15,
    lineHeight: 24,
    textAlign: 'center',
    marginTop: 24,
    marginBottom: 28,
  },
  doneBtn: {
    borderRadius: 16,
    borderWidth: 1,
    paddingVertical: 16,
    alignItems: 'center',
  },
  doneText: { fontFamily: 'Nunito_400Regular', fontSize: 15, letterSpacing: 0.5 },

  // ── Normal reflection styles ───────────────────────────────────────────────
  header: { marginBottom: 20 },
  moodRow: { flexDirection: 'row', alignItems: 'center', gap: 10, marginBottom: 8 },
  reflectionLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.5,
  },
  dateLabel: { fontSize: 13, fontFamily: 'Nunito_400Regular', letterSpacing: 0.3 },

  topicRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginBottom: 28 },
  topicBadge: { borderRadius: 12, borderWidth: 1, paddingHorizontal: 12, paddingVertical: 4 },
  topicText: { fontSize: 11, fontFamily: 'Nunito_400Regular' },

  reflectionCard: { marginBottom: 24 },
  reflectionText: {
    fontSize: 17,
    fontFamily: 'CormorantGaramond_300Light',
    lineHeight: 30,
    marginBottom: 20,
    fontWeight: '300',
  },
  toneRow: { flexDirection: 'row', gap: 24, marginTop: 8, marginBottom: 8 },
  toneItem: { alignItems: 'center', gap: 6 },
  toneDot: { width: 10, height: 10, borderRadius: 5 },
  toneLabel: { fontSize: 11, fontFamily: 'Nunito_400Regular' },

  quoteCard: { borderLeftWidth: 2, paddingLeft: 16, marginBottom: 36 },
  quoteText: {
    fontSize: 14,
    fontFamily: 'CormorantGaramond_400Regular',
    fontStyle: 'italic',
    lineHeight: 22,
  },

  ctaWrap: { gap: 10 },
  tellMoreBtn: {
    borderRadius: 16,
    borderWidth: 1,
    paddingVertical: 16,
    alignItems: 'center',
  },
  tellMoreText: { fontSize: 15, fontFamily: 'Nunito_600SemiBold', letterSpacing: 0.5 },

  shareTherapistBtn: {
    borderRadius: 16,
    borderWidth: 1,
    paddingVertical: 14,
    alignItems: 'center',
  },
  shareTherapistText: { fontSize: 14, fontFamily: 'Nunito_400Regular', letterSpacing: 0.3 },

  goodnightBtn: {
    borderRadius: 16,
    borderWidth: 1,
    paddingVertical: 16,
    alignItems: 'center',
  },
  goodnightText: { fontSize: 15, fontFamily: 'Nunito_400Regular', letterSpacing: 0.5 },
});
