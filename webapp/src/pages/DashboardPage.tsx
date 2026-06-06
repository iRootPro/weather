import type { UseQueryResult } from '@tanstack/react-query';
import type { CurrentWeatherSummary, DashboardSnapshot, NearForecastItem } from '../api/dashboard';
import { dashboardScenarios, getDashboardScenarioLabel, type DashboardScenario } from '../api/mockDashboard';
import { AttentionCard } from '../components/AttentionCard';
import { DashboardSkeleton } from '../components/Skeleton';
import { Headline } from '../components/Headline';
import { QuietSummary } from '../components/QuietSummary';
import { formatClock } from '../utils/time';

export function DashboardPage({ query, scenario }: { query: UseQueryResult<DashboardSnapshot, Error>; scenario?: DashboardScenario }) {
  if (query.isLoading) return <DashboardSkeleton />;

  if (query.isError) return <DashboardError message={query.error.message} onRetry={() => query.refetch()} />;

  const snapshot = query.data;
  if (!snapshot) return null;

  const importantThreshold = 70;
  const attentionCards = snapshot.cards.filter((card) => card.priority >= importantThreshold);
  const contextCards = snapshot.cards.filter((card) => card.priority < importantThreshold);
  const featuredAttention = attentionCards[0];
  const remainingAttention = attentionCards.slice(1);
  const importantCount = attentionCards.length;
  const isCalmMode = snapshot.headline.severity === 'calm' && !featuredAttention;

  return (
    <main className="page-shell">
      <div className="sky-noise" aria-hidden="true" />
      <AppTopbar query={query} scenario={scenario} />
      {scenario && <ScenarioSwitcher active={scenario} />}
      <SectionNav forecastHref={withScenario('/app/forecast', scenario)} />

      {isCalmMode ? (
        <CompactStatus snapshot={snapshot} importantCount={importantCount} />
      ) : (
        <Headline headline={snapshot.headline} station={snapshot.station_status} />
      )}

      <section className="meta-row" aria-label="Метаданные обновления">
        <span>Обновлено: {formatClock(snapshot.generated_at)}</span>
        <span>{importantCount > 0 ? `${importantCount} важных сигналов` : 'важных сигналов нет'}</span>
      </section>

      <section id="now" aria-label="Текущая погода">
        {featuredAttention ? (
          <section className="attention-layout">
            <AttentionCard card={featuredAttention} featured />
            {snapshot.current_weather && <WeatherNow current={snapshot.current_weather} summary={snapshot.summary} compact />}
          </section>
        ) : (
          <CalmOverview current={snapshot.current_weather} snapshot={snapshot} />
        )}
      </section>

      {remainingAttention.length > 0 && (
        <section className="section-block" id="risks">
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
        <section className="section-block" id={remainingAttention.length > 0 ? undefined : 'risks'}>
          <div className="section-heading">
            <span>{remainingAttention.length > 0 ? '02' : '01'}</span>
            <h2>Контекст</h2>
          </div>
          <div className="cards-grid compact">
            {contextCards.map((card) => (
              <AttentionCard key={card.id} card={card} />
            ))}
          </div>
        </section>
      )}

      {snapshot.near_forecast && snapshot.near_forecast.length > 0 && (
        <ForecastStrip items={snapshot.near_forecast} scenario={scenario} />
      )}

      {featuredAttention && <QuietSummary quiet={snapshot.quiet} />}
    </main>
  );
}

function DashboardError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <main className="page-shell error-shell">
      <section className="error-card">
        <span>⚠️</span>
        <h1>Не удалось загрузить дашборд</h1>
        <p>{message}</p>
        <button onClick={onRetry}>Попробовать ещё раз</button>
      </section>
    </main>
  );
}

function AppTopbar({ query, scenario }: { query: UseQueryResult<DashboardSnapshot, Error>; scenario?: DashboardScenario }) {
  return (
    <header className="topbar">
      <div>
        <span className="app-label">Погодный ассистент</span>
        <strong>Армавир сейчас</strong>
      </div>
      <div className="topbar-actions">
        {scenario && <span className="scenario-badge">сценарий: {getDashboardScenarioLabel(scenario)}</span>}
        <button className="refresh-button" onClick={() => query.refetch()} disabled={query.isFetching}>
          {query.isFetching ? 'Обновляю…' : 'Обновить'}
        </button>
      </div>
    </header>
  );
}

