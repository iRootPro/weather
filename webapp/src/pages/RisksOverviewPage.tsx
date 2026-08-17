import type { UseQueryResult } from '@tanstack/react-query';
import type { AttentionCard, DashboardSnapshot, Severity } from '../api/dashboard';
import { getDashboardScenarioLabel, type DashboardScenario } from '../api/mockDashboard';
import { ApiErrorCard } from '../components/ApiErrorCard';
import { AppTabs, withScenario } from '../components/AppTabs';
import { DashboardSkeleton } from '../components/Skeleton';
import { formatClock } from '../utils/time';

type RiskTileConfig = {
  key: string;
  title: string;
  icon: string;
  domains: string[];
  href: string;
  calm: string;
  explainer: string;
};

const riskTiles: RiskTileConfig[] = [
  {
    key: 'wind',
    title: 'Ветер',
    icon: '💨',
    domains: ['wind'],
    href: '/app/wind',
    calm: 'порывов нет',
    explainer: 'Следим за средней скоростью и порывами.'
  },
  {
    key: 'rain',
    title: 'Дождь',
    icon: '🌧️',
    domains: ['rain', 'forecast'],
    href: '/app/rain',
    calm: 'сухо',
    explainer: 'Смотрим текущий дождь и ближайшие часы.'
  },
  {
    key: 'geomagnetic',
    title: 'Геомагнитка',
    icon: '🧲',
    domains: ['geomagnetic'],
    href: '/app/geomagnetic',
    calm: 'бури нет',
    explainer: 'Kp 5+ поднимает карточку в важные.'
  },
  {
    key: 'water',
    title: 'Вода',
    icon: '🌊',
    domains: ['hydro'],
    href: '/app/water',
    calm: 'уровень в норме',
    explainer: 'Важны расстояние до порога и скорость роста.'
  },
  {
    key: 'uv',
    title: 'UV',
    icon: '☀️',
    domains: ['solar'],
    href: '/app/uv',
    calm: 'безопасно сейчас',
    explainer: 'Днём высокий UV станет предупреждением.'
  },
  {
    key: 'station',
    title: 'Станция',
    icon: '📡',
    domains: ['station'],
    href: '/app/station',
    calm: 'данные свежие',
    explainer: 'Если станция молчит, текущие значения теряют надёжность.'
  }
];

export function RisksOverviewPage({ query, scenario }: { query: UseQueryResult<DashboardSnapshot, Error>; scenario?: DashboardScenario }) {
  if (query.isLoading) return <DashboardSkeleton />;

  if (query.isError) return <ApiErrorCard title="Не удалось загрузить риски" message={query.error.message} onRetry={() => query.refetch()} />;

  const snapshot = query.data;
  if (!snapshot) return null;

  const activeCards = snapshot.cards.filter((card) => card.priority >= 70);
  const highest = activeCards[0];
  const heroSeverity = (highest?.severity ?? snapshot.station_status.severity ?? 'calm') as Severity;

  return (
    <main className="page-shell risks-overview-page">
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Погодный ассистент</span>
          <strong>Риски</strong>
        </div>
        <div className="topbar-actions">
          {scenario && <span className="scenario-badge">сценарий: {getDashboardScenarioLabel(scenario)}</span>}
          <button className="refresh-button" onClick={() => query.refetch()} disabled={query.isFetching}>
            {query.isFetching ? 'Обновляю…' : 'Обновить'}
          </button>
        </div>
      </header>
      <AppTabs active="risks" scenario={scenario} />

      <section className={`risks-hero attention-${heroSeverity}`}>
        <div>
          <span className="headline-kicker">сводка контроля</span>
          <h1>{highest ? highest.title : 'Всё под контролем'}</h1>
          <p>{highest ? highest.subtitle || highest.reason : 'Ветер, дождь, вода, геомагнитка, UV и станция не требуют срочных действий.'}</p>
        </div>
        <div className="risks-hero-score">
          <strong>{activeCards.length}</strong>
          <span>{plural(activeCards.length, 'важный сигнал', 'важных сигнала', 'важных сигналов')}</span>
        </div>
      </section>

      <section className="risk-tile-grid" aria-label="Все факторы риска">
        {riskTiles.map((tile) => <RiskTile key={tile.key} tile={tile} snapshot={snapshot} scenario={scenario} />)}
      </section>

      <section className="current-insight-grid">
        <article className="risk-panel action-panel">
          <span className="watch-kicker">если что-то изменится</span>
          <h2>{snapshot.quiet.title || 'Спокойные факторы уйдут в фон'}</h2>
          <p>Как только показатель пересечёт порог, он станет главной карточкой на экране “сейчас” и появится здесь с действием.</p>
        </article>
        <article className="risk-panel">
          <span className="watch-kicker">обновление</span>
          <h2>{snapshot.station_status.label}</h2>
          <p>Последняя сборка дашборда — {formatClock(snapshot.generated_at)}. {snapshot.summary}</p>
        </article>
      </section>
    </main>
  );
}

function RiskTile({ tile, snapshot, scenario }: { tile: RiskTileConfig; snapshot: DashboardSnapshot; scenario?: DashboardScenario }) {
  const card = snapshot.cards.find((item) => tile.domains.includes(item.domain));
  const stationProblem = tile.key === 'station' && !snapshot.station_status.ok;
  const severity = (card?.severity ?? (stationProblem ? snapshot.station_status.severity : 'calm')) as Severity;
  const title = card?.title ?? (stationProblem ? snapshot.station_status.label : tile.calm);
  const subtitle = card?.subtitle ?? (stationProblem ? 'Проверь питание, MQTT и связь с сервером.' : tile.explainer);
  const value = card?.value ?? (tile.key === 'station' && typeof snapshot.station_status.age_minutes === 'number' ? `${snapshot.station_status.age_minutes}` : '');
  const unit = card?.unit ?? (tile.key === 'station' && value ? 'мин' : '');

  return (
    <a className={`risk-tile attention-${severity}`} href={withScenario(tile.href, scenario)}>
      <div className="risk-tile-head">
        <span aria-hidden="true">{card?.icon || tile.icon}</span>
        <b className={`severity-pill severity-${severity}`}>{severityLabel(severity)}</b>
      </div>
      <div>
        <small>{tile.title}</small>
        <h2>{title}</h2>
        <p>{subtitle}</p>
      </div>
      <div className="risk-tile-foot">
        {value ? <strong>{value}<em>{unit}</em></strong> : <strong>OK</strong>}
        <span>подробнее</span>
      </div>
    </a>
  );
}

function severityLabel(severity: Severity) {
  switch (severity) {
    case 'danger': return 'срочно';
    case 'warning': return 'важно';
    case 'info': return 'заметно';
    case 'normal': return 'норма';
    case 'calm':
    default: return 'спокойно';
  }
}

function plural(value: number, one: string, few: string, many: string) {
  const mod10 = value % 10;
  const mod100 = value % 100;
  if (mod10 === 1 && mod100 !== 11) return one;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return few;
  return many;
}
