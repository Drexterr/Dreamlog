import { Fonts } from '../theme';
import { Modal, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useTheme } from '../context/ThemeContext';
import type { StreakInfo } from '../types';
import { T } from '../testIDs';

interface Props {
  visible: boolean;
  onClose: () => void;
  onRenew: () => void;
  streak: StreakInfo | null;
  busy: boolean;
  error: string;
  applied: boolean;
}

export default function RenewStreakModal({ visible, onClose, onRenew, streak, busy, error, applied }: Props) {
  const { colors } = useTheme();
  const count = streak?.freeze_count ?? 0;
  const active = (streak?.current_streak ?? 0) > 0;
  const disabled = busy || (!applied && (active || count === 0));
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={() => { if (!busy) onClose(); }}>
      <ScrollView contentContainerStyle={styles.overlay} style={{ backgroundColor: colors.bg + 'EE' }}>
        <View testID={T.mood.renewalDialog} accessibilityViewIsModal style={[styles.card, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}>
          <Text accessibilityRole="header" style={[styles.title, { color: colors.textPrimary }]}>Renew your streak</Text>
          <Text testID={T.mood.freezeCount} style={[styles.count, { color: colors.info }]}>{count} {count === 1 ? 'freeze' : 'freezes'} available</Text>
          <Text style={[styles.body, { color: colors.textSecondary }]}>{applied
            ? 'Your freeze has been applied. Refresh to load the updated streak.'
            : active ? 'Your streak is active. You only need to renew after a missed day.'
              : count ? 'Use one freeze to protect yesterday and renew your streak. Your journal entries are kept.'
                : 'No freezes are available. You can still start a new streak by journaling today.'}</Text>
          {!!error && <Text accessibilityRole="alert" style={[styles.error, { color: colors.danger }]}>{error}</Text>}
          <View style={styles.actions}>
            <TouchableOpacity testID={T.mood.renewDismiss} accessibilityRole="button" accessibilityState={{ disabled: busy }} disabled={busy} onPress={onClose} style={[styles.button, { backgroundColor: colors.card }]}>
              <Text style={[styles.buttonText, { color: colors.textSecondary }]}>{active ? 'Close' : 'Not now'}</Text>
            </TouchableOpacity>
            <TouchableOpacity testID={T.mood.renewConfirm} accessibilityRole="button" accessibilityState={{ disabled, busy }} disabled={disabled} onPress={onRenew} style={[styles.button, { backgroundColor: colors.brandGlow, borderWidth: 1, borderColor: colors.border, opacity: disabled ? 0.45 : 1 }]}>
              <Text style={[styles.buttonText, { color: colors.info }]}>{busy ? (applied ? 'Refreshing…' : 'Renewing…') : applied ? 'Refresh streak' : 'Renew'}</Text>
            </TouchableOpacity>
          </View>
        </View>
      </ScrollView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flexGrow: 1, justifyContent: 'center', padding: 24 },
  card: { borderRadius: 20, padding: 24, borderWidth: 1 },
  title: { fontFamily: Fonts.serifRegular, fontSize: 23, marginBottom: 14 },
  count: { fontFamily: Fonts.sansSB, fontSize: 16, marginBottom: 12 },
  body: { fontFamily: Fonts.sans, fontSize: 14, lineHeight: 22, marginBottom: 22 },
  error: { fontFamily: Fonts.sans, fontSize: 13, marginBottom: 16 },
  actions: { flexDirection: 'row', gap: 12 },
  button: { flex: 1, minHeight: 44, paddingVertical: 13, alignItems: 'center', justifyContent: 'center', borderRadius: 10 },
  buttonText: { fontFamily: Fonts.sans, fontSize: 14 },
});