function SectionNav({ forecastHref }: { forecastHref: string }) {
  return (
    <nav className="section-nav" aria-label="Разделы дашборда">
      <a href="#now">сейчас</a>
      <a href="#evening">вечер</a>
      <a href="#risks">риски</a>
      <a href={forecastHref}>прогноз</a>
    </nav>
  );
}

function ScenarioSwitcher({ active }: { active: DashboardScenario }) {
  return (
    <nav className="scenario-switcher" aria-label="Тестовые сценарии дашборда">
      <a href="/app/">живые данные</a>
      {dashboardScenarios.map((item) => (
        <a key={item} className={item === active ? 'active' : undefined} href={`/app/?scenario=${item}`}>
          {getDashboardScenarioLabel(item)}
        </a>
      ))}
    </nav>
  );
}

function CompactStatus({ snapshot, importantCount }: { snapshot: DashboardSnapshot; importantCount: number }) {
  return (
    <section className="compact-status status-calm" aria-label="Статус внимания">
      <span className="compact-status-icon" aria-hidden="true">{snapshot.headline.icon || '🟢'}</span>
      <div>
        <strong>{snapshot.headline.title}</strong>
        <span>{snapshot.station_status.label} · {importantCount > 0 ? `${importantCount} важных сигналов` : 'важных сигналов нет'}</span>
      </div>
    </section>
  );
}

function CalmOverview({ current, snapshot }: { current?: CurrentWeatherSummary; snapshot: DashboardSnapshot }) {
  return (
    <section className="calm-overview">
      {current && <WeatherNow current={current} summary={snapshot.summary} />}

      <div className="calm-column">
        <EveningInsight snapshot={snapshot} />
        <ControlStatus quietItems={snapshot.quiet.items} />
      </div>
    </section>
  );
}

function EveningInsight({ snapshot }: { snapshot: DashboardSnapshot }) {
  const insight = buildEveningInsight(snapshot);

  return (
    <section className="calm-card evening-card" id="evening">
      <span className="watch-kicker">сегодня вечером</span>
      <h2>{insight.title}</h2>
      <p>{insight.text}</p>
    </section>
  );
}

function ControlStatus({ quietItems }: { quietItems: string[] }) {
  return (
    <section className="calm-card quiet-focus" id="risks">
      <div className="calm-card-header">
        <span className="quiet-mark">✓</span>
        <div>
          <h2>Под контролем</h2>
          <p>Если один из показателей выйдет из нормы, он станет главной карточкой.</p>
        </div>
      </div>

      {quietItems.length > 0 && (
        <div className="quiet-pills" aria-label="Спокойные показатели">
          {quietItems.map((item) => (
            <span key={item}>{item}</span>
          ))}
        </div>
      )}
    </section>
  );
}

function WeatherNow({ current, summary, compact = false }: { current: CurrentWeatherSummary; summary?: string; compact?: boolean }) {
  return (
    <a className={`weather-now ${compact ? 'compact' : ''}`} href="/detail/temperature">
      <div className="weather-now-top">
        <span className="weather-icon" aria-hidden="true">{current.icon || '🌤️'}</span>
        <div className="weather-status-stack">
          <span className="severity-pill severity-normal">сейчас</span>
          <TemperatureTrend delta={current.temperature_delta} />
        </div>
      </div>

      <div className="weather-now-main">
        <div>
          <h2>{current.title}</h2>
          <p>{current.subtitle}</p>
          {summary && <p className="weather-summary">{compactSummary(summary)}</p>}
          <WeatherFacts current={current} />
        </div>
        <div className="weather-temp">
          <span>{formatNumber(current.temperature)}</span>
          <small>°C</small>
        </div>
      </div>

      <div className="weather-now-footer">
        <span>наблюдение {formatClock(current.observed_at)}</span>
        <span>открыть детали</span>
      </div>
    </a>
  );
}

