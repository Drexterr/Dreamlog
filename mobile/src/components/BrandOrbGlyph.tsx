import { useEffect } from 'react';
import Svg, { Path, Circle } from 'react-native-svg';
import Animated, {
  useSharedValue,
  useAnimatedProps,
  useFrameCallback,
  withTiming,
  withDelay,
  Easing,
} from 'react-native-reanimated';

const AnimatedPath = Animated.createAnimatedComponent(Path);
const AnimatedCircle = Animated.createAnimatedComponent(Circle);

// Same master path as BrandSplash (identity spec, viewBox 0 0 192 64): a
// decaying tremor that resolves into a settled dot. Kept in sync by hand -
// there's only the one canonical path, see BrandSplash.tsx's LINE_D.
const STILL_D =
  'M 8 32 C 18 10, 28 54, 38 32 C 47 15, 56 49, 65 32 C 73 20, 81 44, 89 32 ' +
  'C 96 25, 103 39, 110 32 C 117 28.5, 124 35.5, 131 32 C 141 30, 153 34, 164 32';

const POINTS = 20;
const X_STEP = (164 - 8) / POINTS;

// Rough decaying-tremor approximation of STILL_D, sampled at uniform x steps -
// the morph's starting shape, so the crossfade into it reads as a continuation
// of the real logo rather than a jump to an unrelated curve.
function baseY(i: number) {
  'worklet';
  return 32 + 22 * Math.exp(-i / 5) * Math.sin(i * 1.15);
}

// The live "listening" waveform: a continuously trembling line, amplitude
// tapered at both ends so it still reads as one connected stroke.
function waveY(i: number, t: number) {
  'worklet';
  return 32 + Math.sin((t / 1000) * 6 + i * 0.9) * 7 * (0.4 + 0.6 * Math.sin((i / POINTS) * Math.PI));
}

function buildPath(ys: number[]) {
  'worklet';
  let d = `M 8 ${ys[0].toFixed(2)} `;
  let prevX = 8;
  for (let i = 1; i <= POINTS; i++) {
    const x = 8 + i * X_STEP;
    const midX = ((prevX + x) / 2).toFixed(1);
    const midY = ((ys[i - 1] + ys[i]) / 2).toFixed(2);
    d += `Q ${prevX.toFixed(1)} ${ys[i - 1].toFixed(2)}, ${midX} ${midY} `;
    prevX = x;
  }
  d += `L ${prevX.toFixed(1)} ${ys[POINTS].toFixed(2)}`;
  return d;
}

interface Props {
  /** True while actively recording - triggers the morph into the live trembling wave. */
  listening: boolean;
  /** Stroke/fill color - callers pick based on the surface they're drawing on (e.g. a light
   * tone on a filled colored orb, colors.brandCore on a bright amber orb). */
  color: string;
  size?: number;
}

// The Ode "Breath Line" mark as an interactive glyph: sits perfectly still at
// idle (identical to BrandSplash's resting state), and on `listening` smoothly
// morphs - same path, no hard cut - into a continuously trembling waveform,
// reversing the same way once listening stops.
export default function BrandOrbGlyph({ listening, color, size = 100 }: Props) {
  const stillOpacity = useSharedValue(1);
  const morphOpacity = useSharedValue(0);
  const blend = useSharedValue(0);
  const morphD = useSharedValue(STILL_D);
  const dotR = useSharedValue(5);

  useEffect(() => {
    if (listening) {
      stillOpacity.value = withTiming(0, { duration: 150 });
      morphOpacity.value = withTiming(1, { duration: 150 });
      blend.value = withDelay(100, withTiming(1, { duration: 700, easing: Easing.out(Easing.cubic) }));
    } else {
      stillOpacity.value = withTiming(1, { duration: 200 });
      morphOpacity.value = withTiming(0, { duration: 200 });
      blend.value = withTiming(0, { duration: 200 });
    }
  }, [listening]);

  useFrameCallback((frame) => {
    'worklet';
    if (morphOpacity.value < 0.01) return;
    const t = frame.timestamp;
    const b = blend.value;
    const ys: number[] = [];
    for (let i = 0; i <= POINTS; i++) {
      ys.push(baseY(i) * (1 - b) + waveY(i, t) * b);
    }
    morphD.value = buildPath(ys);
    dotR.value = 5 + b * Math.sin(t / 260) * 1.1;
  });

  const stillProps = useAnimatedProps(() => ({ opacity: stillOpacity.value }));
  const morphProps = useAnimatedProps(() => ({ opacity: morphOpacity.value, d: morphD.value }));
  const dotProps = useAnimatedProps(() => ({ r: dotR.value }));

  const height = size * (64 / 192);

  return (
    <Svg width={size} height={height} viewBox="0 0 192 64">
      <AnimatedPath
        d={STILL_D}
        stroke={color}
        strokeWidth={5}
        strokeLinecap="round"
        fill="none"
        animatedProps={stillProps}
      />
      <AnimatedPath stroke={color} strokeWidth={5} strokeLinecap="round" fill="none" animatedProps={morphProps} />
      <AnimatedCircle cx={179} cy={32} fill={color} animatedProps={dotProps} />
    </Svg>
  );
}
