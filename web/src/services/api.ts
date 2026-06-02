import axios from 'axios';
import type { ClientSummary, ClientBrief, Therapist, UserGoal } from '../types';

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080';

// API Client instance
const client = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Attach JWT token from localStorage to headers
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('dreamlog_therapist_token');
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Helper for Mock Data
const MOCK_CLIENTS: ClientSummary[] = [
  {
    id: 'a3f5a11c-c9d2-4309-80fb-449e21183c51',
    name: 'Sophia Chen',
    goal: 'anxiety',
    moodScore: 58,
    entryCount: 14,
    linkedAt: '2026-05-01T00:00:00Z',
  },
  {
    id: '7d898a12-fc8e-4a61-8b2b-426b659bc8fb',
    name: 'Marcus Thorne',
    goal: 'grief',
    moodScore: 42,
    entryCount: 8,
    linkedAt: '2026-04-18T00:00:00Z',
  },
  {
    id: 'e55d6480-1a2b-4cd3-8e7c-a4422bb9930f',
    name: 'Elena Rostova',
    goal: 'career',
    moodScore: 71,
    entryCount: 22,
    linkedAt: '2026-05-10T00:00:00Z',
  },
];

const MOCK_BRIEFS: Record<string, ClientBrief> = {
  'a3f5a11c-c9d2-4309-80fb-449e21183c51': {
    clientId: 'a3f5a11c-c9d2-4309-80fb-449e21183c51',
    clientName: 'Sophia Chen',
    generatedAt: new Date().toISOString(),
    brief: 'Client journals frequently about experiencing chest tightness and rapid thoughts, primarily associated with an upcoming project launch. However, in recent logs, she shows high emotional resilience by noticing that sitting in quiet environments helps ground her nervous system.',
    topEmotions: ['Anxious anticipation', 'Quiet resilience', 'Physical fatigue'],
    moodTrend: 'stable',
    avgMood7d: 58,
    entryCount: 14,
    recentEntries: [
      {
        id: '1',
        recordedAt: 'Today at 11:15 AM',
        durationSecs: 102,
        moodScore: 58,
        summary: 'Experienced sudden chest tightness while preparing slides for the stakeholder review. Sat in the conference room in the dark for five minutes, which slowed down the rapid heart rate.',
        keyQuotes: ['I just sat in the dark. It helped, actually.', 'I felt stupid for caring so much.'],
        topics: ['Work Setback', 'Anxiety Trigger', 'Somatic Grounding'],
        emotions: [{ name: 'Anxiety', intensity: 0.75 }, { name: 'Resilience', intensity: 0.5 }],
      },
      {
        id: '2',
        recordedAt: 'May 27, 2026 at 9:30 PM',
        durationSecs: 135,
        moodScore: 68,
        summary: 'Expressed relief after receiving confirmation that the deadline was pushed. Felt a sense of space to breathe and catch up.',
        keyQuotes: ['It feels like a massive weight has been lifted.', 'I can sleep tonight.'],
        topics: ['Work Pressure', 'Relief'],
        emotions: [{ name: 'Relief', intensity: 0.8 }, { name: 'Peace', intensity: 0.6 }],
      },
      {
        id: '3',
        recordedAt: 'May 25, 2026 at 8:00 AM',
        durationSecs: 58,
        moodScore: 45,
        summary: 'Logged a short entry immediately after waking up. Anticipated massive workload for the week and expressed fear of failing client expectations.',
        keyQuotes: ['I woke up with this knot in my stomach.', 'Just looking at my calendar makes me want to shrink.'],
        topics: ['Anticipation', 'Morning Anxiety'],
        emotions: [{ name: 'Anxiety', intensity: 0.85 }],
      },
    ],
  },
  '7d898a12-fc8e-4a61-8b2b-426b659bc8fb': {
    clientId: '7d898a12-fc8e-4a61-8b2b-426b659bc8fb',
    clientName: 'Marcus Thorne',
    generatedAt: new Date().toISOString(),
    brief: 'Marcus is processing deep grief following a recent bereavement. His logs show a very slow speaking pace and low energy levels. He describes a feeling of numbness alternating with intense waves of sadness when encountering familiar items in the house.',
    topEmotions: ['Bereavement / Sorrow', 'Isolation', 'Emotional numbness'],
    moodTrend: 'heavy',
    avgMood7d: 42,
    entryCount: 8,
    recentEntries: [
      {
        id: '11',
        recordedAt: 'Yesterday at 8:45 PM',
        durationSecs: 98,
        moodScore: 40,
        summary: 'Spent the evening packing things in the living room. Encountered an old postcard which triggered a long quiet reflection. Felt deeply lonely.',
        keyQuotes: ['I found this card from 2022. It felt like a punch in the chest.', 'The house is just so incredibly quiet.'],
        topics: ['Bereavement', 'Loneliness', 'Triggers'],
        emotions: [{ name: 'Sorrow', intensity: 0.85 }, { name: 'Isolation', intensity: 0.6 }],
      },
      {
        id: '12',
        recordedAt: 'May 26, 2026 at 7:00 PM',
        durationSecs: 74,
        moodScore: 35,
        summary: 'Felt completely detached from projects during team standups. Felt like a bystander observing a different life.',
        keyQuotes: ['Everyone is laughing about meeting KPIs.', 'I\'m just sitting here trying to remember to breathe.'],
        topics: ['Disconnection', 'Workplace Isolation'],
        emotions: [{ name: 'Numbness', intensity: 0.75 }],
      },
    ],
  },
  'e55d6480-1a2b-4cd3-8e7c-a4422bb9930f': {
    clientId: 'e55d6480-1a2b-4cd3-8e7c-a4422bb9930f',
    clientName: 'Elena Rostova',
    generatedAt: new Date().toISOString(),
    brief: 'Elena journals to navigate a significant promotion and career path transition. Her tone is energetic, showing high motivation mixed with mild impostor syndrome. She is processing details about establishing boundaries as she steps into a leadership role.',
    topEmotions: ['Ambitious drive', 'Impostor anxiety', 'Excitement'],
    moodTrend: 'improving',
    avgMood7d: 71,
    entryCount: 22,
    recentEntries: [
      {
        id: '21',
        recordedAt: 'Today at 8:30 AM',
        durationSecs: 110,
        moodScore: 71,
        summary: 'Conducted her first 1-on-1 meetings as team lead. Felt nervous initially but managed to guide discussions productively. Felt proud of setting clear expectations.',
        keyQuotes: ['I was sweating before it started, but I think I actually held my ground.', 'They listened to me.'],
        topics: ['Leadership Transition', 'Boundary Setting'],
        emotions: [{ name: 'Pride', intensity: 0.7 }, { name: 'Apprehension', intensity: 0.45 }],
      },
      {
        id: '22',
        recordedAt: 'May 28, 2026 at 10:15 PM',
        durationSecs: 92,
        moodScore: 70,
        summary: 'Reflected on the shift in relationships with former peers. Felt some sadness about no longer being \'one of them\' but accepted it as a necessary step.',
        keyQuotes: ['There\'s this weird distance now at lunch.', 'It\'s lonely, but I know it is what it is.'],
        topics: ['Peer Dynamics', 'Acceptance'],
        emotions: [{ name: 'Acceptance', intensity: 0.8 }],
      },
    ],
  },
};

