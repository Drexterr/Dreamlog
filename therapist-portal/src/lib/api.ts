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
  Cookies.set(TOKEN_KEY, token, { expires: 7, sameSite: 'strict' });
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
