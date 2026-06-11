# DreamLog Testing Strategy

## Current State (updated 2026-06-11)

**600+ unit/integration tests exist** covering Priorities 1–9 below. Coverage by package:

| Package | Coverage | Notes |
|---|---|---|
| `internal/models` | 100% | pure logic |
| `internal/middleware` | ~87% | incl. JWKS/ES256 path; uncovered = live-network branches |
| `internal/handlers` | ~77% | all routes incl. billing verification, therapy, exports |
| `internal/workers` | ~78% | full pipeline; uncovered = long-running `Run()` scheduler loops |
| `internal/services` | ~70% | crisis.go ~100% (CI gate ≥90% ✅); uncovered = thin pgx-backed wrappers + external clients (TTS; FCM credential/OAuth handling unit-tested in `fcm_test.go` since 2026-06-11, live send still external) |
| `pkg/apierr` | 100% | pure logic |
| `internal/repositories`, `pkg/queue`, `pkg/storage` | 0% | need real Postgres/Redis/MinIO — integration tier, see Test Database below |
| `cmd/*`, `internal/config` | 0% | process entry points; exercised by `make dev` smoke run |

The repository/queue/storage tier is intentionally untested at unit level — mocking
the DB is forbidden (see Test Database). Closing that gap requires the
testcontainers-go suite described below, which needs Docker and belongs in CI.

Run all backend tests: `go test ./...`
Run specific package: `go test ./internal/services/...`

---

## Priority 1 - Safety-Critical (Build First)

### Crisis Detection (`internal/services/crisis.go`)

This is the highest-priority test target. A bug here can harm a real user.

```go
// Must cover:
TestCrisisStage1_KeywordMatch          // each of the 20+ phrases triggers stage 1
TestCrisisStage1_NormalText_NoMatch    // benign transcript passes through
TestCrisisStage2_ClaudeConfirmsYes     // ambiguous → Claude says yes → crisis
TestCrisisStage2_ClaudeConfirmsNo      // ambiguous → Claude says no → not crisis
TestCrisisStage2_ClaudeUnreachable     // network error → defaults to crisis (fail-safe)
TestCrisisStage2_ClaudeTimeout         // timeout → defaults to crisis (fail-safe)
TestCrisis_EntryMarkedCorrectly        // is_crisis=true stored in DB
```

Use a mock Claude client for Stage 2 tests. The mock should be injectable via interface.

---

## Priority 2 - Core Pipeline (`internal/workers/transcription.go`)

The entire app's value runs through this file.

```go
TestPipeline_HappyPath                 // transcribe → crisis check → context → claude → analysis stored, status=completed
TestPipeline_TranscriptionFails        // whisper error → status=failed, dead letter job created
TestPipeline_CrisisEntry               // crisis detected → is_crisis=true, reflection NOT generated, status=completed
TestPipeline_ClaudeJSONMalformed       // claude returns invalid JSON → status=failed, retry counted
TestPipeline_MaxRetriesExceeded        // retry_count >= WORKER_MAX_RETRIES → dead letter, no infinite loop
TestPipeline_AudioDeletedAfterSuccess  // storage.Delete called after successful transcription
TestPipeline_AudioNotDeletedOnFailure  // storage.Delete NOT called if transcription fails
```

Use table-driven tests. Mock Whisper, Claude, and storage clients via interfaces.

---

## Priority 3 - Claude Service (`internal/services/claude.go`)

```go
TestAnalyzeEntry_ParsesAllFields       // valid JSON → all 7 fields populated in AnalysisResult
TestAnalyzeEntry_MoodScoreBounds       // mood_score always 1-100
TestAnalyzeEntry_EmotionalToneFormat   // emotional_tone is array of {emotion, intensity}
TestAnalyzeEntry_MalformedJSON         // non-JSON response → returns error, not panic
TestAnalyzeEntry_CrisisJSON            // {"crisis": true} response → returns crisis flag
TestAnalyzeEntry_StubMode              // STUB_AI_ANALYSIS=true → returns valid stub without API call
TestFollowUp_TurnCountEnforced         // 4th turn rejected before calling Claude
TestFollowUp_ContextInjected           // original transcript + reflection in system prompt
```

---

## Priority 4 - HTTP Handlers (Integration Tests)

Use `net/http/httptest` with Gin in test mode. Use a real test PostgreSQL instance (see below) or mock repositories.

