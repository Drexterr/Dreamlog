/**
 * ProcessingScreen — shown while the backend processes the audio.
 * Uses a breathing pulse animation — no spinner, intentional feel.
 */

import React, { useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Animated,
  SafeAreaView,
  StatusBar,
} from 'react-native';

const BREATHING_MESSAGES = [
  'Listening to your words…',
  'Finding the threads…',
  'Sitting with what you shared…',
  'Almost ready…',
];

export function ProcessingScreen() {
  const scaleAnim = useRef(new Animated.Value(1)).current;
  const opacityAnim = useRef(new Animated.Value(0.4)).current;
  const messageIndexRef = useRef(0);
  const [messageIndex, setMessageIndex] = React.useState(0);

  useEffect(() => {
    // Breathing animation: inhale 3s, exhale 3s, repeat.
    const breathe = () =>
      Animated.sequence([
        Animated.parallel([
          Animated.timing(scaleAnim, {
            toValue: 1.35,
            duration: 3000,
            useNativeDriver: true,
          }),
          Animated.timing(opacityAnim, {
            toValue: 0.9,
            duration: 3000,
            useNativeDriver: true,
          }),
        ]),
        Animated.parallel([
          Animated.timing(scaleAnim, {
            toValue: 1,
            duration: 3000,
            useNativeDriver: true,
          }),
          Animated.timing(opacityAnim, {
            toValue: 0.4,
            duration: 3000,
            useNativeDriver: true,
          }),
        ]),
      ]);

    const loop = Animated.loop(breathe());
    loop.start();
    return () => loop.stop();
  }, []);

  // Rotate through messages every 4 seconds.
  useEffect(() => {
    const interval = setInterval(() => {
      messageIndexRef.current = (messageIndexRef.current + 1) % BREATHING_MESSAGES.length;
      setMessageIndex(messageIndexRef.current);
    }, 4000);
    return () => clearInterval(interval);
  }, []);

  return (
    <SafeAreaView style={styles.container}>
      <StatusBar barStyle="light-content" />
      <View style={styles.center}>
        {/* Breathing orb */}
        <View style={styles.orbWrapper}>
          <Animated.View
            style={[
              styles.orbOuter,
              { transform: [{ scale: scaleAnim }], opacity: opacityAnim },
            ]}
          />
          <View style={styles.orbInner} />
        </View>

        <Text style={styles.message}>{BREATHING_MESSAGES[messageIndex]}</Text>
        <Text style={styles.subtext}>This usually takes 10–15 seconds</Text>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0f0f1a' },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 36 },

  orbWrapper: { width: 120, height: 120, alignItems: 'center', justifyContent: 'center' },
  orbOuter: {
    position: 'absolute',
    width: 100,
    height: 100,
    borderRadius: 50,
    backgroundColor: '#1e3a5f',
  },
  orbInner: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: '#3b82f6',
    opacity: 0.9,
  },

  message: {
    fontSize: 17,
    color: '#d1d5db',
    fontWeight: '300',
    textAlign: 'center',
    paddingHorizontal: 40,
  },
  subtext: { fontSize: 13, color: '#374151' },
});
