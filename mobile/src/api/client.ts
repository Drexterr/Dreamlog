import axios, { AxiosInstance, AxiosError } from 'axios';
import * as SecureStore from 'expo-secure-store';
import {
  AgeRange,
  AnnualReview,
  AnnualReviewListResponse,
  BillingPlanResponse,
  Conversation,
  CreatePaymentIntentResponse,
  CreateShareLinkResult,
  Entry,
  EntryAnalysis,
  EntryMode,
  InsightCardData,
  InsightShareResult,
  JourneyListResponse,
  JourneySession,
  JourneySessionsResponse,
  ListEntriesResponse,
  MoodHistoryResponse,
  PatternRadarResponse,
  Plan,
  PresignResponse,
  ShareLinksResponse,
  StreakInfo,
  TimelineResponse,
  User,
  UserGoal,
  WeeklyMoodResponse,
  WeeklyReview,
  WeeklyReviewListResponse,
} from '../types';

const BASE_URL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';
const TOKEN_KEY = 'dreamlog_access_token';

// ── Token storage ────────────────────────────────────────────────────────────
export const storeToken = (token: string) => SecureStore.setItemAsync(TOKEN_KEY, token);
export const getToken = () => SecureStore.getItemAsync(TOKEN_KEY);
export const clearToken = () => SecureStore.deleteItemAsync(TOKEN_KEY);

// ── Axios instance ───────────────────────────────────────────────────────────
const http: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 30_000,
  headers: { 'Content-Type': 'application/json', 'ngrok-skip-browser-warning': '1' },
});

