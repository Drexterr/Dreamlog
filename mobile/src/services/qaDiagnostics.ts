const screens = new Set([
  'onboarding', 'auth', 'record', 'processing', 'reflection', 'followup',
  'therapy', 'share', 'export', 'upgrade', 'journeys', 'settings', 'timeline',
  'mood', 'habits', 'relationships', 'reviews',
]);

function emit(event: string, stage: string) {
  if (process.env.EXPO_PUBLIC_QA_DIAGNOSTICS !== '1') return;
  // Only fixed categories are accepted by the public helpers; never log route IDs or user content.
  console.info(`[ODE_QA] ${JSON.stringify({ event, stage })}`);
}

export function logQaScreen(segments: readonly string[]) {
  const category = segments.find(segment => screens.has(segment));
  emit('screen_entered', category || (segments.includes('(tabs)') ? 'home' : 'other'));
}

export function logQaOnboardingStep(step: number | 'intro') {
  const stages = ['welcome', 'goal', 'reveal', 'name', 'age', 'country', 'mode'];
  emit('onboarding_step_entered', step === 'intro' ? 'intro' : stages[step] || 'unknown');
}
