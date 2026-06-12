import React, { createContext, useContext, useState, useRef, useEffect } from 'react';
import { StyleSheet, Animated, Dimensions, Easing } from 'react-native';
import { THEMES, moodToColor } from '../theme';
import type { ThemeKey } from '../theme';
import { api } from '../api/client';

interface ThemeContextProps {
  theme: ThemeKey;
  colors: typeof THEMES[ThemeKey];
  moodToColor: (score: number) => string;
  setTheme: (theme: ThemeKey) => void;
  /** Expand a flood-fill circle from (tapX, tapY), swap theme when it covers the screen. */
  setThemeWithBubble: (theme: ThemeKey, tapX: number, tapY: number, onComplete?: () => void) => void;
  /** @deprecated No longer needed - overlay is rendered at the provider root. */
  ThemeBubbleOverlay: React.FC;
}

const ThemeContext = createContext<ThemeContextProps | undefined>(undefined);

const { width: SW, height: SH } = Dimensions.get('window');
const BUBBLE_RADIUS = 50;

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<ThemeKey>('curious');
  const [bubbleColor, setBubbleColor] = useState<string>(THEMES.curious.bg);
  const [isAnimating, setIsAnimating] = useState(false);

  const bubbleScale    = useRef(new Animated.Value(0)).current;
  const bubbleOpacity  = useRef(new Animated.Value(0)).current;
  const bubblePosition = useRef({ x: SW / 2, y: SH / 2 });

  // Align theme with persisted goal on mount.
  useEffect(() => {
    (async () => {
      try {
        const user = await api.me();
        if (user.goal && THEMES[user.goal]) {
          setTheme(user.goal as ThemeKey);
          setBubbleColor(THEMES[user.goal as ThemeKey].bg);
        }
      } catch { /* keep default */ }
    })();
  }, []);

  const setThemeWithBubble = (
    newTheme: ThemeKey,
    tapX: number,
    tapY: number,
    onComplete?: () => void,
  ) => {
    if (isAnimating) return;
    setIsAnimating(true);

    const ox = tapX > 0 ? tapX : SW / 2;
    const oy = tapY > 0 ? tapY : SH / 2;

    setBubbleColor(THEMES[newTheme].bg);
    bubblePosition.current = { x: ox, y: oy };

    bubbleScale.setValue(0);
    bubbleOpacity.setValue(1);

    const dx = Math.max(ox, SW - ox);
    const dy = Math.max(oy, SH - oy);
    const targetScale = (Math.sqrt(dx * dx + dy * dy) / BUBBLE_RADIUS) * 1.15;

    Animated.timing(bubbleScale, {
      toValue: targetScale,
      duration: 1250,
      easing: Easing.bezier(0.35, 0.01, 0.08, 1),
      useNativeDriver: true,
    }).start(() => {
      // Circle fully covers the screen - swap the theme while hidden, then
      // fade the circle out to reveal the re-themed UI (same soft reveal as
      // the onboarding goal animation). An instant removal here makes the
      // screen flash one flat color and then snap, which feels broken.
      setTheme(newTheme);
      api.updateMe({ goal: newTheme }).catch(() => {});
      Animated.timing(bubbleOpacity, {
        toValue: 0,
        duration: 450,
        easing: Easing.out(Easing.quad),
        useNativeDriver: true,
      }).start(() => {
        bubbleScale.setValue(0);
        bubbleOpacity.setValue(0);
        setIsAnimating(false);
        onComplete?.();
      });
    });
  };

  const currentColors = THEMES[theme];

  return (
    <ThemeContext.Provider
      value={{
        theme,
        colors: currentColors,
        moodToColor: (score) => moodToColor(score, currentColors),
        setTheme,
        setThemeWithBubble,
        ThemeBubbleOverlay: () => null,
      }}
    >
      {children}

      {/*
        Rendered AFTER children so it sits above EVERYTHING - tab bar, modals,
        navigation headers. This ensures the flood fill covers the full screen
        with no jarring snap when setTheme fires.
      */}
      {isAnimating && (
        <Animated.View
          pointerEvents="none"
          style={[StyleSheet.absoluteFill, styles.overlayContainer]}
        >
          <Animated.View
            style={[
              styles.bubble,
              {
                backgroundColor: bubbleColor,
                left: bubblePosition.current.x - BUBBLE_RADIUS,
                top:  bubblePosition.current.y - BUBBLE_RADIUS,
                width:  BUBBLE_RADIUS * 2,
                height: BUBBLE_RADIUS * 2,
                borderRadius: BUBBLE_RADIUS,
                transform: [{ scale: bubbleScale }],
                opacity: bubbleOpacity,
              },
            ]}
          />
        </Animated.View>
      )}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) throw new Error('useTheme must be used within a ThemeProvider');
  return context;
}

const styles = StyleSheet.create({
  overlayContainer: {
    zIndex: 9999,
  },
  bubble: {
    position: 'absolute',
  },
});