http.interceptors.request.use(async (config) => {
  const token = await getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// ── Auth API response ────────────────────────────────────────────────────────
export interface AuthResponse {
  user: User;
  token: string;
}

// ── API surface ──────────────────────────────────────────────────────────────
export const api = {
  // ── Auth ──────────────────────────────────────────────────────────────────
  register: (email: string, name: string, password: string): Promise<AuthResponse> =>
    http.post<AuthResponse>('/auth/register', { email, name, password }).then((r) => r.data),

  login: (email: string, password: string): Promise<AuthResponse> =>
    http.post<AuthResponse>('/auth/login', { email, password }).then((r) => r.data),

  // ── User ──────────────────────────────────────────────────────────────────
  me: (): Promise<User> =>
    http.get<User>('/me').then((r) => r.data),

  updateMe: (fields: {
    name?: string;
    preferred_name?: string;
    timezone?: string;
    fcm_nudge_hour?: number;
    nudge_enabled?: boolean;
    goal?: UserGoal;
    age_range?: AgeRange;
    country?: string;
  }): Promise<User> =>
    http.put<User>('/me', fields).then((r) => r.data),

  deleteAccount: (): Promise<void> =>
    http.delete('/me').then(() => undefined),

  // ── Entries ───────────────────────────────────────────────────────────────
  presign: (): Promise<PresignResponse> =>
    http.post<PresignResponse>('/entries/presign').then((r) => r.data),

  createEntry: (payload: {
    audio_key: string;
    audio_size_bytes: number;
    duration_sec: number;
    mode?: EntryMode;
  }): Promise<Entry> =>
    http.post<Entry>('/entries', payload).then((r) => r.data),

  listEntries: (page = 1, pageSize = 20): Promise<ListEntriesResponse> =>
    http
      .get<ListEntriesResponse>('/entries', { params: { page, page_size: pageSize } })
      .then((r) => r.data),

  getEntry: (id: string): Promise<Entry> =>
    http.get<Entry>(`/entries/${id}`).then((r) => r.data),

  // ── Analysis ──────────────────────────────────────────────────────────────
  getAnalysis: (entryId: string): Promise<EntryAnalysis> =>
    http.get<EntryAnalysis>(`/entries/${entryId}/analysis`).then((r) => r.data),

  // ── Timeline ──────────────────────────────────────────────────────────────
  getTimeline: (page = 1, pageSize = 20): Promise<TimelineResponse> =>
    http
      .get<TimelineResponse>('/timeline', { params: { page, page_size: pageSize } })
      .then((r) => r.data),

  // ── Search ────────────────────────────────────────────────────────────────
  searchEntries: (query: string, limit = 20): Promise<{ entries: Entry[]; query: string }> =>
    http
      .get('/entries/search', { params: { q: query, limit } })
      .then((r) => r.data),

  // ── Conversations ─────────────────────────────────────────────────────────
  getOrCreateConversation: (entryId: string): Promise<Conversation> =>
    http.post<Conversation>(`/entries/${entryId}/conversation`).then((r) => r.data),

  sendConversationMessage: (convId: string, content: string): Promise<Conversation> =>
    http
      .post<Conversation>(`/conversations/${convId}/messages`, { content })
      .then((r) => r.data),

  // ── Mood ──────────────────────────────────────────────────────────────────
  weeklyMood: (): Promise<WeeklyMoodResponse> =>
    http.get<WeeklyMoodResponse>('/mood/weekly').then((r) => r.data),

  streak: (): Promise<StreakInfo> =>
    http.get<StreakInfo>('/mood/streak').then((r) => r.data),

  // ── Devices ───────────────────────────────────────────────────────────────
  registerDevice: (fcm_token: string, platform: 'ios' | 'android'): Promise<void> =>
    http.post('/devices', { fcm_token, platform }).then(() => undefined),

  // ── Streak freeze ─────────────────────────────────────────────────────────
  useStreakFreeze: (freezeDate: string): Promise<{ freeze_count: number; freeze_date: string }> =>
    http
      .post('/streak/freeze', { freeze_date: freezeDate })
      .then((r) => r.data),

  // ── Life Graph ────────────────────────────────────────────────────────────
  moodHistory: (range: '30d' | '90d' | '365d' = '30d'): Promise<MoodHistoryResponse> =>
    http.get<MoodHistoryResponse>('/mood/history', { params: { range } }).then((r) => r.data),

  // ── Weekly Reviews ────────────────────────────────────────────────────────
  getLatestWeeklyReview: (): Promise<WeeklyReview> =>
    http.get<WeeklyReview>('/reviews/weekly/latest').then((r) => r.data),

  listWeeklyReviews: (): Promise<WeeklyReviewListResponse> =>
    http.get<WeeklyReviewListResponse>('/reviews/weekly').then((r) => r.data),

  // ── Therapist Share Links (5a) ────────────────────────────────────────────
  createShareLink: (): Promise<CreateShareLinkResult> =>
    http.post<CreateShareLinkResult>('/share').then((r) => r.data),

  listShareLinks: (): Promise<ShareLinksResponse> =>
    http.get<ShareLinksResponse>('/share').then((r) => r.data),

  revokeShareLink: (id: string): Promise<void> =>
    http.delete(`/share/${id}`).then(() => undefined),

  // ── Pattern Radar ─────────────────────────────────────────────────────────────
  getPatternRadar: (range: '30d' | '90d' | '365d' = '30d'): Promise<PatternRadarResponse> =>
    http.get<PatternRadarResponse>('/mood/patterns', { params: { range } }).then((r) => r.data),

  // ── Insight Cards (4d) ───────────────────────────────────────────────────────
  getInsightCard: (): Promise<InsightCardData> =>
    http.get<InsightCardData>('/insights/card').then((r) => r.data),

  trackInsightShare: (weekStart?: string): Promise<InsightShareResult> =>
    http
      .post<InsightShareResult>('/insights/share', weekStart ? { week_start: weekStart } : {})
      .then((r) => r.data),

  // ── Annual Reviews ────────────────────────────────────────────────────────
  getLatestAnnualReview: (): Promise<AnnualReview> =>
    http.get<AnnualReview>('/reviews/annual/latest').then((r) => r.data),

  listAnnualReviews: (): Promise<AnnualReviewListResponse> =>
    http.get<AnnualReviewListResponse>('/reviews/annual').then((r) => r.data),

  // ── Guided Journeys ───────────────────────────────────────────────────────
  listJourneys: (): Promise<JourneyListResponse> =>
    http.get<JourneyListResponse>('/journeys').then((r) => r.data),

  startJourney: (journeyID: string): Promise<JourneySession> =>
    http.post<JourneySession>(`/journeys/${journeyID}/start`).then((r) => r.data),

  listJourneySessions: (): Promise<JourneySessionsResponse> =>
    http.get<JourneySessionsResponse>('/journeys/sessions').then((r) => r.data),

  getJourneySession: (sessionID: string): Promise<JourneySession> =>
    http.get<JourneySession>(`/journeys/sessions/${sessionID}`).then((r) => r.data),

  advanceJourneySession: (sessionID: string, entryID: string): Promise<JourneySession> =>
    http
      .post<JourneySession>(`/journeys/sessions/${sessionID}/advance`, { entry_id: entryID })
      .then((r) => r.data),

  // ── PDF Export (5d) ───────────────────────────────────────────────────────
  // Returns the full URL + auth header needed by expo-file-system.downloadAsync.
  exportPDFParams: async (period: 'monthly' | 'yearly'): Promise<{ url: string; headers: Record<string, string> }> => {
    const token = await getToken();
    const url = `${BASE_URL}/export/pdf?period=${period}`;
    return {
      url,
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    };
  },

  // ── Therapy Mode (Phase 6 + 8) ───────────────────────────────────────────
  startTherapySession: (persona?: import('../types').TherapyPersona): Promise<import('../types').StartSessionResponse> =>
    http.post<import('../types').StartSessionResponse>('/therapy/sessions', persona ? { persona } : {}).then((r) => r.data),

  listTherapySessions: (): Promise<import('../types').ListTherapySessionsResponse> =>
    http.get<import('../types').ListTherapySessionsResponse>('/therapy/sessions').then((r) => r.data),

  getTherapySession: (id: string): Promise<import('../types').TherapySession> =>
    http.get<import('../types').TherapySession>(`/therapy/sessions/${id}`).then((r) => r.data),

  presignTherapyAudio: (id: string, filename: string, contentType: string): Promise<import('../types').TherapyPresignResponse> =>
    http
      .post<import('../types').TherapyPresignResponse>(`/therapy/sessions/${id}/presign`, { filename, content_type: contentType })
      .then((r) => r.data),

  sendTherapyMessage: (
    id: string,
    payload: { content?: string; audio_key?: string; input_mode: 'voice' | 'text' },
  ): Promise<import('../types').SendTherapyMessageResponse> =>
    http
      .post<import('../types').SendTherapyMessageResponse>(`/therapy/sessions/${id}/messages`, payload)
      .then((r) => r.data),

  endTherapySession: (id: string): Promise<import('../types').EndSessionResponse> =>
    http.post<import('../types').EndSessionResponse>(`/therapy/sessions/${id}/end`).then((r) => r.data),

  // ── Billing (Phase 7a) ────────────────────────────────────────────────────
  getBillingPlan: (): Promise<BillingPlanResponse> =>
    http.get<BillingPlanResponse>('/billing/plan').then((r) => r.data),

  createPaymentIntent: (plan: 'plus' | 'pro', currency: 'inr' | 'usd'): Promise<CreatePaymentIntentResponse> =>
    http
      .post<CreatePaymentIntentResponse>('/billing/create-payment-intent', { plan, currency })
      .then((r) => r.data),

  upgradePlan: (plan: Plan, expiresAt?: string): Promise<BillingPlanResponse> =>
    http
      .post<BillingPlanResponse>('/billing/upgrade', { plan, expires_at: expiresAt ?? null })
      .then((r) => r.data),
};

// ── Typed error helper ───────────────────────────────────────────────────────
export function isApiError(err: unknown): err is AxiosError<{ message: string }> {
  return axios.isAxiosError(err);
}
