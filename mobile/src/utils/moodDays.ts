// Preserve the current analytics API's UTC calendar-date buckets.
export function sevenDaySlots<T extends { day: string }>(days: T[], now = new Date()) {
  const byDate = new Map(days.map(day => [day.day, day]));
  return Array.from({ length: 7 }, (_, index) => {
    const date = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - 6 + index));
    const day = date.toISOString().slice(0, 10);
    return { date, day, value: byDate.get(day) };
  });
}

export function filterSevenDays<T extends { day: string }>(days: T[], now = new Date()) {
  return sevenDaySlots(days, now).flatMap(slot => slot.value ? [slot.value] : []);
}
