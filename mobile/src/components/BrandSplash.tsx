import { useEffect, useRef } from 'react';
import { AccessibilityInfo, Animated, Easing, StyleSheet } from 'react-native';
import Svg, { Circle, Path } from 'react-native-svg';
import { useTheme } from '../context/ThemeContext';
import { Fonts } from '../theme';

const AnimatedPath = Animated.createAnimatedComponent(Path);
const AnimatedCircle = Animated.createAnimatedComponent(Circle);

// The Breath Line — master path from the identity spec (viewBox 0 0 192 64):
// a decaying tremor that resolves into a settled dot.
const LINE_D =
  'M 8 32 C 18 10, 28 54, 38 32 C 47 15, 56 49, 65 32 C 73 20, 81 44, 89 32 ' +
  'C 96 25, 103 39, 110 32 C 117 28.5, 124 35.5, 131 32 C 141 30, 153 34, 164 32';
// react-native-svg has no pathLength, so the dash trick needs a real length.
// Slight overestimate: the draw completes just before the timing curve does.
const LINE_LEN = 230;
const DOT_R = 5;

interface Props {
  /** Shown as "Hello, {name}" in the closing beat; null = brand mark only. */
  name: string | null;
  onDone: () => void;
}

// Cold-start brand splash: the line draws itself trembling → steadies, the dot
// lands, "ode" rises above it, then (for a returning user) the greeting fades
// in beneath. pointerEvents="none" so it never blocks interaction.
export default function BrandSplash({ name, onDone }: Props) {
  const { colors } = useTheme();
  const dash = useRef(new Animated.Value(LINE_LEN)).current;
  const dotR = useRef(new Animated.Value(0.01)).current;
  const word = useRef(new Animated.Value(0)).current;
  const hello = useRef(new Animated.Value(0)).current;
  const overlay = useRef(new Animated.Value(1)).current;
  const done = useRef(onDone);
  done.current = onDone;

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const reduceMotion = await AccessibilityInfo.isReduceMotionEnabled().catch(() => false);
      if (cancelled) return;

      if (reduceMotion) {
        dash.setValue(0);
        dotR.setValue(DOT_R);
        word.setValue(1);
        hello.setValue(1);
        Animated.sequence([
          Animated.delay(1400),
          Animated.timing(overlay, { toValue: 0, duration: 400, useNativeDriver: true }),
        ]).start(() => done.current());
        return;
      }

      Animated.parallel([
        // SVG props can't ride the native driver
        Animated.timing(dash, {
          toValue: 0,
          duration: 1600,
          easing: Easing.bezier(0.4, 0, 0.25, 1),
          useNativeDriver: false,
        }),
        Animated.sequence([
          Animated.delay(1450),
          Animated.spring(dotR, { toValue: DOT_R, friction: 5, tension: 120, useNativeDriver: false }),
        ]),
        Animated.sequence([
          Animated.delay(1750),
          Animated.timing(word, { toValue: 1, duration: 500, useNativeDriver: true }),
        ]),
        ...(name
          ? [
              Animated.sequence([
                Animated.delay(2250),
                Animated.timing(hello, { toValue: 1, duration: 400, useNativeDriver: true }),
              ]),
            ]
          : []),
        Animated.sequence([
          Animated.delay(name ? 3400 : 2900),
          Animated.timing(overlay, { toValue: 0, duration: 400, useNativeDriver: true }),
        ]),
      ]).start(() => done.current());
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Animated.View
      pointerEvents="none"
      style={[
        StyleSheet.absoluteFillObject,
        {
          backgroundColor: colors.bg,
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 999,
          opacity: overlay,
        },
      ]}
    >
      <Animated.Text
        style={{
          fontFamily: Fonts.serif,
          fontSize: 80,
          color: colors.textPrimary,
          letterSpacing: 0.5,
          opacity: word,
          transform: [{ translateY: word.interpolate({ inputRange: [0, 1], outputRange: [10, 0] }) }],
          marginBottom: 2,
        }}
      >
        ode
      </Animated.Text>
      <Svg width={200} height={67} viewBox="0 0 192 64">
        <AnimatedPath
          d={LINE_D}
          stroke={colors.brand}
          strokeWidth={5}
          strokeLinecap="round"
          fill="none"
          strokeDasharray={`${LINE_LEN}`}
          strokeDashoffset={dash}
        />
        <AnimatedCircle cx={179} cy={32} r={dotR} fill={colors.purple400} />
      </Svg>
      {name ? (
        <Animated.Text
          style={{
            fontFamily: 'HankenGrotesk_300Light',
            fontSize: 13,
            color: colors.textMuted,
            letterSpacing: 0.5,
            opacity: hello,
            marginTop: 28,
          }}
        >
          Hello, {name}
        </Animated.Text>
      ) : null}
    </Animated.View>
  );
}
