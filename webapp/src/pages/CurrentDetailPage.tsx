import { useMemo } from 'react';
import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { CurrentWeatherSummary, DashboardSnapshot, NearForecastItem } from '../api/dashboard';
import { fetchWeatherChart, type WeatherChartData } from '../api/weather';
import { getDashboardScenarioLabel, type DashboardScenario } from '../api/mockDashboard';
import { ApiErrorCard } from '../components/ApiErrorCard';
import { AppTabs, withScenario } from '../components/AppTabs';
import { DashboardSkeleton } from '../components/Skeleton';
import { WeatherGlyph } from '../components/WeatherGlyph';
import { formatClock } from '../utils/time';
import { displayForecastIcon } from './DashboardPage';

const chartFields = ['temp_outdoor', 'humidity_outdoor', 'pressure_relative', 'wind_speed', 'rain_rate', 'uv_index'];

const fieldLabels: Record<string, { label: string; unit: string; tone: string; precision?: number }> = {
  temp_outdoor: { label: 'Температура', unit: '°C', tone: 'temperature', precision: 1 },
  humidity_outdoor: { label: 'Влажность', unit: '%', tone: 'humidity' },
  pressure_relative: { label: 'Давление', unit: 'мм', tone: 'pressure' },
  wind_speed: { label: 'Ветер', unit: 'м/с', tone: 'wind', precision: 1 },
  rain_rate: { label: 'Дождь', unit: 'мм/ч', tone: 'rain', precision: 1 },
  uv_index: { label: 'UV', unit: '', tone: 'uv', precision: 0 }
};

export function CurrentDetailPage({ query, scenario }: { query: UseQueryResult<DashboardSnapshot, Error>; scenario?: DashboardScenario }) {
  if (query.isLoading) return <DashboardSkeleton />;

  if (query.isError) return <ApiErrorCard title="Не удалось загрузить детали" message={query.error.message} onRetry={() => query.refetch()} />;

  const snapshot = query.data;
  const current = snapshot?.current_weather;
  if (!snapshot) return null;
  if (!current) return <NoCurrentWeather snapshot={snapshot} scenario={scenario} onRefresh={() => query.refetch()} isRefreshing={query.isFetching} />;

  return <CurrentDetailContent snapshot={snapshot} current={current} scenario={scenario} onRefresh={() => query.refetch()} isRefreshing={query.isFetching} />;
}

function NoCurrentWeather({
  snapshot,
  scenario,
  onRefresh,
  isRefreshing
}: {
  snapshot: DashboardSnapshot;
  scenario?: DashboardScenario;
  onRefresh: () => void;
  isRefreshing: boolean;
}) {
  return (
    <main className="page-shell current-page">
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Погодный ассистент</span>
          <strong>Сейчас подробно</strong>
        </div>
        <div className="topbar-actions">
          {scenario && <span className="scenario-badge">сценарий: {getDashboardScenarioLabel(scenario)}</span>}
          <button className="refresh-button" onClick={onRefresh} disabled={isRefreshing}>
            {isRefreshing ? 'Обновляю…' : 'Обновить'}
          </button>
          <a className="refresh-button" href={withScenario('/app/', scenario)}>Назад</a>
        </div>
      </header>
      <AppTabs active="now" scenario={scenario} />

      <section className="current-hero no-current-hero">
        <div className="current-hero-copy">
          <span className="headline-kicker">текущие измерения недоступны</span>
          <h1>{snapshot.headline.title || 'Станция молчит'}</h1>
          <p>{snapshot.summary || 'Нет свежих показаний метеостанции. Прогноз и остальные источники могут быть доступны отдельно.'}</p>
          <div className="current-hero-pills">
            <span>{snapshot.station_status.label}</span>
            <span>{snapshot.near_forecast?.length ? 'прогноз доступен' : 'прогноза нет'}</span>
          </div>
        </div>
        <div className="current-temp-dial" aria-hidden="true">
          <span className="current-temp-icon">📡</span>
          <strong>—</strong>
          <small>нет данных</small>
        </div>
      </section>

      <section className="current-insight-grid">
        <article className="risk-panel action-panel">
          <span className="watch-kicker">что сделать</span>
          <h2>Проверь питание, MQTT и связь</h2>
          <p>Если сервер жив, но метеостанция не присылает измерения, текущие значения лучше не использовать для решений.</p>
        </article>
        <article className="risk-panel">
          <span className="watch-kicker">статус</span>
          <h2>{snapshot.station_status.label}</h2>
          <p>Дашборд собран в {formatClock(snapshot.generated_at)}.</p>
        </article>
      </section>
    </main>
  );
}

