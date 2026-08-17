import type { UseQueryResult } from '@tanstack/react-query';
import type { CurrentWeatherSummary, DashboardSnapshot, NearForecastItem } from '../api/dashboard';
import { dashboardScenarios, getDashboardScenarioLabel, type DashboardScenario } from '../api/mockDashboard';
import { ApiErrorCard } from '../components/ApiErrorCard';
import { AppTabs, withScenario } from '../components/AppTabs';
import { AttentionCard } from '../components/AttentionCard';
import { DashboardSkeleton } from '../components/Skeleton';
import { QuietSummary } from '../components/QuietSummary';
import { WeatherGlyph } from '../components/WeatherGlyph';
import { formatClock } from '../utils/time';

export function DashboardPage({ query, scenario }: { query: UseQueryResult<DashboardSnapshot, Error>; scenario?: DashboardScenario }) {
  if (query.isLoading) return <DashboardSkeleton />;

  if (query.isError) return <ApiErrorCard title="Не удалось загрузить дашборд" message={query.error.message} onRetry={() => query.refetch()} />;

  const snapshot = query.data;
  if (!snapshot) return null;

  const importantThreshold = 70;
  const attentionCards = snapshot.cards.filter((card) => card.priority >= importantThreshold);
  const featuredAttention = attentionCards[0];
  const contextCards = snapshot.cards.filter((card) => card.priority < importantThreshold && card.domain !== featuredAttention?.domain);
  const remainingAttention = attentionCards.slice(1);
  const importantCount = attentionCards.length;

  return (
    <main className="page-shell">
      <div className="sky-noise" aria-hidden="true" />
      <AppTopbar query={query} scenario={scenario} snapshot={snapshot} importantCount={importantCount} />
      {scenario && <ScenarioSwitcher active={scenario} />}
      <AppTabs active="now" scenario={scenario} />

      <section className="meta-row" aria-label="Метаданные обновления">
        <span>Обновлено: {formatClock(snapshot.generated_at)}</span>
        <span>{importantCount > 0 ? `${importantCount} важных сигналов` : 'без критичных рисков'}</span>
      </section>

      <section id="now" aria-label="Текущая погода">
        <NowOverview current={snapshot.current_weather} snapshot={snapshot} scenario={scenario} featuredAttention={featuredAttention} />
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
        <section className="section-block dashboard-context" id={remainingAttention.length > 0 ? undefined : 'risks'}>
          <div className="section-heading">
            <span>{remainingAttention.length > 0 ? '02' : '01'}</span>
            <h2>Контекст и рекомендации</h2>
          </div>
          <div className="cards-grid compact insight-grid">
            {contextCards.map((card) => (
              <HandoffInsightCard key={card.id} card={card} scenario={scenario} />
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

function AppTopbar({
  query,
  scenario,
  snapshot,
  importantCount
}: {
  query: UseQueryResult<DashboardSnapshot, Error>;
  scenario?: DashboardScenario;
  snapshot: DashboardSnapshot;
  importantCount: number;
}) {
  return (
    <header className="topbar">
      <div>
        <span className="app-label">Погодный ассистент</span>
        <h1>Армавир сейчас</h1>
        <span className="topbar-meta">{snapshot.station_status.label} · {importantCount > 0 ? `${importantCount} важных сигналов` : 'без критичных рисков'} · {formatClock(snapshot.generated_at)}</span>
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

function HeaderStatus({ snapshot, importantCount }: { snapshot: DashboardSnapshot; importantCount: number }) {
  return (
    <div className={`header-status header-status-${snapshot.station_status.severity}`} aria-label="Статус данных">
      <span aria-hidden="true" />
      <div>
        <b>{importantCount > 0 ? `${importantCount} важных` : 'без критичных рисков'}</b>
        <small>{snapshot.station_status.label} · {formatClock(snapshot.generated_at)}</small>
      </div>
    </div>
  );
}

function HandoffInsightCard({ card, scenario }: { card: AttentionCardType; scenario?: DashboardScenario }) {
  return (
    <a className={`handoff-insight-card handoff-insight-${card.severity}`} href={withScenario(signalHref(card), scenario)}>
      <div className="handoff-insight-head">
        <span className="handoff-insight-icon" aria-hidden="true">{card.icon || '•'}</span>
        <span className={`severity-pill severity-${card.severity}`}>{severityLabel(card.severity)}</span>
      </div>
      <div className="handoff-insight-body">
        <div>
          <h3>{card.title}</h3>
          {card.subtitle && <p>{card.subtitle}</p>}
        </div>
        {card.value && <strong>{card.value}<small>{card.unit}</small></strong>}
      </div>
      {card.action && <p className="handoff-insight-action"><span>Что сделать</span>{card.action}</p>}
      <p className="handoff-insight-foot">{card.reason || 'оценка важности'}</p>
    </a>
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

function NowOverview({
  current,
  snapshot,
  scenario,
  featuredAttention
}: {
  current?: CurrentWeatherSummary;
  snapshot: DashboardSnapshot;
  scenario?: DashboardScenario;
  featuredAttention?: AttentionCardType;
}) {
  return (
    <section className={`calm-overview now-overview ${featuredAttention ? 'has-signal' : 'is-calm'}`}>
      {featuredAttention && <SignalFocus card={featuredAttention} scenario={scenario} />}
      {current && <WeatherNow current={current} snapshot={snapshot} scenario={scenario} />}

      <div className="calm-column">
        <EveningInsight snapshot={snapshot} />
        {!featuredAttention && <ControlStatus quietItems={snapshot.quiet.items} />}
        {current && <DailySummaryCard current={current} snapshot={snapshot} />}
      </div>
    </section>
  );
}

type AttentionCardType = DashboardSnapshot['cards'][number];

function SignalFocus({ card, scenario }: { card: AttentionCardType; scenario?: DashboardScenario }) {
  return (
    <a className={`calm-card signal-focus attention-${card.severity}`} href={withScenario(signalHref(card), scenario)}>
      <div className="signal-focus-head">
        <span className="card-icon" aria-hidden="true">{card.icon || '•'}</span>
        <span className={`severity-pill severity-${card.severity}`}>{severityLabel(card.severity)}</span>
      </div>
      <div className="signal-focus-main">
        <div>
          <span className="watch-kicker">главный сигнал</span>
          <h2>{card.title}</h2>
          {card.subtitle && <p>{card.subtitle}</p>}
        </div>
        {card.value && <strong>{card.value}<small>{card.unit}</small></strong>}
      </div>
      {card.action && <p className="signal-focus-action"><span>что сделать</span>{card.action}</p>}
    </a>
  );
}

function signalHref(card: AttentionCardType) {
  switch (card.domain) {
    case 'geomagnetic': return '/app/geomagnetic';
    case 'hydro': return '/app/water';
    case 'rain':
    case 'forecast': return '/app/rain';
    case 'wind': return '/app/wind';
    case 'solar': return '/app/uv';
    case 'station': return '/app/station';
    default: return card.detail_url || '/app/risks';
  }
}

function severityLabel(severity: AttentionCardType['severity']) {
  switch (severity) {
    case 'danger': return 'срочно';
    case 'warning': return 'важно';
    case 'info': return 'заметно';
    case 'normal': return 'норма';
    case 'calm':
    default: return 'спокойно';
  }
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
          <h2>Критичных рисков нет</h2>
          <p>Ветер, вода и геомагнитка без опасных значений; дождь отслеживаю отдельно.</p>
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

function DailySummaryCard({ current, snapshot }: { current: CurrentWeatherSummary; snapshot: DashboardSnapshot }) {
  const rainPeak = snapshot.near_forecast?.reduce<NearForecastItem | undefined>((best, item) => {
    if (!best || item.precipitation_probability > best.precipitation_probability) return item;
    return best;
  }, undefined);

  return (
    <section className="calm-card daily-summary-card">
      <span className="watch-kicker">кратко о дне</span>
      <dl>
        <div>
          <dt>Температура</dt>
          <dd>{formatTemperature(current.temperature)}</dd>
        </div>
        <div>
          <dt>Вер. осадков</dt>
          <dd>{rainPeak ? `${rainPeak.precipitation_probability}%` : 'нет данных'}</dd>
        </div>
        <div>
          <dt>Ветер</dt>
          <dd>{typeof current.wind_speed === 'number' ? `${current.wind_speed.toFixed(1)} м/с` : 'нет данных'}</dd>
        </div>
        <div>
          <dt>UV</dt>
          <dd>{typeof current.uv_index === 'number' ? current.uv_index.toFixed(0) : 'нет данных'}</dd>
        </div>
      </dl>
    </section>
  );
}

function WeatherNow({ current, snapshot, scenario, compact = false }: { current: CurrentWeatherSummary; snapshot: DashboardSnapshot; scenario?: DashboardScenario; compact?: boolean }) {
  const nowStory = buildNowStory(current, snapshot);
  const focusMetric = buildTemperatureMetric(current);

  return (
    <a className={`weather-now weather-now-${nowStory.tone} ${compact ? 'compact' : ''}`} href={withScenario('/app/current', scenario)}>
      <div className="weather-now-top">
        <WeatherGlyph icon={current.icon || '🌤️'} className="weather-icon" />
        <div className="weather-status-stack">
          <span className="severity-pill severity-normal">сейчас</span>
        </div>
      </div>

      <div className="weather-now-main">
        <div className="weather-copy">
          <h2>{current.title}</h2>
          <p>{current.subtitle}</p>
          <TemperatureTrend delta={current.temperature_delta} />
          {nowStory.nextEvent && <p className={`weather-primary-signal signal-${nowStory.tone}`}><span>важно сейчас</span>{primarySignalLabel(nowStory)}</p>}
        </div>
        <div className="weather-temp">
          <span>{focusMetric.value}</span>
          {focusMetric.unit && <small>{focusMetric.unit}</small>}
          <b>{focusMetric.label}</b>
        </div>
      </div>

      <NowStory story={nowStory} />

      <WeatherFacts current={current} story={nowStory} />

      <div className="weather-now-footer">
        <span>наблюдение {formatClock(current.observed_at)}</span>
        <span>открыть детали</span>
      </div>
    </a>
  );
}

type NowStoryTone = 'calm' | 'rain' | 'wind' | 'heat' | 'cold' | 'stale';

type NowStory = {
  tone: NowStoryTone;
  headline: string;
  details: string[];
  metric?: FocusMetric;
  nextEvent?: string;
  action?: string;
};

type FocusMetric = {
  value: string;
  unit?: string;
  label: string;
};

function NowStory({ story }: { story: NowStory }) {
  return (
    <div className="now-story">
      <p className="weather-summary">{story.headline}</p>
      <div className="now-story-details" aria-label="Что важно сейчас">
        {story.details.map((detail) => <span key={detail}>{detail}</span>)}
      </div>
      {story.nextEvent && <p className="now-story-next"><span>следующее</span>{story.nextEvent}</p>}
      {story.action && <p className="now-story-action"><span>что сделать</span>{story.action}</p>}
    </div>
  );
}

function primarySignalLabel(story: NowStory) {
  if (story.tone === 'rain' && story.nextEvent) {
    return story.nextEvent.replace(/^осадки /, 'Дождь ').replace(';', ',');
  }
  return story.nextEvent || story.headline;
}

function buildNowStory(current: CurrentWeatherSummary, snapshot: DashboardSnapshot): NowStory {
  if (!snapshot.station_status.ok) {
    return {
      tone: 'stale',
      headline: 'Текущие значения могут быть устаревшими.',
      details: [snapshot.station_status.label, snapshot.near_forecast?.length ? 'прогноз доступен' : 'прогноза нет'],
      metric: buildStationMetric(snapshot),
      nextEvent: 'показываю прогноз отдельно от датчиков станции',
      action: 'Проверь питание станции, MQTT и связь с сервером'
    };
  }

  const forecast = snapshot.near_forecast ?? [];
  const rainNow = typeof current.rain_rate === 'number' && current.rain_rate >= 0.1;
  const rainSoon = !rainNow ? forecast.find((item) => item.precipitation_probability >= 40 || item.precipitation >= 0.5) : undefined;
  const windGust = typeof current.wind_gust === 'number' ? current.wind_gust : undefined;
  const windSpeed = typeof current.wind_speed === 'number' ? current.wind_speed : undefined;
  const windy = (windGust ?? 0) >= 12 || (windSpeed ?? 0) >= 8;
  const hot = typeof current.temperature === 'number' && current.temperature >= 30;
  const cold = typeof current.temperature === 'number' && current.temperature <= 0;
  const details = buildNowDetails(current, snapshot, rainSoon);

  if (rainNow) {
    return {
      tone: 'rain',
      headline: `Дождь идёт сейчас: ${current.rain_rate?.toFixed(1)} мм/ч.`,
      details,
      metric: { value: current.rain_rate?.toFixed(1) ?? '—', unit: 'мм/ч', label: 'дождь сейчас' },
      nextEvent: buildRainEndsEvent(forecast),
      action: 'Закрой окна и перенеси вещи с улицы, если они под дождём'
    };
  }

  if (rainSoon) {
    return {
      tone: 'rain',
      headline: `Сейчас сухо, но к ${formatClock(rainSoon.time)} дождь вероятен ${rainSoon.precipitation_probability}%.`,
      details,
      metric: { value: rainSoon.precipitation_probability.toFixed(0), unit: '%', label: `дождь к ${formatClock(rainSoon.time)}` },
      nextEvent: `осадки ${relativeTime(rainSoon.time, snapshot.generated_at)}; пик около ${formatClock(rainSoon.time)}`,
      action: 'Если уходишь надолго, закрой окна заранее'
    };
  }

  if (windy) {
    return {
      tone: 'wind',
      headline: windGust ? `Порывы до ${windGust.toFixed(1)} м/с.` : `Ветер ${windSpeed?.toFixed(1)} м/с.`,
      details,
      metric: { value: (windGust ?? windSpeed ?? 0).toFixed(1), unit: 'м/с', label: windGust ? 'порывы ветра' : 'ветер сейчас' },
      nextEvent: buildWindNextEvent(forecast),
      action: 'Закрепи лёгкие предметы на улице и балконе'
    };
  }

  if (hot) {
    return {
      tone: 'heat',
      headline: 'Главный фактор сейчас — жара и нагрузка от солнца.',
      details,
      metric: buildTemperatureMetric(current),
      nextEvent: buildTemperatureNextEvent(forecast),
      action: typeof current.uv_index === 'number' && current.uv_index >= 6 ? 'Лучше держаться тени и пить воду' : undefined
    };
  }

  if (cold) {
    return {
      tone: 'cold',
      headline: 'Главный фактор сейчас — холодный воздух.',
      details,
      metric: buildTemperatureMetric(current),
      nextEvent: buildTemperatureNextEvent(forecast),
      action: 'Одевайся теплее, особенно если ветер усилится'
    };
  }

  return {
    tone: 'calm',
    headline: compactSummary(snapshot.summary || 'Сейчас без срочных погодных сигналов.'),
    details,
    metric: buildTemperatureMetric(current),
    nextEvent: buildCalmNextEvent(forecast)
  };
}

function buildTemperatureMetric(current: CurrentWeatherSummary): FocusMetric {
  return {
    value: formatNumber(current.temperature),
    unit: '°C',
    label: typeof current.feels_like === 'number' ? `ощущается ${current.feels_like.toFixed(1)}°` : 'температура сейчас'
  };
}

function buildStationMetric(snapshot: DashboardSnapshot): FocusMetric {
  if (typeof snapshot.station_status.age_minutes !== 'number') {
    return { value: '—', label: 'нет свежести' };
  }

  if (snapshot.station_status.age_minutes >= 60) {
    return { value: `${Math.round(snapshot.station_status.age_minutes / 60)}ч+`, label: 'без данных' };
  }

  return { value: `${snapshot.station_status.age_minutes}`, unit: 'мин', label: 'без данных' };
}

function buildRainEndsEvent(forecast: NearForecastItem[]) {
  const dry = forecast.find((item) => item.precipitation_probability < 30 && item.precipitation < 0.2);
  return dry ? `осадки ослабнут около ${formatClock(dry.time)}` : 'осадки могут держаться ближайшие часы';
}

function buildWindNextEvent(forecast: NearForecastItem[]) {
  if (forecast.length === 0) return 'следи за порывами на открытых местах';
  const max = forecast.reduce((best, item) => item.wind_speed > best.wind_speed ? item : best, forecast[0]);
  return `пик ветра около ${formatClock(max.time)}: ${max.wind_speed.toFixed(1)} м/с`;
}

function buildTemperatureNextEvent(forecast: NearForecastItem[]) {
  if (forecast.length < 2) return undefined;
  const first = forecast[0];
  const last = forecast[forecast.length - 1];
  const delta = last.temperature - first.temperature;
  if (Math.abs(delta) < 2) return `температура останется около ${Math.round(last.temperature)}°`;
  return `${delta > 0 ? 'потеплеет' : 'похолодает'} до ${Math.round(last.temperature)}° к ${formatClock(last.time)}`;
}

function buildCalmNextEvent(forecast: NearForecastItem[]) {
  if (forecast.length === 0) return undefined;
  const rainy = forecast.find((item) => item.precipitation_probability >= 30 || item.precipitation >= 0.2);
  if (rainy) return `возможны осадки около ${formatClock(rainy.time)} (${rainy.precipitation_probability}%)`;
  const windy = forecast.find((item) => item.wind_speed >= 7);
  if (windy) return `ветер заметнее около ${formatClock(windy.time)}: ${windy.wind_speed.toFixed(1)} м/с`;
  const last = forecast[forecast.length - 1];
  return `до ${formatClock(last.time)} без заметных погодных сигналов`;
}

function relativeTime(target: string, base: string) {
  const targetTime = new Date(target).getTime();
  const baseTime = new Date(base).getTime();
  const diffMinutes = Math.round((targetTime - baseTime) / 60_000);
  if (!Number.isFinite(diffMinutes)) return `к ${formatClock(target)}`;
  if (diffMinutes <= 0) return 'уже рядом';
  if (diffMinutes < 60) return `через ${diffMinutes} мин`;
  const hours = Math.floor(diffMinutes / 60);
  const minutes = diffMinutes % 60;
  return minutes > 0 ? `через ${hours} ч ${minutes} мин` : `через ${hours} ч`;
}

function formatTemperature(value?: number) {
  return typeof value === 'number' ? `${value.toFixed(1)}°C` : 'нет данных';
}

function buildNowDetails(current: CurrentWeatherSummary, snapshot: DashboardSnapshot, rainSoon?: NearForecastItem) {
  const details: string[] = [];

  if (typeof current.rain_rate === 'number') {
    details.push(current.rain_rate >= 0.1 ? `дождь ${current.rain_rate.toFixed(1)} мм/ч` : 'сейчас сухо');
  }
  if (rainSoon) {
    details.push(`осадки около ${formatClock(rainSoon.time)}`);
  }
  if (typeof current.wind_speed === 'number') {
    details.push(current.wind_gust && current.wind_gust >= 8 ? `ветер ${current.wind_speed.toFixed(1)}, порывы ${current.wind_gust.toFixed(1)} м/с` : `ветер ${current.wind_speed.toFixed(1)} м/с`);
  }
  if (typeof current.pressure_delta === 'number' && Math.abs(current.pressure_delta) >= 1) {
    details.push(current.pressure_delta > 0 ? `давление растёт ${current.pressure_delta.toFixed(1)} мм/ч` : `давление падает ${Math.abs(current.pressure_delta).toFixed(1)} мм/ч`);
  }
  if (typeof current.uv_index === 'number' && current.uv_index >= 6) {
    details.push(`UV ${current.uv_index.toFixed(0)} высокий`);
  }
  if (snapshot.quiet.items.length > 0 && details.length < 4) {
    details.push(`${snapshot.quiet.items.slice(0, 3).join(', ')} в норме`);
  }

  return details.slice(0, 4);
}

function TemperatureTrend({ delta }: { delta?: number }) {
  if (typeof delta !== 'number' || Math.abs(delta) < 0.2) {
    return <span className="trend-chip trend-flat">температура стабильна</span>;
  }

  const falling = delta < 0;
  return (
    <span className={`trend-chip ${falling ? 'trend-down' : 'trend-up'}`} title={`${falling ? 'Холодает' : 'Теплеет'} на ${Math.abs(delta).toFixed(1)}° за час`}>
      {falling ? '↓' : '↑'} {Math.abs(delta).toFixed(1)}°/ч
    </span>
  );
}

function WeatherFacts({ current, story }: { current: CurrentWeatherSummary; story: NowStory }) {
  const rainFactValue = buildRainFactValue(current, story);
  const facts = [
    typeof current.humidity === 'number' ? { key: 'humidity', icon: 'drop', label: 'Влажность', value: `${current.humidity}%` } : null,
    typeof current.pressure === 'number' ? { key: 'pressure', icon: 'gauge', label: 'Давление', value: `${Math.round(current.pressure)} мм` } : null,
    typeof current.wind_speed === 'number' ? { key: 'wind', icon: 'wind', label: 'Ветер', value: `${current.wind_speed.toFixed(1)} м/с` } : null,
    typeof current.rain_rate === 'number' && current.rain_rate > 0
      ? { key: 'rain', icon: 'umbrella', label: 'Осадки', value: `${current.rain_rate.toFixed(1)} мм/ч` }
      : { key: 'rain', icon: 'umbrella', label: story.tone === 'rain' ? 'Дождь' : 'Осадки', value: rainFactValue },
    typeof current.uv_index === 'number' ? { key: 'uv', icon: 'sun', label: 'UV индекс', value: current.uv_index.toFixed(0) } : null
  ].filter((item): item is { key: string; icon: MetricIconName; label: string; value: string } => Boolean(item));
  const primaryKey = primaryFactKey(story.tone);
  const sortedFacts = [...facts].sort((a, b) => {
    if (a.key === primaryKey) return -1;
    if (b.key === primaryKey) return 1;
    return 0;
  });

  return (
    <div className="weather-facts">
      {sortedFacts.map((fact) => (
        <span key={fact.key} className={fact.key === primaryKey ? 'primary' : undefined}>
          <MetricIcon name={fact.icon} />
          <small>{fact.label}</small>
          <b>{fact.value}</b>
        </span>
      ))}
    </div>
  );
}

function buildRainFactValue(current: CurrentWeatherSummary, story: NowStory) {
  if (typeof current.rain_rate === 'number' && current.rain_rate > 0) return `${current.rain_rate.toFixed(1)} мм/ч`;
  if (story.tone === 'rain' && story.nextEvent) {
    const match = story.nextEvent.match(/через ([^;]+)/);
    if (match) return `через ${match[1]}`;
    if (story.nextEvent.includes('уже рядом')) return 'скоро';
  }
  return 'нет';
}

function primaryFactKey(tone: NowStoryTone) {
  switch (tone) {
    case 'rain': return 'rain';
    case 'wind': return 'wind';
    case 'heat': return 'uv';
    case 'cold': return 'wind';
    case 'stale': return 'pressure';
    case 'calm':
    default: return 'humidity';
  }
}

type MetricIconName = 'drop' | 'gauge' | 'wind' | 'umbrella' | 'sun';

function MetricIcon({ name }: { name: MetricIconName }) {
  const common = { fill: 'none', stroke: 'currentColor', strokeWidth: 1.8, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const };
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      {name === 'drop' && <path d="M12 3.8s6 6.2 6 10.3a6 6 0 0 1-12 0C6 10 12 3.8 12 3.8Z" {...common} />}
      {name === 'gauge' && <><path d="M4.8 16.8a8 8 0 1 1 14.4 0" {...common} /><path d="m12 14 3.2-4.2" {...common} /><path d="M8 16.8h8" {...common} /></>}
      {name === 'wind' && <><path d="M3.8 8.5h10.6a2.5 2.5 0 1 0-2.5-2.5" {...common} /><path d="M3.8 13h15.4a2.7 2.7 0 1 1-2.7 2.7" {...common} /><path d="M3.8 17.5h8.4" {...common} /></>}
      {name === 'umbrella' && <><path d="M4 12a8 8 0 0 1 16 0H4Z" {...common} /><path d="M12 12v5.2a2 2 0 0 0 4 0" {...common} /></>}
      {name === 'sun' && <><circle cx="12" cy="12" r="3.6" {...common} /><path d="M12 3.2v2M12 18.8v2M4.3 4.3l1.4 1.4M18.3 18.3l1.4 1.4M3.2 12h2M18.8 12h2M4.3 19.7l1.4-1.4M18.3 5.7l1.4-1.4" {...common} /></>}
    </svg>
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
          <article className={`forecast-hour ${item.precipitation_probability >= 50 ? 'forecast-hour-rain' : ''}`} key={item.time} role="listitem">
            <time>{formatClock(item.time)}</time>
            <WeatherGlyph icon={displayForecastIcon(item)} className="forecast-icon" />
            <strong>{Math.round(item.temperature)}°</strong>
            {forecastMicrocopy(item) && <small>{forecastMicrocopy(item)}</small>}
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

  const eveningForecast = forecast.filter((item) => {
    const hour = new Date(item.time).getHours();
    return hour >= 18 || hour < 5;
  });
  const relevantForecast = eveningForecast.length > 0 ? eveningForecast : forecast;
  const first = relevantForecast[0];
  const last = relevantForecast[relevantForecast.length - 1];
  const rainy = relevantForecast.find((item) => item.precipitation_probability >= 40 || item.precipitation >= 0.5);
  const maxWind = Math.max(...relevantForecast.map((item) => item.wind_speed));
  const tempDelta = last.temperature - first.temperature;
  const tempPhrase = Math.abs(tempDelta) >= 1
    ? `${tempDelta < 0 ? 'прохладнее' : 'теплее'}, около ${Math.round(last.temperature)}°`
    : `около ${Math.round(last.temperature)}°`;
  const rainPhrase = rainy
    ? `дождь вероятен около ${formatClock(rainy.time)}`
    : 'дождя почти нет';
  const windPhrase = maxWind >= 8
    ? 'ветер будет заметным'
    : 'ветер слабый';

  return {
    title: `К вечеру ${tempPhrase}`,
    text: `${rainPhrase[0].toUpperCase()}${rainPhrase.slice(1)}, ${windPhrase}.`
  };
}

function forecastMicrocopy(item: NearForecastItem) {
  if (item.precipitation_probability >= 15) return `${item.precipitation_probability}% дождь`;
  if (item.wind_speed >= 7) return `${item.wind_speed.toFixed(1)} м/с`;
  return '';
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

function formatNumber(value?: number) {
  if (typeof value !== 'number') return '—';
  return value.toFixed(1);
}
