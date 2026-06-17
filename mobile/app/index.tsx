import { View } from 'react-native';

// Default entry point. The root layout (_layout.tsx) handles all routing decisions
// (onboarding vs tabs) via useEffect once session + fonts are ready.
// This screen is only visible for the fraction of a second before that redirect fires.
export default function Index() {
  return <View style={{ flex: 1, backgroundColor: '#0f0c1e' }} />;
}
