import axios from 'axios';
import Cookies from 'js-cookie';

const TOKEN_KEY = 'dreamlog_therapist_token';

export const http = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080',
  timeout: 30_000,
  headers: { 'Content-Type': 'application/json' },
});

http.interceptors.request.use((config) => {
  const token = Cookies.get(TOKEN_KEY);
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

export function saveToken(token: string) {
  Cookies.set(TOKEN_KEY, token, {
    expires: 7,
    sameSite: 'strict',
    // Only transmit the token over HTTPS in production. Kept off on localhost
    // (http) so dev login still works.
    secure: typeof window !== 'undefined' && window.location.protocol === 'https:',
  });
}
export function clearToken() {
  Cookies.remove(TOKEN_KEY);
}
export function getToken(): string | undefined {
  return Cookies.get(TOKEN_KEY);
}

// ── Types ────────────────────────────────────────────────────────────────────

export interface Therapist {
  id: string;
  user_id: string;
  name: string;
  email: string;
  credentials: string;
  plan: string;
  created_at: string;
}

export interface ClientSummary {
  client_id: string;
  name: string;
  linked_at: string;
  last_entry_at: string | null;
  avg_mood_30d: number | null;
  entry_count: number;
}

export interface ClientBrief {
  client_id: string;
  client_name: string;
  generated_at: string;
  brief: string;
  top_emotions: string[];
  mood_trend: 'improving' | 'declining' | 'stable' | 'insufficient_data';
  avg_mood_7d: number | null;
  entry_count: number;
  recent_entries: {
    date: string;
    summary: string;
    mood_score: number;
    topics: string[];
    key_quote: string;
  }[];
}

// ── API calls ─────────────────────────────────────────────────────────────────

export const api = {
  login: (email: string, password: string) =>
    http.post<{ token: string; user: { id: string; name: string; email: string } }>(
      '/auth/login', { email, password }
    ).then(r => r.data),

  authRegister: (email: string, password: string, name: string) =>
    http.post<{ token: string; user: { id: string; name: string; email: string } }>(
      '/auth/register', { email, password, name }
    ).then(r => r.data),

  registerTherapist: (name: string, email: string, credentials: string) =>
    http.post<Therapist>('/therapists/register', { name, email, credentials }).then(r => r.data),

  listClients: () =>
    http.get<{ clients: ClientSummary[] }>('/therapists/clients').then(r => r.data.clients),

  linkClient: (clientId: string) =>
    http.post('/therapists/clients/link', { client_id: clientId }).then(r => r.data),

  unlinkClient: (clientId: string) =>
    http.delete(`/therapists/clients/${clientId}`).then(r => r.data),

  getClientBrief: (clientId: string) =>
    http.get<ClientBrief>(`/therapists/clients/${clientId}/brief`).then(r => r.data),
};

// ── Therapist Workspace: external clients + session notes ────────────────────

export interface ExternalClient {
  id: string;
  therapist_id: string;
  name: string;
  role: string;
  archived: boolean;
  session_count: number;
  last_session_at?: string | null;
  created_at: string;
  updated_at: string;
}

export type ClientSessionStatus = 'pending' | 'processing' | 'completed' | 'failed';

export interface ClientSession {
  id: string;
  therapist_id: string;
  external_client_id?: string;
  linked_user_id?: string;
  session_date: string;
  status: ClientSessionStatus;
  raw_text?: string;
  bullets: string[];
  summary?: string;
  error_msg?: string;
  created_at: string;
  updated_at: string;
}

export interface TherapistOverview {
  external_clients: number;
  linked_clients: number;
  sessions_this_week: number;
  sessions_this_month: number;
  total_sessions: number;
  last_session_at?: string | null;
}

export const notesApi = {
  me: () =>
    http.get<{ therapist: Therapist; client_consent_accepted: boolean }>('/therapists/me').then(r => r.data),

  acceptConsent: () =>
    http.post<{ client_consent_accepted: boolean }>('/therapists/consent').then(r => r.data),

  overview: () =>
    http.get<TherapistOverview>('/therapists/overview').then(r => r.data),

  listExternalClients: () =>
    http.get<{ clients: ExternalClient[] }>('/therapists/external-clients').then(r => r.data.clients),

  createExternalClient: (name: string, role?: string) =>
    http.post<ExternalClient>('/therapists/external-clients', { name, role }).then(r => r.data),

  updateExternalClient: (id: string, patch: { name?: string; role?: string; archived?: boolean }) =>
    http.patch<ExternalClient>(`/therapists/external-clients/${id}`, patch).then(r => r.data),

  deleteExternalClient: (id: string) =>
    http.delete(`/therapists/external-clients/${id}`).then(() => undefined),

  presignNote: (filename: string, contentType: string) =>
    http
      .post<{ upload_url: string; image_key: string }>('/therapists/sessions/presign', {
        filename,
        content_type: contentType,
      })
      .then(r => r.data),

  createSession: (input: {
    external_client_id?: string;
    linked_client_id?: string;
    session_date?: string;
    image_key?: string;
    bullets?: string[];
  }) => http.post<ClientSession>('/therapists/sessions', input).then(r => r.data),

  listSessions: (params?: { external_client_id?: string; linked_client_id?: string }) =>
    http.get<{ sessions: ClientSession[] }>('/therapists/sessions', { params }).then(r => r.data.sessions),

  getSession: (id: string) =>
    http.get<ClientSession>(`/therapists/sessions/${id}`).then(r => r.data),

  updateBullets: (id: string, bullets: string[]) =>
    http.patch<ClientSession>(`/therapists/sessions/${id}`, { bullets }).then(r => r.data),

  summarize: (id: string) =>
    http.post<ClientSession>(`/therapists/sessions/${id}/summarize`).then(r => r.data),

  deleteSession: (id: string) =>
    http.delete(`/therapists/sessions/${id}`).then(() => undefined),
};
