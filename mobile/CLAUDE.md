# Mobile - Claude Guidance

Read this before touching any React Native / Expo code in this directory.

Also read:
- `../docs/API_CONTRACT.md` - backend API shapes the mobile consumes
- `../docs/ROADMAP.md` - Phase 4 features planned for mobile

---

## Project Structure

```
app/                        expo-router file-based routes
  _layout.tsx               root layout - fonts, auth guard, theme provider
  auth.tsx                  JWT paste screen (dev) / Supabase auth screen (prod)
  record.tsx                audio recording UI
  processing/[id].tsx       polling screen - waits for status=completed
  reflection/[id].tsx       displays AI reflection
  followup/[id].tsx         3-turn follow-up conversation UI
  (tabs)/
    _layout.tsx             bottom tab navigator
    index.tsx               home / journal feed
    timeline.tsx            paginated entry list with analysis previews
    mood.tsx                7-day mood chart + streak display
    settings.tsx            user preferences

src/
  api/client.ts             typed Axios instance - USE THIS for all API calls
  hooks/useRecorder.ts      audio recording state machine - do not re-implement recording logic
  services/upload.ts        presign → PUT → POST orchestration with backoff
  services/offlineQueue.ts  AsyncStorage-based retry queue
  theme.ts                  design tokens - always import from here
  types/index.ts            shared TypeScript types
  screens/                  screen-level components (used by app/ routes)
  components/               reusable UI components
```

---

## Rules - Always Follow

### API Calls
- ALWAYS use `src/api/client.ts` for API calls - never import Axios directly
- The client reads JWT from expo-secure-store and injects the Authorization header automatically
- Add new typed API functions to `src/api/client.ts`, not inline in screens

### Navigation
- Use expo-router's typed `href` - `experiments.typedRoutes: true` is enabled in `app.json`
- Do NOT use `router.push('/some-string')` - use the typed form
- File-based routing: adding a screen means adding a file in `app/`, not registering anything manually

### Styling & Theme
- ALWAYS import colors, fonts, and spacing from `src/theme.ts`
- Never hardcode hex colors or font names inline
- Dark purple palette - do not introduce new colors without adding them to `theme.ts` first
- Fonts: Cormorant Garamond (serif, headings) + Nunito (sans, body)

### State Management
- No Redux, Zustand, or any global state library
- Component state via `useState` / `useReducer`
- Cross-screen persistence: AsyncStorage (non-sensitive) or expo-secure-store (JWT only)
- If a new feature seems to need global state, first try lifting state to the nearest common parent

### Audio Recording
- `src/hooks/useRecorder.ts` is the single recording state machine - do not add recording logic elsewhere
- Format: AAC, 44.1 kHz, mono
- Max duration: 30 minutes (enforced in useRecorder)
- Do not use `expo-av` directly in screens - go through the hook

### Upload Flow
- Audio upload is a 3-step sequence: presign → PUT to storage → POST to /entries
- This lives in `src/services/upload.ts` with 3-attempt exponential backoff
- Do not re-implement this inline in screens
- Failed uploads queue in `src/services/offlineQueue.ts` - auto-flush on reconnect

### TypeScript
- All API response types are defined in `src/types/index.ts` - keep them in sync with `../docs/API_CONTRACT.md`
- Never use `any` - use `unknown` and narrow if needed
- All new screens and components need proper TypeScript typing

---

## Adding a New Screen

1. Create file in `app/` (or `app/(tabs)/` for tab screens)
2. expo-router auto-registers it - no manual registration
3. Use typed `href` from expo-router for navigation to it
4. Pull data via `src/api/client.ts`
5. Style using `src/theme.ts` tokens

## Adding a New API Call

1. Add the function to `src/api/client.ts` with proper TypeScript return type
2. Add the response type to `src/types/index.ts`
3. Use the function in your screen/component

---

## Audio Permissions

Both platforms require permissions before recording. The useRecorder hook handles permission requests. If you're adding a new recording surface, check that permissions are granted before showing record UI.

- iOS: `NSMicrophoneUsageDescription` in `app.json` (already set)
- Android: `RECORD_AUDIO` permission in `app.json` (already set)

---

## Key Dependencies

```json
expo-av           audio recording + playback
expo-router       file-based navigation with typed routes
expo-secure-store JWT storage
expo-font         font loading (Cormorant Garamond + Nunito)
axios             HTTP client (always via src/api/client.ts)
@react-native-async-storage/async-storage   offline queue + non-sensitive persistence
```

---

## Dev Setup

```bash
npm install
npx expo start

# Platform-specific
npx expo start --android
npx expo start --ios

# Set API URL in mobile/.env
EXPO_PUBLIC_API_URL=http://10.0.2.2:8080    # Android emulator
EXPO_PUBLIC_API_URL=http://localhost:8080    # iOS simulator / web
```

Auth in dev: generate JWT at jwt.io with payload `{"sub":"test-user-001","email":"test@dreamlog.dev"}` signed with `SUPABASE_JWT_SECRET` from root `.env`. Paste into the auth screen.

---

## What Not To Do

- Do not `import axios from 'axios'` directly in any screen or component
- Do not hardcode `http://localhost:8080` anywhere - always read from `EXPO_PUBLIC_API_URL`
- Do not use `StyleSheet.create` with hardcoded color values - use theme tokens
- Do not add a global state library without a concrete case that hooks can't solve
- Do not skip TypeScript types for API responses - they catch API contract breakage at compile time