// Check if using Mock Mode (disabled if actual API is live)
const isMockMode = () => {
  return localStorage.getItem('dreamlog_mock_mode') === 'true' || true; // Fallback to true for demonstration
};

export const api = {
  // Auth API
  async login(email: string, password: string): Promise<{ token: string }> {
    if (isMockMode()) {
      await new Promise(r => setTimeout(r, 800));
      if (email.includes('error')) throw new Error('Invalid email or password.');
      return { token: 'mock-jwt-therapist-token' };
    }
    const res = await client.post('/auth/login', { email, password });
    return res.data;
  },

  async register(email: string, name: string, credentials?: string): Promise<Therapist> {
    if (isMockMode()) {
      await new Promise(r => setTimeout(r, 1000));
      return {
        id: 'mock-therapist-uuid-1',
        userId: 'mock-user-uuid-1',
        name,
        email,
        credentials,
        createdAt: new Date().toISOString(),
      };
    }
    const res = await client.post('/therapists/register', { name, email, credentials });
    return res.data;
  },

  // Clients API
  async getClients(): Promise<ClientSummary[]> {
    if (isMockMode()) {
      const stored = localStorage.getItem('dreamlog_mock_clients');
      return stored ? JSON.parse(stored) : MOCK_CLIENTS;
    }
    const res = await client.get('/therapists/clients');
    return res.data.clients;
  },

  async linkClient(clientId: string, name?: string, goal?: UserGoal): Promise<any> {
    if (isMockMode()) {
      await new Promise(r => setTimeout(r, 600));
      const clients = await this.getClients();
      
      const newClient: ClientSummary = {
        id: clientId,
        name: name || 'New Connection',
        goal: goal || 'anxiety',
        moodScore: 65,
        entryCount: 1,
        linkedAt: new Date().toISOString(),
      };

      clients.push(newClient);
      localStorage.setItem('dreamlog_mock_clients', JSON.stringify(clients));

      // Store a brief for them
      const briefs = { ...MOCK_BRIEFS };
      briefs[clientId] = {
        clientId,
        clientName: newClient.name,
        generatedAt: new Date().toISOString(),
        brief: `Initial pre-session brief for ${newClient.name}. Client recently linked their DreamLog account. No journal patterns have been analyzed yet.`,
        topEmotions: ['Cooperative', 'Receptive'],
        moodTrend: 'stable',
        avgMood7d: 65,
        entryCount: 1,
        recentEntries: [
          {
            id: 'new-1',
            recordedAt: 'Today at 12:00 PM',
            durationSecs: 42,
            moodScore: 65,
            summary: 'Client linked their account to their therapist portal.',
            keyQuotes: ['I am ready to share my mood summaries.'],
            topics: ['Baseline Connection'],
            emotions: [{ name: 'Cooperation', intensity: 0.7 }],
          }
        ]
      };
      
      return newClient;
    }
    const res = await client.post('/therapists/clients/link', { client_id: clientId });
    return res.data;
  },

  async unlinkClient(clientId: string): Promise<void> {
    if (isMockMode()) {
      const clients = await this.getClients();
      const filtered = clients.filter(c => c.id !== clientId);
      localStorage.setItem('dreamlog_mock_clients', JSON.stringify(filtered));
      return;
    }
    await client.delete(`/therapists/clients/${clientId}`);
  },

  async getClientBrief(clientId: string): Promise<ClientBrief> {
    if (isMockMode()) {
      await new Promise(r => setTimeout(r, 300));
      return MOCK_BRIEFS[clientId] || {
        clientId,
        clientName: 'Unknown Client',
        generatedAt: new Date().toISOString(),
        brief: 'No data available.',
        topEmotions: [],
        moodTrend: 'stable',
        entryCount: 0,
        recentEntries: [],
      };
    }
    const res = await client.get(`/therapists/clients/${clientId}/brief`);
    return res.data;
  },

  async regenerateClientBrief(clientId: string): Promise<ClientBrief> {
    await new Promise(r => setTimeout(r, 1500)); // Simulate slow AI generation
    const brief = await this.getClientBrief(clientId);
    return {
      ...brief,
      generatedAt: new Date().toISOString(),
      brief: brief.brief + ' Re-evaluation shows consistent use of cognitive coping mechanisms when processing daily stressors.',
    };
  }
};
