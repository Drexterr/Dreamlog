/**
 * PDF Export screen.
 *
 * Lets the user choose a period (monthly / yearly), download a PDF of
 * their emotional journal, and share it via the native share sheet.
 */

import { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  SafeAreaView,
  StatusBar,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { useRouter } from 'expo-router';
import * as FileSystem from 'expo-file-system';
import * as Sharing from 'expo-sharing';
import { api } from '../src/api/client';
import { useTheme } from '../src/context/ThemeContext';

type Period = 'monthly' | 'yearly';

const PERIODS: { key: Period; label: string; description: string }[] = [
  { key: 'monthly', label: 'Last 30 days', description: 'Mood arc, top emotions, and all entries from the past month.' },
  { key: 'yearly', label: 'Last 365 days', description: 'Full-year emotional journey — a complete annual retrospective.' },
];

export default function ExportScreen() {
  const router = useRouter();
  const { colors } = useTheme();

  const [selected, setSelected] = useState<Period>('monthly');
  const [loading, setLoading] = useState(false);

  const handleExport = async () => {
    setLoading(true);
    try {
      const { url, headers } = await api.exportPDFParams(selected);

      const filename = `dreamlog-${selected}-${new Date().toISOString().slice(0, 7)}.pdf`;
      const cacheDir = (FileSystem as unknown as Record<string, string | null>)['cacheDirectory'] ?? '';
      const dest = cacheDir + filename;

      const result = await FileSystem.downloadAsync(url, dest, { headers });

      if (result.status !== 200) {
        Alert.alert('Export failed', 'Could not generate your PDF. Please try again.');
        return;
      }

      const canShare = await Sharing.isAvailableAsync();
      if (!canShare) {
        Alert.alert('Saved', `PDF saved to ${dest}`);
        return;
      }

      await Sharing.shareAsync(result.uri, {
        mimeType: 'application/pdf',
        dialogTitle: 'Share your DreamLog journal',
        UTI: 'com.adobe.pdf',
      });
    } catch {
      Alert.alert('Error', 'Something went wrong. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <View style={styles.inner}>

          <TouchableOpacity onPress={() => router.back()} style={styles.backBtn}>
            <Text style={[styles.backText, { color: colors.textMuted }]}>← Back</Text>
          </TouchableOpacity>

          <Text style={[styles.title, { color: colors.textPrimary }]}>Export Journal</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
            Download a beautifully formatted PDF of your emotional journal — your mood arc, AI entry summaries, and key quotes.
          </Text>

          {/* What's included */}
          <View style={[styles.infoBox, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
            <Text style={[styles.infoTitle, { color: colors.brand }]}>What's in the PDF</Text>
            {[
              'Cover page with your mood stats',
              'Daily mood chart and trend',
              'Top 5 recurring emotions',
              'All journal entry summaries',
              'Key quotes from each entry',
            ].map((item) => (
              <Text key={item} style={[styles.infoItem, { color: colors.textSecondary }]}>✓  {item}</Text>
            ))}
            <Text style={[styles.infoNote, { color: colors.textMuted }]}>
              Raw transcripts and voice recordings are never included.
            </Text>
          </View>

          {/* Period selector */}
          <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>SELECT PERIOD</Text>
          {PERIODS.map((p) => {
            const isActive = selected === p.key;
            return (
              <TouchableOpacity
                key={p.key}
                style={[
                  styles.periodCard,
                  { backgroundColor: colors.card, borderColor: isActive ? colors.brand : colors.borderFaint },
                ]}
                onPress={() => setSelected(p.key)}
                activeOpacity={0.8}
              >
                <View style={styles.periodRow}>
                  <View style={styles.periodText}>
                    <Text style={[styles.periodLabel, { color: isActive ? colors.brand : colors.textPrimary }]}>
                      {p.label}
                    </Text>
                    <Text style={[styles.periodDesc, { color: colors.textMuted }]}>{p.description}</Text>
                  </View>
                  <View style={[styles.radio, { borderColor: isActive ? colors.brand : colors.borderFaint }]}>
                    {isActive && <View style={[styles.radioDot, { backgroundColor: colors.brand }]} />}
                  </View>
                </View>
              </TouchableOpacity>
            );
          })}

          {/* Export button */}
          <TouchableOpacity
            style={[styles.exportBtn, { backgroundColor: colors.brand }, loading && { opacity: 0.6 }]}
            onPress={handleExport}
            disabled={loading}
            activeOpacity={0.85}
          >
            {loading ? (
              <ActivityIndicator color={colors.bg} />
            ) : (
              <Text style={[styles.exportBtnText, { color: colors.bg }]}>Generate & Download PDF</Text>
            )}
          </TouchableOpacity>

          <Text style={[styles.footer, { color: colors.textMuted }]}>
            PDFs are generated on demand and not stored on our servers.
          </Text>
        </View>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  inner: { flex: 1, padding: 24 },

  backBtn: { marginBottom: 20 },
  backText: { fontSize: 14, fontFamily: 'Nunito_400Regular' },

  title: {
    fontFamily: 'CormorantGaramond_300Light',
    fontSize: 28,
    fontWeight: '300',
    marginBottom: 10,
  },
  subtitle: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 14,
    lineHeight: 22,
    marginBottom: 24,
  },

  infoBox: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 18,
    marginBottom: 28,
  },
  infoTitle: {
    fontFamily: 'Nunito_700Bold',
    fontSize: 11,
    letterSpacing: 1,
    marginBottom: 10,
  },
  infoItem: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 13,
    marginBottom: 5,
    lineHeight: 20,
  },
  infoNote: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 11,
    marginTop: 8,
    lineHeight: 18,
  },

  sectionLabel: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 10,
    letterSpacing: 1.5,
    marginBottom: 10,
  },

  periodCard: {
    borderRadius: 14,
    borderWidth: 1.5,
    padding: 16,
    marginBottom: 10,
  },
  periodRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  periodText: { flex: 1 },
  periodLabel: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 14,
    marginBottom: 2,
  },
  periodDesc: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 12,
    lineHeight: 18,
  },
  radio: {
    width: 20,
    height: 20,
    borderRadius: 10,
    borderWidth: 2,
    alignItems: 'center',
    justifyContent: 'center',
  },
  radioDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
  },

  exportBtn: {
    borderRadius: 16,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 24,
    marginBottom: 16,
  },
  exportBtnText: {
    fontFamily: 'Nunito_700Bold',
    fontSize: 15,
  },

  footer: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 11,
    lineHeight: 18,
    textAlign: 'center',
  },
});
