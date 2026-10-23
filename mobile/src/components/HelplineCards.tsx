import { Linking, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { helplineHref, type CountryHelplines } from '../services/helplines';
import type { ThemeColors } from '../theme';

/**
 * Tap-to-call crisis helpline cards for one country. Shared by every crisis
 * surface (reflection crisis card, follow-up conversation) so the resources
 * look and behave the same wherever crisis screening fires.
 */
export function HelplineCards({ resource, colors, testID }: { resource: CountryHelplines; colors: ThemeColors; testID?: string }) {
  const openURL = (url: string) => {
    Linking.canOpenURL(url).then((ok) => {
      if (ok) Linking.openURL(url);
    });
  };

  return (
    <View testID={testID}>
      <Text style={[styles.resourceSectionLabel, { color: colors.textMuted }]}>
        CRISIS HELPLINES{resource.code !== 'INTL' ? ` · ${resource.countryName.toUpperCase()}` : ''}
      </Text>
      <View style={styles.hotlineList}>
        {resource.lines.map((h) => (
          <TouchableOpacity
            key={h.name}
            accessibilityRole="button"
            accessibilityLabel={`${h.name}, ${h.detail}`}
            style={[styles.hotlineCard, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}
            onPress={() => openURL(helplineHref(h.action))}
            activeOpacity={0.75}
          >
            <View style={{ flex: 1 }}>
              <Text style={[styles.hotlineName, { color: colors.textPrimary }]}>{h.name}</Text>
              <Text style={[styles.hotlineHours, { color: colors.textMuted }]}>
                {h.hours ? `${h.detail} · ${h.hours}` : h.detail}
              </Text>
            </View>
            <View style={[styles.callPill, { backgroundColor: `${colors.brand}22`, borderColor: `${colors.brand}55` }]}>
              <Text style={[styles.callPillText, { color: colors.brand }]}>
                {h.action.kind === 'link' ? '🔗 Open' : h.action.kind === 'text' ? '💬 Text' : '📞 Call'}
              </Text>
            </View>
          </TouchableOpacity>
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  resourceSectionLabel: {
    fontFamily: 'HankenGrotesk_600SemiBold',
    fontSize: 10,
    letterSpacing: 1.5,
    marginBottom: 10,
  },
  hotlineList: { gap: 8, marginBottom: 28 },
  hotlineCard: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 16,
    borderWidth: 1,
    padding: 16,
    gap: 12,
  },
  hotlineName: { fontFamily: 'HankenGrotesk_600SemiBold', fontSize: 14, marginBottom: 2 },
  hotlineHours: { fontFamily: 'HankenGrotesk_400Regular', fontSize: 11 },
  callPill: {
    borderRadius: 10,
    borderWidth: 1,
    paddingHorizontal: 10,
    paddingVertical: 6,
  },
  callPillText: { fontFamily: 'HankenGrotesk_700Bold', fontSize: 12 },
});
