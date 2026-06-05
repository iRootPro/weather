import type { DashboardHeadline, StationStatus } from '../api/dashboard';
import { formatRelativeTime } from '../utils/time';

export function Headline({ headline, station }: { headline: DashboardHeadline; station: StationStatus }) {
  return (
    <section className={`headline-panel headline-${headline.severity}`}>
      <div className="orbital-ring" aria-hidden="true" />
      <div className="headline-kicker">Армавир · метеостанция</div>
      <div className="headline-content">
        <div className="headline-icon" aria-hidden="true">{headline.icon || '🌤️'}</div>
        <div>
          <h1>{headline.title}</h1>
          {headline.summary && <p>{headline.summary}</p>}
        </div>
      </div>
      <div className="station-strip">
        <span className={`station-dot station-${station.severity}`} />
        <span>{station.label}</span>
        {station.last_seen_at && <span className="muted">· {formatRelativeTime(station.last_seen_at)}</span>}
      </div>
    </section>
  );
}
