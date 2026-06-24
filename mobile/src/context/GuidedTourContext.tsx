import AsyncStorage from '@react-native-async-storage/async-storage';
import {
  createContext,
  useCallback,
  useContext,
  useRef,
  useState,
  type RefObject,
} from 'react';
import type { View } from 'react-native';

export const TOUR_PENDING_KEY = '@dreamlog/tour_pending';
const TOUR_DONE_KEY = '@dreamlog/tour_done';

export interface Measurement {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface TourStep {
  key: string;
  title: string;
  description: string;
}

export const TOUR_STEPS: TourStep[] = [
  {
    key: 'record',
    title: 'Start here',
    description: 'Tap the mic and speak your mind. 30 seconds or 30 minutes — your voice becomes your journal.',
  },
  {
    key: 'week_strip',
    title: 'Your emotional week',
    description: 'Each bar is a day. Record an entry and your mood arc starts building here.',
  },
  {
    key: 'tab_explore',
    title: 'Tools & features',
    description: 'Therapy Mode, Dream Decoder, and Guided Journeys are all one tap away.',
  },
  {
    key: 'tab_mood',
    title: 'See your patterns',
    description: 'Your streak, mood history, and emotional trends over time.',
  },
  {
    key: 'tab_settings',
    title: 'Your space',
    description: 'Profile, plan, and data export — everything personal lives here.',
  },
];

interface GuidedTourContextValue {
  tourActive: boolean;
  currentStep: number;
  measurement: Measurement | null;
  registerRef: (key: string, ref: RefObject<View>) => void;
  nextStep: () => void;
  skipTour: () => void;
  checkAndStartTour: () => void;
}

const Ctx = createContext<GuidedTourContextValue | null>(null);

export function useGuidedTour() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error('useGuidedTour must be inside GuidedTourProvider');
  return ctx;
}

export function GuidedTourProvider({ children }: { children: React.ReactNode }) {
  const [tourActive, setTourActive] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);
  const [measurement, setMeasurement] = useState<Measurement | null>(null);
  const refs = useRef<Record<string, RefObject<View>>>({});
  const measureTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const measureStep = useCallback((stepIndex: number) => {
    const key = TOUR_STEPS[stepIndex]?.key;
    if (!key) return;
    if (measureTimer.current) clearTimeout(measureTimer.current);
    measureTimer.current = setTimeout(() => {
      const ref = refs.current[key];
      if (ref?.current) {
        ref.current.measureInWindow((x, y, w, h) => {
          if (w > 0 && h > 0) {
            setMeasurement({ x, y, width: w, height: h });
          }
        });
      }
    }, 150);
  }, []);

  const startTour = useCallback(() => {
    setCurrentStep(0);
    setMeasurement(null);
    setTourActive(true);
    measureStep(0);
  }, [measureStep]);

  const endTour = useCallback(async () => {
    setTourActive(false);
    setMeasurement(null);
    await AsyncStorage.setItem(TOUR_DONE_KEY, '1');
  }, []);

  const nextStep = useCallback(() => {
    setCurrentStep((s) => {
      const next = s + 1;
      if (next >= TOUR_STEPS.length) {
        endTour();
        return s;
      }
      setMeasurement(null);
      measureStep(next);
      return next;
    });
  }, [endTour, measureStep]);

  const skipTour = useCallback(() => { endTour(); }, [endTour]);

  const registerRef = useCallback((key: string, ref: RefObject<View>) => {
    refs.current[key] = ref;
  }, []);

  const checkAndStartTour = useCallback(async () => {
    try {
      const [pending, done] = await Promise.all([
        AsyncStorage.getItem(TOUR_PENDING_KEY),
        AsyncStorage.getItem(TOUR_DONE_KEY),
      ]);
      if (pending === '1' && done !== '1') {
        await AsyncStorage.removeItem(TOUR_PENDING_KEY);
        setTimeout(startTour, 600);
      }
    } catch {
      // AsyncStorage failures are non-fatal
    }
  }, [startTour]);

  return (
    <Ctx.Provider value={{ tourActive, currentStep, measurement, registerRef, nextStep, skipTour, checkAndStartTour }}>
      {children}
    </Ctx.Provider>
  );
}
