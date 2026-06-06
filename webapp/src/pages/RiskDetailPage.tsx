import type { UseQueryResult } from '@tanstack/react-query';
import type { AttentionCard, DashboardSnapshot } from '../api/dashboard';
import { getDashboardScenarioLabel, type DashboardScenario } from '../api/mockDashboard';
import { DashboardSkeleton } from '../components/Skeleton';
import { formatClock } from '../utils/time';

type RiskKind = 'geomagnetic' | 'water' | 'rain' | 'wind' | 'uv' | 'station';

const riskCopy: Record<RiskKind, { title: string; domain: string[]; icon: string; quiet: string; explainer: string }> = {
  geomagnetic: {
    title: 'Геомагнитка',
    domain: ['geomagnetic'],
    icon: '🧲',
    quiet: 'Магнитное поле сейчас не требует внимания.',
    explainer: 'Kp 5 и выше считается магнитной бурей. Чем выше индекс, тем заметнее возможное влияние на самочувствие и радиосвязь.'
  },
  water: {
    title: 'Вода',
    domain: ['hydro'],
    icon: '🌊',
    quiet: 'Уровень воды сейчас не требует внимания.',
    explainer: 'Главное — расстояние до неблагоприятного уровня и скорость изменения. Быстрый рост важнее абсолютной цифры.'
  },
  rain: {
    title: 'Дождь',
    domain: ['rain', 'forecast'],
    icon: '🌧️',
    quiet: 'Осадки сейчас не требуют внимания.',
    explainer: 'Смотрим на текущую интенсивность и ближайший почасовой прогноз. Высокая вероятность рядом по времени поднимается наверх.'
  },
  wind: {
    title: 'Ветер',
    domain: ['wind'],
    icon: '💨',
    quiet: 'Ветер сейчас не требует внимания.',
    explainer: 'Для безопасности важны порывы: они могут быть существенно выше средней скорости ветра.'
  },
  uv: {
    title: 'UV',
    domain: ['solar'],
    icon: '☀️',
    quiet: 'UV сейчас не требует внимания.',
    explainer: 'UV 6+ уже требует осторожности на прямом солнце, UV 8+ — высокий риск обгорания.'
  },
  station: {
    title: 'Станция',
    domain: ['station'],
    icon: '📡',
    quiet: 'Данные станции свежие.',
    explainer: 'Если станция молчит, текущие показания становятся менее надёжными. Прогноз продолжает работать отдельно.'
  }
};

export function RiskDetailPage({ query, scenario, kind }: { query: UseQueryResult<DashboardSnapshot, Error>; scenario?: DashboardScenario; kind: RiskKind }) {
  if (query.isLoading) return <DashboardSkeleton />;

  if (query.isError) {
    return (
      <main className="page-shell error-shell">
        <section className="error-card">
          <span>⚠️</span>
          <h1>Не удалось загрузить раздел</h1>
          <p>{query.error.message}</p>
          <button onClick={() => query.refetch()}>Попробовать ещё раз</button>
        </section>
      </main>
    );
  }

  const snapshot = query.data;
  if (!snapshot) return null;

  const copy = riskCopy[kind];
  const cards = snapshot.cards.filter((card) => copy.domain.includes(card.domain));
  const primary = cards[0];
  const severity = primary?.severity ?? 'calm';

  return (
    <main className={`page-shell risk-page risk-page-${severity}`}>
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Погодный ассистент</span>
          <strong>{copy.title}</strong>
        </div>
        <div className="topbar-actions">
          {scenario && <span className="scenario-badge">сценарий: {getDashboardScenarioLabel(scenario)}</span>}
          <a className="refresh-button" href={scenario ? `/app/?scenario=${scenario}` : '/app/'}>Назад</a>
        </div>
      </header>

      <section className={`risk-hero attention-${severity}`}>
        <span className="risk-hero-icon" aria-hidden="true">{primary?.icon || copy.icon}</span>
        <div>
          <span className="headline-kicker">{severity === 'calm' ? 'под контролем' : 'требует внимания'}</span>
          <h1>{primary?.title || copy.quiet}</h1>
          <p>{primary?.subtitle || copy.explainer}</p>
        </div>
        {primary?.value && (
          <div className="risk-metric">
            <strong>{primary.value}</strong>
            {primary.unit && <span>{primary.unit}</span>}
          </div>
        )}
      </section>

      <section className="risk-detail-grid">
        <article className="risk-panel action-panel">
          <span className="watch-kicker">что сделать</span>
          <h2>{primary?.action || 'Действий не требуется'}</h2>
          <p>{primary?.reason || copy.explainer}</p>
        </article>

        <article className="risk-panel">
          <span className="watch-kicker">статус</span>
          <h2>{snapshot.station_status.label}</h2>
          <p>Дашборд обновлён в {formatClock(snapshot.generated_at)}. {snapshot.summary}</p>
        </article>
      </section>

      {cards.length > 1 && (
        <section className="risk-panel risk-related">
          <span className="watch-kicker">связанные сигналы</span>
          {cards.slice(1).map((card) => <RiskMiniCard key={card.id} card={card} />)}
        </section>
      )}
    </main>
  );
}

function RiskMiniCard({ card }: { card: AttentionCard }) {
  return (
    <article className="risk-mini-card">
      <span>{card.icon}</span>
      <div>
        <strong>{card.title}</strong>
        <p>{card.subtitle}</p>
      </div>
      <b>{card.priority}</b>
    </article>
  );
}

export type { RiskKind };
