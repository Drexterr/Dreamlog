// DreamLog design tokens — warm espresso

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
    // Soft lavender/muted purple — mourning, spirituality, gentle memory, transition
    bg: '#1C1820',
    bgDeep: '#231E2A',
    card: 'rgba(155, 138, 184, 0.06)',
    cardSolid: '#2A2434',
    textPrimary: '#EDE8F5',
    textSecondary: 'rgba(237, 232, 245, 0.7)',
    textMuted: 'rgba(237, 232, 245, 0.45)',
    textFaint: 'rgba(237, 232, 245, 0.15)',
    border: 'rgba(155, 138, 184, 0.18)',
    borderFaint: 'rgba(155, 138, 184, 0.08)',
    brand: '#9B8AB8',
    brandGlow: 'rgba(155, 138, 184, 0.3)',
    brandCore: '#2A2434',
    moodGreen: '#90B0C0',
    moodYellow: '#C0B0D8',
    moodOrange: '#8870A0',
    moodRed: '#C4A5B0',
    danger: '#C4A5B0',
    info: '#B0A0CC',
    purple900: '#1C1820',
    purple700: '#2A2434',
    purple600: '#9B8AB8',
    purple500: '#9B8AB8',
    purple400: '#9B8AB8',
    purple300: '#C0B0D8',
    purple200: '#EDE8F5',
  },
  depression: {
    // Warm golden saffron — hope, inner warmth, light emerging from darkness
    bg: '#1A160A',
    bgDeep: '#221C0E',
    card: 'rgba(196, 158, 52, 0.06)',
    cardSolid: '#28200F',
    textPrimary: '#F5EDD0',
    textSecondary: 'rgba(245, 237, 208, 0.7)',
    textMuted: 'rgba(245, 237, 208, 0.45)',
    textFaint: 'rgba(245, 237, 208, 0.15)',
    border: 'rgba(196, 158, 52, 0.18)',
    borderFaint: 'rgba(196, 158, 52, 0.08)',
    brand: '#C49E34',
    brandGlow: 'rgba(196, 158, 52, 0.3)',
    brandCore: '#28200F',
    moodGreen: '#A0B848',
    moodYellow: '#E8D070',
    moodOrange: '#C88030',
    moodRed: '#C08080',
    danger: '#C08080',
    info: '#D8C050',
    purple900: '#1A160A',
    purple700: '#28200F',
    purple600: '#C49E34',
    purple500: '#C49E34',
    purple400: '#C49E34',
    purple300: '#E8D070',
    purple200: '#F5EDD0',
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
    // Deep teal — ambition, clarity, focus, professional growth
    bg: '#0C1A1A',
    bgDeep: '#142222',
    card: 'rgba(64, 148, 148, 0.06)',
    cardSolid: '#182C2C',
    textPrimary: '#E0EEF0',
    textSecondary: 'rgba(224, 238, 240, 0.7)',
    textMuted: 'rgba(224, 238, 240, 0.45)',
    textFaint: 'rgba(224, 238, 240, 0.15)',
    border: 'rgba(64, 148, 148, 0.18)',
    borderFaint: 'rgba(64, 148, 148, 0.08)',
    brand: '#409494',
    brandGlow: 'rgba(64, 148, 148, 0.3)',
    brandCore: '#182C2C',
    moodGreen: '#50C0A8',
    moodYellow: '#A0C8B8',
    moodOrange: '#508888',
    moodRed: '#C08888',
    danger: '#C08888',
    info: '#70B8B0',
    purple900: '#0C1A1A',
    purple700: '#182C2C',
    purple600: '#409494',
    purple500: '#409494',
    purple400: '#409494',
    purple300: '#A0C8B8',
    purple200: '#E0EEF0',
  },
  trauma: {
    // Warm terracotta/rust — earth, resilience, grounding, healing foundations
    bg: '#1C1410',
    bgDeep: '#261A14',
    card: 'rgba(184, 108, 72, 0.06)',
    cardSolid: '#2E1E16',
    textPrimary: '#F0E4D8',
    textSecondary: 'rgba(240, 228, 216, 0.7)',
    textMuted: 'rgba(240, 228, 216, 0.45)',
    textFaint: 'rgba(240, 228, 216, 0.15)',
    border: 'rgba(184, 108, 72, 0.18)',
    borderFaint: 'rgba(184, 108, 72, 0.08)',
    brand: '#B86C48',
    brandGlow: 'rgba(184, 108, 72, 0.3)',
    brandCore: '#2E1E16',
    moodGreen: '#90A870',
    moodYellow: '#C8A858',
    moodOrange: '#A86040',
    moodRed: '#C07070',
    danger: '#C07070',
    info: '#C89870',
    purple900: '#1C1410',
    purple700: '#2E1E16',
    purple600: '#B86C48',
    purple500: '#B86C48',
    purple400: '#B86C48',
    purple300: '#C8A858',
    purple200: '#F0E4D8',
  },
  curious: {
    bg:       '#18150f',
    bgDeep:   '#1f1c14',
    card:     'rgba(200,149,90,0.06)',
    cardSolid:'#26221a',
    textPrimary:   '#e8ddd0',
    textSecondary: 'rgba(232,221,208,0.5)',
    textMuted:     'rgba(232,221,208,0.28)',
    textFaint:     'rgba(232,221,208,0.14)',
    border:     'rgba(200,149,90,0.12)',
    borderFaint:'rgba(200,149,90,0.06)',
    brand: '#c8955a',
    brandGlow: 'rgba(200,149,90,0.22)',
    brandCore: '#16120a',
    moodGreen:  '#88b490',
    moodYellow: '#b8a060',
    moodOrange: '#b07868',
    moodRed:    '#c47878',
    danger: '#db8b8b',
    info:   '#88b490',
    purple900: '#18150f',
    purple700: '#26221a',
    purple600: '#c8955a',
    purple500: '#c8955a',
    purple400: '#d4a870',
    purple300: 'rgba(232,221,208,0.5)',
    purple200: '#e8ddd0',
  }
};

// Default palette — warm espresso (curious theme)
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

