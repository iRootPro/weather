import { useEffect, useState } from 'react';
import type { DashboardSnapshot } from '../api/dashboard';

export function LiveRefreshBadge({ snapshot, isFetching }: { snapshot?: DashboardSnapshot; isFetching: boolean }) {
  const [, forceTick] = useState(0);

  useEffect(() => {
    const timer = window.setInterval(() => forceTick((value) => value + 1), 60_000);
    return () => window.clearInterval(timer);
  }, []);

  if (!snapshot) return null;

  const generated = new Date(snapshot.generated_at);
  const ageMs = Date.now() - generated.getTime();
  const ageMinutes = Number.isFinite(ageMs) ? Math.max(0, Math.floor(ageMs / 60_000)) : 0;
  const stale = ageMinutes >= 30 || !snapshot.station_status.ok;

  return (
    <div className={`live-refresh-badge ${isFetching ? 'refreshing' : ''} ${stale ? 'stale' : ''}`} role="status" aria-live="polite">
      <span aria-hidden="true" />
      <b>{isFetching ? 'обновляю данные' : ageMinutes === 0 ? 'обновлено только что' : `обновлено ${ageMinutes} мин назад`}</b>
      <small>{snapshot.station_status.label}</small>
    </div>
  );
}
