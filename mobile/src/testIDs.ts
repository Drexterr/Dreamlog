/**
 * Central registry of stable UI test identifiers.
 *
 * Every value here is applied to a component via `testID` (and, for icon-only
 * or otherwise text-ambiguous controls, `accessibilityLabel`). Automation
 * (Maestro, Detox, etc.) should target these instead of visible copy, so that
 * marketing/wording changes never break the E2E suite.
 *
 * Rules:
 *  - Keep values kebab-cased and screen-prefixed (e.g. `onboarding-goal-stress`).
 *  - Never reuse a value across two distinct elements.
 *  - When a list renders one row per item, suffix the item key
 *    (see `onboardingGoal(goal)` helpers below).
 *
 * On iOS `testID` becomes the accessibilityIdentifier; on Android it becomes the
 * view resource-id. Maestro matches both via `id:`.
 */
export const T = {
  // ── Bottom tab bar (app/(tabs)/_layout.tsx) ──
  tab: {
    home:     'tab-home',
    explore:  'tab-explore',
    mood:     'tab-mood',
    settings: 'tab-settings',
  },

  // ── Onboarding (app/onboarding.tsx) ──
  onboarding: {
    welcome:        'onboarding-welcome',
    begin:          'onboarding-begin',
    introSkip:      'onboarding-intro-skip',
    introNext:      'onboarding-intro-next',      // Continue / Get started
    goalList:       'onboarding-goal-list',
    revealContinue: 'onboarding-reveal-continue',
    nameInput:      'onboarding-name-input',
    nameContinue:   'onboarding-name-continue',
    ageList:        'onboarding-age-list',
    ageContinue:    'onboarding-age-continue',
    countryInput:   'onboarding-country-input',
    countryContinue:'onboarding-country-continue',
    modeJournal:    'onboarding-mode-journal',
    modeTherapy:    'onboarding-mode-therapy',
    back:           'onboarding-back',
  },

  // ── Home tab (app/(tabs)/index.tsx) ──
  home: {
    screen:      'home-screen',
    recordBtn:   'home-record-button',
    lastEntry:   'home-last-entry',
    weekStrip:   'home-week-strip',
  },

  // ── Record screen (app/record.tsx) ──
  record: {
    screen:  'record-screen',
    cancel:  'record-cancel',
    orb:     'record-orb',
    modeGrid:'record-mode-grid',
  },

  // ── AuthSheet (src/components/AuthSheet.tsx) ──
  auth: {
    sheet:        'auth-sheet',
    google:       'auth-google',
    apple:        'auth-apple',
    useEmail:     'auth-use-email',
    tabLogin:     'auth-tab-login',
    tabRegister:  'auth-tab-register',
    nameInput:    'auth-name-input',
    emailInput:   'auth-email-input',
    passwordInput:'auth-password-input',
    submit:       'auth-submit',
    error:        'auth-error',
    later:        'auth-later',
    backToSocial: 'auth-back-to-social',
  },

  // ── Explore tab (app/(tabs)/timeline.tsx) ──
  explore: {
    screen:        'explore-screen',
    entries:       'explore-entries',
    therapy:       'explore-therapy',
    dream:         'explore-dream',
    journeys:      'explore-journeys',
    relationships: 'explore-relationships',
  },

  // ── Settings tab (app/(tabs)/settings.tsx) ──
  settings: {
    screen:        'settings-screen',
    profileCard:   'settings-profile-card',
    upgrade:       'settings-upgrade',
    goalRow:       'settings-goal-row',
    nudgeSwitch:   'settings-nudge-switch',
    nudgeAutoTimeSwitch: 'settings-nudge-auto-time-switch',
    nudgeHourRow:  'settings-nudge-hour-row',
    voiceLangRow:  'settings-voice-language-row',
    exportRow:     'settings-export-row',
    deleteRow:     'settings-delete-row',
    helpRow:       'settings-help-row',
    aboutRow:      'settings-about-row',
    signOut:       'settings-sign-out',
    profileModalClose: 'settings-profile-modal-close',
  },

  // ── Home tab additions (app/(tabs)/index.tsx) ──
  flashback: {
    card: 'home-flashback-card',
  },

  // ── Reflection screen (app/reflection/[id].tsx) ──
  reflection: {
    checkinButton: 'reflection-checkin-button',
  },

  // ── Auth role pill (app/auth.tsx) ──
  authRole: {
    me:        'auth-role-me',
    therapist: 'auth-role-therapist',
  },

  // ── Therapist workspace (app/therapist/*.tsx) ──
  therapistPortal: {
    // register.tsx
    registerNameInput:        'therapist-register-name-input',
    registerEmailInput:       'therapist-register-email-input',
    registerCredentialsInput: 'therapist-register-credentials-input',
    registerConsentCheckbox:  'therapist-register-consent-checkbox',
    registerSubmit:           'therapist-register-submit',
    registerGoToJournal:      'therapist-register-go-to-journal',

    // index.tsx (dashboard)
    dashboardScreen:      'therapist-dashboard-screen',
    dashboardGoToJournal: 'therapist-dashboard-go-to-journal',
    dashboardNewSession:  'therapist-dashboard-new-session',
    dashboardMyClients:   'therapist-dashboard-my-clients',

    // clients.tsx
    clientsScreen:      'therapist-clients-screen',
    clientsAddButton:   'therapist-clients-add-button',
    clientsAddNameInput:'therapist-clients-add-name-input',
    clientsAddSubmit:   'therapist-clients-add-submit',

    // client/[id].tsx
    clientDetailScreen:      'therapist-client-detail-screen',
    clientDetailAddSession:  'therapist-client-detail-add-session',
    clientDetailDelete:      'therapist-client-detail-delete',

    // add-session.tsx
    addSessionPhotoTab:    'therapist-add-session-photo-tab',
    addSessionTypeTab:     'therapist-add-session-type-tab',
    addSessionBulletsInput:'therapist-add-session-bullets-input',
    addSessionSubmit:      'therapist-add-session-submit',

    // session/[id].tsx
    sessionDetailScreen:    'therapist-session-detail-screen',
    sessionSummarizeButton: 'therapist-session-summarize-button',
    sessionAddBullet:       'therapist-session-add-bullet',
    sessionSaveChanges:     'therapist-session-save-changes',
    sessionDelete:          'therapist-session-delete',
  },

  // ── Therapy persona picker (app/therapy/persona-picker.tsx) ──
  persona: {
    screen: 'persona-screen',
    back:   'persona-back',
    start:  'persona-start',
  },

  // ── Therapy session (app/therapy/session.tsx) ──
  therapy: {
    chatButton:  'therapy-chat-button',
    voiceButton: 'therapy-voice-button',
    textInput:   'therapy-text-input',
    sendButton:  'therapy-send-button',
    endButton:   'therapy-end-button',
  },
} as const;

// ── Per-item id builders (lists) ──
export const onboardingGoalID = (goal: string) => `onboarding-goal-${goal}`;
export const onboardingAgeID  = (age: string)  => `onboarding-age-${age}`;
export const recordModeID     = (mode: string) => `record-mode-${mode}`;
export const personaCardID    = (persona: string) => `persona-card-${persona}`;