function CurrentDetailContent({
  snapshot,
  current,
  scenario,
  onRefresh,
  isRefreshing
}: {
  snapshot: DashboardSnapshot;
  current: CurrentWeatherSummary;
  scenario?: DashboardScenario;
  onRefresh: () => void;
  isRefreshing: boolean;
}) {
  const observedAt = useMemo(() => new Date(current.observed_at), [current.observed_at]);
  const chartRange = useMemo(() => {
    const to = Number.isNaN(observedAt.getTime()) ? new Date() : observedAt;
    return { to, from: new Date(to.getTime() - 24 * 60 * 60 * 1000) };
  }, [observedAt]);

  const liveChartQuery = useQuery({
    queryKey: ['weather-chart-24h', chartRange.from.toISOString(), chartRange.to.toISOString()],
    queryFn: () => fetchWeatherChart({ from: chartRange.from, to: chartRange.to, interval: '1h', fields: chartFields }),
    enabled: !scenario
  });

  const chart = scenario ? buildMockChart(snapshot) : liveChartQuery.data;
  const explanation = buildComfortExplanation(snapshot, current);

  return (
    <main className="page-shell current-page">
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Погодный ассистент</span>
          <strong>Сейчас подробно</strong>
        </div>
        <div className="topbar-actions">
          {scenario && <span className="scenario-badge">сценарий: {getDashboardScenarioLabel(scenario)}</span>}
          <button className="refresh-button" onClick={onRefresh} disabled={isRefreshing}>
            {isRefreshing ? 'Обновляю…' : 'Обновить'}
          </button>
          <a className="refresh-button" href={withScenario('/app/', scenario)}>Назад</a>
        </div>
      </header>
      <AppTabs active="now" scenario={scenario} />

      <section className="current-hero">
        <div className="current-hero-copy">
          <span className="headline-kicker">Армавир · наблюдение {formatClock(current.observed_at)}</span>
          <h1>{current.title}</h1>
          <p>{current.subtitle}. {snapshot.summary}</p>
          <div className="current-hero-pills" aria-label="Ключевые показатели">
            {metricPills(current).map((pill) => <span key={pill}>{pill}</span>)}
          </div>
        </div>
        <div className="current-temp-dial" aria-label={`Температура ${formatMetric(current.temperature, 1)} градусов`}>
          <WeatherGlyph icon={current.icon || '🌤️'} className="current-temp-icon" />
          <strong>{formatMetric(current.temperature, 1)}</strong>
          <small>°C</small>
        </div>
      </section>

      <section className="metric-detail-grid" aria-label="Детальные показатели">
        {buildMetricCards(current, snapshot).map((metric) => (
          <article className={`metric-detail-card metric-${metric.tone}`} key={metric.label}>
            <span>{metric.label}</span>
            <strong>{metric.value}</strong>
            <p>{metric.hint}</p>
          </article>
        ))}
      </section>

      <section className="current-insight-grid">
        <article className="risk-panel action-panel">
          <span className="watch-kicker">почему такой статус</span>
          <h2>{explanation.title}</h2>
          <p>{explanation.text}</p>
        </article>
        <article className="risk-panel">
          <span className="watch-kicker">свежесть данных</span>
          <h2>{snapshot.station_status.label}</h2>
          <p>{freshnessText(snapshot)} Дашборд собран в {formatClock(snapshot.generated_at)}.</p>
        </article>
      </section>

      <section className="section-block">
        <div className="section-heading forecast-heading">
          <div>
            <span>сутки</span>
            <h2>Как менялись показатели</h2>
          </div>
          {liveChartQuery.isFetching && !scenario && <small className="chart-state">загружаю графики…</small>}
          {liveChartQuery.isError && !scenario && <small className="chart-state chart-error">графики недоступны</small>}
        </div>
        {chart ? <ChartGrid chart={chart} /> : <ChartSkeletonGrid />}
      </section>

      {snapshot.near_forecast && snapshot.near_forecast.length > 0 && (
        <section className="section-block">
          <div className="section-heading forecast-heading">
            <div>
              <span>прогноз</span>
              <h2>Что будет рядом</h2>
            </div>
            <a href={withScenario('/app/forecast', scenario)}>Открыть прогноз</a>
          </div>
          <div className="current-forecast-mini">
            {snapshot.near_forecast.slice(0, 5).map((item) => <MiniForecast key={item.time} item={item} />)}
          </div>
        </section>
      )}
    </main>
  );
}

