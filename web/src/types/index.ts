export type UserGoal = 'anxiety' | 'stress' | 'grief' | 'depression' | 'relationships' | 'career' | 'trauma' | 'curious';

export interface User {
  id: string;
  email: string;
  name: string;
  preferredName?: string;
  goal?: UserGoal;
  plan: 'free' | 'plus' | 'pro';
  createdAt: string;
}

export interface Therapist {
  id: string;
  userId: string;
  name: string;
  email: string;
  credentials?: string;
  createdAt: string;
}

export interface ClientSummary {
  id: string; // client's user UUID
  name: string;
  goal: UserGoal;
  moodScore: number;
  entryCount: number;
  linkedAt: string;
}

export interface Emotion {
  name: string;
  intensity: number; // 0.0 - 1.0
}

export interface JournalEntry {
  id: string;
  recordedAt: string;
  durationSecs: number;
  moodScore: number;
  summary: string;
  keyQuotes: string[];
  topics: string[];
  emotions: Emotion[];
}

export interface ClientBrief {
  clientId: string;
  clientName: string;
  generatedAt: string;
  brief: string;
  topEmotions: string[];
  moodTrend: string;
  avgMood7d?: number;
  entryCount: number;
  recentEntries: JournalEntry[];
}
