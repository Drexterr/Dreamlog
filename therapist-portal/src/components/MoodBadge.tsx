interface Props {
  score: number;
  label?: string;
}

export default function MoodBadge({ score, label }: Props) {
  const color = score >= 71 ? '#4ade80' : score >= 46 ? '#facc15' : score >= 26 ? '#fb923c' : '#f87171';

  return (
    <div style={{ textAlign: 'center', flexShrink: 0 }}>
      <div style={{
        width: 42, height: 42, borderRadius: '50%',
        border: `2px solid ${color}`,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 13, fontWeight: 700, color,
      }}>
        {score}
      </div>
      {label && (
        <div style={{ fontSize: 9, color: 'var(--muted)', marginTop: 3 }}>{label}</div>
      )}
    </div>
  );
}
