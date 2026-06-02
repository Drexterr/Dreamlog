type Trend = 'improving' | 'declining' | 'stable' | 'insufficient_data';

const ICONS: Record<Trend, { icon: string; color: string }> = {
  improving:         { icon: '↑', color: '#4ade80' },
  declining:         { icon: '↓', color: '#f87171' },
  stable:            { icon: '→', color: '#facc15' },
  insufficient_data: { icon: '—', color: '#9b8ec4' },
};

export default function MoodTrendIcon({ trend }: { trend: Trend }) {
  const { icon, color } = ICONS[trend] ?? ICONS.insufficient_data;
  return <span style={{ color, fontWeight: 700 }}>{icon}</span>;
}
