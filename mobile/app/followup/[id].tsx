import { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TextInput,
  FlatList,
  TouchableOpacity,
  KeyboardAvoidingView,
  Platform,
  StyleSheet,
  
  StatusBar,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { api } from '../../src/api/client';
import { useTheme } from '../../src/context/ThemeContext';
import type { Conversation, ConversationMessage } from '../../src/types';

const MAX_TURNS = 3;

function MessageBubble({ message, colors }: { message: ConversationMessage; colors: any }) {
  const isUser = message.role === 'user';
  return (
    <View style={[
      styles.bubble,
      isUser
        ? [styles.userBubble, { backgroundColor: colors.brandGlow, borderColor: colors.border }]
        : [styles.assistantBubble, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint }],
    ]}>
      <Text style={[
        styles.bubbleText,
        { color: isUser ? colors.textPrimary : colors.textPrimary },
        isUser ? styles.userText : styles.assistantText,
      ]}>
        {message.content}
      </Text>
    </View>
  );
}

function TypingBubble({ colors }: { colors: any }) {
  return (
    <View style={[styles.assistantBubble, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint }]}>
      <Text style={[styles.assistantText, { color: colors.textPrimary, letterSpacing: 4, fontSize: 18 }]}>···</Text>
    </View>
  );
}

