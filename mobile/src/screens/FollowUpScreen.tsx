/**
 * FollowUpScreen - "Tell me more" bounded conversation.
 *
 * Rules:
 * - Max 3 user turns (enforced by backend + shown in UI)
 * - Opens with the reflection question already displayed as first assistant message
 * - Ends gracefully when turns are exhausted or user taps "Goodnight"
 */

import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TextInput,
  FlatList,
  TouchableOpacity,
  KeyboardAvoidingView,
  Platform,
  StyleSheet,
  SafeAreaView,
  ActivityIndicator,
} from 'react-native';
import { api } from '../api/client';
import { Conversation, ConversationMessage, Entry, EntryAnalysis } from '../types';

const MAX_TURNS = 3;

interface Props {
  entry: Entry;
  analysis: EntryAnalysis;
  onClose: () => void;
}

export function FollowUpScreen({ entry, analysis, onClose }: Props) {
  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [initializing, setInitializing] = useState(true);
  const listRef = useRef<FlatList>(null);

  useEffect(() => {
    initConversation();
  }, []);

  const initConversation = async () => {
    try {
      const conv = await api.getOrCreateConversation(entry.id);
      setConversation(conv);

      // If no messages yet, seed the opening question as the first assistant message.
      if (conv.messages.length === 0) {
        // Extract the question from the reflection (last sentence).
        const question = extractQuestion(analysis.reflection);
        setMessages([
          {
            id: 'seed',
            conversation_id: conv.id,
            role: 'assistant',
            content: question,
            created_at: new Date().toISOString(),
          },
        ]);
      } else {
        setMessages(conv.messages);
      }
    } catch (err) {
      console.error('init conversation', err);
    } finally {
      setInitializing(false);
    }
  };

  const sendMessage = useCallback(async () => {
    if (!conversation || !input.trim() || loading) return;

    const userMsg = input.trim();
    setInput('');
    setLoading(true);

    // Optimistic: add user message immediately.
    const optimistic: ConversationMessage = {
      id: String(Date.now()),
      conversation_id: conversation.id,
      role: 'user',
      content: userMsg,
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, optimistic]);
    setTimeout(() => listRef.current?.scrollToEnd({ animated: true }), 100);

    try {
      const updated = await api.sendConversationMessage(conversation.id, userMsg);
      setConversation(updated);
      setMessages(updated.messages);
    } catch (err) {
      console.error('send message', err);
      // Remove optimistic message on error.
      setMessages((prev) => prev.filter((m) => m.id !== optimistic.id));
      setInput(userMsg);
    } finally {
      setLoading(false);
      setTimeout(() => listRef.current?.scrollToEnd({ animated: true }), 100);
    }
  }, [conversation, input, loading]);

  const turnsUsed = conversation?.turn_count ?? 0;
  const turnsLeft = MAX_TURNS - turnsUsed;
  const isClosed = conversation?.is_closed ?? false;
  const canSend = !isClosed && turnsLeft > 0 && input.trim().length > 0 && !loading;

  if (initializing) {
    return (
      <SafeAreaView style={styles.container}>
        <ActivityIndicator color="#3b82f6" style={{ marginTop: 80 }} />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Continue</Text>
        <TouchableOpacity onPress={onClose} style={styles.closeButton}>
          <Text style={styles.closeText}>Goodnight</Text>
        </TouchableOpacity>
      </View>

      {/* Turn counter */}
      {!isClosed && (
        <View style={styles.turnsRow}>
          {Array.from({ length: MAX_TURNS }).map((_, i) => (
            <View
              key={i}
              style={[styles.turnDot, i < turnsUsed && styles.turnDotUsed]}
            />
          ))}
          <Text style={styles.turnsLabel}>
            {turnsLeft > 0 ? `${turnsLeft} exchange${turnsLeft !== 1 ? 's' : ''} left` : 'Last exchange'}
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
        renderItem={({ item }) => <MessageBubble message={item} />}
        ListFooterComponent={loading ? <TypingIndicator /> : null}
      />

      {/* Closed state */}
      {isClosed ? (
        <View style={styles.closedBanner}>
          <Text style={styles.closedText}>This reflection is complete. Goodnight.</Text>
          <TouchableOpacity style={styles.goodnightBtn} onPress={onClose}>
            <Text style={styles.goodnightBtnText}>Close</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          keyboardVerticalOffset={8}
        >
          <View style={styles.inputRow}>
            <TextInput
              style={styles.input}
              value={input}
              onChangeText={setInput}
              placeholder="Type a response…"
              placeholderTextColor="#4b5563"
              multiline
              maxLength={2000}
              returnKeyType="send"
              onSubmitEditing={sendMessage}
            />
            <TouchableOpacity
              style={[styles.sendButton, !canSend && styles.sendButtonDisabled]}
              onPress={sendMessage}
              disabled={!canSend}
            >
              <Text style={styles.sendText}>→</Text>
            </TouchableOpacity>
          </View>
        </KeyboardAvoidingView>
      )}
    </SafeAreaView>
  );
}

function MessageBubble({ message }: { message: ConversationMessage }) {
  const isUser = message.role === 'user';
  return (
    <View style={[styles.bubble, isUser ? styles.userBubble : styles.assistantBubble]}>
      <Text style={[styles.bubbleText, isUser ? styles.userText : styles.assistantText]}>
        {message.content}
      </Text>
    </View>
  );
}

function TypingIndicator() {
  return (
    <View style={styles.assistantBubble}>
      <Text style={[styles.assistantText, { letterSpacing: 4 }]}>···</Text>
    </View>
  );
}

// Extract the last sentence (question) from the reflection string.
function extractQuestion(reflection: string): string {
  const sentences = reflection.split(/(?<=[.!?])\s+/);
  return sentences[sentences.length - 1]?.trim() ?? reflection;
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0f0f1a' },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 20,
    paddingVertical: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#1f2937',
  },
  headerTitle: { fontSize: 17, color: '#f3f4f6', fontWeight: '600' },
  closeButton: { paddingVertical: 6, paddingHorizontal: 12 },
  closeText: { fontSize: 15, color: '#6b7280' },

  turnsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 20,
    paddingVertical: 10,
    gap: 6,
  },
  turnDot: { width: 6, height: 6, borderRadius: 3, backgroundColor: '#1f2937' },
  turnDotUsed: { backgroundColor: '#3b82f6' },
  turnsLabel: { fontSize: 12, color: '#4b5563', marginLeft: 4 },

  messageList: { paddingHorizontal: 16, paddingTop: 16, paddingBottom: 24 },
  bubble: { maxWidth: '82%', borderRadius: 18, padding: 14, marginBottom: 12 },
  userBubble: { alignSelf: 'flex-end', backgroundColor: '#1e3a5f' },
  assistantBubble: { alignSelf: 'flex-start', backgroundColor: '#161625' },
  bubbleText: { fontSize: 15, lineHeight: 24 },
  userText: { color: '#bfdbfe' },
  assistantText: { color: '#e5e7eb', fontWeight: '300' },

  inputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderTopWidth: 1,
    borderTopColor: '#1f2937',
    gap: 10,
  },
  input: {
    flex: 1,
    backgroundColor: '#161625',
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    color: '#f3f4f6',
    fontSize: 15,
    maxHeight: 120,
  },
  sendButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: '#1e3a5f',
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendButtonDisabled: { opacity: 0.3 },
  sendText: { color: '#93c5fd', fontSize: 18 },

  closedBanner: {
    padding: 24,
    alignItems: 'center',
    borderTopWidth: 1,
    borderTopColor: '#1f2937',
    gap: 16,
  },
  closedText: { fontSize: 15, color: '#6b7280', textAlign: 'center' },
  goodnightBtn: {
    paddingVertical: 12,
    paddingHorizontal: 32,
    borderRadius: 24,
    backgroundColor: '#1f2937',
  },
  goodnightBtnText: { color: '#9ca3af', fontSize: 15 },
});