```go
TestPresign_ReturnsUploadURL           // POST /entries/presign → 200 with upload_url and audio_key
TestCreateEntry_CreatesRowAndQueuesJob // POST /entries → 201, DB row exists, Redis job pushed
TestGetEntry_NotFound                  // GET /entries/unknown-id → 404
TestGetEntry_WrongUser                 // entry exists but belongs to different user → 404
TestGetAnalysis_EntryPending           // GET /entries/:id/analysis when status=pending → 409
TestGetAnalysis_EntryCompleted         // GET /entries/:id/analysis when completed → 200 with all fields
TestSendMessage_ThirdTurnClosesConvo   // 3rd message → turn_count=3, is_closed=true
TestSendMessage_FourthTurnRejected     // 4th message → 409
TestMoodWeekly_ExcludesCrisis          // crisis entries not counted in avg_mood
TestSearch_FullText                    // GET /entries/search?q=anxiety → matches transcript
TestAuth_InvalidJWT                    // missing or malformed Authorization → 401
TestAuth_ExpiredJWT                    // expired token → 401

// User profile - age_range
TestUpdateMe_AgeRange_Valid            // PUT /me { age_range: "25_34" } → 200, field persisted
TestUpdateMe_AgeRange_InvalidValue     // PUT /me { age_range: "22_30" } → 400
TestUpdateMe_AgeRange_Omitted          // PUT /me without age_range → age_range unchanged (null stays null)
TestGetMe_AgeRange_Returned            // GET /me → age_range present when set, omitted when null
TestUpdateMe_AllFieldsEmpty            // PUT /me with no fields → 400 "at least one field required"
```

---

## Priority 5 - Auth (`internal/services/auth.go`)

```go
TestRegister_HashesPassword            // stored hash != plaintext
TestRegister_DuplicateEmail            // second register with same email → error
TestLogin_CorrectPassword              // returns valid JWT
TestLogin_WrongPassword                // returns error
TestLogin_UnknownEmail                 // returns error (same message as wrong password - no enumeration)
TestJWT_ValidToken                     // minted token passes middleware validation
TestJWT_ExpiredToken                   // expired token rejected
TestJWT_WrongSecret                    // token signed with wrong secret rejected
```

---

## Priority 6 - Context Builder (`internal/services/context_builder.go`)

```go
TestContextBuilder_NewUser             // 0 past entries → empty trends, no panic
TestContextBuilder_FiveEntries         // returns exactly 5 summaries, oldest→newest
TestContextBuilder_ExcludesCrisis      // crisis entries not included in context
TestContextBuilder_EmotionTrend        // correctly aggregates top emotions across entries
TestContextBuilder_TopicTrend          // correctly aggregates top topics across entries
```

---

## Priority 7 - Nudge Scheduler (`internal/workers/nudge_scheduler.go`)

```go
TestNudge_SendsPendingNudges           // nudge with scheduled_at in past → FCM called, status=sent
TestNudge_IgnoresFutureNudges          // scheduled_at in future → FCM not called
TestNudge_MarksSentAfterDispatch       // status updated to sent after FCM success
TestNudge_MarksFailedOnFCMError        // FCM error → status=failed, error_msg set
TestNudge_NoDuplicates                 // same nudge not sent twice
TestNudge_TimezoneAware                // scheduled_at respects user's timezone
```

---

## Test Infrastructure

### Test Database

Use a real PostgreSQL instance for integration tests. Options:
1. **Docker in CI**: `docker run -e POSTGRES_PASSWORD=test -p 5433:5432 postgres:16`
2. **`testcontainers-go`**: spin up a Postgres container per test suite (preferred)

Never mock the database in integration tests - mocked DB tests have failed to catch real bugs.

Pattern for test DB setup:
```go
func setupTestDB(t *testing.T) *pgxpool.Pool {
    // connect to test DB
    // run migrations
    // t.Cleanup(func() { /* truncate tables */ })
}
```

### Mock Interfaces

Key interfaces to define for testing:

```go
// services/claude.go
type AIClient interface {
    AnalyzeEntry(ctx context.Context, input AnalyzeEntryInput) (*AnalysisResult, error)
    SendMessage(ctx context.Context, input ConversationInput) (string, error)
}

// services/transcription.go
type TranscriptionClient interface {
    Transcribe(ctx context.Context, audioURL string) (string, string, error) // transcript, language, error
}

// pkg/storage/s3.go
type StorageClient interface {
    GetPresignedURL(ctx context.Context, key string) (string, error)
    Delete(ctx context.Context, key string) error
}
```

### Running Tests

