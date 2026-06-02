import '@testing-library/jest-native/extend-expect';

// Mock Async Storage
jest.mock('@react-native-async-storage/async-storage', () =>
  require('@react-native-async-storage/async-storage/jest/async-storage-mock')
);

// Mock NetInfo
jest.mock('@react-native-community/netinfo', () => ({
  addEventListener: jest.fn(),
  fetch: jest.fn(() => Promise.resolve({ isConnected: true })),
  useNetInfo: jest.fn(() => ({ isConnected: true })),
}));

// Mock Expo Secure Store
jest.mock('expo-secure-store', () => ({
  setItemAsync: jest.fn(() => Promise.resolve()),
  getItemAsync: jest.fn(() => Promise.resolve('mock-token')),
  deleteItemAsync: jest.fn(() => Promise.resolve()),
}));

// Mock Expo AV (Audio Video)
jest.mock('expo-av', () => ({
  Audio: {
    Recording: jest.fn().mockImplementation(() => ({
      prepareToRecordAsync: jest.fn(() => Promise.resolve()),
      startAsync: jest.fn(() => Promise.resolve()),
      stopAndUnloadAsync: jest.fn(() => Promise.resolve()),
      getURI: jest.fn(() => 'mock-uri.m4a'),
      setOnRecordingStatusUpdate: jest.fn(),
    })),
    setAudioModeAsync: jest.fn(() => Promise.resolve()),
    RecordingOptionsPresets: {
      HIGH_QUALITY: {},
    },
  },
}));

// Mock Expo Font
jest.mock('expo-font', () => ({
  loadAsync: jest.fn(() => Promise.resolve()),
  isLoaded: jest.fn(() => true),
}));

// Mock react-native-worklets
jest.mock('react-native-worklets', () => ({
  Worklets: {
    createRunInJS: (fn) => fn,
    createRunInWorklet: (fn) => fn,
  },
  createSerializable: (val) => val,
  isWorklet: () => false,
  useWorklet: (fn) => fn,
  serializableMappingCache: new Map(),
  RuntimeKind: {},
  isWorkletFunction: () => false,
  runOnUI: (fn) => fn,
}));

// Mock React Native Reanimated
require('react-native-reanimated/mock');

// Mock React Native View Shot (used for shareable cards)
jest.mock('react-native-view-shot', () => 'ViewShot');

// Mock Safe Area Context
jest.mock('react-native-safe-area-context', () => {
  const inset = { top: 0, right: 0, bottom: 0, left: 0 };
  return {
    SafeAreaProvider: ({ children }) => children,
    SafeAreaView: ({ children }) => children,
    useSafeAreaInsets: () => inset,
    useSafeAreaFrame: () => ({ x: 0, y: 0, width: 390, height: 844 }),
  };
});