export default function FollowUpScreen() {
  const { id: entryId } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { colors } = useTheme();

  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [initializing, setInitializing] = useState(true);
  const listRef = useRef<FlatList>(null);

  useEffect(() => {
    if (!entryId) return;
    api.getOrCreateConversation(entryId)
      .then((conv) => {
        setConversation(conv);
        if (conv.messages.length > 0) {
          setMessages(conv.messages);
        }
        if (conv.messages.length === 0) {
          setMessages([{
            id: 'seed',
            conversation_id: conv.id,
            role: 'assistant',
            content: "What would you like to explore further from your reflection tonight?",
            created_at: new Date().toISOString(),
          }]);
        }
      })
      .catch(() => {})
      .finally(() => setInitializing(false));
  }, [entryId]);

  const sendMessage = useCallback(async () => {
    if (!conversation || !input.trim() || loading) return;

    const content = input.trim();
    setInput('');
    setLoading(true);

    const optimistic: ConversationMessage = {
      id: String(Date.now()),
      conversation_id: conversation.id,
      role: 'user',
      content,
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, optimistic]);
    setTimeout(() => listRef.current?.scrollToEnd({ animated: true }), 80);

    try {
      const updated = await api.sendConversationMessage(conversation.id, content);
      setConversation(updated);
      setMessages(updated.messages);
    } catch {
      setMessages((prev) => prev.filter((m) => m.id !== optimistic.id));
      setInput(content);
    } finally {
      setLoading(false);
      setTimeout(() => listRef.current?.scrollToEnd({ animated: true }), 80);
    }
  }, [conversation, input, loading]);

  const turnsUsed = conversation?.turn_count ?? 0;
  const turnsLeft = MAX_TURNS - turnsUsed;
  const isClosed = conversation?.is_closed ?? false;
  const canSend = !isClosed && turnsLeft > 0 && input.trim().length > 0 && !loading;

  if (initializing) {
    return (
      <View style={[styles.container, { backgroundColor: colors.bg }]}>
        <SafeAreaView style={styles.center}>
          <ActivityIndicator color={colors.purple400} />
        </SafeAreaView>
      </View>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.bg }]}>
      <StatusBar barStyle="light-content" />
      <SafeAreaView style={{ flex: 1 }}>
        {/* Header */}
        <View style={[styles.header, { borderBottomColor: colors.borderFaint }]}>
          <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Continue</Text>
          <TouchableOpacity onPress={() => router.back()} style={styles.closeBtn}>
            <Text style={[styles.closeText, { color: colors.textMuted }]}>Goodnight</Text>
          </TouchableOpacity>
        </View>

        {/* Turn counter dots */}
        {!isClosed && (
          <View style={styles.turnsRow}>
            {Array.from({ length: MAX_TURNS }).map((_, i) => (
              <View
                key={i}
                style={[
                  styles.turnDot,
                  { backgroundColor: colors.cardSolid, borderColor: colors.border },
                  i < turnsUsed && { backgroundColor: colors.purple500, borderColor: colors.purple500 },
                ]}
              />
            ))}
            <Text style={[styles.turnsLabel, { color: colors.textMuted }]}>
              {turnsLeft > 0
                ? `${turnsLeft} exchange${turnsLeft !== 1 ? 's' : ''} left`
                : 'Last exchange'}
            </Text>
          </View>
        )}

        {/* Messages */}
        <FlatList
          ref={listRef}
          data={messages}
          keyExtractor={(m) => m.id}
          contentContainerStyle={styles.messageList}
          onContentSizeChange={() => listRef.current?.scrollToEnd({ animated: false })}
          removeClippedSubviews
          maxToRenderPerBatch={10}
          windowSize={5}
          renderItem={({ item }) => <MessageBubble message={item} colors={colors} />}
          ListFooterComponent={loading ? <TypingBubble colors={colors} /> : null}
        />

        {/* Input / closed banner */}
        {isClosed ? (
          <View style={[styles.closedBanner, { borderTopColor: colors.borderFaint }]}>
            <Text style={[styles.closedText, { color: colors.textMuted }]}>This reflection is complete. Goodnight.</Text>
            <TouchableOpacity
              style={[styles.goodnightBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
              onPress={() => router.replace('/(tabs)')}
              activeOpacity={0.8}
            >
              <Text style={[styles.goodnightText, { color: colors.purple300 }]}>Close</Text>
            </TouchableOpacity>
          </View>
        ) : (
          <KeyboardAvoidingView
            behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
            keyboardVerticalOffset={8}
          >
            <View style={[styles.inputRow, { borderTopColor: colors.borderFaint }]}>
              <TextInput
                style={[styles.input, { backgroundColor: colors.cardSolid, borderColor: colors.borderFaint, color: colors.textPrimary }]}
                value={input}
                onChangeText={setInput}
                placeholder="Type a response…"
                placeholderTextColor={colors.textFaint}
                multiline
                maxLength={2000}
                returnKeyType="send"
                blurOnSubmit={false}
                onSubmitEditing={sendMessage}
              />
              <TouchableOpacity
                style={[styles.sendBtn, { backgroundColor: colors.card, borderColor: colors.border }, !canSend && styles.sendBtnDisabled]}
                onPress={sendMessage}
                disabled={!canSend}
                activeOpacity={0.8}
              >
                <Text style={[styles.sendArrow, { color: colors.purple300 }]}>→</Text>
              </TouchableOpacity>
            </View>
          </KeyboardAvoidingView>
        )}
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },

  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderBottomWidth: 1,
  },
  headerTitle: {
    fontSize: 17,
    fontFamily: 'Nunito_600SemiBold',
  },
  closeBtn: { padding: 6 },
  closeText: {
    fontSize: 15,
    fontFamily: 'Nunito_400Regular',
  },

  turnsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 20,
    paddingVertical: 10,
    gap: 6,
  },
  turnDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    borderWidth: 1,
  },
  turnsLabel: {
    fontSize: 12,
    fontFamily: 'Nunito_400Regular',
    marginLeft: 4,
  },

  messageList: { paddingHorizontal: 16, paddingTop: 16, paddingBottom: 20 },
  bubble: {
    maxWidth: '82%',
    borderRadius: 18,
    padding: 14,
    marginBottom: 10,
    borderWidth: 1,
  },
  userBubble: { alignSelf: 'flex-end' },
  assistantBubble: { alignSelf: 'flex-start' },
  bubbleText: { fontSize: 15, lineHeight: 24 },
  userText: { fontFamily: 'Nunito_400Regular' },
  assistantText: {
    fontFamily: 'CormorantGaramond_300Light',
    fontWeight: '300',
  },

  inputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderTopWidth: 1,
    gap: 10,
  },
  input: {
    flex: 1,
    borderRadius: 20,
    borderWidth: 1,
    paddingHorizontal: 16,
    paddingVertical: 10,
    fontFamily: 'Nunito_400Regular',
    fontSize: 15,
    maxHeight: 120,
  },
  sendBtn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    borderWidth: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendBtnDisabled: { opacity: 0.3 },
  sendArrow: {
    fontSize: 18,
    lineHeight: 22,
  },

  closedBanner: {
    padding: 24,
    alignItems: 'center',
    borderTopWidth: 1,
    gap: 16,
  },
  closedText: {
    fontSize: 15,
    fontFamily: 'CormorantGaramond_400Regular',
    textAlign: 'center',
    lineHeight: 24,
  },
  goodnightBtn: {
    paddingVertical: 12,
    paddingHorizontal: 32,
    borderRadius: 14,
    borderWidth: 1,
  },
  goodnightText: {
    fontSize: 15,
    fontFamily: 'Nunito_600SemiBold',
  },
});
