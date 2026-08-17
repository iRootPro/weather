import type { CSSProperties } from 'react';
import type { UseQueryResult } from '@tanstack/react-query';
import type { DashboardSnapshot, NearForecastItem } from '../api/dashboard';
import { getDashboardScenarioLabel, type DashboardScenario } from '../api/mockDashboard';
import { ApiErrorCard } from '../components/ApiErrorCard';
import { AppTabs } from '../components/AppTabs';
import { DashboardSkeleton } from '../components/Skeleton';
import { WeatherGlyph } from '../components/WeatherGlyph';
import { formatClock } from '../utils/time';
import { buildEveningInsight, displayForecastIcon } from './DashboardPage';

export function EveningPage({ query, scenario }: { query: UseQueryResult<DashboardSnapshot, Error>; scenario?: DashboardScenario }) {
  if (query.isLoading) return <DashboardSkeleton />;

  if (query.isError) return <ApiErrorCard title="Не удалось загрузить вечер" message={query.error.message} onRetry={() => query.refetch()} />;

  const snapshot = query.data;
  if (!snapshot) return null;

  const forecast = snapshot.near_forecast ?? [];
  const insight = buildEveningInsight(snapshot);
  const recommendations = buildEveningRecommendations(snapshot);
  const stats = buildEveningStats(forecast);

  return (
    <main className="page-shell evening-page">
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Погодный ассистент</span>
          <strong>Вечер</strong>
        </div>
        <div className="topbar-actions">
          {scenario && <span className="scenario-badge">сценарий: {getDashboardScenarioLabel(scenario)}</span>}
          <button className="refresh-button" onClick={() => query.refetch()} disabled={query.isFetching}>
            {query.isFetching ? 'Обновляю…' : 'Обновить'}
          </button>
        </div>
      </header>
      <AppTabs active="evening" scenario={scenario} />

      <section className="evening-hero">
        <div>
          <span className="headline-kicker">план на ближайшие часы</span>
          <h1>{insight.title}</h1>
          <p>{insight.text} {snapshot.summary}</p>
          <div className="forecast-hero-stats">
            {stats.map((stat) => <span key={stat}>{stat}</span>)}
          </div>
        </div>
        <div className="evening-moon" aria-hidden="true">
          <span>🌙</span>
          <small>{forecast[0] ? `${formatClock(forecast[0].time)} → ${formatClock(forecast[forecast.length - 1].time)}` : 'вечер'}</small>
        </div>
      </section>

      <section className="evening-plan-grid">
        <article className="risk-panel action-panel">
          <span className="watch-kicker">что учесть</span>
          <h2>{recommendations.primary}</h2>
          <p>{recommendations.text}</p>
        </article>
        <article className="risk-panel">
          <span className="watch-kicker">дом и улица</span>
          <h2>{recommendations.home}</h2>
          <p>{recommendations.extra}</p>
        </article>
      </section>

      <section className="section-block">
        <div className="section-heading forecast-heading">
          <div>
            <span>вечер</span>
            <h2>Почасовая картина</h2>
          </div>
        </div>
        <div className="evening-timeline">
          {forecast.map((item, index) => <EveningTimelineItem key={item.time} item={item} index={index} total={forecast.length} />)}
        </div>
      </section>
    </main>
  );
}

function EveningTimelineItem({ item, index, total }: { item: NearForecastItem; index: number; total: number }) {
  const rain = item.precipitation_probability >= 40 || item.precipitation >= 0.5;
  const width = total <= 1 ? 0 : (index / (total - 1)) * 100;

  return (
    <article className={`evening-timeline-item ${rain ? 'rainy' : ''}`} style={{ '--progress': `${width}%` } as CSSProperties}>
      <time>{formatClock(item.time)}</time>
      <WeatherGlyph icon={displayForecastIcon(item)} />
      <strong>{Math.round(item.temperature)}°</strong>
      <p>{item.weather_description || 'прогноз'}</p>
      <small>{item.precipitation_probability}% дождь · {item.wind_speed.toFixed(1)} м/с</small>
    </article>
  );
}

function buildEveningStats(forecast: NearForecastItem[]) {
  if (forecast.length === 0) return ['прогноз пока недоступен'];
  const minTemp = Math.min(...forecast.map((item) => item.temperature));
  const maxTemp = Math.max(...forecast.map((item) => item.temperature));
  const maxRain = Math.max(...forecast.map((item) => item.precipitation_probability));
  const maxWind = Math.max(...forecast.map((item) => item.wind_speed));
  return [`${Math.round(minTemp)}–${Math.round(maxTemp)}°`, `дождь до ${maxRain}%`, `ветер до ${maxWind.toFixed(1)} м/с`];
}

function buildEveningRecommendations(snapshot: DashboardSnapshot) {
  const forecast = snapshot.near_forecast ?? [];
  const rainy = forecast.find((item) => item.precipitation_probability >= 40 || item.precipitation >= 0.5);
  const windy = forecast.find((item) => item.wind_speed >= 8);
  const last = forecast[forecast.length - 1];
  const first = forecast[0];
  const tempDrop = first && last ? first.temperature - last.temperature : 0;

  if (rainy) {
    return {
      primary: `Зонт лучше взять до ${formatClock(rainy.time)}`,
      text: 'Прогноз показывает заметную вероятность дождя рядом по времени. Лучше закрыть окна и убрать вещи с улицы заранее.',
      home: 'Закрой окна заранее',
      extra: 'Если выходишь надолго, ориентируйся не на текущую сухую погоду, а на ближайший пик осадков.'
    };
  }

  if (windy) {
    return {
      primary: 'Закрепи лёгкие предметы',
      text: 'Вечером ветер может быть заметным, особенно на открытых местах и балконах.',
      home: 'Осторожнее у деревьев',
      extra: 'Порывы обычно важнее средней скорости, поэтому лучше убрать всё, что может унести.'
    };
  }

  if (tempDrop >= 4) {
    return {
      primary: 'Возьми слой потеплее',
      text: `Температура заметно снизится к ${last ? formatClock(last.time) : 'утру'}. На короткой прогулке это может ощущаться резко.`,
      home: 'Осадков почти нет',
      extra: 'Главное изменение вечера — похолодание, а не дождь или ветер.'
    };
  }

  return {
    primary: 'Можно без специальных мер',
    text: 'Ближайшие часы выглядят спокойными: без сильного ветра, ливня и резких скачков.',
    home: 'Окна и двор в порядке',
    extra: 'Следи только за обычным похолоданием к ночи.'
  };
}
