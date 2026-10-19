import { filterSevenDays, sevenDaySlots } from '../moodDays';

it('uses UTC calendar dates across leap days and year boundaries', () => {
  const leap = sevenDaySlots([], new Date('2024-03-01T00:01:00Z'));
  expect(leap.map(slot => slot.day)).toEqual(['2024-02-24', '2024-02-25', '2024-02-26', '2024-02-27', '2024-02-28', '2024-02-29', '2024-03-01']);
  const newYear = sevenDaySlots([], new Date('2026-01-01T23:59:59Z'));
  expect(newYear[0].day).toBe('2025-12-26');
  expect(newYear[6].day).toBe('2026-01-01');
});

it('orders only the seven current days and preserves empty slots without inventing mood scores', () => {
  const now = new Date('2026-10-19T01:00:00+05:30'); // Still October 18 in UTC.
  const days = [{ day: '2026-10-18', mood: 0 }, { day: '2026-10-11', mood: 99 }, { day: '2026-10-19', mood: 90 }, { day: '2026-10-12', mood: 60 }];
  expect(filterSevenDays(days, now)).toEqual([days[3], days[0]]);
  const slots = sevenDaySlots(days, now);
  expect(slots.filter(slot => slot.value === undefined)).toHaveLength(5);
  expect(slots[6].value?.mood).toBe(0);
});
