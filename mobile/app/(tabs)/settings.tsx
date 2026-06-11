import { useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  Alert,
  Switch,
  Modal,
  FlatList,
  Linking,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { BlurView } from 'expo-blur';
import { useRouter } from 'expo-router';
import { clearToken, api } from '../../src/api/client';
import { supabase } from '../../src/lib/supabase';
import { useTheme } from '../../src/context/ThemeContext';
import type { AgeRange, Plan, User, UserGoal } from '../../src/types';

const GOAL_META: Record<UserGoal, { label: string; emoji: string }> = {
  anxiety:       { label: 'Working through anxiety',     emoji: '🌱' },
  stress:        { label: 'Managing stress',             emoji: '🌊' },
  grief:         { label: 'Processing grief',            emoji: '🕊️' },
  depression:    { label: 'Lifting low mood',            emoji: '☀️' },
  relationships: { label: 'Understanding relationships', emoji: '❤️' },
  career:        { label: 'Career & purpose',            emoji: '🌲' },
  trauma:        { label: 'Processing past / trauma',    emoji: '🩹' },
  curious:       { label: 'Just exploring',              emoji: '🌌' },
};

const PLAN_LABELS: Record<Plan, string> = {
  free: 'Free',
  plus: 'DreamLog+',
  pro:  'DreamLog Pro',
  b2b:  'B2B Wellness',
};

const PLAN_BADGE_COLORS: Record<Plan, string> = {
  free: '#6b7280',
  plus: '#a78bfa',
  pro:  '#c084fc',
  b2b:  '#34d399',
};

const AGE_RANGE_LABELS: Record<AgeRange, string> = {
  under_18: 'Under 18',
  '18_24':  '18 – 24',
  '25_34':  '25 – 34',
  '35_44':  '35 – 44',
  '45_plus': '45 or older',
};

const CRISIS_HOTLINES = [
  { name: 'iCall', tel: 'tel:9152987821', info: '9152987821 · India' },
  { name: 'Vandrevala Foundation', tel: 'tel:18602662345', info: '1860-2662-345 · India · 24/7' },
  { name: '988 Lifeline', tel: 'tel:988', info: '988 · US · 24/7' },
];

function SettingRow({
  label,
  sub,
  right,
  danger = false,
  onPress,
  colors,
}: {
  label: string;
  sub?: string;
  right?: React.ReactNode;
  danger?: boolean;
  onPress?: () => void;
  colors: ReturnType<typeof useTheme>['colors'];
}) {
  return (
    <TouchableOpacity
      onPress={onPress}
      disabled={!onPress && !right}
      activeOpacity={onPress ? 0.7 : 1}
      style={styles.settingRow}
    >
      <View style={{ flex: 1 }}>
        <Text style={[styles.settingLabel, { color: danger ? colors.danger : colors.textPrimary }]}>
          {label}
        </Text>
        {sub ? <Text style={[styles.settingSub, { color: colors.textMuted }]}>{sub}</Text> : null}
      </View>
      {right ?? (onPress ? <Text style={[styles.chevron, { color: colors.textMuted }]}>›</Text> : null)}
    </TouchableOpacity>
  );
}

function formatHour(h: number): string {
  if (h === 0) return '12:00 AM';
  if (h < 12) return `${h}:00 AM`;
  if (h === 12) return '12:00 PM';
  return `${h - 12}:00 PM`;
}

function initials(name: string): string {
  return name
    .split(' ')
    .map((w) => w[0] ?? '')
    .join('')
    .toUpperCase()
    .slice(0, 2);
}

const NUDGE_HOURS = Array.from({ length: 18 }, (_, i) => i + 6);

const GOALS_LIST: { key: UserGoal; label: string; description: string; emoji: string }[] = [
  { key: 'anxiety',       label: 'Working through anxiety',     description: 'Worry, uncertainty, restless thoughts',               emoji: '🌱' },
  { key: 'stress',        label: 'Managing stress',             description: 'Overwhelm, pressure, too much on my plate',           emoji: '🌊' },
  { key: 'grief',         label: 'Processing grief',            description: "Loss, endings, things that can't be undone",          emoji: '🕊️' },
  { key: 'depression',    label: 'Lifting low mood',            description: 'Sadness, low motivation, feeling flat',               emoji: '☀️' },
  { key: 'relationships', label: 'Understanding relationships', description: 'Connection, conflict, how I show up for others',      emoji: '❤️' },
  { key: 'career',        label: 'Career & purpose',            description: "Work, direction, what I'm building toward",           emoji: '🌲' },
  { key: 'trauma',        label: 'Processing past / trauma',    description: 'Difficult memories, healing, working through trauma', emoji: '🩹' },
  { key: 'curious',       label: 'Just exploring',              description: "No agenda - I'm curious about my inner life",         emoji: '🌌' },
];

export default function SettingsScreen() {
  const router = useRouter();
  const { theme, colors, setThemeWithBubble } = useTheme();

  const [user, setUser] = useState<User | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [entryCount, setEntryCount] = useState(0);
  const [currentPlan, setCurrentPlan] = useState<Plan>('free');
  const [planExpiresAt, setPlanExpiresAt] = useState<string | null>(null);
  const [nudgeEnabled, setNudgeEnabled] = useState(true);
  const [nudgeHour, setNudgeHour] = useState(8);
  const [showHourPicker, setShowHourPicker] = useState(false);
  const [showCrisisModal, setShowCrisisModal] = useState(false);
  const [showProfileModal, setShowProfileModal] = useState(false);
  const [showGoalModal, setShowGoalModal] = useState(false);
  const [savingHour, setSavingHour] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const loadProfile = () => {
    setLoadError(false);
    api.me()
      .then((u) => {
        setUser(u);
        setNudgeEnabled(u.nudge_enabled ?? true);
        setNudgeHour(u.fcm_nudge_hour ?? 8);
      })
      .catch(() => setLoadError(true));
    api.getBillingPlan()
      .then((b) => {
        setCurrentPlan(b.plan);
        setPlanExpiresAt(b.plan_expires_at ?? null);
      })
      .catch(() => {});
    api.listEntries(1, 1)
      .then((r) => setEntryCount(r.total))
      .catch(() => {});
  };

  useEffect(() => { loadProfile(); }, []);

  const handleNudgeToggle = async (value: boolean) => {
    setNudgeEnabled(value);
    try {
      await api.updateMe({ nudge_enabled: value });
    } catch {
      setNudgeEnabled(!value);
      Alert.alert('Could not save', 'Please try again.');
    }
  };

  const handleSelectHour = async (h: number) => {
    setShowHourPicker(false);
    if (h === nudgeHour) return;
    setSavingHour(true);
    const prev = nudgeHour;
    setNudgeHour(h);
    try {
      await api.updateMe({ fcm_nudge_hour: h });
    } catch {
      setNudgeHour(prev);
      Alert.alert('Could not save', 'Please try again.');
    } finally {
      setSavingHour(false);
    }
  };

  const handleSignOut = () => {
    Alert.alert('Sign out', 'This will clear your stored session.', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Sign out',
        style: 'destructive',
        onPress: async () => {
          await supabase.auth.signOut();
          // clearToken() is called by the onAuthStateChange listener in supabase.ts
          router.replace('/auth');
        },
      },
    ]);
  };

  const handleDeleteData = () => {
    Alert.alert(
      'Delete all data',
      'This will permanently delete your account, all journal entries, reflections, and analysis. This cannot be undone.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete everything',
          style: 'destructive',
          onPress: () => {
            Alert.alert(
              'Are you absolutely sure?',
              'There is no recovery. All your data will be gone forever.',
              [
                { text: 'Cancel', style: 'cancel' },
                {
                  text: 'Yes, delete my account',
                  style: 'destructive',
                  onPress: async () => {
                    setDeleting(true);
                    try {
                      await api.deleteAccount();
                      await supabase.auth.signOut();
                      router.replace('/auth');
                    } catch {
                      setDeleting(false);
                      Alert.alert('Error', 'Could not delete account. Please try again.');
                    }
                  },
                },
              ],
            );
          },
        },
      ],
    );
  };

  // ── Goal selection: full-screen flood fill via ThemeContext ─────────────────
  const handleGoalSelectFromModal = (goal: UserGoal, pageX: number, pageY: number) => {
    setShowGoalModal(false);
    if (goal === theme) return;

    // Optimistically update the goal label immediately.
    setUser(prev => prev ? { ...prev, goal } : prev);

    // Wait for the modal slide-down, then trigger the global flood fill.
    // ThemeContext's overlay renders above everything (tab bar included), so
    // no element snaps when setTheme fires at the end of the animation.
    setTimeout(() => {
      setThemeWithBubble(goal, pageX, pageY);
    }, 300);
  };

  const goalMeta = user?.goal
    ? (GOAL_META[user.goal] ?? { label: 'Just exploring', emoji: '🌌' })
    : { label: 'Not set', emoji: '🌌' };

  const displayName = user?.preferred_name || user?.name || '-';
  const avatarText = user ? initials(user.preferred_name || user.name || '?') : '?';
  const profileSub = loadError
    ? 'Could not load profile - tap to retry'
    : user
      ? `${entryCount} ${entryCount === 1 ? 'entry' : 'entries'} · ${PLAN_LABELS[currentPlan]}`
      : 'Loading…';

  if (deleting) {
    return (
      <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]}>
        <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', gap: 16 }}>
          <ActivityIndicator color={colors.brand} />
          <Text style={{ color: colors.textMuted, fontFamily: 'Nunito_400Regular', fontSize: 14 }}>
            Deleting your account…
          </Text>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Settings</Text>

          {/* Profile card */}
          <TouchableOpacity
            style={[styles.profileCard, { backgroundColor: colors.card, borderColor: loadError ? colors.danger : colors.border }]}
            onPress={() => loadError ? loadProfile() : setShowProfileModal(true)}
            activeOpacity={0.75}
          >
            <View style={[styles.avatar, { backgroundColor: colors.brand }]}>
              <Text style={styles.avatarText}>{avatarText}</Text>
            </View>
            <View style={{ flex: 1 }}>
              <Text style={[styles.profileName, { color: colors.textPrimary }]} numberOfLines={1}>
                {displayName}
              </Text>
              <Text style={[styles.profileSub, { color: colors.textMuted }]} numberOfLines={1}>
                {profileSub}
              </Text>
            </View>
            <Text style={[styles.chevron, { color: colors.textMuted }]}>›</Text>
          </TouchableOpacity>

          {/* Subscription */}
          <Text style={[styles.sectionLabel, { color: colors.textSecondary }]}>SUBSCRIPTION</Text>
          <View style={[styles.section, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
            <View style={styles.settingRow}>
              <View style={{ flex: 1 }}>
                <Text style={[styles.settingLabel, { color: colors.textPrimary }]}>Current plan</Text>
                <Text style={[styles.settingSub, { color: colors.textMuted }]}>
                  {currentPlan === 'free'
                    ? 'Upgrade for unlimited entries & more'
                    : planExpiresAt
                      ? `Active until ${new Date(planExpiresAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })} · does not auto-renew`
                      : 'Active'}
                </Text>
              </View>
              <View style={[styles.planBadge, {
                backgroundColor: `${PLAN_BADGE_COLORS[currentPlan]}22`,
                borderColor: `${PLAN_BADGE_COLORS[currentPlan]}55`,
              }]}>
                <Text style={[styles.planBadgeText, { color: PLAN_BADGE_COLORS[currentPlan] }]}>
                  {PLAN_LABELS[currentPlan]}
                </Text>
              </View>
            </View>
            {currentPlan !== 'pro' && currentPlan !== 'b2b' && (
              <>
                <View style={[styles.rowDivider, { backgroundColor: colors.borderFaint }]} />
                <TouchableOpacity
                  style={[styles.upgradeRow, { backgroundColor: colors.brandGlow }]}
                  onPress={() => router.push('/upgrade' as never)}
                  activeOpacity={0.8}
                >
                  <Text style={[styles.upgradeText, { color: colors.purple300 }]}>
                    {currentPlan === 'free' ? '✦  Upgrade to DreamLog+' : '✦  Upgrade to DreamLog Pro'}
                  </Text>
                  <Text style={[styles.chevron, { color: colors.purple300 }]}>›</Text>
                </TouchableOpacity>
              </>
            )}
          </View>

          {/* Emotional Goal */}
          <Text style={[styles.sectionLabel, { color: colors.textSecondary }]}>EMOTIONAL GOAL</Text>
          <View style={[styles.section, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
            <SettingRow
              label={`${goalMeta.emoji}  ${goalMeta.label}`}
              sub="Shapes how your reflections are written"
              colors={colors}
              onPress={() => setShowGoalModal(true)}
            />
          </View>

          {/* Reminders */}
          <Text style={[styles.sectionLabel, { color: colors.textSecondary }]}>REMINDERS</Text>
          <View style={[styles.section, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
            <SettingRow
              label="Morning nudge"
              sub="Sent after your first journal of the day"
              colors={colors}
              right={
                <Switch
                  value={nudgeEnabled}
                  onValueChange={handleNudgeToggle}
                  trackColor={{ false: colors.cardSolid, true: colors.brand }}
                  thumbColor={nudgeEnabled ? colors.purple300 : colors.textMuted}
                />
              }
            />
            {nudgeEnabled && (
              <>
                <View style={[styles.rowDivider, { backgroundColor: colors.borderFaint }]} />
                <SettingRow
                  label="Nudge hour"
                  sub="What time to send your morning nudge"
                  colors={colors}
                  onPress={() => setShowHourPicker(true)}
                  right={
                    <Text style={[styles.valueText, { color: savingHour ? colors.textMuted : colors.purple300 }]}>
                      {formatHour(nudgeHour)}
                    </Text>
                  }
                />
              </>
            )}
          </View>

          {/* Privacy */}
          <Text style={[styles.sectionLabel, { color: colors.textSecondary }]}>PRIVACY</Text>
          <View style={[styles.section, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
            <SettingRow label="Export my data" colors={colors} onPress={() => router.push('/export')} />
            <View style={[styles.rowDivider, { backgroundColor: colors.borderFaint }]} />
            <SettingRow label="Delete all data" danger colors={colors} onPress={handleDeleteData} />
          </View>

          {/* Support */}
          <Text style={[styles.sectionLabel, { color: colors.textSecondary }]}>SUPPORT</Text>
          <View style={[styles.section, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
            <SettingRow
              label="Get help now"
              sub="Crisis resources and helplines"
              colors={colors}
              onPress={() => setShowCrisisModal(true)}
            />
            <View style={[styles.rowDivider, { backgroundColor: colors.borderFaint }]} />
            <SettingRow
              label="About DreamLog"
              colors={colors}
              right={<Text style={[styles.valueText, { color: colors.purple300 }]}>v1.0.0</Text>}
            />
          </View>

          {/* Sign out */}
          <TouchableOpacity
            onPress={handleSignOut}
            style={[styles.signOutBtn, { borderColor: `${colors.danger}33`, backgroundColor: `${colors.danger}11` }]}
            activeOpacity={0.8}
          >
            <Text style={[styles.signOutText, { color: colors.danger }]}>Sign out</Text>
          </TouchableOpacity>
        </ScrollView>
      </SafeAreaView>

      {/* Hour picker modal */}
      <Modal
        visible={showHourPicker}
        transparent
        animationType="slide"
        onRequestClose={() => setShowHourPicker(false)}
      >
        <BlurView intensity={55} tint="dark" style={styles.modalOverlay}>
          <TouchableOpacity
            style={styles.modalDismiss}
            activeOpacity={1}
            onPress={() => setShowHourPicker(false)}
          />
          <View
            style={[styles.modalSheet, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}
            onStartShouldSetResponder={() => true}
          >
            <View style={[styles.modalHandle, { backgroundColor: colors.border }]} />
            <Text style={[styles.modalTitle, { color: colors.textPrimary }]}>Morning Nudge Time</Text>
            <Text style={[styles.modalSub, { color: colors.textMuted }]}>
              When should we send your nudge?
            </Text>
            <FlatList
              data={NUDGE_HOURS}
              keyExtractor={(h) => String(h)}
              showsVerticalScrollIndicator={false}
              style={styles.hourList}
              initialScrollIndex={Math.max(0, NUDGE_HOURS.indexOf(nudgeHour))}
              getItemLayout={(_, index) => ({ length: 52, offset: 52 * index, index })}
              renderItem={({ item: h }) => {
                const isSelected = h === nudgeHour;
                return (
                  <TouchableOpacity
                    style={[
                      styles.hourRow,
                      { borderBottomColor: colors.borderFaint },
                      isSelected && { backgroundColor: colors.brandGlow },
                    ]}
                    onPress={() => handleSelectHour(h)}
                    activeOpacity={0.7}
                  >
                    <Text style={[styles.hourLabel, { color: isSelected ? colors.purple300 : colors.textPrimary }]}>
                      {formatHour(h)}
                    </Text>
                    {isSelected && (
                      <Text style={[styles.hourCheck, { color: colors.brand }]}>✓</Text>
                    )}
                  </TouchableOpacity>
                );
              }}
            />
          </View>
        </BlurView>
      </Modal>

      {/* Crisis / Get Help modal */}
      <Modal
        visible={showCrisisModal}
        transparent
        animationType="slide"
        onRequestClose={() => setShowCrisisModal(false)}
      >
        <BlurView intensity={55} tint="dark" style={styles.modalOverlay}>
          <TouchableOpacity
            style={styles.modalDismiss}
            activeOpacity={1}
            onPress={() => setShowCrisisModal(false)}
          />
          <View
            style={[styles.modalSheet, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}
            onStartShouldSetResponder={() => true}
          >
            <View style={[styles.modalHandle, { backgroundColor: colors.border }]} />
            <Text style={[styles.modalTitle, { color: colors.textPrimary }]}>You're not alone.</Text>
            <Text style={[styles.modalSub, { color: colors.textMuted }]}>
              If you're in crisis or need to talk to someone right now, please reach out.
            </Text>
            {CRISIS_HOTLINES.map((h) => (
              <TouchableOpacity
                key={h.name}
                style={[styles.crisisCard, { backgroundColor: colors.bg, borderColor: colors.danger }]}
                onPress={() => Linking.openURL(h.tel)}
                activeOpacity={0.8}
              >
                <Text style={[styles.crisisName, { color: colors.textPrimary }]}>{h.name}</Text>
                <Text style={[styles.crisisInfo, { color: colors.textMuted }]}>{h.info}</Text>
              </TouchableOpacity>
            ))}
            <TouchableOpacity
              style={[styles.closeBtn, { borderColor: colors.borderFaint }]}
              onPress={() => setShowCrisisModal(false)}
            >
              <Text style={[styles.closeBtnText, { color: colors.textMuted }]}>Close</Text>
            </TouchableOpacity>
          </View>
        </BlurView>
      </Modal>

      {/* Goal picker modal */}
      <Modal
        visible={showGoalModal}
        transparent
        animationType="slide"
        onRequestClose={() => setShowGoalModal(false)}
      >
        <BlurView intensity={55} tint="dark" style={styles.modalOverlay}>
          <TouchableOpacity
            style={styles.modalDismiss}
            activeOpacity={1}
            onPress={() => setShowGoalModal(false)}
          />
          <View
            style={[styles.modalSheet, styles.goalSheet, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}
            onStartShouldSetResponder={() => true}
          >
            <View style={[styles.modalHandle, { backgroundColor: colors.border }]} />
            <Text style={[styles.modalTitle, { color: colors.textPrimary }]}>Emotional Goal</Text>
            <Text style={[styles.modalSub, { color: colors.textMuted }]}>
              Shapes how your reflections are written.
            </Text>
            <ScrollView showsVerticalScrollIndicator={false} style={styles.goalScrollList}>
              {GOALS_LIST.map((g) => {
                const isSelected = theme === g.key;
                return (
                  <TouchableOpacity
                    key={g.key}
                    style={[
                      styles.goalPickerCard,
                      {
                        backgroundColor: isSelected ? colors.brandGlow : colors.card,
                        borderColor: isSelected ? colors.brand : colors.border,
                      },
                    ]}
                    onPress={(e) => handleGoalSelectFromModal(g.key, e.nativeEvent.pageX, e.nativeEvent.pageY)}
                    activeOpacity={0.7}
                  >
                    <View style={styles.goalPickerRow}>
                      <Text style={[styles.goalPickerLabel, { color: isSelected ? colors.purple300 : colors.textPrimary }]}>
                        {g.label}
                      </Text>
                      {isSelected
                        ? <Text style={[styles.goalPickerCheck, { color: colors.brand }]}>✓</Text>
                        : <Text style={{ fontSize: 15 }}>{g.emoji}</Text>
                      }
                    </View>
                    <Text style={[styles.goalPickerDesc, { color: colors.textMuted }]}>{g.description}</Text>
                  </TouchableOpacity>
                );
              })}
            </ScrollView>
          </View>
        </BlurView>
      </Modal>

      {/* Profile info modal */}
      <Modal
        visible={showProfileModal}
        transparent
        animationType="slide"
        onRequestClose={() => setShowProfileModal(false)}
      >
        <BlurView intensity={55} tint="dark" style={styles.modalOverlay}>
          <TouchableOpacity
            style={styles.modalDismiss}
            activeOpacity={1}
            onPress={() => setShowProfileModal(false)}
          />
          <View
            style={[styles.modalSheet, { backgroundColor: colors.cardSolid, borderColor: colors.border }]}
            onStartShouldSetResponder={() => true}
          >
            <View style={[styles.modalHandle, { backgroundColor: colors.border }]} />

            {/* Avatar + name header */}
            <View style={styles.profileModalHeader}>
              <View style={[styles.profileModalAvatar, { backgroundColor: colors.brand }]}>
                <Text style={styles.profileModalAvatarText}>{avatarText}</Text>
              </View>
              <Text style={[styles.profileModalName, { color: colors.textPrimary }]}>{displayName}</Text>
            </View>

            {/* Info rows */}
            <View style={[styles.profileInfoCard, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}>
              <View style={styles.profileInfoRow}>
                <Text style={[styles.profileInfoLabel, { color: colors.textMuted }]}>Email</Text>
                <Text style={[styles.profileInfoValue, { color: colors.textPrimary }]} numberOfLines={1}>
                  {user?.email ?? '-'}
                </Text>
              </View>

              <View style={[styles.rowDivider, { backgroundColor: colors.borderFaint }]} />

              <View style={styles.profileInfoRow}>
                <Text style={[styles.profileInfoLabel, { color: colors.textMuted }]}>Name</Text>
                <Text style={[styles.profileInfoValue, { color: colors.textPrimary }]}>
                  {user?.name ?? '-'}
                </Text>
              </View>

              <View style={[styles.rowDivider, { backgroundColor: colors.borderFaint }]} />

              <View style={styles.profileInfoRow}>
                <Text style={[styles.profileInfoLabel, { color: colors.textMuted }]}>Age range</Text>
                <Text style={[styles.profileInfoValue, { color: user?.age_range ? colors.textPrimary : colors.textMuted }]}>
                  {user?.age_range ? AGE_RANGE_LABELS[user.age_range] : 'Not set'}
                </Text>
              </View>
            </View>

            {/* Change email */}
            <TouchableOpacity
              style={[styles.profileActionRow, { backgroundColor: colors.card, borderColor: colors.borderFaint }]}
              activeOpacity={0.75}
              onPress={() => {
                setShowProfileModal(false);
                Alert.alert(
                  'Change email address',
                  'To change your email, contact us at support@dreamlog.app and we\'ll update it for you.',
                  [{ text: 'OK' }],
                );
              }}
            >
              <Text style={[styles.settingLabel, { color: colors.textPrimary }]}>Change email address</Text>
              <Text style={[styles.chevron, { color: colors.textMuted }]}>›</Text>
            </TouchableOpacity>

            <TouchableOpacity
              style={[styles.closeBtn, { borderColor: colors.borderFaint, marginTop: 8 }]}
              onPress={() => setShowProfileModal(false)}
            >
              <Text style={[styles.closeBtnText, { color: colors.textMuted }]}>Close</Text>
            </TouchableOpacity>
          </View>
        </BlurView>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 20, paddingBottom: 60 },

  goalSheet: {
    maxHeight: '80%',
  },
  goalScrollList: {
    flexGrow: 0,
  },
  goalPickerCard: {
    borderWidth: 1,
    borderRadius: 12,
    padding: 14,
    marginBottom: 8,
  },
  goalPickerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 3,
  },
  goalPickerLabel: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 14,
    flex: 1,
  },
  goalPickerCheck: {
    fontSize: 15,
    fontFamily: 'Nunito_600SemiBold',
  },
  goalPickerDesc: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 12,
    lineHeight: 17,
  },

  title: {
    fontSize: 26,
    fontFamily: 'CormorantGaramond_300Light',
    marginBottom: 24,
  },

  profileCard: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 16,
    borderRadius: 20,
    borderWidth: 1,
    padding: 20,
    marginBottom: 28,
  },
  avatar: {
    width: 48, height: 48, borderRadius: 24,
    alignItems: 'center', justifyContent: 'center',
    flexShrink: 0,
  },
  avatarText: {
    fontSize: 18,
    color: '#fff',
    fontFamily: 'CormorantGaramond_500Medium',
  },
  profileName: {
    fontSize: 16,
    fontFamily: 'Nunito_600SemiBold',
    marginBottom: 2,
  },
  profileSub: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
  },

  sectionLabel: {
    fontSize: 10,
    fontFamily: 'Nunito_600SemiBold',
    letterSpacing: 1.5,
    marginBottom: 8,
    marginTop: 4,
  },
  section: {
    borderRadius: 16,
    borderWidth: 1,
    marginBottom: 24,
    overflow: 'hidden',
  },
  settingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 18,
    paddingVertical: 14,
    gap: 12,
  },
  rowDivider: {
    height: 1,
    marginLeft: 18,
  },
  settingLabel: {
    fontSize: 15,
    fontFamily: 'Nunito_400Regular',
  },
  settingSub: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    marginTop: 2,
    lineHeight: 18,
  },
  chevron: {
    fontSize: 20,
    lineHeight: 22,
  },
  valueText: {
    fontSize: 14,
    fontFamily: 'Nunito_400Regular',
  },

  planBadge: {
    borderRadius: 10,
    borderWidth: 1,
    paddingHorizontal: 10,
    paddingVertical: 4,
  },
  planBadgeText: {
    fontSize: 12,
    fontFamily: 'Nunito_600SemiBold',
  },
  upgradeRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 18,
    paddingVertical: 14,
  },
  upgradeText: {
    fontSize: 14,
    fontFamily: 'Nunito_600SemiBold',
  },

  signOutBtn: {
    borderRadius: 14,
    borderWidth: 1,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 8,
  },
  signOutText: {
    fontSize: 15,
    fontFamily: 'Nunito_600SemiBold',
  },

  modalOverlay: {
    flex: 1,
    justifyContent: 'flex-end',
  },
  modalDismiss: {
    flex: 1,
  },
  modalSheet: {
    borderTopLeftRadius: 28,
    borderTopRightRadius: 28,
    borderWidth: 1,
    borderBottomWidth: 0,
    paddingTop: 12,
    paddingHorizontal: 20,
    paddingBottom: 40,
    maxHeight: '70%',
  },
  modalHandle: {
    width: 36,
    height: 4,
    borderRadius: 2,
    alignSelf: 'center',
    marginBottom: 20,
  },
  modalTitle: {
    fontSize: 18,
    fontFamily: 'CormorantGaramond_500Medium',
    marginBottom: 4,
  },
  modalSub: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
    marginBottom: 16,
    lineHeight: 20,
  },
  hourList: { flexGrow: 0 },
  hourRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 14,
    borderBottomWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 4,
  },
  hourLabel: {
    fontSize: 16,
    fontFamily: 'Nunito_400Regular',
  },
  hourCheck: {
    fontSize: 16,
    fontFamily: 'Nunito_600SemiBold',
  },

  profileModalHeader: {
    alignItems: 'center',
    marginBottom: 20,
  },
  profileModalAvatar: {
    width: 64,
    height: 64,
    borderRadius: 32,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 10,
  },
  profileModalAvatarText: {
    fontSize: 24,
    color: '#fff',
    fontFamily: 'CormorantGaramond_500Medium',
  },
  profileModalName: {
    fontSize: 20,
    fontFamily: 'CormorantGaramond_500Medium',
  },
  profileInfoCard: {
    borderRadius: 14,
    borderWidth: 1,
    marginBottom: 12,
    overflow: 'hidden',
  },
  profileInfoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 13,
  },
  profileInfoLabel: {
    fontFamily: 'Nunito_400Regular',
    fontSize: 13,
  },
  profileInfoValue: {
    fontFamily: 'Nunito_600SemiBold',
    fontSize: 14,
    maxWidth: '65%',
    textAlign: 'right',
  },
  profileActionRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    borderRadius: 14,
    borderWidth: 1,
    paddingHorizontal: 16,
    paddingVertical: 14,
    marginBottom: 4,
  },
  crisisCard: {
    borderWidth: 1,
    borderRadius: 12,
    padding: 14,
    marginBottom: 10,
  },
  crisisName: {
    fontSize: 15,
    fontFamily: 'Nunito_700Bold',
    marginBottom: 2,
  },
  crisisInfo: {
    fontSize: 13,
    fontFamily: 'Nunito_400Regular',
  },
  closeBtn: {
    borderWidth: 1,
    borderRadius: 12,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 4,
  },
  closeBtnText: {
    fontSize: 14,
    fontFamily: 'Nunito_600SemiBold',
  },
});
