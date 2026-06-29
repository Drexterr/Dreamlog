/**
 * ShareInsightModal - shows a preview of InsightCard and handles capture + share.
 * Uses react-native-view-shot to capture the card as a PNG, then shares via native sheet.
 *
 * Requires a dev build (not Expo Go). See InsightCard.tsx for details.
 */

import { useRef, useState } from 'react';
import {
  Modal,
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  Share,
  Platform,
  ScrollView,
} from 'react-native';
import { captureRef } from 'react-native-view-shot';
import InsightCard, { CARD_WIDTH, CARD_HEIGHT } from './InsightCard';
import { useTheme } from '../context/ThemeContext';
import type { ThemeColors } from '../theme';
import { api } from '../api/client';
import type { MoodArcDay } from '../types';

export interface ShareInsightModalProps {
  visible: boolean;
  onClose: () => void;
  weekLabel: string;
  weekStart?: string;   // YYYY-MM-DD - passed to backend share tracking
  moodArc: MoodArcDay[];
  topEmotions: string[];
  streak: number;
  entryCount: number;
}

export default function ShareInsightModal({
  visible,
  onClose,
  weekLabel,
  weekStart,
  moodArc,
  topEmotions,
  streak,
  entryCount,
}: ShareInsightModalProps) {
  const cardRef = useRef<View>(null);
  const [sharing, setSharing] = useState(false);
  const { colors } = useTheme();
  const styles = getStyles(colors);

  const handleShare = async () => {
    if (!cardRef.current) return;
    setSharing(true);
    try {
      const uri = await captureRef(cardRef, {
        format: 'png',
        quality: 1,
        width: CARD_WIDTH,
        height: CARD_HEIGHT,
      });

      let shared = false;
      // On iOS Share.share supports url; on Android we pass the file URI as a message fallback.
      if (Platform.OS === 'ios') {
        const result = await Share.share({ url: uri });
        shared = result.action !== Share.dismissedAction;
      } else {
        const result = await Share.share({
          message: `My week in review - ${weekLabel} - via DreamLog`,
        });
        shared = result.action !== Share.dismissedAction;
      }

      if (shared) {
        // Fire-and-forget: track the share event on the backend.
        api.trackInsightShare(weekStart).catch(() => undefined);
      }
    } catch {
      // User cancelled or share failed - no-op.
    } finally {
      setSharing(false);
    }
  };

  return (
    <Modal
      visible={visible}
      transparent
      animationType="slide"
      onRequestClose={onClose}
    >
      <View style={styles.overlay}>
        <View style={styles.sheet}>
          <Text style={styles.sheetTitle}>Your Week in Review</Text>
          <Text style={styles.sheetSub}>Preview your shareable insight card</Text>

          {/* Card preview - scrollable if screen is small */}
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={styles.previewScroll}
          >
            <View style={styles.cardWrapper}>
              <InsightCard
                ref={cardRef}
                weekLabel={weekLabel}
                moodArc={moodArc}
                topEmotions={topEmotions}
                streak={streak}
                entryCount={entryCount}
              />
            </View>
          </ScrollView>

          {/* Actions */}
          <View style={styles.actions}>
            <TouchableOpacity style={styles.cancelBtn} onPress={onClose}>
              <Text style={styles.cancelText}>Cancel</Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={[styles.shareBtn, sharing && styles.shareBtnDisabled]}
              onPress={handleShare}
              disabled={sharing}
            >
              {sharing ? (
                <ActivityIndicator color="#fff" size="small" />
              ) : (
                <Text style={styles.shareText}>Share Card</Text>
              )}
            </TouchableOpacity>
          </View>
        </View>
      </View>
    </Modal>
  );
}

const getStyles = (colors: ThemeColors) =>
  StyleSheet.create({
    overlay: {
      flex: 1,
      backgroundColor: 'rgba(10,5,20,0.9)',
      justifyContent: 'flex-end',
    },
    sheet: {
      backgroundColor: colors.cardSolid,
      borderTopLeftRadius: 24,
      borderTopRightRadius: 24,
      paddingTop: 24,
      paddingBottom: 40,
      paddingHorizontal: 20,
      borderTopWidth: 1,
      borderTopColor: colors.borderFaint,
    },
    sheetTitle: {
      fontSize: 20,
      color: colors.textPrimary,
      fontFamily: 'CormorantGaramond_400Regular',
      textAlign: 'center',
      marginBottom: 4,
    },
    sheetSub: {
      fontSize: 12,
      color: colors.textMuted,
      fontFamily: 'Nunito_400Regular',
      textAlign: 'center',
      marginBottom: 20,
    },
    previewScroll: {
      paddingHorizontal: 4,
      paddingBottom: 8,
    },
    cardWrapper: {
      borderRadius: 16,
      overflow: 'hidden',
      // Scale down the 375-wide card to fit narrower screens while keeping aspect ratio.
      transform: [{ scale: 0.85 }],
      transformOrigin: 'top left',
      marginRight: -CARD_WIDTH * 0.15, // compensate for scale shrink
    },
    actions: {
      flexDirection: 'row',
      gap: 12,
      marginTop: 16,
    },
    cancelBtn: {
      flex: 1,
      paddingVertical: 14,
      borderRadius: 14,
      backgroundColor: colors.card,
      alignItems: 'center',
    },
    cancelText: {
      color: colors.textSecondary,
      fontFamily: 'Nunito_400Regular',
      fontSize: 15,
    },
    shareBtn: {
      flex: 2,
      paddingVertical: 14,
      borderRadius: 14,
      backgroundColor: colors.brand,
      alignItems: 'center',
    },
    shareBtnDisabled: {
      opacity: 0.6,
    },
    shareText: {
      color: '#fff',
      fontFamily: 'Nunito_600SemiBold',
      fontSize: 15,
    },
  });