function TemperatureTrend({ delta }: { delta?: number }) {
  if (typeof delta !== 'number' || Math.abs(delta) < 0.2) {
    return <span className="trend-chip trend-flat">температура стабильна</span>;
  }

  const falling = delta < 0;
  return (
    <span className={`trend-chip ${falling ? 'trend-down' : 'trend-up'}`}>
      {falling ? '↓ холодает' : '↑ теплеет'} {Math.abs(delta).toFixed(1)}°/ч
    </span>
  );
}

function WeatherFacts({ current }: { current: CurrentWeatherSummary }) {
  const facts = [
    typeof current.humidity === 'number' ? `влажность ${current.humidity}%` : null,
    typeof current.pressure === 'number' ? `давление ${Math.round(current.pressure)} мм` : null,
    typeof current.wind_speed === 'number' ? `ветер ${current.wind_speed.toFixed(1)} м/с` : null,
    typeof current.rain_rate === 'number' && current.rain_rate > 0 ? `дождь ${current.rain_rate.toFixed(1)} мм/ч` : 'дождя нет',
    typeof current.uv_index === 'number' ? `UV ${current.uv_index.toFixed(0)}` : null
  ].filter(Boolean);

  return (
    <div className="weather-facts">
      {facts.map((fact) => (
        <span key={fact}>{fact}</span>
      ))}
    </div>
  );
}

function ForecastStrip({ items, scenario }: { items: NearForecastItem[]; scenario?: DashboardScenario }) {
  return (
    <section className="forecast-strip-section" id="forecast">
      <div className="section-heading forecast-heading">
        <div>
          <span>прогноз</span>
          <h2>Ближайшие часы</h2>
        </div>
        <a href={withScenario('/app/forecast', scenario)}>Открыть подробно</a>
      </div>
      <div className="forecast-strip" role="list" aria-label="Прогноз на ближайшие часы">
        {items.map((item) => (
          <article className="forecast-hour" key={item.time} role="listitem">
            <time>{formatClock(item.time)}</time>
            <span className="forecast-icon" aria-hidden="true">{displayForecastIcon(item)}</span>
            <strong>{Math.round(item.temperature)}°</strong>
            <small>{item.precipitation_probability > 0 ? `${item.precipitation_probability}% дождь` : item.weather_description || 'прогноз'}</small>
          </article>
        ))}
      </div>
    </section>
  );
}

export function buildEveningInsight(snapshot: DashboardSnapshot) {
  const forecast = snapshot.near_forecast ?? [];
  if (forecast.length === 0) {
    return {
      title: 'Без резких изменений',
      text: snapshot.summary || 'Пока нет прогноза на ближайшие часы.'
    };
  }

  const first = forecast[0];
  const last = forecast[forecast.length - 1];
  const rainy = forecast.find((item) => item.precipitation_probability >= 40 || item.precipitation >= 0.5);
  const maxWind = Math.max(...forecast.map((item) => item.wind_speed));
  const tempDelta = last.temperature - first.temperature;
  const tempPhrase = Math.abs(tempDelta) >= 1
    ? `${tempDelta < 0 ? 'похолодает' : 'потеплеет'} до ${Math.round(last.temperature)}°`
    : `останется около ${Math.round(last.temperature)}°`;
  const rainPhrase = rainy
    ? `дождь вероятен около ${formatClock(rainy.time)}`
    : 'дождя почти нет';
  const windPhrase = maxWind >= 8
    ? 'ветер будет заметным'
    : 'ветер слабый';

  return {
    title: `К ${formatClock(last.time)} ${tempPhrase}`,
    text: `${rainPhrase}, ${windPhrase}.`
  };
}

export function displayForecastIcon(item: NearForecastItem) {
  const hour = new Date(item.time).getHours();
  const isNight = hour >= 21 || hour < 5;
  if (!isNight) return item.icon;

  if (item.icon === '☀️') return '🌙';
  if (item.icon === '🌤️') return '🌙';
  if (item.icon === '⛅') return '☁️';
  return item.icon;
}

function compactSummary(summary: string) {
  return summary.replace(/^сейчас [^;]+;\s*/i, '').replace(/;\s*/g, ' · ');
}

function withScenario(path: string, scenario?: DashboardScenario) {
  return scenario ? `${path}?scenario=${scenario}` : path;
}

function formatNumber(value?: number) {
  if (typeof value !== 'number') return '—';
  return value.toFixed(1);
}
