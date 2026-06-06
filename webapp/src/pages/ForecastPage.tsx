import type { UseQueryResult } from '@tanstack/react-query';
import type { DashboardSnapshot, NearForecastItem } from '../api/dashboard';
import { getDashboardScenarioLabel, type DashboardScenario } from '../api/mockDashboard';
import { DashboardSkeleton } from '../components/Skeleton';
import { formatClock } from '../utils/time';
import { buildEveningInsight, displayForecastIcon } from './DashboardPage';

export function ForecastPage({ query, scenario }: { query: UseQueryResult<DashboardSnapshot, Error>; scenario?: DashboardScenario }) {
  if (query.isLoading) return <DashboardSkeleton />;

  if (query.isError) {
    return (
      <main className="page-shell error-shell">
        <section className="error-card">
          <span>⚠️</span>
          <h1>Не удалось загрузить прогноз</h1>
          <p>{query.error.message}</p>
          <button onClick={() => query.refetch()}>Попробовать ещё раз</button>
        </section>
      </main>
    );
  }

  const snapshot = query.data;
  if (!snapshot) return null;

  const items = snapshot.near_forecast ?? [];
  const insight = buildEveningInsight(snapshot);
  const maxRain = Math.max(0, ...items.map((item) => item.precipitation_probability));
  const maxWind = Math.max(0, ...items.map((item) => item.wind_speed));
  const minTemp = Math.min(...items.map((item) => item.temperature));
  const maxTemp = Math.max(...items.map((item) => item.temperature));

  return (
    <main className="page-shell forecast-page">
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Погодный ассистент</span>
          <strong>Прогноз</strong>
        </div>
        <div className="topbar-actions">
          {scenario && <span className="scenario-badge">сценарий: {getDashboardScenarioLabel(scenario)}</span>}
          <a className="refresh-button" href={scenario ? `/app/?scenario=${scenario}` : '/app/'}>Назад</a>
        </div>
      </header>

      <section className="forecast-hero">
        <span className="headline-kicker">Армавир · ближайшие часы</span>
        <h1>{insight.title}</h1>
        <p>{insight.text}</p>
        {items.length > 0 && (
          <div className="forecast-hero-stats">
            <span>{Math.round(minTemp)}–{Math.round(maxTemp)}°</span>
            <span>дождь до {maxRain}%</span>
            <span>ветер до {maxWind.toFixed(1)} м/с</span>
          </div>
        )}
      </section>

      <section className="forecast-detail-list" aria-label="Подробный прогноз по часам">
        {items.map((item) => <ForecastRow key={item.time} item={item} />)}
      </section>
    </main>
  );
}

function ForecastRow({ item }: { item: NearForecastItem }) {
  return (
    <article className="forecast-row">
      <div className="forecast-row-time">
        <time>{formatClock(item.time)}</time>
        <span>{formatDay(item.time)}</span>
      </div>
      <span className="forecast-row-icon" aria-hidden="true">{displayForecastIcon(item)}</span>
      <div className="forecast-row-main">
        <strong>{Math.round(item.temperature)}°</strong>
        <span>{item.weather_description || 'прогноз'}</span>
      </div>
      <div className="forecast-row-metrics">
        <span>{item.precipitation_probability}% дождь</span>
        <span>{item.wind_speed.toFixed(1)} м/с</span>
        <span>ощущ. {Math.round(item.feels_like)}°</span>
      </div>
    </article>
  );
}

function formatDay(value: string) {
  return new Intl.DateTimeFormat('ru-RU', { weekday: 'short', day: '2-digit', month: 'short' }).format(new Date(value));
}
