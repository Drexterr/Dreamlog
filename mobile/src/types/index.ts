export type EntryStatus = 'pending' | 'processing' | 'completed' | 'failed';
export type EntryMode = 'processing' | 'rant' | 'gratitude' | 'decision';

export interface EmotionalTone {
  emotion: string;
  intensity: number; // 0.0 – 1.0
}

export interface Entry {
  id: string;
  user_id: string;
  audio_key: string;
  audio_size_bytes: number;
  duration_sec: number;
  status: EntryStatus;
  mode: EntryMode;
  transcript?: string;
  language?: string;
  error_msg?: string;
  retry_count: number;
  created_at: string;
  updated_at: string;
}

export interface EntryAnalysis {
  id: string;
  entry_id: string;
  mood_score: number;        // 1-100 - shown only as color, never as number
  emotional_tone: EmotionalTone[];
  topics: string[];
  key_quotes: string[];
  summary: string;
  reflection: string;
  morning_nudge: string;
  is_crisis: boolean;
  created_at: string;
  updated_at: string;
}

export interface PresignResponse {
  upload_url: string;
  audio_key: string;
  expires_in: number;
}

export interface ListEntriesResponse {
  entries: Entry[];
  total: number;
  page: number;
  page_size: number;
  has_more: boolean;
}

export interface TimelineEntry {
  entry: Entry;
  analysis?: EntryAnalysis;
}

export interface TimelineResponse {
  entries: TimelineEntry[];
  total: number;
  page: number;
  page_size: number;
  has_more: boolean;
}

export type UserGoal = 'stress' | 'anxiety' | 'grief' | 'relationships' | 'career' | 'curious' | 'depression' | 'trauma';
export type AgeRange = 'under_18' | '18_24' | '25_34' | '35_44' | '45_plus';

export type Plan = 'free' | 'plus' | 'pro' | 'b2b';

// Therapy TTS voice language. 'auto' follows the language the user speaks each
// turn. Keep in sync with models.SupportedVoiceLanguages on the backend.
export type VoiceLanguage =
  | 'auto' | 'english' | 'hindi' | 'arabic' | 'bengali' | 'chinese' | 'dutch'
  | 'french' | 'german' | 'greek' | 'gujarati' | 'indonesian' | 'italian'
  | 'japanese' | 'kannada' | 'korean' | 'malayalam' | 'marathi' | 'polish'
  | 'portuguese' | 'punjabi' | 'russian' | 'spanish' | 'swedish' | 'tamil'
  | 'telugu' | 'thai' | 'turkish' | 'ukrainian' | 'urdu' | 'vietnamese';

export interface User {
  id: string;
  supabase_id: string;
  email: string;
  name: string;
  preferred_name?: string;
  timezone: string;
  fcm_nudge_hour: number;
  nudge_enabled: boolean;
  goal?: UserGoal;
  age_range?: AgeRange;
  country?: string;            // ISO 3166-1 alpha-2 (e.g. "IN", "US", "DE")
  voice_language: VoiceLanguage;
  streak_freeze_count: number;
  plan: Plan;
  plan_expires_at?: string;    // RFC3339, null if no expiry
  is_deleted: boolean;
  first_joined_at?: string;    // RFC3339 - original registration date
  reregistered_at?: string;    // RFC3339 - set if account was deleted and re-registered
  reregistration_count: number;
  created_at: string;
  updated_at: string;
}

// ── Phase 4d: Shareable Insight Cards ────────────────────────────────────────

// ── Pattern Radar ────────────────────────────────────────────────────────────

export interface EmotionPattern {
  emotion: string;
  frequency: number;      // count of entries where this appeared
  avg_intensity: number;  // 0.0 – 1.0
  score: number;          // normalized 0.0 – 1.0 for radar axis
}

export interface MoodDistribution {
  high: number;     // mood_score >= 70
  neutral: number;  // 40–69
  low: number;      // < 40
}

export interface PatternRadarResponse {
  range: '30d' | '90d' | '365d';
  emotions: EmotionPattern[];
  total_entries: number;
  mood_distribution: MoodDistribution;
}

