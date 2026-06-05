import type { UseQueryResult } from '@tanstack/react-query';
import type { DashboardSnapshot } from '../api/dashboard';
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

  const [featured, ...restCards] = snapshot.cards;
  const importantCards = restCards.filter((card) => card.priority >= 55);
  const contextCards = restCards.filter((card) => card.priority < 55);

  return (
    <main className="page-shell">
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Weather attention</span>
          <strong>Важно сейчас</strong>
        </div>
        <button className="refresh-button" onClick={() => query.refetch()} disabled={query.isFetching}>
          {query.isFetching ? 'Обновляю…' : 'Обновить'}
        </button>
      </header>

      <Headline headline={snapshot.headline} station={snapshot.station_status} />

      <section className="meta-row" aria-label="Метаданные обновления">
        <span>Snapshot: {formatClock(snapshot.generated_at)}</span>
        <span>{snapshot.cards.length} активных карточек</span>
      </section>

      {featured ? (
        <section className="featured-grid">
          <AttentionCard card={featured} featured />
          <aside className="assistant-note">
            <span>логика внимания</span>
            <p>
              Backend уже отсортировал показатели по важности. Низкоприоритетные состояния уходят в спокойный блок,
              а резкие изменения и риски поднимаются наверх.
            </p>
          </aside>
        </section>
      ) : (
        <section className="empty-attention">
          <span>🟢</span>
          <h2>Нет важных карточек</h2>
          <p>Система не нашла показателей, которые требуют внимания.</p>
        </section>
      )}

      {importantCards.length > 0 && (
        <section className="section-block">
          <div className="section-heading">
            <span>01</span>
            <h2>Требует внимания</h2>
          </div>
          <div className="cards-grid">
            {importantCards.map((card) => (
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

      <QuietSummary quiet={snapshot.quiet} />
    </main>
  );
}