function buildMetricCards(current: CurrentWeatherSummary, snapshot: DashboardSnapshot) {
  const stationHint = snapshot.station_status.ok ? 'можно доверять текущим значениям' : 'текущие значения могут быть устаревшими';

  return [
    {
      label: 'Ощущается',
      value: `${formatMetric(current.feels_like ?? current.temperature, 1)}°`,
      hint: temperatureHint(current),
      tone: 'temperature'
    },
    {
      label: 'Влажность',
      value: typeof current.humidity === 'number' ? `${current.humidity}%` : '—',
      hint: humidityHint(current.humidity),
      tone: 'humidity'
    },
    {
      label: 'Давление',
      value: typeof current.pressure === 'number' ? `${Math.round(current.pressure)} мм` : '—',
      hint: pressureHint(current.pressure, current.pressure_delta),
      tone: 'pressure'
    },
    {
      label: 'Ветер',
      value: typeof current.wind_speed === 'number' ? `${current.wind_speed.toFixed(1)} м/с` : '—',
      hint: typeof current.wind_gust === 'number' ? `порывы до ${current.wind_gust.toFixed(1)} м/с` : 'порывы не переданы',
      tone: 'wind'
    },
    {
      label: 'Дождь',
      value: typeof current.rain_rate === 'number' ? `${current.rain_rate.toFixed(1)} мм/ч` : '—',
      hint: current.rain_rate && current.rain_rate > 0 ? 'осадки идут прямо сейчас' : 'по станции сейчас сухо',
      tone: 'rain'
    },
    {
      label: 'Станция',
      value: snapshot.station_status.ok ? 'онлайн' : 'пауза',
      hint: stationHint,
      tone: snapshot.station_status.ok ? 'station' : 'danger'
    }
  ];
}

function ChartGrid({ chart }: { chart: WeatherChartData }) {
  return (
    <div className="chart-grid">
      {chartFields.map((field) => {
        const meta = fieldLabels[field];
        const values = compactSeries(chart.datasets[field] ?? []);
        const latest = lastFinite(values);
        return (
          <article className={`spark-card spark-${meta.tone}`} key={field}>
            <div className="spark-card-head">
              <span>{meta.label}</span>
              <strong>{typeof latest === 'number' ? formatMetric(latest, meta.precision ?? 0) : '—'}{meta.unit && <small> {meta.unit}</small>}</strong>
            </div>
            <Sparkline values={values} />
          </article>
        );
      })}
    </div>
  );
}

function Sparkline({ values }: { values: number[] }) {
  const finite = values.filter((value) => Number.isFinite(value));
  if (finite.length < 2) {
    return <div className="spark-empty">мало данных</div>;
  }

  const min = Math.min(...finite);
  const max = Math.max(...finite);
  const range = max - min || 1;
  const points = values.map((value, index) => {
    const x = values.length === 1 ? 0 : (index / (values.length - 1)) * 100;
    const safe = Number.isFinite(value) ? value : min;
    const y = 84 - ((safe - min) / range) * 68;
    return `${x.toFixed(2)},${y.toFixed(2)}`;
  }).join(' ');

  return (
    <svg className="sparkline" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
      <polyline className="spark-area" points={`0,100 ${points} 100,100`} />
      <polyline className="spark-line" points={points} />
    </svg>
  );
}

function ChartSkeletonGrid() {
  return (
    <div className="chart-grid">
      {chartFields.map((field) => <div className="skeleton spark-card" key={field} />)}
    </div>
  );
}

function MiniForecast({ item }: { item: NearForecastItem }) {
  return (
    <article className="mini-forecast-card">
      <time>{formatClock(item.time)}</time>
      <WeatherGlyph icon={displayForecastIcon(item)} />
      <strong>{Math.round(item.temperature)}°</strong>
      <small>{item.precipitation_probability > 0 ? `${item.precipitation_probability}% дождь` : item.weather_description}</small>
    </article>
  );
}

function buildComfortExplanation(snapshot: DashboardSnapshot, current: CurrentWeatherSummary) {
  const important = snapshot.cards.find((card) => card.priority >= 70);
  if (important) {
    return {
      title: important.title,
      text: `${important.reason || important.subtitle || 'Есть показатель, который вышел выше обычного уровня.'}${important.action ? ` ${important.action}.` : ''}`
    };
  }

  const factors = [
    current.rain_rate && current.rain_rate > 0 ? 'идёт дождь' : 'дождя сейчас нет',
    typeof current.wind_speed === 'number' && current.wind_speed >= 8 ? 'ветер заметный' : 'ветер слабый',
    typeof current.uv_index === 'number' && current.uv_index >= 6 ? 'UV высокий' : 'UV не мешает',
    snapshot.station_status.ok ? 'данные свежие' : 'данные устарели'
  ];

  return {
    title: snapshot.headline.title || current.title,
    text: `${factors.join(', ')}. Поэтому дашборд оставляет главный статус спокойным и показывает показатели как контекст.`
  };
}

