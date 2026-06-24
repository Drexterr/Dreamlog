import { useEffect, useRef } from 'react';
import {
  Animated,
  Dimensions,
  Pressable,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { useTheme } from '../context/ThemeContext';
import { useGuidedTour, TOUR_STEPS } from '../context/GuidedTourContext';

const { width: SW, height: SH } = Dimensions.get('window');
const OVERLAY = 'rgba(10, 7, 24, 0.82)';
const PAD = 10;
const RING_R = 14;
const TOOLTIP_W = SW - 48;

export default function GuidedTour() {
  const { colors } = useTheme();
  const { tourActive, currentStep, measurement, nextStep, skipTour } = useGuidedTour();

  const overlayOpacity = useRef(new Animated.Value(0)).current;
  const tooltipOpacity = useRef(new Animated.Value(0)).current;
  const tooltipY       = useRef(new Animated.Value(8)).current;

  // Animate overlay in/out
  useEffect(() => {
    Animated.timing(overlayOpacity, {
      toValue: tourActive ? 1 : 0,
      duration: 280,
      useNativeDriver: true,
    }).start();
  }, [tourActive]);

  // Animate tooltip whenever step / measurement changes
  useEffect(() => {
    if (!tourActive || !measurement) return;
    tooltipOpacity.setValue(0);
    tooltipY.setValue(8);
    Animated.parallel([
      Animated.timing(tooltipOpacity, { toValue: 1, duration: 260, useNativeDriver: true }),
      Animated.spring(tooltipY, { toValue: 0, tension: 80, friction: 10, useNativeDriver: true }),
    ]).start();
  }, [currentStep, measurement, tourActive]);

  if (!tourActive) return null;

  const step = TOUR_STEPS[currentStep];
  const isLast = currentStep === TOUR_STEPS.length - 1;

  // Spotlight rect (padded)
  const spot = measurement
    ? {
        x: measurement.x - PAD,
        y: measurement.y - PAD,
        w: measurement.width + PAD * 2,
        h: measurement.height + PAD * 2,
      }
    : null;

  // Tooltip positioned above or below spotlight
  const spotBottom = spot ? spot.y + spot.h : SH * 0.6;
  const spaceBelow = SH - spotBottom;
  const tooltipAbove = spaceBelow < 200;
  const tooltipTop = tooltipAbove
    ? (spot ? spot.y - 160 : SH * 0.3)
    : spotBottom + 20;

  return (
    <Animated.View
      style={[StyleSheet.absoluteFillObject, styles.root, { opacity: overlayOpacity }]}
      pointerEvents={tourActive ? 'box-none' : 'none'}
    >
      {/* Touch blocker — swallows all touches except our own buttons */}
      <Pressable style={StyleSheet.absoluteFillObject} onPress={() => {}} />

      {/* 4 dark rectangles creating the spotlight cutout */}
      {spot ? (
        <>
          {/* Top */}
          <View style={[styles.shadow, { top: 0, left: 0, right: 0, height: spot.y }]} />
          {/* Left */}
          <View style={[styles.shadow, { top: spot.y, left: 0, width: spot.x, height: spot.h }]} />
          {/* Right */}
          <View style={[styles.shadow, { top: spot.y, left: spot.x + spot.w, right: 0, height: spot.h }]} />
          {/* Bottom */}
          <View style={[styles.shadow, { top: spot.y + spot.h, left: 0, right: 0, bottom: 0 }]} />
          {/* Spotlight ring */}
          <View
            style={[
              styles.ring,
              {
                top: spot.y,
                left: spot.x,
                width: spot.w,
                height: spot.h,
                borderRadius: RING_R,
                borderColor: colors.purple300 + 'cc',
              },
            ]}
          />
        </>
      ) : (
        // No measurement yet — full dark overlay
        <View style={[StyleSheet.absoluteFillObject, styles.shadow]} />
      )}

      {/* Tooltip */}
      <Animated.View
        style={[
          styles.tooltip,
          {
            top: tooltipTop,
            left: 24,
            width: TOOLTIP_W,
            backgroundColor: colors.cardSolid ?? colors.card,
            borderColor: colors.border,
            opacity: tooltipOpacity,
            transform: [{ translateY: tooltipY }],
          },
        ]}
        pointerEvents="box-none"
      >
        {/* Step indicator */}
        <View style={styles.stepRow}>
          {TOUR_STEPS.map((_, i) => (
            <View
              key={i}
              style={[
                styles.stepDot,
                {
                  backgroundColor: i === currentStep ? colors.purple300 : colors.border,
                  width: i === currentStep ? 16 : 5,
                },
              ]}
            />
          ))}
          <Text style={[styles.stepCount, { color: colors.textMuted }]}>
            {currentStep + 1} / {TOUR_STEPS.length}
          </Text>
        </View>

        {/* Content */}
        <Text style={[styles.title, { color: colors.textPrimary }]}>{step.title}</Text>
        <Text style={[styles.desc, { color: colors.textSecondary }]}>{step.description}</Text>

        {/* Actions */}
        <View style={styles.actions}>
          <TouchableOpacity onPress={skipTour} hitSlop={{ top: 10, bottom: 10, left: 10, right: 10 }}>
            <Text style={[styles.skipText, { color: colors.textMuted }]}>Skip tour</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.nextBtn, { backgroundColor: colors.purple600 }]}
            onPress={nextStep}
            activeOpacity={0.8}
          >
            <Text style={[styles.nextText, { color: colors.purple200 ?? colors.textPrimary }]}>
              {isLast ? 'Done' : 'Next →'}
            </Text>
          </TouchableOpacity>
        </View>
      </Animated.View>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  root: {
    zIndex: 9000,
  },
  shadow: {
    position: 'absolute',
    backgroundColor: OVERLAY,
  },
  ring: {
    position: 'absolute',
    borderWidth: 1.5,
  },
  tooltip: {
    position: 'absolute',
    borderRadius: 18,
    borderWidth: 1,
    padding: 20,
    gap: 10,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.4,
    shadowRadius: 20,
    elevation: 12,
  },
  stepRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 5,
  },
  stepDot: {
    height: 5,
    borderRadius: 3,
  },
  stepCount: {
    marginLeft: 4,
    fontSize: 10,
    fontFamily: 'Nunito_400Regular',
    letterSpacing: 0.5,
  },
  title: {
    fontSize: 20,
    fontFamily: 'CormorantGaramond_500Medium',
    letterSpacing: 0.2,
    lineHeight: 24,
  },
  desc: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 21,
  },
  actions: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: 6,
  },
  skipText: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
  },
  nextBtn: {
    paddingHorizontal: 20,
    paddingVertical: 9,
    borderRadius: 20,
  },
  nextText: {
    fontSize: 13,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 0.3,
  },
});
