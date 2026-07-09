import { useEffect, useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
  Image,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useLocalSearchParams } from 'expo-router';
import * as ImagePicker from 'expo-image-picker';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import { T } from '../../src/testIDs';
import type { ExternalClient } from '../../src/types';

type NoteMode = 'photo' | 'type';

// New session notes: pick a client, then either photograph handwritten notes
// (uploaded → OCR → editable bullets) or type them directly.
export default function AddSessionScreen() {
  const params = useLocalSearchParams<{ clientId?: string }>();
  const [clients, setClients] = useState<ExternalClient[]>([]);
  const [selectedClient, setSelectedClient] = useState<string | null>(params.clientId ?? null);
  const [noteMode, setNoteMode] = useState<NoteMode>('photo');
  const [photo, setPhoto] = useState<ImagePicker.ImagePickerAsset | null>(null);
  const [typedNotes, setTypedNotes] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const router = useRouter();
  const { colors } = useTheme();

  useEffect(() => {
    api.listExternalClients()
      .then(r => setClients(r.clients))
      .catch(() => {});
  }, []);

  const pickImage = async (fromCamera: boolean) => {
    setError('');
    const options: ImagePicker.ImagePickerOptions = {
      mediaTypes: ['images'],
      quality: 0.8,
      // JPEG keeps uploads small and is always accepted by the backend.
      allowsEditing: false,
    };
    const result = fromCamera
      ? await ImagePicker.launchCameraAsync(options)
      : await ImagePicker.launchImageLibraryAsync(options);
    if (!result.canceled && result.assets.length > 0) {
      setPhoto(result.assets[0]);
    }
  };

  const handleSubmit = async () => {
    if (!selectedClient) {
      setError('Pick a client first.');
      return;
    }
    if (noteMode === 'photo' && !photo) {
      setError('Take or choose a photo of your notes.');
      return;
    }
    const bullets = typedNotes
      .split('\n')
      .map(l => l.replace(/^[-•*]\s*/, '').trim())
      .filter(Boolean);
    if (noteMode === 'type' && bullets.length === 0) {
      setError('Type at least one note.');
      return;
    }

    setSubmitting(true);
    setError('');
    try {
      if (noteMode === 'photo' && photo) {
        const contentType =
          photo.mimeType === 'image/png' || photo.mimeType === 'image/webp'
            ? photo.mimeType
            : 'image/jpeg';
        const { upload_url, image_key } = await api.presignNotePhoto(
          photo.fileName ?? 'notes.jpg',
          contentType,
        );
        const blob = await (await fetch(photo.uri)).blob();
        const putResp = await fetch(upload_url, {
          method: 'PUT',
          headers: { 'Content-Type': contentType },
          body: blob,
        });
        if (!putResp.ok) throw new Error('Photo upload failed. Please try again.');
        const session = await api.createClientSession({
          external_client_id: selectedClient,
          image_key,
        });
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        router.replace(`/therapist/session/${session.id}` as any);
      } else {
        const session = await api.createClientSession({
          external_client_id: selectedClient,
          bullets,
        });
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        router.replace(`/therapist/session/${session.id}` as any);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not create the session.');
      setSubmitting(false);
    }
  };

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.bg }]}>
      <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
        <Text style={[styles.title, { color: colors.textPrimary }]}>New session notes</Text>

        {/* Client picker */}
        <Text style={[styles.label, { color: colors.textMuted }]}>CLIENT</Text>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.clientRow}>
          {clients.map(c => (
            <TouchableOpacity
              key={c.id}
              style={[
                styles.clientChip,
                { backgroundColor: colors.card, borderColor: colors.border },
                selectedClient === c.id && { backgroundColor: colors.purple600, borderColor: colors.purple600 },
              ]}
              onPress={() => { setSelectedClient(c.id); setError(''); }}
            >
              <Text
                style={[
                  styles.clientChipText,
                  { color: colors.textSecondary },
                  selectedClient === c.id && { color: '#fff' },
                ]}
              >
                {c.name}
              </Text>
            </TouchableOpacity>
          ))}
          <TouchableOpacity
            style={[styles.clientChip, { backgroundColor: 'transparent', borderColor: colors.border, borderStyle: 'dashed' }]}
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            onPress={() => router.push('/therapist/clients' as any)}
          >
            <Text style={[styles.clientChipText, { color: colors.textMuted }]}>＋ New client</Text>
          </TouchableOpacity>
        </ScrollView>

        {/* Mode switch */}
        <Text style={[styles.label, { color: colors.textMuted }]}>NOTES</Text>
        <View style={[styles.modeTabs, { backgroundColor: colors.card, borderColor: colors.border }]}>
          <TouchableOpacity
            testID={T.therapistPortal.addSessionPhotoTab}
            style={[styles.modeTab, noteMode === 'photo' && { backgroundColor: colors.purple600 }]}
            onPress={() => setNoteMode('photo')}
          >
            <Text style={[styles.modeTabText, { color: colors.textMuted }, noteMode === 'photo' && { color: '#fff' }]}>
              📷 Photo of notes
            </Text>
          </TouchableOpacity>
          <TouchableOpacity
            testID={T.therapistPortal.addSessionTypeTab}
            style={[styles.modeTab, noteMode === 'type' && { backgroundColor: colors.purple600 }]}
            onPress={() => setNoteMode('type')}
          >
            <Text style={[styles.modeTabText, { color: colors.textMuted }, noteMode === 'type' && { color: '#fff' }]}>
              ⌨️ Type notes
            </Text>
          </TouchableOpacity>
        </View>

        {noteMode === 'photo' ? (
          <View>
            {photo ? (
              <View style={[styles.previewWrap, { borderColor: colors.border }]}>
                <Image source={{ uri: photo.uri }} style={styles.preview} resizeMode="cover" />
                <TouchableOpacity style={styles.removePhoto} onPress={() => setPhoto(null)}>
                  <Text style={styles.removePhotoText}>✕ Remove</Text>
                </TouchableOpacity>
              </View>
            ) : (
              <View style={styles.photoBtns}>
                <TouchableOpacity
                  style={[styles.photoBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
                  onPress={() => pickImage(true)}
                >
                  <Text style={styles.photoBtnEmoji}>📷</Text>
                  <Text style={[styles.photoBtnText, { color: colors.textPrimary }]}>Camera</Text>
                </TouchableOpacity>
                <TouchableOpacity
                  style={[styles.photoBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
                  onPress={() => pickImage(false)}
                >
                  <Text style={styles.photoBtnEmoji}>🖼️</Text>
                  <Text style={[styles.photoBtnText, { color: colors.textPrimary }]}>Gallery</Text>
                </TouchableOpacity>
              </View>
            )}
            <Text style={[styles.hint, { color: colors.textMuted }]}>
              We extract the text from your photo into an editable bullet list, then permanently
              delete the photo. Notes are stored encrypted.
            </Text>
          </View>
        ) : (
          <View>
            <TextInput
              testID={T.therapistPortal.addSessionBulletsInput}
              style={[styles.notesInput, { backgroundColor: colors.card, borderColor: colors.border, color: colors.textPrimary }]}
              value={typedNotes}
              onChangeText={v => { setTypedNotes(v); setError(''); }}
              placeholder={'One note per line, e.g.\nClient reported better sleep\nDiscussed workplace boundary'}
              placeholderTextColor={colors.textFaint}
              multiline
              textAlignVertical="top"
            />
            <Text style={[styles.hint, { color: colors.textMuted }]}>Each line becomes one bullet.</Text>
          </View>
        )}

        {!!error && <Text style={styles.errorText}>{error}</Text>}

        <TouchableOpacity
          testID={T.therapistPortal.addSessionSubmit}
          style={[styles.button, { backgroundColor: colors.purple600, shadowColor: colors.purple500 }, submitting && styles.buttonDisabled]}
          onPress={handleSubmit}
          disabled={submitting}
          activeOpacity={0.85}
        >
          {submitting ? (
            <ActivityIndicator color="#fff" size="small" />
          ) : (
            <Text style={styles.buttonText}>{noteMode === 'photo' ? 'Upload & extract' : 'Save notes'}</Text>
          )}
        </TouchableOpacity>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 20, paddingBottom: 48 },
  title: { fontSize: 28, fontFamily: 'CormorantGaramond_500Medium', marginBottom: 22 },
  label: { fontSize: 11, fontFamily: 'Nunito_700Bold', letterSpacing: 1.5, marginBottom: 10 },
  clientRow: { gap: 8, paddingBottom: 20 },
  clientChip: {
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  clientChipText: { fontSize: 13.5, fontFamily: 'Nunito_600SemiBold' },
  modeTabs: {
    flexDirection: 'row',
    borderRadius: 14,
    borderWidth: 1,
    padding: 4,
    marginBottom: 16,
  },
  modeTab: { flex: 1, paddingVertical: 10, alignItems: 'center', borderRadius: 10 },
  modeTabText: { fontSize: 13.5, fontFamily: 'Nunito_600SemiBold' },
  photoBtns: { flexDirection: 'row', gap: 10 },
  photoBtn: {
    flex: 1,
    borderRadius: 16,
    borderWidth: 1,
    alignItems: 'center',
    paddingVertical: 26,
  },
  photoBtnEmoji: { fontSize: 28, marginBottom: 6 },
  photoBtnText: { fontSize: 14, fontFamily: 'Nunito_600SemiBold' },
  previewWrap: { borderRadius: 16, borderWidth: 1, overflow: 'hidden' },
  preview: { width: '100%', height: 260 },
  removePhoto: {
    position: 'absolute',
    top: 10,
    right: 10,
    backgroundColor: 'rgba(0,0,0,0.65)',
    borderRadius: 8,
    paddingHorizontal: 10,
    paddingVertical: 6,
  },
  removePhotoText: { color: '#fff', fontSize: 12, fontFamily: 'Nunito_600SemiBold' },
  hint: { fontSize: 12, fontFamily: 'Nunito_400Regular', lineHeight: 17, marginTop: 10 },
  notesInput: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 16,
    minHeight: 180,
    fontSize: 14.5,
    fontFamily: 'Nunito_400Regular',
    lineHeight: 21,
  },
  errorText: { fontSize: 13, color: '#ef4444', fontFamily: 'Nunito_400Regular', marginTop: 14 },
  button: {
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 20,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 12,
    elevation: 6,
  },
  buttonDisabled: { opacity: 0.6 },
  buttonText: { color: '#fff', fontSize: 16, fontFamily: 'Nunito_600SemiBold', letterSpacing: 0.5 },
});
