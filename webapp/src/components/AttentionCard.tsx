import type { AttentionCard as AttentionCardType, Severity } from '../api/dashboard';

const severityLabel: Record<Severity, string> = {
  calm: 'спокойно',
  normal: 'норма',
  info: 'заметно',
  warning: 'важно',
  danger: 'срочно'
};

export function AttentionCard({ card, featured = false }: { card: AttentionCardType; featured?: boolean }) {
  const content = (
    <>
      <div className="card-topline">
        <span className="card-icon" aria-hidden="true">{card.icon || '•'}</span>
        <span className={`severity-pill severity-${card.severity}`}>{severityLabel[card.severity]}</span>
      </div>

      <div className="card-main">
        <div>
          <h3>{card.title}</h3>
          {card.subtitle && <p className="card-subtitle">{card.subtitle}</p>}
          {card.action && <p className="card-action"><span>Что сделать</span>{card.action}</p>}
        </div>

        {card.value && (
          <div className="metric-lockup" aria-label={`${card.value} ${card.unit || ''}`}>
            <span className="metric-value">{card.value}</span>
            {card.unit && <span className="metric-unit">{card.unit}</span>}
          </div>
        )}
      </div>

      <div className="card-footer">
        <span>{card.reason || 'оценка важности'}</span>
        <span className="priority">{card.priority}</span>
      </div>
    </>
  );

  const className = `attention-card attention-${card.severity} ${featured ? 'featured' : ''}`;
  const href = riskDetailHref(card);

  if (href) {
    return (
      <a className={className} href={href}>
        {content}
      </a>
    );
  }

  return <article className={className}>{content}</article>;
}

function riskDetailHref(card: AttentionCardType) {
  const scenario = new URLSearchParams(window.location.search).get('scenario');
  const suffix = scenario ? `?scenario=${encodeURIComponent(scenario)}` : '';

  switch (card.domain) {
    case 'geomagnetic':
      return `/app/geomagnetic${suffix}`;
    case 'hydro':
      return `/app/water${suffix}`;
    case 'rain':
    case 'forecast':
      return `/app/rain${suffix}`;
    case 'wind':
      return `/app/wind${suffix}`;
    case 'solar':
      return `/app/uv${suffix}`;
    case 'station':
      return `/app/station${suffix}`;
    default:
      return card.detail_url;
  }
}
