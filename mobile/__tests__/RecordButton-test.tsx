import React from 'react';
import { render, fireEvent } from '@testing-library/react-native';
import { RecordButton } from '../src/components/RecordButton';

describe('RecordButton', () => {
  it('renders "Hold to Record" in idle state', () => {
    const { getByText } = render(
      <RecordButton
        state="idle"
        durationMs={0}
        onPress={() => {}}
      />
    );
    expect(getByText('Hold to Record')).toBeTruthy();
  });

  it('renders formatted duration in recording state', () => {
    const { getByText } = render(
      <RecordButton
        state="recording"
        durationMs={65000} // 1m 5s
        onPress={() => {}}
      />
    );
    expect(getByText('01:05')).toBeTruthy();
  });

  it('renders "Processing…" in stopped state', () => {
    const { getByText } = render(
      <RecordButton
        state="stopped"
        durationMs={0}
        onPress={() => {}}
      />
    );
    expect(getByText('Processing…')).toBeTruthy();
  });

  it('triggers onPress callback when tapped', () => {
    const onPressMock = jest.fn();
    const { getByText } = render(
      <RecordButton
        state="idle"
        durationMs={0}
        onPress={onPressMock}
      />
    );

    fireEvent.press(getByText('Hold to Record'));
    expect(onPressMock).toHaveBeenCalledTimes(1);
  });
});
