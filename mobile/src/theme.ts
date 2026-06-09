// DreamLog design tokens - dark purple night aesthetic

export type ThemeKey = 'stress' | 'anxiety' | 'grief' | 'depression' | 'relationships' | 'career' | 'trauma' | 'curious';

export interface ThemeColors {
  readonly bg: string;
  readonly bgDeep: string;
  readonly card: string;
  readonly cardSolid: string;
  readonly textPrimary: string;
  readonly textSecondary: string;
  readonly textMuted: string;
  readonly textFaint: string;
  readonly border: string;
  readonly borderFaint: string;
  readonly brand: string;
  readonly brandGlow: string;
  readonly brandCore: string;
  readonly moodGreen: string;
  readonly moodYellow: string;
  readonly moodOrange: string;
  readonly moodRed: string;
  readonly danger: string;
  readonly info: string;
  readonly purple900: string;
  readonly purple700: string;
  readonly purple600: string;
  readonly purple500: string;
  readonly purple400: string;
  readonly purple300: string;
  readonly purple200: string;
}

export const THEMES: Record<ThemeKey, ThemeColors> = {
  anxiety: {
    bg: '#1A2B20',
    bgDeep: '#233428',
    card: 'rgba(123, 158, 135, 0.06)',
    cardSolid: '#233428',
    textPrimary: '#F0EDEA',
    textSecondary: 'rgba(240, 237, 234, 0.7)',
    textMuted: 'rgba(240, 237, 234, 0.45)',
    textFaint: 'rgba(240, 237, 234, 0.15)',
    border: 'rgba(123, 158, 135, 0.18)',
    borderFaint: 'rgba(123, 158, 135, 0.08)',
    brand: '#7B9E87',
    brandGlow: 'rgba(123, 158, 135, 0.3)',
    brandCore: '#233428',
    moodGreen: '#90C5A0',
    moodYellow: '#C5D9CE',
    moodOrange: '#7A8F6A',
    moodRed: '#db8b8b',
    danger: '#db8b8b',
    info: '#A8C5B0',
    purple900: '#1A2B20',
    purple700: '#233428',
    purple600: '#7B9E87',
    purple500: '#7B9E87',
    purple400: '#7B9E87',
    purple300: '#A8C5B0',
    purple200: '#C5D9CE',
  },
  stress: {
    bg: '#0F1E2E',
    bgDeep: '#162535',
    card: 'rgba(91, 141, 184, 0.06)',
    cardSolid: '#1A2D40',
    textPrimary: '#E8F0F5',
    textSecondary: 'rgba(232, 240, 245, 0.7)',
    textMuted: 'rgba(232, 240, 245, 0.45)',
    textFaint: 'rgba(232, 240, 245, 0.15)',
    border: 'rgba(91, 141, 184, 0.18)',
    borderFaint: 'rgba(91, 141, 184, 0.08)',
    brand: '#5B8DB8',
    brandGlow: 'rgba(91, 141, 184, 0.3)',
    brandCore: '#1A2D40',
    moodGreen: '#70B8D8',
    moodYellow: '#B0D0E8',
    moodOrange: '#5A7A98',
    moodRed: '#db8b8b',
    danger: '#db8b8b',
    info: '#7AAECC',
    purple900: '#0F1E2E',
    purple700: '#1A2D40',
    purple600: '#5B8DB8',
    purple500: '#5B8DB8',
    purple400: '#5B8DB8',
    purple300: '#7AAECC',
    purple200: '#B0D0E8',
  },
  grief: {
    bg: '#1A2535',
    bgDeep: '#202d3e',
    card: 'rgba(143, 168, 196, 0.06)',
    cardSolid: '#253347',
    textPrimary: '#EDE8F0',
    textSecondary: 'rgba(237, 232, 240, 0.7)',
    textMuted: 'rgba(237, 232, 240, 0.45)',
    textFaint: 'rgba(237, 232, 240, 0.15)',
    border: 'rgba(143, 168, 196, 0.18)',
    borderFaint: 'rgba(143, 168, 196, 0.08)',
    brand: '#8FA8C4',
    brandGlow: 'rgba(143, 168, 196, 0.3)',
    brandCore: '#253347',
    moodGreen: '#A8C0D8',
    moodYellow: '#B5A8C8',
    moodOrange: '#8B7AAB',
    moodRed: '#C4A5A8',
    danger: '#C4A5A8',
    info: '#8FA8C4',
    purple900: '#1A2535',
    purple700: '#253347',
    purple600: '#8FA8C4',
    purple500: '#8FA8C4',
    purple400: '#8FA8C4',
    purple300: '#B5A8C8',
    purple200: '#EDE8F0',
  },
  depression: {
    bg: '#201A12',
    bgDeep: '#292218',
    card: 'rgba(200, 150, 90, 0.06)',
    cardSolid: '#2E2318',
    textPrimary: '#F5EDD8',
    textSecondary: 'rgba(245, 237, 216, 0.7)',
    textMuted: 'rgba(245, 237, 216, 0.45)',
    textFaint: 'rgba(245, 237, 216, 0.15)',
    border: 'rgba(200, 150, 90, 0.18)',
    borderFaint: 'rgba(200, 150, 90, 0.08)',
    brand: '#C8965A',
    brandGlow: 'rgba(200, 150, 90, 0.3)',
    brandCore: '#2E2318',
    moodGreen: '#E8A840',
    moodYellow: '#F0D898',
    moodOrange: '#A07840',
    moodRed: '#db8b8b',
    danger: '#db8b8b',
    info: '#E8C878',
    purple900: '#201A12',
    purple700: '#2E2318',
    purple600: '#C8965A',
    purple500: '#C8965A',
    purple400: '#C8965A',
    purple300: '#E8C878',
    purple200: '#F5EDD8',
  },
  relationships: {
    bg: '#221518',
    bgDeep: '#2d1c20',
    card: 'rgba(200, 127, 127, 0.06)',
    cardSolid: '#32201E',
    textPrimary: '#F5EAE8',
    textSecondary: 'rgba(245, 234, 232, 0.7)',
    textMuted: 'rgba(245, 234, 232, 0.45)',
    textFaint: 'rgba(245, 234, 232, 0.15)',
    border: 'rgba(200, 127, 127, 0.18)',
    borderFaint: 'rgba(200, 127, 127, 0.08)',
    brand: '#C87F7F',
    brandGlow: 'rgba(200, 127, 127, 0.3)',
    brandCore: '#32201E',
    moodGreen: '#E89080',
    moodYellow: '#EED0C8',
    moodOrange: '#A87070',
    moodRed: '#db8b8b',
    danger: '#db8b8b',
    info: '#E0A898',
    purple900: '#221518',
    purple700: '#32201E',
    purple600: '#C87F7F',
    purple500: '#C87F7F',
    purple400: '#C87F7F',
    purple300: '#EED0C8',
    purple200: '#F5EAE8',
  },
  career: {
    bg: '#181D15',
    bgDeep: '#1f261b',
    card: 'rgba(92, 122, 98, 0.06)',
    cardSolid: '#222B1C',
    textPrimary: '#F0EDE8',
    textSecondary: 'rgba(240, 237, 232, 0.7)',
    textMuted: 'rgba(240, 237, 232, 0.45)',
    textFaint: 'rgba(240, 237, 232, 0.15)',
    border: 'rgba(92, 122, 98, 0.18)',
    borderFaint: 'rgba(92, 122, 98, 0.08)',
    brand: '#5C7A62',
    brandGlow: 'rgba(92, 122, 98, 0.3)',
    brandCore: '#222B1C',
    moodGreen: '#70A878',
    moodYellow: '#C0A880',
    moodOrange: '#6A7A58',
    moodRed: '#db8b8b',
    danger: '#db8b8b',
    info: '#8C7055',
    purple900: '#181D15',
    purple700: '#222B1C',
    purple600: '#5C7A62',
    purple500: '#5C7A62',
    purple400: '#5C7A62',
    purple300: '#C0A880',
    purple200: '#F0EDE8',
  },
  trauma: {
    bg: '#1C1A18',
    bgDeep: '#25221f',
    card: 'rgba(160, 144, 128, 0.06)',
    cardSolid: '#282420',
    textPrimary: '#F0E8D8',
    textSecondary: 'rgba(240, 232, 216, 0.7)',
    textMuted: 'rgba(240, 232, 216, 0.45)',
    textFaint: 'rgba(240, 232, 216, 0.15)',
    border: 'rgba(160, 144, 128, 0.18)',
    borderFaint: 'rgba(160, 144, 128, 0.08)',
    brand: '#A09080',
    brandGlow: 'rgba(160, 144, 128, 0.3)',
    brandCore: '#282420',
    moodGreen: '#B8A070',
    moodYellow: '#C8B898',
    moodOrange: '#907868',
    moodRed: '#db8b8b',
    danger: '#db8b8b',
    info: '#C8B898',
    purple900: '#1C1A18',
    purple700: '#282420',
    purple600: '#A09080',
    purple500: '#A09080',
    purple400: '#A09080',
    purple300: '#C8B898',
    purple200: '#F0E8D8',
  },
  curious: {
    bg:       '#0f0c1e',
    bgDeep:   '#130f28',
    card:     'rgba(139,92,246,0.06)',
    cardSolid:'#161625',
    textPrimary:   '#e8e0f0',
    textSecondary: 'rgba(196,181,253,0.6)',
    textMuted:     'rgba(196,181,253,0.35)',
    textFaint:     'rgba(196,181,253,0.2)',
    border:     'rgba(139,92,246,0.15)',
    borderFaint:'rgba(139,92,246,0.08)',
    brand: '#7B6FA0',
    brandGlow: 'rgba(123, 111, 160, 0.3)',
    brandCore: '#221530',
    moodGreen:  '#86efac',
    moodYellow: '#fde68a',
    moodOrange: '#fdba74',
    moodRed:    '#fca5a5',
    danger: '#f87171',
    info:   '#93c5fd',
    purple900: '#4c1d95',
    purple700: '#6d28d9',
    purple600: '#7c3aed',
    purple500: '#8b5cf6',
    purple400: '#a78bfa',
    purple300: '#c4b5fd',
    purple200: '#ddd6fe',
  }
};

// Default fallbacks to purple (curious) theme
export const Colors: ThemeColors = THEMES.curious;

export function moodToColor(score: number, currentColors: ThemeColors = Colors): string {
  if (score >= 71) return currentColors.moodGreen;
  if (score >= 46) return currentColors.moodYellow;
  if (score >= 26) return currentColors.moodOrange;
  return currentColors.moodRed;
}

// Font families - loaded via expo-font in _layout.tsx
export const Fonts = {
  serif:  'CormorantGaramond_300Light',
  sans:   'Nunito_400Regular',
  sansSB: 'Nunito_600SemiBold',
  sansBold:'Nunito_700Bold',
  mono:   undefined as string | undefined, // falls back to system mono
} as const;

