# Approved Mood refinement

Implemented from the approved Desktop preview on 19 October 2026.

## App behavior

- Mood Trend always renders seven UTC calendar days including today, matching the analytics API's date buckets. Missing journal days remain gaps. Older/future responses are excluded from the trend and its mood labels.
- Trend/history bars are at most 5px; annual review bars and Emotion Patterns tracks are 3px. Dense history ranges use smaller gaps.
- Dedicated streak sharing and the milestone Share action are removed. Insight-card sharing remains.
- Renew streak shows the available freeze count. Active streaks and zero freezes explain why renewal is disabled. Busy actions cannot be submitted twice.
- The lost-streak notice lives above navigation. It checks once per app launch after authenticated startup data is available, and waits for the brand splash, auth sheet and force-update gate. Visiting tabs, resuming, or signing in again within that mounted launch does not repeat it. Manual renewal remains available.
- After a successful freeze acknowledgment, a failed streak refresh retains that acknowledgment. Refresh streak only reads the result; dismissing/reopening the dialog does not consume another freeze. This is a client-side guarantee for acknowledged responses, not server idempotency for an ambiguous network timeout.
- Weekly/yearly reviews now live in Explore > Reflections, alongside the existing journal and tools. Mood links directly to the top of that section. Explore refetches on focus and distinguishes an absent/pending review from a service failure, with retry and partial success.
- Weekly text uses actual native line measurement. More than three lines gets Show more / Show less, with an accessible 44px action. The hidden full-text measurement is excluded from accessibility.
- Real data comes from the existing typed API client. Preview sample data and adapters were not added to the app.

## Implementation boundaries

`StreakProvider` is mounted in the existing root layout. It clears private data on sign-out and ignores stale reads or renewal responses from an earlier authenticated session. Mood refreshes the shared streak on focus. Its existing graph range loading/error/race handling is retained.

Reusable components: `ReflectionsSection`, `ReviewCards`, `WeeklyNarrative`, `RenewStreakModal`. Date-window helpers live in `src/utils/moodDays.ts`. No new dependencies, API endpoints, backend changes or publication/deployment.

## Verification

- Full mobile Jest suite: **25 suites / 284 tests passed**. A final milestone-preservation refinement was then checked with **all 9 Mood tests passing**.
- TypeScript: `npx tsc --noEmit` passed.
- Android Metro export with Hermes bytecode passed (2,191 modules). The sandbox temp-folder write failure was resolved by using the task workspace for TEMP/TMP; no compiler checks were disabled.
- Device-runner helper tests: **14 passed**.
- All **41 Maestro YAML files** parse and their subflow references resolve.
- Mood device flows now inspect/dismiss renewal without spending real freezes, check removal of streak sharing, and navigate to review sharing/expansion in Explore. Existing tool navigation flows explicitly scroll their controls into view now that reviews occupy space in Explore.
- No Android device was connected; Maestro/native interaction, visual appearance and real-service smoke checks remain pending. The startup-dismiss helper is conditional and should be exercised with a broken-streak fixture when a device is available.

The design assessment remains **65/100 provisional**, with AI-generated appearance **4/10** (lower is better). This implementation does not establish a new visual score.
