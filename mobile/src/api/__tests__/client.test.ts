import axios from 'axios';
import { api } from '../client';

jest.mock('axios', () => {
  const mockAxios = {
    create: jest.fn(),
    isAxiosError: jest.fn(),
  };
  const instance = {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    interceptors: {
      request: { use: jest.fn() },
      response: { use: jest.fn() },
    },
  };
  mockAxios.create.mockReturnValue(instance);
  return { ...mockAxios, default: mockAxios };
});

// Pull the mocked instance out after module load.
const mockInstance = (axios.create as jest.Mock)();

beforeEach(() => {
  jest.clearAllMocks();
});

// ── moodHistory ────────────────────────────────────────────────────────────────

describe('api.moodHistory', () => {
  const makeResponse = (overrides = {}) => ({
    days: [{ day: '2026-04-28', avg_mood: 70, entry_count: 2 }],
    range: '30d' as const,
    avg_mood: 70,
    prev_avg_mood: 62,
    mood_delta: 8,
    top_emotions: ['hopeful', 'anxious'],
    entry_count: 14,
    ...overrides,
  });

  it('calls GET /mood/history with default range 30d', async () => {
    mockInstance.get.mockResolvedValueOnce({ data: makeResponse() });

    const result = await api.moodHistory();

    expect(mockInstance.get).toHaveBeenCalledWith('/mood/history', {
      params: { range: '30d' },
    });
    expect(result.range).toBe('30d');
  });

  it('passes explicit range param to the API', async () => {
    mockInstance.get.mockResolvedValueOnce({ data: makeResponse({ range: '90d' }) });

    await api.moodHistory('90d');

    expect(mockInstance.get).toHaveBeenCalledWith('/mood/history', {
      params: { range: '90d' },
    });
  });

  it('passes 365d range param correctly', async () => {
    mockInstance.get.mockResolvedValueOnce({ data: makeResponse({ range: '365d' }) });

    await api.moodHistory('365d');

    expect(mockInstance.get).toHaveBeenCalledWith('/mood/history', {
      params: { range: '365d' },
    });
  });

  it('returns avg_mood and mood_delta from response', async () => {
    mockInstance.get.mockResolvedValueOnce({ data: makeResponse({ avg_mood: 75, mood_delta: 5 }) });

    const result = await api.moodHistory('30d');

    expect(result.avg_mood).toBe(75);
    expect(result.mood_delta).toBe(5);
  });

  it('returns null avg_mood and mood_delta when no data', async () => {
    mockInstance.get.mockResolvedValueOnce({
      data: makeResponse({ avg_mood: null, prev_avg_mood: null, mood_delta: null, days: [], entry_count: 0 }),
    });

    const result = await api.moodHistory('30d');

    expect(result.avg_mood).toBeNull();
    expect(result.mood_delta).toBeNull();
    expect(result.days).toHaveLength(0);
  });

  it('returns top_emotions array', async () => {
    mockInstance.get.mockResolvedValueOnce({
      data: makeResponse({ top_emotions: ['hopeful', 'anxious', 'calm'] }),
    });

    const result = await api.moodHistory('90d');

    expect(result.top_emotions).toEqual(['hopeful', 'anxious', 'calm']);
  });

  it('propagates network errors', async () => {
    mockInstance.get.mockRejectedValueOnce(new Error('Network Error'));

    await expect(api.moodHistory()).rejects.toThrow('Network Error');
  });
});
