import React, { createContext, useContext, useState, useRef, useEffect, useCallback } from 'react';
import { AccessibilityInfo, Animated, Easing, Image, StyleSheet, View, useWindowDimensions } from 'react-native';
import { captureRef, releaseCapture } from 'react-native-view-shot';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { THEMES, moodToColor } from '../theme';
import type { ThemeKey } from '../theme';
import { api } from '../api/client';

const THEME_KEY = 'dreamlog:theme';
export const THEME_TRANSITION_MS = 280;
type ThemeTransitionOptions = { onComplete?: () => void; onApply?: () => void; syncAccount?: boolean };

interface ThemeContextProps {
  theme: ThemeKey;
  colors: typeof THEMES[ThemeKey];
  moodToColor: (score: number) => string;
  setTheme: (theme: ThemeKey) => void;
  setThemeWithTransition: (theme: ThemeKey, options?: ThemeTransitionOptions) => void;
  /** Crossfade the complete interface. Tap coordinates are retained for compatibility. */
  setThemeWithBubble: (theme: ThemeKey, tapX: number, tapY: number, onComplete?: () => void) => void;
  /** @deprecated The transition is rendered at the provider root. */
  ThemeBubbleOverlay: React.FC;
}
type Change = ThemeTransitionOptions & { theme: ThemeKey };
type Transition = Change & { id: number; uri?: string; phase: 'capturing' | 'loading' | 'fading'; committed: boolean };
const ThemeContext = createContext<ThemeContextProps | undefined>(undefined);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<ThemeKey>('curious');
  const [snapshot, setSnapshot] = useState<{ id: number; uri: string } | null>(null);
  const content = useRef<View>(null);
  const opacity = useRef(new Animated.Value(1)).current;
  const current = useRef<ThemeKey>('curious');
  const active = useRef<Transition | null>(null);
  const pending = useRef<Change | null>(null);
  const revision = useRef(0);
  const mounted = useRef(true);
  const reduceMotion = useRef(false);
  const savedGoal = useRef<Promise<unknown>>(Promise.resolve());
  const watchdog = useRef<ReturnType<typeof setTimeout> | null>(null);
  const frame = useRef<number | null>(null);
  const startNext = useRef<(change: Change) => void>(() => {});
  const { width, height } = useWindowDimensions();

  const clearTransition = useCallback(() => {
    if (watchdog.current) clearTimeout(watchdog.current);
    watchdog.current = null;
    if (frame.current !== null) cancelAnimationFrame(frame.current);
    frame.current = null;
    opacity.stopAnimation();
    const previous = active.current;
    active.current = null;
    if (previous?.uri) releaseCapture(previous.uri);
    if (mounted.current) setSnapshot(null);
  }, [opacity]);

  const commit = useCallback((change: Transition) => {
    if (change.committed) return;
    change.committed = true;
    current.current = change.theme;
    setThemeState(change.theme);
    AsyncStorage.setItem(THEME_KEY, change.theme).catch(() => {});
    if (change.syncAccount !== false) {
      savedGoal.current = savedGoal.current
        .then(() => api.updateMe({ goal: change.theme }))
        .catch(() => {});
    }
    change.onApply?.();
  }, []);

  const finish = useCallback((id: number) => {
    const change = active.current;
    if (!change || change.id !== id || !mounted.current) return;
    commit(change);
    clearTransition();
    change.onComplete?.();
    const next = pending.current;
    pending.current = null;
    if (next && mounted.current) startNext.current(next);
  }, [clearTransition, commit]);

  startNext.current = (change) => {
    if (change.theme === current.current) {
      change.onApply?.();
      change.onComplete?.();
      return;
    }
    const transition: Transition = { ...change, id: ++revision.current, phase: 'capturing', committed: false };
    active.current = transition;
    if (reduceMotion.current || !content.current) {
      finish(transition.id);
      return;
    }
    // A failed capture/decode must never leave an old screen covering the app.
    watchdog.current = setTimeout(() => finish(transition.id), 1200);
    captureRef(content, { format: 'png', result: 'tmpfile' }).then(uri => {
      if (!mounted.current || active.current !== transition) {
        releaseCapture(uri);
        return;
      }
      transition.uri = uri;
      transition.phase = 'loading';
      opacity.setValue(1);
      setSnapshot({ id: transition.id, uri });
    }).catch(() => finish(transition.id));
  };

  const setThemeWithTransition = useCallback((next: ThemeKey, options: ThemeTransitionOptions = {}) => {
    if (!THEMES[next]) return;
    revision.current++;
    if (active.current) pending.current = { ...options, theme: next };
    else startNext.current({ ...options, theme: next });
  }, []);

  const setThemeWithBubble = useCallback((next: ThemeKey, _x: number, _y: number, onComplete?: () => void) => {
    setThemeWithTransition(next, { onComplete });
  }, [setThemeWithTransition]);

  const setTheme = useCallback((next: ThemeKey) => {
    if (!THEMES[next]) return;
    revision.current++;
    pending.current = null;
    clearTransition();
    current.current = next;
    setThemeState(next);
    AsyncStorage.setItem(THEME_KEY, next).catch(() => {});
  }, [clearTransition]);

  const reveal = (id: number) => {
    const change = active.current;
    if (!change || change.id !== id || change.phase !== 'loading') return;
    change.phase = 'fading';
    commit(change);
    // Paint the new palette underneath the decoded old-screen image first.
    frame.current = requestAnimationFrame(() => {
      frame.current = requestAnimationFrame(() => {
        if (active.current !== change) return;
        Animated.timing(opacity, {
          toValue: 0, duration: THEME_TRANSITION_MS,
          easing: Easing.bezier(0.22, 1, 0.36, 1), useNativeDriver: true,
        }).start(({ finished }) => { if (finished) finish(id); });
      });
    });
  };

  useEffect(() => {
    mounted.current = true;
    AccessibilityInfo.isReduceMotionEnabled().then(value => {
      if (mounted.current) reduceMotion.current = value;
    }).catch(() => {});
    const subscription = AccessibilityInfo.addEventListener('reduceMotionChanged', value => {
      reduceMotion.current = value;
      if (value && active.current) finish(active.current.id);
    });
    return () => {
      mounted.current = false;
      revision.current++;
      pending.current = null;
      clearTransition();
      subscription.remove();
    };
  }, [clearTransition, finish]);

  useEffect(() => {
    if (active.current) finish(active.current.id);
  }, [width, height, finish]);

  useEffect(() => {
    const initialRevision = revision.current;
    void (async () => {
      try {
        const saved = await AsyncStorage.getItem(THEME_KEY);
        if (!mounted.current || revision.current !== initialRevision) return;
        if (saved && THEMES[saved as ThemeKey]) {
          current.current = saved as ThemeKey;
          setThemeState(saved as ThemeKey);
          return;
        }
      } catch { /* Continue with the account's saved goal. */ }
      try {
        const user = await api.me();
        if (!mounted.current || revision.current !== initialRevision) return;
        if (user.goal && THEMES[user.goal as ThemeKey]) {
          current.current = user.goal as ThemeKey;
          setThemeState(user.goal as ThemeKey);
          AsyncStorage.setItem(THEME_KEY, user.goal).catch(() => {});
        }
      } catch { /* Keep the local default when offline. */ }
    })();
  }, []);

  const colors = THEMES[theme];
  return (
    <ThemeContext.Provider value={{ theme, colors, moodToColor: score => moodToColor(score, colors), setTheme, setThemeWithTransition, setThemeWithBubble, ThemeBubbleOverlay: () => null }}>
      <View style={styles.root}>
        <View ref={content} collapsable={false} style={styles.root}>{children}</View>
        {snapshot && (
          <Animated.View pointerEvents="none" accessibilityElementsHidden importantForAccessibility="no-hide-descendants" style={[StyleSheet.absoluteFill, styles.transition, { opacity }]}>
            <Image key={snapshot.id} testID="theme-transition-snapshot" source={{ uri: snapshot.uri }} style={StyleSheet.absoluteFill} resizeMode="stretch" fadeDuration={0} onLoad={() => reveal(snapshot.id)} onError={() => finish(snapshot.id)} />
          </Animated.View>
        )}
      </View>
    </ThemeContext.Provider>
  );
}
export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) throw new Error('useTheme must be used within a ThemeProvider');
  return context;
}
const styles = StyleSheet.create({ root: { flex: 1 }, transition: { zIndex: 1 } });
