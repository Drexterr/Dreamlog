type Trend = 'improving' | 'declining' | 'stable' | 'insufficient_data';

const ICONS: Record<Trend, { icon: string; color: string }> = {
  improving:         { icon: '↑', color: '#5A9367' },
  declining:         { icon: '↓', color: '#C05B4D' },
  stable:            { icon: '→', color: '#B08A3E' },
  insufficient_data: { icon: '–', color: '#7E8280' },
};

export default function MoodTrendIcon({ trend }: { trend: Trend }) {
  const { icon, color } = ICONS[trend] ?? ICONS.insufficient_data;
  return <span style={{ color, fontWeight: 700 }}>{icon}</span>;
}
