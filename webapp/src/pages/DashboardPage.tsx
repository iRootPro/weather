import type { UseQueryResult } from '@tanstack/react-query';
import type { AttentionCard as AttentionCardType, DashboardSnapshot } from '../api/dashboard';
import { AttentionCard } from '../components/AttentionCard';
import { DashboardSkeleton } from '../components/Skeleton';
import { Headline } from '../components/Headline';
import { QuietSummary } from '../components/QuietSummary';
import { formatClock } from '../utils/time';

export function DashboardPage({ query }: { query: UseQueryResult<DashboardSnapshot, Error> }) {
  if (query.isLoading) return <DashboardSkeleton />;

  if (query.isError) {
    return (
      <main className="page-shell error-shell">
        <section className="error-card">
          <span>⚠️</span>
          <h1>Не удалось загрузить дашборд</h1>
          <p>{query.error.message}</p>
          <button onClick={() => query.refetch()}>Попробовать ещё раз</button>
        </section>
      </main>
    );
  }

  const snapshot = query.data;
  if (!snapshot) return null;

  const weatherCard = snapshot.cards.find((card) => card.id === 'weather-current');
  const attentionCards = snapshot.cards.filter((card) => card.id !== 'weather-current' && card.priority >= 55);
  const contextCards = snapshot.cards.filter((card) => card.id !== 'weather-current' && card.priority < 55);
  const featuredAttention = attentionCards[0];
  const remainingAttention = attentionCards.slice(1);
  const importantCount = attentionCards.length;

  return (
    <main className="page-shell">
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Погодный ассистент</span>
          <strong>Армавир сейчас</strong>
        </div>
        <button className="refresh-button" onClick={() => query.refetch()} disabled={query.isFetching}>
          {query.isFetching ? 'Обновляю…' : 'Обновить'}
        </button>
      </header>

      <Headline headline={snapshot.headline} station={snapshot.station_status} />

      <section className="meta-row" aria-label="Метаданные обновления">
        <span>Обновлено: {formatClock(snapshot.generated_at)}</span>
        <span>{importantCount > 0 ? `${importantCount} важных сигналов` : 'важных сигналов нет'}</span>
      </section>

      {featuredAttention ? (
        <section className="attention-layout">
          <AttentionCard card={featuredAttention} featured />
          {weatherCard && <WeatherNow card={weatherCard} compact />}
        </section>
      ) : (
        <CalmOverview weatherCard={weatherCard} snapshot={snapshot} />
      )}

      {remainingAttention.length > 0 && (
        <section className="section-block">
          <div className="section-heading">
            <span>01</span>
            <h2>Ещё требует внимания</h2>
          </div>
          <div className="cards-grid">
            {remainingAttention.map((card) => (
              <AttentionCard key={card.id} card={card} />
            ))}
          </div>
        </section>
      )}

      {contextCards.length > 0 && (
        <section className="section-block">
          <div className="section-heading">
            <span>02</span>
            <h2>Контекст</h2>
          </div>
          <div className="cards-grid compact">
            {contextCards.map((card) => (
              <AttentionCard key={card.id} card={card} />
            ))}
          </div>
        </section>
      )}

      {featuredAttention && <QuietSummary quiet={snapshot.quiet} />}
    </main>
  );
}

function CalmOverview({ weatherCard, snapshot }: { weatherCard?: AttentionCardType; snapshot: DashboardSnapshot }) {
  return (
    <section className="calm-overview">
      {weatherCard && <WeatherNow card={weatherCard} />}

      <div className="calm-column">
        <section className="calm-card quiet-focus">
          <div className="calm-card-header">
            <span className="quiet-mark">✓</span>
            <div>
              <h2>Можно не отвлекаться</h2>
              <p>Система следит за показателями и поднимет наверх только то, что стало важным.</p>
            </div>
          </div>

          {snapshot.quiet.items.length > 0 && (
            <div className="quiet-pills" aria-label="Спокойные показатели">
              {snapshot.quiet.items.map((item) => (
                <span key={item}>{item}</span>
              ))}
            </div>
          )}
        </section>

        <section className="calm-card watch-card">
          <span className="watch-kicker">наблюдение</span>
          <h2>Если что-то изменится — экран перестроится</h2>
          <p>Магнитная буря, рост воды, порывы ветра, дождь или устаревшие данные станут отдельной крупной карточкой.</p>
        </section>
      </div>
    </section>
  );
}

function WeatherNow({ card, compact = false }: { card: AttentionCardType; compact?: boolean }) {
  return (
    <a className={`weather-now ${compact ? 'compact' : ''}`} href={card.detail_url || '/detail/temperature'}>
      <div className="weather-now-top">
        <span className="weather-icon" aria-hidden="true">{card.icon || '🌤️'}</span>
        <span className={`severity-pill severity-${card.severity}`}>сейчас</span>
      </div>

      <div className="weather-now-main">
        <div>
          <h2>{card.title}</h2>
          {card.subtitle && <p>{card.subtitle}</p>}
        </div>
        <div className="weather-temp">
          <span>{card.value || '—'}</span>
          <small>{card.unit || ''}</small>
        </div>
      </div>

      <div className="weather-now-footer">
        <span>{card.reason || 'текущая погода'}</span>
        <span className="priority">{card.priority}</span>
      </div>
    </a>
  );
}
