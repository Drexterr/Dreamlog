import { useEffect, useState } from 'react';
import { StyleProp, Text, TextStyle, TouchableOpacity, View } from 'react-native';

interface Props {
  text: string;
  style: StyleProp<TextStyle>;
  linkStyle: StyleProp<TextStyle>;
}

/** Measure the full text at the actual width and font size, including font scaling. */
export default function WeeklyNarrative({ text, style, linkStyle }: Props) {
  const [overflow, setOverflow] = useState(false);
  const [expanded, setExpanded] = useState(false);
  useEffect(() => { setExpanded(false); }, [text]);

  return (
    <View>
      <Text
        testID="weekly-review-full-measure"
        accessible={false}
        accessibilityElementsHidden
        importantForAccessibility="no-hide-descendants"
        pointerEvents="none"
        onTextLayout={({ nativeEvent }) => setOverflow(nativeEvent.lines.length > 3)}
        style={[style, { position: 'absolute', left: 0, right: 0, top: 0, opacity: 0 }]}
      >{text}</Text>
      <Text testID="weekly-review-visible-text" numberOfLines={expanded ? undefined : 3} ellipsizeMode="tail" style={style}>{text}</Text>
      {overflow && (
        <TouchableOpacity
          testID="weekly-review-show-more"
          accessibilityRole="button"
          accessibilityState={{ expanded }}
          onPress={() => setExpanded(value => !value)}
          style={{ minHeight: 44, minWidth: 80, alignSelf: 'flex-start', justifyContent: 'center', marginBottom: 10 }}
        ><Text style={[linkStyle, { fontSize: 13 }]}>{expanded ? 'Show less' : 'Show more'}</Text></TouchableOpacity>
      )}
    </View>
  );
}
