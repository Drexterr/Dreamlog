import React from 'react';
import { fireEvent, render } from '@testing-library/react-native';
import WeeklyNarrative from '../WeeklyNarrative';

it('only offers expansion above three rendered lines, toggles and resets for a different review', () => {
  const view = render(<WeeklyNarrative text="A weekly reflection" style={{ fontSize: 15 }} linkStyle={{}} />);
  const measure = (count: number) => fireEvent(view.getByTestId('weekly-review-full-measure', { includeHiddenElements: true }), 'textLayout', { nativeEvent: { lines: Array.from({ length: count }, () => ({})) } });
  measure(3);
  expect(view.queryByText('Show more')).toBeNull();
  measure(4);
  expect(view.getByTestId('weekly-review-visible-text').props.numberOfLines).toBe(3);
  fireEvent.press(view.getByText('Show more'));
  expect(view.getByText('Show less')).toBeTruthy();
  expect(view.getByTestId('weekly-review-visible-text').props.numberOfLines).toBeUndefined();
  fireEvent.press(view.getByText('Show less'));
  expect(view.getByTestId('weekly-review-visible-text').props.numberOfLines).toBe(3);
  fireEvent.press(view.getByText('Show more'));
  view.rerender(<WeeklyNarrative text="A different week" style={{ fontSize: 15 }} linkStyle={{}} />);
  expect(view.getByTestId('weekly-review-visible-text').props.numberOfLines).toBe(3);
  expect(view.getByTestId('weekly-review-show-more').props.accessibilityState).toEqual({ expanded: false });
  measure(2);
  expect(view.queryByTestId('weekly-review-show-more')).toBeNull();
});
