import { Tabs } from 'expo-router';
import { useEffect, useRef } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from '../../src/context/ThemeContext';
import { useGuidedTour } from '../../src/context/GuidedTourContext';

// SVG-free tab icons using React Native Views + Text
function HomeIcon({ focused }: { focused: boolean }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.iconWrap, focused && styles.iconWrapActive]}>
      <Text style={[styles.iconEmoji, { color: focused ? colors.purple300 : colors.textMuted }]}>⌂</Text>
    </View>
  );
}

function ExploreIcon({ focused }: { focused: boolean }) {
  const { colors } = useTheme();
  const { registerRef } = useGuidedTour();
  const ref = useRef<View>(null);
  useEffect(() => { registerRef('tab_explore', ref); }, []);
  const c = focused ? colors.purple300 : colors.textMuted;
  return (
    <View ref={ref} collapsable={false} style={[styles.iconWrap, focused && styles.iconWrapActive]}>
      <View style={styles.gridWrap}>
        <View style={styles.gridRow}>
          <View style={[styles.gridDot, { backgroundColor: c }]} />
          <View style={[styles.gridDot, { backgroundColor: c }]} />
        </View>
        <View style={styles.gridRow}>
          <View style={[styles.gridDot, { backgroundColor: c }]} />
          <View style={[styles.gridDot, { backgroundColor: c }]} />
        </View>
      </View>
    </View>
  );
}

function MoodIcon({ focused }: { focused: boolean }) {
  const { colors } = useTheme();
  const { registerRef } = useGuidedTour();
  const ref = useRef<View>(null);
  useEffect(() => { registerRef('tab_mood', ref); }, []);
  return (
    <View ref={ref} collapsable={false} style={[styles.iconWrap, focused && styles.iconWrapActive]}>
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
  const { registerRef } = useGuidedTour();
  const ref = useRef<View>(null);
  useEffect(() => { registerRef('tab_settings', ref); }, []);
  return (
    <View ref={ref} collapsable={false} style={[styles.iconWrap, focused && styles.iconWrapActive]}>
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
          title: 'Explore',
          tabBarIcon: ({ focused }) => <ExploreIcon focused={focused} />,
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

  // Explore icon (2×2 grid)
  gridWrap: { gap: 3.5 },
  gridRow: { flexDirection: 'row', gap: 3.5 },
  gridDot: { width: 5, height: 5, borderRadius: 1.5 },

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