// ── (original 4d below) ───────────────────────────────────────────────────────

export interface InsightCardData {
  week_label: string;    // e.g. "May 26 – Jun 1"
  week_start: string;    // YYYY-MM-DD (Monday)
  mood_arc: MoodArcDay[];
  top_emotions: string[];
  streak: number;
  entry_count: number;
  share_count: number;
}

export interface InsightShareResult {
  total_shares: number;
  week_start: string;
}

export interface ConversationMessage {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

export interface Conversation {
  id: string;
  entry_id: string;
  user_id: string;
  turn_count: number;
  is_closed: boolean;
  messages: ConversationMessage[];
  created_at: string;
  updated_at: string;
}

export interface DailyMood {
  day: string;        // YYYY-MM-DD
  avg_mood: number;
  entry_count: number;
}

export interface WeeklyMoodResponse {
  days: DailyMood[];
}

export interface StreakInfo {
  current_streak: number;
  longest_streak: number;
  total_days: number;
  next_milestone: number;  // 0 when all milestones reached
  freeze_count: number;    // available streak freezes
}

export type WeeklyReviewStatus = 'pending' | 'completed' | 'failed';

export interface MoodArcDay {
  date: string;       // YYYY-MM-DD
  avg_mood: number;
}

export interface WeeklyReview {
  id: string;
  user_id: string;
  week_start: string;  // date string
  narrative: string;
  top_emotions: string[];
  mood_arc: MoodArcDay[];
  entry_count: number;
  status: WeeklyReviewStatus;
  scheduled_at: string;
  generated_at?: string;
  created_at: string;
}

export interface WeeklyReviewListResponse {
  reviews: WeeklyReview[];
}

export interface MoodHistoryResponse {
  days: DailyMood[];
  range: '30d' | '90d' | '365d';
  avg_mood: number | null;
  prev_avg_mood: number | null;
  mood_delta: number | null;
  top_emotions: string[];
  entry_count: number;
}

// ── Phase 5a: Therapist share links ──────────────────────────────────────────

export interface ShareLink {
  id: string;
  token: string;
  url: string;
  expires_at: string; // RFC3339
}

export interface CreateShareLinkResult {
  token: string;
  passcode: string; // plaintext, shown once
  url: string;
  expires_at: string;
}

export interface ShareLinksResponse {
  links: ShareLink[];
}

// ── Year in Review ────────────────────────────────────────────────────────────

export type AnnualReviewStatus = 'pending' | 'completed' | 'failed';

export interface MonthlyMoodArcDay {
  month: string;       // YYYY-MM
  avg_mood: number;    // 1-100
  entry_count: number;
}

export interface AnnualReview {
  id: string;
  user_id: string;
  year: number;
  narrative: string;
  top_emotions: string[];
  top_topics: string[];
  mood_arc: MonthlyMoodArcDay[];
  entry_count: number;
  avg_mood: number | null;
  status: AnnualReviewStatus;
  scheduled_at: string;
  generated_at?: string;
  created_at: string;
}

export interface AnnualReviewListResponse {
  reviews: AnnualReview[];
}

// ── Guided Journeys ───────────────────────────────────────────────────────────

export interface JourneyTemplate {
  id: string;
  title: string;
  description: string;
  step_count: number;
  estimated_minutes: number;
  tags: string[];
  prompts: string[];
}

export type JourneySessionStatus = 'in_progress' | 'completed';

export interface JourneyStep {
  step_index: number;
  prompt: string;
  entry_id?: string;
  completed: boolean;
}

export interface JourneySession {
  id: string;
  user_id: string;
  journey_id: string;
  journey_title: string;
  current_step: number;
  total_steps: number;
  status: JourneySessionStatus;
  steps: JourneyStep[];
  created_at: string;
  updated_at: string;
}

export interface JourneyListResponse {
  journeys: JourneyTemplate[];
}

export interface JourneySessionsResponse {
  sessions: JourneySession[];
}

export interface OfflineQueueItem {
  id: string;
  localUri: string;
  durationSec: number;
  sizeBytes: number;
  createdAt: string;
  attempts: number;
  mode?: EntryMode; // entry mode at record time; defaults to 'processing' when absent
}

// ── Phase 6 + 8: Therapy Mode ─────────────────────────────────────────────────

export type TherapySessionStatus = 'active' | 'completed' | 'expired' | 'crisis_detected';

export type TherapyPersona = 'comforting' | 'rational' | 'cbt' | 'mindful';

export const PERSONA_META: Record<TherapyPersona, { label: string; tagline: string; emoji: string }> = {
  comforting: { label: 'Comforting', tagline: 'Warm, gentle, feelings-first', emoji: '🌿' },
  rational:   { label: 'Rational',   tagline: 'Structured, Socratic, clear-headed', emoji: '🧭' },
  cbt:        { label: 'CBT-Style',  tagline: 'Notices thought patterns, offers reframes', emoji: '🔍' },
  mindful:    { label: 'Mindful',    tagline: 'Grounding, present-moment, breath-aware', emoji: '🍃' },
};

export interface TherapyContextSnapshot {
  mood_avg_30d: number | null;
  top_emotions: string[];
  top_topics: string[];
  recent_summaries: string[];
  past_session_summaries: string[];
}

export interface TherapySessionMessage {
  id: string;
  session_id: string;
  role: 'user' | 'assistant';
  content: string;
  input_mode: 'voice' | 'text' | 'system';
  tts_url?: string; // presigned GET URL to AI voice audio; present only on assistant messages when TTS is enabled
  created_at: string;
}

export interface TherapySessionState {
  status: TherapySessionStatus;
  turn_count: number;
  time_remaining_sec: number;
  is_crisis: boolean;
  crisis_warnings: number;
}

export interface TherapySession {
  id: string;
  user_id: string;
  status: TherapySessionStatus;
  persona: TherapyPersona;
  started_at: string;
  expires_at: string;
  ended_at?: string;
  duration_sec?: number;
  turn_count: number;
  crisis_warnings: number;
  context_snapshot: TherapyContextSnapshot;
  post_session_summary?: string;
  billing_amount_paise: number;
  time_remaining_sec: number;
  messages?: TherapySessionMessage[];
  created_at: string;
}

export interface TherapySessionSummary {
  id: string;
  status: TherapySessionStatus;
  persona?: TherapyPersona;
  started_at: string;
  ended_at?: string;
  duration_sec?: number;
  turn_count: number;
  post_session_summary?: string;
}

export interface StartSessionResponse {
  id: string;
  status: string;
  persona: TherapyPersona;
  started_at: string;
  expires_at: string;
  context_loaded: boolean;
  has_session_history: boolean;
  billing_amount_paise: number;
}

export interface TherapyPresignResponse {
  upload_url: string;
  audio_key: string;
}

export interface SendTherapyMessageResponse {
  user_message: TherapySessionMessage;
  assistant_message: TherapySessionMessage;
  session_state: TherapySessionState;
}

export interface EndSessionResponse {
  session_id: string;
  status: string;
  duration_sec: number;
  turn_count: number;
  post_session_summary: string;
}

export interface ListTherapySessionsResponse {
  sessions: TherapySessionSummary[];
}

// ── Billing / Subscription ────────────────────────────────────────────────────

export interface PlanLimits {
  plan: Plan;
  monthly_entries: number;       // -1 = unlimited
  monthly_shares: number;        // -1 = unlimited; 0 = not allowed
  has_pdf_export: boolean;
  has_weekly_review: boolean;
  has_mood_history: boolean;
  has_hindi: boolean;
  has_all_modes: boolean;
  has_streak_freeze: boolean;
  has_therapist_share: boolean;
  display_name: string;
  price: string;
}

export interface BillingPlanResponse {
  plan: Plan;
  plan_expires_at: string | null;
  limits: PlanLimits;
  all_plans: Record<Plan, PlanLimits>;
}

export interface CreatePaymentIntentResponse {
  client_secret: string;
  amount: number;
  currency: string;
  publishable_key: string;
}

export interface VersionInfo {
  minimum_version: string;
  android_store_url: string;
  ios_store_url: string;
}
