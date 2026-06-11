import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { Colors, Fonts } from '../theme';
import { VersionInfo } from '../types';
import { openStoreListing } from '../services/version';

interface Props {
  info: VersionInfo;
}

// Full-screen blocking gate shown when the installed app version is below the
// backend's minimum_version. There is intentionally no way to dismiss it.
export default function ForceUpdateScreen({ info }: Props) {
  return (
    <View style={styles.container}>
      <Text style={styles.emoji}>✦</Text>
      <Text style={styles.title}>Time for an update</Text>
      <Text style={styles.body}>
        This version of DreamLog is no longer supported. Update to the latest
        version to keep journaling.
      </Text>
      <TouchableOpacity
        style={styles.button}
        activeOpacity={0.85}
        onPress={() => openStoreListing(info)}
      >
        <Text style={styles.buttonText}>Update Now</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: Colors.bg,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 36,
    zIndex: 1000,
  },
  emoji: {
    fontSize: 40,
    color: Colors.purple400,
    marginBottom: 20,
  },
  title: {
    fontFamily: Fonts.serif,
    fontSize: 32,
    color: Colors.textPrimary,
    marginBottom: 14,
    textAlign: 'center',
  },
  body: {
    fontFamily: Fonts.sans,
    fontSize: 15,
    lineHeight: 22,
    color: Colors.textSecondary,
    textAlign: 'center',
    marginBottom: 32,
  },
  button: {
    backgroundColor: Colors.purple600,
    borderRadius: 28,
    paddingVertical: 15,
    paddingHorizontal: 48,
  },
  buttonText: {
    fontFamily: Fonts.sansSB,
    fontSize: 16,
    color: Colors.textPrimary,
  },
});
