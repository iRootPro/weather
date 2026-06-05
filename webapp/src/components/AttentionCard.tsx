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

  if (card.detail_url) {
    return (
      <a className={className} href={card.detail_url}>
        {content}
      </a>
    );
  }

  return <article className={className}>{content}</article>;
}