```bash
# All tests
go test ./...

# With race detector (always use in CI)
go test -race ./...

# Specific package
go test ./internal/services/...

# Verbose output
go test -v ./internal/services/crisis_test.go

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Priority 8 - Therapy Mode (`internal/services/therapy.go`, `internal/handlers/therapy.go`)

### Safety-Critical (same bar as Priority 1)

```go
TestTherapy_CrisisDetectedMidSession          // crisis message → session ends, crisis resources returned, no further messages accepted
TestTherapy_CrisisFailSafe_ClaudeUnreachable  // Claude unreachable during Stage 2 → treated as crisis, session closed
TestTherapy_CrisisFailSafe_Timeout            // Claude timeout during Stage 2 → treated as crisis, session closed
```

### Session Lifecycle

```go
TestTherapy_StartSession_LoadsJournalContext   // POST /therapy/sessions → context_snapshot populated from last 5 entries
TestTherapy_StartSession_NoEntries             // new user with 0 entries → context_loaded=false, session still starts
TestTherapy_ExpiredSession_RejectsMessages     // message after expires_at → 410
TestTherapy_ExpiryEnforcedServerSide           // expires_at derived from started_at, not any client value
TestTherapy_EndSession_GeneratesSummary        // POST /therapy/sessions/:id/end → post_session_summary set
TestTherapy_EndSession_AlreadyEnded            // double-end → 409
TestTherapy_GetSession_ReturnsFullHistory      // GET /therapy/sessions/:id → all messages in order
```

### Message Processing

```go
TestTherapy_VoiceInput_TranscribesAndResponds  // audio_key provided → Whisper called, transcript stored as user message
TestTherapy_TextInput_SkipsWhisper             // content provided → Whisper NOT called
TestTherapy_AudioDeletedAfterTranscription     // storage.Delete called after Whisper succeeds
TestTherapy_AudioNotDeletedOnTranscriptionFail // storage.Delete NOT called if Whisper errors
TestTherapy_WhisperTimeout_Returns504          // Whisper takes >30s → handler returns 504, no message stored
TestTherapy_ClaudeTimeout_Returns504           // Claude takes >60s → handler returns 504, no assistant message stored
TestTherapy_ContextInjectedFromSnapshot        // Claude system prompt contains context_snapshot fields
TestTherapy_HistorySentEachTurn                // full message history sent to Claude on each turn
```

### Billing

```go
TestTherapy_BillingDeducted_OnStart            // session creation deducts therapy_sessions_remaining or charges
TestTherapy_NoBillingCredit_Returns402         // no credits + not Pro → 402, no session created
TestTherapy_ProPlan_AllowsTwoSessions          // Pro user gets 2 free sessions/month
```

### Mock Interfaces for Therapy Tests

```go
// services/therapy.go - add to AIClient interface
type AIClient interface {
    AnalyzeEntry(ctx context.Context, input AnalyzeEntryInput) (*AnalysisResult, error)
    SendMessage(ctx context.Context, input ConversationInput) (string, error)
    TherapyTurn(ctx context.Context, input TherapyTurnInput) (string, error)
    TherapySummary(ctx context.Context, messages []TherapyMessage) (string, error)
}

