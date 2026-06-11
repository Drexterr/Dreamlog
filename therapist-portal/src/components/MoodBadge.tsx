interface Props {
  score: number;
  label?: string;
}

const moodColor = (score: number) =>
  score >= 71 ? '#5A9367' : score >= 46 ? '#B08A3E' : score >= 26 ? '#C0703D' : '#C05B4D';

export default function MoodBadge({ score, label }: Props) {
  const color = moodColor(score);

  return (
    <div style={{ textAlign: 'center', flexShrink: 0 }}>
      <div style={{
        width: 42, height: 42, borderRadius: '50%',
        border: `2px solid ${color}`,
        background: `${color}14`,
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
