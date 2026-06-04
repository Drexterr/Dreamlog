import React, { createContext, useContext, useState, useRef, useEffect, useCallback } from 'react';
import { StyleSheet, View, Animated, Dimensions, Easing } from 'react-native';
import { THEMES, moodToColor } from '../theme';
import type { ThemeKey } from '../theme';
import { api } from '../api/client';

interface ThemeContextProps {
  theme: ThemeKey;
  colors: typeof THEMES[ThemeKey];
  moodToColor: (score: number) => string;
  setThemeWithBubble: (theme: ThemeKey, tapX: number, tapY: number) => void;
  ThemeBubbleOverlay: React.FC;
}

const ThemeContext = createContext<ThemeContextProps | undefined>(undefined);

const { width: SW, height: SH } = Dimensions.get('window');
const BUBBLE_RADIUS = 50; // Initial radius of the bubble

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<ThemeKey>('curious');
  const [bubbleColor, setBubbleColor] = useState<string>(THEMES.curious.bg);
  
  // Animation values
  const bubbleScale = useRef(new Animated.Value(0)).current;
  const bubbleOpacity = useRef(new Animated.Value(0)).current;
  const bubblePosition = useRef({ x: 0, y: 0 }).current;
  const [isAnimating, setIsAnimating] = useState(false);

  // Fetch current user goal on mount to align active theme
  useEffect(() => {
    (async () => {
      try {
        const user = await api.me();
        if (user.goal && THEMES[user.goal]) {
          setTheme(user.goal as ThemeKey);
          setBubbleColor(THEMES[user.goal as ThemeKey].bg);
        }
      } catch {
        // Fallback to default
      }
    })();
  }, []);

  const setThemeWithBubble = (newTheme: ThemeKey, tapX: number, tapY: number) => {
    if (isAnimating) return;
    setIsAnimating(true);

    setBubbleColor(THEMES[newTheme].bg);
    bubblePosition.x = tapX;
    bubblePosition.y = tapY;

    bubbleScale.setValue(0);
    bubbleOpacity.setValue(1);

    const dx = Math.max(tapX, SW - tapX);
    const dy = Math.max(tapY, SH - tapY);
    const maxDistance = Math.sqrt(dx * dx + dy * dy);
    const targetScale = (maxDistance / BUBBLE_RADIUS) * 1.2;

    Animated.timing(bubbleScale, {
      toValue: targetScale,
      duration: 600,
      easing: Easing.bezier(0.16, 1, 0.3, 1),
      useNativeDriver: true,
    }).start(() => {
      // Bubble has fully covered the screen behind the content.
      // Switch theme now — new bg matches bubble color so it's seamless.
      setTheme(newTheme);
      api.updateMe({ goal: newTheme }).catch(() => {});
      bubbleScale.setValue(0);
      bubbleOpacity.setValue(0);
      setIsAnimating(false);
    });
  };

  const ThemeBubbleOverlay = useCallback(() => {
    if (!isAnimating) return null;

    return (
      <View style={[StyleSheet.absoluteFill, styles.overlay]} pointerEvents="none">
        <Animated.View
          style={[
            styles.bubble,
            {
              backgroundColor: bubbleColor,
              left: bubblePosition.x - BUBBLE_RADIUS,
              top: bubblePosition.y - BUBBLE_RADIUS,
              width: BUBBLE_RADIUS * 2,
              height: BUBBLE_RADIUS * 2,
              borderRadius: BUBBLE_RADIUS,
              transform: [{ scale: bubbleScale }],
              opacity: bubbleOpacity,
            },
          ]}
        />
      </View>
    );
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAnimating, bubbleColor]);

  const currentColors = THEMES[theme];

  return (
    <ThemeContext.Provider
      value={{
        theme,
        colors: currentColors,
        moodToColor: (score) => moodToColor(score, currentColors),
        setThemeWithBubble,
        ThemeBubbleOverlay,
      }}
    >
      {/* Bubble renders BEFORE children so it sits behind all screen content */}
      <ThemeBubbleOverlay />
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

const styles = StyleSheet.create({
  overlay: {
    // No zIndex — rendered before children so it naturally sits behind them
  },
  bubble: {
    position: 'absolute',
  },
});
