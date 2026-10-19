import { createContext, ReactNode, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import RenewStreakModal from '../components/RenewStreakModal';
import type { StreakInfo } from '../types';

interface StreakContextValue {
  streak: StreakInfo | null;
  renewed: boolean;
  openRenewal: () => void;
  refreshStreak: () => void;
}
const StreakContext = createContext<StreakContextValue>({ streak: null, renewed: false, openRenewal: () => {}, refreshStreak: () => {} });

/** Lives above navigation: one successful startup check per launch, never per tab visit. */
export function StreakProvider({ children, enabled, startupReady }: { children: ReactNode; enabled: boolean; startupReady: boolean }) {
  const [streak, setStreak] = useState<StreakInfo | null>(null);
  const [visible, setVisible] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [applied, setApplied] = useState(false);
  const [renewed, setRenewed] = useState(false);
  const checked = useRef(false);
  const inFlight = useRef(false);
  const generation = useRef(0);
  const readVersion = useRef(0);

  const refreshStreak = useCallback(() => {
    if (!enabled || inFlight.current) return;
    const current = generation.current;
    const request = ++readVersion.current;
    api.streak().then(value => {
      if (generation.current === current && readVersion.current === request) setStreak(value);
    }).catch(() => {});
  }, [enabled]);

  useEffect(() => {
    generation.current++;
    inFlight.current = false;
    setStreak(null); setVisible(false); setBusy(false); setError(''); setApplied(false); setRenewed(false);
    if (enabled) refreshStreak();
    return () => { generation.current++; };
  }, [enabled, refreshStreak]);

  useEffect(() => {
    if (!enabled || !startupReady || !streak || checked.current) return;
    checked.current = true;
    if (streak.current_streak === 0 && streak.total_days > 0) setVisible(true);
  }, [enabled, startupReady, streak]);

  const openRenewal = useCallback(() => { if (enabled && streak) { setError(''); setVisible(true); } }, [enabled, streak]);
  const renew = async () => {
    if (!enabled || !streak || inFlight.current || (!applied && (streak.freeze_count <= 0 || streak.current_streak > 0))) return;
    const current = generation.current;
    readVersion.current++; // A pending read must not overwrite the renewal result.
    inFlight.current = true; setBusy(true); setError('');
    let acknowledged = applied;
    try {
      if (!acknowledged) {
        const yesterday = new Date(); yesterday.setUTCDate(yesterday.getUTCDate() - 1);
        const result = await api.useStreakFreeze(yesterday.toISOString().slice(0, 10));
        if (generation.current !== current) return;
        acknowledged = true;
        setApplied(true);
        setStreak(previous => previous ? { ...previous, freeze_count: result.freeze_count } : previous);
      }
      const updated = await api.streak();
      if (generation.current !== current) return;
      setStreak(updated); setVisible(false); setRenewed(true); setApplied(false);
    } catch {
      if (generation.current === current) setError(acknowledged
        ? 'Your freeze was applied. Refresh to load the renewed streak; another freeze will not be used.'
        : 'Could not renew your streak. Try again.');
    } finally {
      if (generation.current === current) { inFlight.current = false; setBusy(false); }
    }
  };

  return (
    <StreakContext.Provider value={{ streak: enabled ? streak : null, renewed: enabled && renewed, openRenewal, refreshStreak }}>
      {children}
      <RenewStreakModal visible={enabled && startupReady && visible} streak={streak} busy={busy} error={error} applied={applied} onClose={() => setVisible(false)} onRenew={renew} />
    </StreakContext.Provider>
  );
}

export const useStreak = () => useContext(StreakContext);