function buildMockChart(snapshot: DashboardSnapshot): WeatherChartData {
  const current = snapshot.current_weather;
  const labels: string[] = [];
  const datasets: Record<string, number[]> = Object.fromEntries(chartFields.map((field) => [field, []]));
  const end = current ? new Date(current.observed_at) : new Date();
  const startTemp = (current?.temperature ?? 22) - 2;
  const forecast = snapshot.near_forecast ?? [];

  for (let index = 0; index < 13; index += 1) {
    const t = new Date(end.getTime() - (12 - index) * 2 * 60 * 60 * 1000);
    labels.push(t.toISOString());
    const wave = Math.sin(index / 2.1);
    datasets.temp_outdoor.push(round(startTemp + wave * 1.8 + index * 0.12, 1));
    datasets.humidity_outdoor.push(round((current?.humidity ?? 66) + Math.cos(index / 2) * 7, 0));
    datasets.pressure_relative.push(round((current?.pressure ?? 744) + Math.sin(index / 3) * 1.8, 0));
    datasets.wind_speed.push(round(Math.max(0, (current?.wind_speed ?? 2) + Math.sin(index) * 1.1), 1));
    datasets.rain_rate.push(0);
    datasets.uv_index.push(round(Math.max(0, (current?.uv_index ?? 0) + Math.sin((index - 4) / 2) * 1.4), 0));
  }

  if (forecast.some((item) => item.precipitation_probability >= 40)) {
    datasets.rain_rate[datasets.rain_rate.length - 2] = 1.2;
    datasets.rain_rate[datasets.rain_rate.length - 1] = 0.6;
  }

  return { labels, datasets };
}

function metricPills(current: CurrentWeatherSummary) {
  return [
    typeof current.feels_like === 'number' ? `ощущается ${current.feels_like.toFixed(1)}°` : null,
    typeof current.humidity === 'number' ? `влажность ${current.humidity}%` : null,
    typeof current.pressure === 'number' ? `давление ${Math.round(current.pressure)} мм` : null,
    typeof current.wind_speed === 'number' ? `ветер ${current.wind_speed.toFixed(1)} м/с` : null,
    typeof current.rain_rate === 'number' && current.rain_rate > 0 ? `дождь ${current.rain_rate.toFixed(1)} мм/ч` : 'дождя нет'
  ].filter((item): item is string => Boolean(item));
}

function temperatureHint(current: CurrentWeatherSummary) {
  if (typeof current.temperature_delta === 'number' && Math.abs(current.temperature_delta) >= 0.2) {
    return current.temperature_delta < 0 ? `холодает на ${Math.abs(current.temperature_delta).toFixed(1)}°/ч` : `теплеет на ${current.temperature_delta.toFixed(1)}°/ч`;
  }
  return 'температура почти не меняется';
}

function humidityHint(value?: number) {
  if (typeof value !== 'number') return 'датчик не передал значение';
  if (value < 35) return 'сухой воздух';
  if (value > 80) return 'очень влажно';
  return 'комфортный диапазон';
}

function pressureHint(value?: number, delta?: number) {
  if (typeof value !== 'number') return 'датчик не передал значение';
  if (typeof delta === 'number' && Math.abs(delta) >= 1) {
    return delta > 0 ? `растёт на ${delta.toFixed(1)} мм/ч` : `падает на ${Math.abs(delta).toFixed(1)} мм/ч`;
  }
  return 'без резкого изменения';
}

function freshnessText(snapshot: DashboardSnapshot) {
  if (typeof snapshot.station_status.age_minutes === 'number') {
    return snapshot.station_status.age_minutes < 60
      ? `Последнее наблюдение было ${snapshot.station_status.age_minutes} мин назад.`
      : `Последнее наблюдение было примерно ${Math.round(snapshot.station_status.age_minutes / 60)} ч назад.`;
  }
  return snapshot.station_status.ok ? 'Станция отвечает штатно.' : 'Нет точного времени последнего наблюдения.';
}

function compactSeries(values: number[]) {
  return values.filter((value) => Number.isFinite(value));
}

function lastFinite(values: number[]) {
  for (let index = values.length - 1; index >= 0; index -= 1) {
    if (Number.isFinite(values[index])) return values[index];
  }
  return undefined;
}

function formatMetric(value?: number, precision = 0) {
  if (typeof value !== 'number' || Number.isNaN(value)) return '—';
  return value.toFixed(precision);
}

function round(value: number, precision: number) {
  const power = 10 ** precision;
  return Math.round(value * power) / power;
}