// services/tts.go
type TTSClient interface {
    Synthesize(ctx context.Context, text string) (audioURL string, err error)
}
```

---

## Priority 9 - Enhanced Therapy Mode (Phase 8)

### Safety-Critical - Layered Crisis (blocking, same bar as Priority 1 and 8)

```go
// Layered crisis - de-escalate first, then hard stop
TestTherapy_LayeredCrisis_FirstDetection_DeEscalates         // first crisis → session stays active, crisis_warnings=1, de-escalation sent
TestTherapy_LayeredCrisis_SecondDetection_HardStop           // second crisis → status=crisis_detected, helpline resources returned, no further messages
TestTherapy_LayeredCrisis_FailSafe_ClaudeUnreachableOnFirst  // Claude unreachable on first detection → treat as hard stop immediately (fail safe)
TestTherapy_LayeredCrisis_FailSafe_TimeoutOnFirst            // Claude timeout on first detection → treat as hard stop immediately (fail safe)
TestTherapy_LayeredCrisis_SessionOpenAfterFirstWarning       // after first crisis warning, session still accepts next message
TestTherapy_LayeredCrisis_SessionClosedAfterSecondWarning    // after second crisis, 409 returned on any subsequent message
```

### Personas

```go
TestTherapy_StartSession_DefaultPersona            // POST /therapy/sessions with no persona field → persona="comforting"
TestTherapy_StartSession_AllPersonasAccepted       // comforting, rational, cbt, mindful all accepted; invalid value → 400
TestTherapy_PersonaStoredInSession                 // persona returned in POST response and GET session response
TestTherapy_PersonaSystemPromptRouting             // each persona routes to correct buildTherapyPersonaSystemPrompt_* function
TestTherapy_PersonaPrompt_Comforting_ToneWords     // comforting prompt contains warmth-oriented language cues
TestTherapy_PersonaPrompt_Rational_ToneWords       // rational prompt contains logic/structure cues
TestTherapy_PersonaPrompt_CBT_ToneWords            // cbt prompt contains thought-pattern language
TestTherapy_PersonaPrompt_Mindful_ToneWords        // mindful prompt contains grounding/present-moment language
```

### Session Continuity

```go
TestTherapy_StartSession_InjectsPastSummaries      // 3 past session summaries fetched and stored in context_snapshot.past_session_summaries
TestTherapy_StartSession_NoPastSessions            // user's first session → has_session_history=false, no past summaries injected, no panic
TestTherapy_StartSession_FewerThanThreeSessions    // user has 1-2 past sessions → those fetched, no error
TestTherapy_PastSummaryInSystemPrompt              // system prompt includes past session summaries section when present
TestTherapy_PastSummariesFromSnapshotNotLive       // past summaries read from context_snapshot, not re-queried on each turn
```

### Wind-Down

```go
TestTherapy_WindDown_PromptIncludesTimeRemaining   // system prompt includes time_remaining_sec on each turn
TestTherapy_WindDown_SubTenMinuteInstructions      // when time_remaining_sec < 600, prompt instructs gentle wind-down
TestTherapy_WindDown_SubTwoMinuteInstructions      // when time_remaining_sec < 120, prompt instructs wrap-up this turn
TestTherapy_WindDown_TimeRemainingAccurate         // time_remaining_sec in session_state = max(0, expires_at - now)
```

---

## Priority 10 - User Profile & UX (`app/(tabs)/settings.tsx`, `app/_layout.tsx`, `app/onboarding.tsx`)

These are mobile-only. Verify manually with Expo dev server; no backend tests required.

### Onboarding - Age Range Step

```
OnboardingAgeRange_StepAppearsAfterName   // step 3 shown after preferred name step
OnboardingAgeRange_SkipButton_WhenNone    // button label is "Skip" when nothing selected
OnboardingAgeRange_ContinueButton_WhenSet // button label is "Continue" when a range selected
OnboardingAgeRange_Deselects_OnRetap      // tapping selected range deselects it (toggle)
OnboardingAgeRange_SavedToDB              // PUT /me called with age_range value on step 3 completion
OnboardingAgeRange_OmittedWhenSkipped     // PUT /me called without age_range when skipped
OnboardingAgeRange_BackNavigatesToName    // Back button returns to step 2 (name)
OnboardingAgeRange_ContinuesToModeGate    // continuing advances to step 4 (Journal/Therapy)
```

### Settings - Profile Card & Modal

```
Settings_ProfileCard_ShowsNameNotEmail    // card sub-text shows "N entries · Plan" not email
Settings_ProfileCard_IsClickable          // tapping card opens profile modal
Settings_ProfileCard_ShowsChevron         // › chevron visible on profile card
Settings_ProfileModal_ShowsEmail          // modal displays user's email address
Settings_ProfileModal_ShowsName           // modal displays user's full name
Settings_ProfileModal_ShowsAgeRange       // modal displays formatted age range when set
Settings_ProfileModal_AgeRange_NotSet     // modal shows "Not set" in muted text when age_range is null
Settings_ProfileModal_ChangeEmail_Alert   // tapping "Change email address" shows support contact alert
Settings_ProfileModal_DismissOnBackdrop   // tapping outside modal closes it
Settings_ProfileModal_DismissOnClose      // tapping Close button closes it
```

### Greeting Splash

```
Greeting_ShowsOnColdStart_AuthedUser      // "Hello, [name]" overlay visible on app open when logged in
Greeting_UsesPreferredName_IfSet          // shows preferred_name over name when both exist
Greeting_FadesIn_ThenFadesOut            // opacity: 0 → 1 → 0 over ~2.2s total
Greeting_NotShown_WhenNeedsOnboarding    // no greeting if user hasn't completed onboarding
Greeting_NotShown_WhenNoToken            // no greeting on auth screen
Greeting_DoesNotBlockInteraction         // overlay gone after animation; tabs are fully interactive
```

### Therapy Index - Hero Redesign (`app/therapy/index.tsx`)

```
TherapyIndex_AmbientGlow_Visible              // pulsating purple glow renders behind hero, not on top of content
TherapyIndex_FeatureChips_Horizontal          // 5 feature pills scroll horizontally without clipping
TherapyIndex_ResumeBanner_HiddenNoSession     // active session banner not rendered when no active session exists
TherapyIndex_ResumeBanner_ShownWhenActive     // banner appears and taps into session when status=active
TherapyIndex_PersonaCards_AllFourVisible      // 4 persona cards scroll horizontally; emoji + name + tagline each visible
TherapyIndex_StatsBar_HiddenFirstVisit        // stats bar not shown when user has no sessions
TherapyIndex_StatsBar_ShowsAfterSession       // stats bar shows total sessions / turns / completed after first session
TherapyIndex_SessionCard_AccentBar_Active     // active session card has brand-color left accent bar
TherapyIndex_SessionCard_AccentBar_Other      // non-active session card has muted accent bar
TherapyIndex_SessionCard_StatusBadge          // status badge visible on each session card
TherapyIndex_StartCTA_NavigatesToPersonaPicker // "Start a Session" CTA → persona picker screen
```

### Therapy Session Screen - Voice-First Redesign (`app/therapy/session.tsx`)

```
TherapySession_OrbVisible_VoiceMode           // SessionOrb visible in center when inputMode=voice
TherapySession_OrbBreathing_WhenIdle          // orb pulses subtly (1.03x scale) when idle, not recording
TherapySession_OrbPulse_WhenRecording         // orb shows strong pulse + red glow when recording=true
TherapySession_OrbSpinner_WhenThinking        // ActivityIndicator shown on orb when waiting for AI response
TherapySession_Waveform_WhenRecording         // 9 animated bars shown when recording
TherapySession_ChatButton_OpensChatMode       // tapping Chat button switches inputMode to text
TherapySession_TextMode_ShowsFullChat         // text mode: full-height message list + text input at bottom
TherapySession_TextMode_BackToVoice           // sending a text message auto-switches back to voice mode
TherapySession_VoiceButton_InTextMode         // "🎙 Voice" header button returns to voice mode without sending
TherapySession_MessageScroll_MaxHeight        // voice mode message scroll limited to ≤38% of screen height
TherapySession_CrisisState_ShowsResources     // crisis state renders hotline cards, no input allowed
TherapySession_EndedState_NoInput             // ended/expired session shows summary, no orb or text input
```

### Therapy Pricing Screen (`app/therapy/pricing.tsx`)

```
TherapyPricing_ShowsLoader_WhileDetectingRegion  // ActivityIndicator shown until detectAndCacheRegion resolves
TherapyPricing_AutoINR_IndiaDevice               // device locale IN → prices shown in ₹ with no toggle
TherapyPricing_AutoUSD_NonIndiaDevice            // non-IN locale → prices shown in $ with no toggle
TherapyPricing_NoManualCurrencyToggle            // currency toggle UI is NOT present on the screen
TherapyPricing_FourOptions_Visible               // Single, 5-Pack, 12-Pack, Pro cards all render
TherapyPricing_PopularBadge_On5Pack              // 5-Pack card shows "POPULAR" badge
TherapyPricing_BestValueBadge_OnPro             // Pro card shows "BEST VALUE" badge
TherapyPricing_SessionPack_NavigatesToPersona    // tapping Single / 5-Pack / 12-Pack CTA → persona picker
TherapyPricing_Pro_NavigatesToUpgrade            // tapping "Get Pro" CTA → upgrade screen
TherapyPricing_PersonaChips_HorizontalScroll     // 4 persona chips scroll horizontally before options
TherapyPricing_EverySessionIncludes_Visible      // info box with 5 included features renders below options
TherapyPricing_SafetyDisclaimer_Visible          // crisis/not-therapy disclaimer text at bottom
```

### Therapy - 402 Credit Redirect

```
TherapyPersonaPicker_402_RedirectsToPricing      // starting session with no credits → navigates to /therapy/pricing
TherapyPersonaPicker_OtherError_ShowsAlert       // non-402 errors still show Alert (not redirected)
```

---

### CI Requirements

Before any PR is merged:
- `go test -race ./...` must pass
- `go vet ./...` must pass
- Crisis detection tests must pass (blocking - no merges if these fail)
- Coverage for `internal/services/crisis.go` must be ≥ 90%
- Therapy mode crisis tests must pass (blocking - same severity as Priority 1)
- Layered crisis tests (Priority 9) must pass (blocking - same severity)
