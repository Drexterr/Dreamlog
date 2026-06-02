import { Tabs } from 'expo-router';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from '../../src/context/ThemeContext';

// SVG-free tab icons using React Native Views + Text
function HomeIcon({ focused }: { focused: boolean }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.iconWrap, focused && styles.iconWrapActive]}>
      <Text style={[styles.iconEmoji, { color: focused ? colors.purple300 : colors.textMuted }]}>⌂</Text>
    </View>
  );
}

function TimelineIcon({ focused }: { focused: boolean }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.iconWrap, focused && styles.iconWrapActive]}>
      <View style={styles.tlWrap}>
        {[0, 1, 2].map((i) => (
          <View key={i} style={styles.tlRow}>
            <View style={[styles.tlDot, { backgroundColor: focused ? colors.purple300 : colors.textMuted }]} />
            <View style={[styles.tlLine, { backgroundColor: focused ? `${colors.purple300}80` : colors.textFaint }]} />
          </View>
        ))}
      </View>
    </View>
  );
}

function MoodIcon({ focused }: { focused: boolean }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.iconWrap, focused && styles.iconWrapActive]}>
      <View style={styles.chartWrap}>
        {[6, 10, 4, 12, 8].map((h, i) => (
          <View
            key={i}
            style={[
              styles.chartBar,
              { height: h, backgroundColor: focused ? colors.purple300 : colors.textMuted },
            ]}
          />
        ))}
      </View>
    </View>
  );
}

function SettingsIcon({ focused }: { focused: boolean }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.iconWrap, focused && styles.iconWrapActive]}>
      <View style={[styles.gearOuter, { borderColor: focused ? colors.purple300 : colors.textMuted }]}>
        <View style={[styles.gearInner, { backgroundColor: focused ? colors.purple300 : colors.textMuted }]} />
      </View>
    </View>
  );
}

export default function TabsLayout() {
  const { colors } = useTheme();
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: styles.tabBar,
        tabBarBackground: () => <View style={[styles.tabBarBg, { backgroundColor: colors.bg, borderTopColor: colors.borderFaint }]} />,
        tabBarActiveTintColor: colors.purple300,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarLabelStyle: styles.tabLabel,
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Home',
          tabBarIcon: ({ focused }) => <HomeIcon focused={focused} />,
        }}
      />
      <Tabs.Screen
        name="timeline"
        options={{
          title: 'Timeline',
          tabBarIcon: ({ focused }) => <TimelineIcon focused={focused} />,
        }}
      />
      <Tabs.Screen
        name="mood"
        options={{
          title: 'Mood',
          tabBarIcon: ({ focused }) => <MoodIcon focused={focused} />,
        }}
      />
      <Tabs.Screen
        name="settings"
        options={{
          title: 'Settings',
          tabBarIcon: ({ focused }) => <SettingsIcon focused={focused} />,
        }}
      />
    </Tabs>
  );
}

const styles = StyleSheet.create({
  tabBar: {
    backgroundColor: 'transparent',
    borderTopWidth: 0,
    elevation: 0,
    height: 80,
    paddingBottom: 16,
  },
  tabBarBg: {
    ...StyleSheet.absoluteFillObject,
    borderTopWidth: 1,
  },
  tabLabel: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 10,
    letterSpacing: 0.4,
  },

  // Icon containers
  iconWrap: { width: 32, height: 24, alignItems: 'center', justifyContent: 'center' },
  iconWrapActive: {},
  iconEmoji: { fontSize: 18 },

  // Timeline icon
  tlWrap: { gap: 3 },
  tlRow: { flexDirection: 'row', alignItems: 'center', gap: 4 },
  tlDot: { width: 5, height: 5, borderRadius: 2.5 },
  tlLine: { width: 12, height: 1.5, borderRadius: 1 },

  // Mood icon (mini bars)
  chartWrap: { flexDirection: 'row', alignItems: 'flex-end', gap: 2, height: 14 },
  chartBar: { width: 4, borderRadius: 2 },

  // Settings icon (gear-like)
  gearOuter: {
    width: 16, height: 16, borderRadius: 8,
    borderWidth: 2,
    alignItems: 'center', justifyContent: 'center',
  },
  gearInner: {
    width: 5, height: 5, borderRadius: 2.5,
  },
});
