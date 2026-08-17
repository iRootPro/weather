import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { type WeatherArchive, type WeatherArchiveDay, type WeatherArchiveSummary, fetchWeatherArchive } from '../api/weather';
import { ApiErrorCard } from '../components/ApiErrorCard';
import { AppTabs } from '../components/AppTabs';
import type { DashboardScenario } from '../api/mockDashboard';

type ArchivePeriod = WeatherArchive['period'];

const periodOptions: Array<{ value: ArchivePeriod; label: string }> = [
  { value: 'month', label: 'месяц' },
  { value: 'season', label: 'сезон' },
  { value: 'year', label: 'год' }
];

export function ArchivePage({ scenario }: { scenario?: DashboardScenario }) {
  const [period, setPeriod] = useState<ArchivePeriod>('month');
  const [month, setMonth] = useState(currentMonth());
  const archiveQuery = useQuery({
    queryKey: ['weather-archive', period, month],
    queryFn: () => fetchWeatherArchive({ period, month }),
    staleTime: 10 * 60_000,
    refetchInterval: false
  });

  return (
    <main className="page-shell archive-page">
      <div className="sky-noise" aria-hidden="true" />
      <header className="topbar">
        <div>
          <span className="app-label">Погодный ассистент</span>
          <strong>Архив станции</strong>
        </div>
      </header>
      <AppTabs active="archive" scenario={scenario} />

      <section className="archive-intro" aria-labelledby="archive-title">
        <div>
          <span className="watch-kicker">наблюдения станции</span>
          <h1 id="archive-title">Погода за период</h1>
          <p>Средние, экстремумы и осадки по собственным измерениям станции — без климатической «нормы».</p>
        </div>
        <div className="archive-controls" aria-label="Период архива">
          <div className="archive-period-switch" role="group" aria-label="Тип периода">
            {periodOptions.map((option) => (
              <button key={option.value} className={period === option.value ? 'active' : undefined} onClick={() => setPeriod(option.value)}>
                {option.label}
              </button>
            ))}
          </div>
          <label>
            <span>Опорный месяц</span>
            <input type="month" value={month} max={currentMonth()} onChange={(event) => setMonth(event.target.value)} />
          </label>
        </div>
      </section>

      {archiveQuery.isPending && <ArchiveLoading />}
      {archiveQuery.isError && <ApiErrorCard title="Не удалось загрузить архив" message={archiveQuery.error.message} onRetry={() => archiveQuery.refetch()} />}
      {archiveQuery.data && <ArchiveContent archive={archiveQuery.data} isRefreshing={archiveQuery.isFetching} />}
    </main>
  );
}

function ArchiveContent({ archive, isRefreshing }: { archive: WeatherArchive; isRefreshing: boolean }) {
  const { summary } = archive;
  const temperatureDelta = comparisonDelta(summary.temp_avg, archive.comparison.summary.temp_avg);
  const rainDelta = archive.comparison.available ? summary.rain_total - archive.comparison.summary.rain_total : undefined;

  return (
    <>
      <div className="archive-period-meta">
        <strong>{archive.label}</strong>
        <span>{formatPeriodDates(archive.start_date, archive.end_date)}</span>
        {isRefreshing && <small>обновляю…</small>}
      </div>

      <section className="archive-summary-grid" aria-label="Сводка выбранного периода">
        <SummaryCard label="Средняя температура" value={formatTemperature(summary.temp_avg)} unit="°C" note={summary.temp_avg === undefined ? 'температурных данных нет' : 'среднее суточных средних'} tone="temperature" />
        <SummaryCard label="Минимум" value={formatTemperature(summary.temp_min)} unit="°C" note="абсолютный минимум периода" tone="cold" />
        <SummaryCard label="Максимум" value={formatTemperature(summary.temp_max)} unit="°C" note="абсолютный максимум периода" tone="heat" />
        <SummaryCard label="Осадки" value={summary.rain_total.toFixed(1)} unit="мм" note={summary.rain_total > 0 ? 'сумма по дням' : 'по станции сухо'} tone="rain" />
      </section>

      <section className="archive-context-grid">
        <article className="archive-comparison-card">
          <span className="watch-kicker">сравнение</span>
          <h2>{archive.comparison.label}</h2>
          {archive.comparison.available ? (
            <div className="archive-delta-grid">
              <Delta label="Температура" value={temperatureDelta} unit="°C" />
              <Delta label="Осадки" value={rainDelta} unit="мм" />
            </div>
          ) : <p>Для сравнения пока недостаточно данных станции за предыдущий период.</p>}
        </article>
        <article className="archive-coverage-card">
          <span className="watch-kicker">покрытие</span>
          <strong>{summary.coverage_percent}%</strong>
          <p>{summary.days_with_data} из {summary.days_in_period} календарных дней содержат наблюдения.</p>
          <div className="archive-coverage-rail" aria-label={`Покрытие данных ${summary.coverage_percent}%`}><i style={{ width: `${summary.coverage_percent}%` }} /></div>
        </article>
      </section>

      <section className="section-block archive-temperature-section">
        <div className="section-heading">
          <div>
            <span>температура по дням</span>
            <h2>Минимум → средняя → максимум</h2>
          </div>
          <small>{archive.days.length ? `${archive.days.length} дней с данными` : 'нет наблюдений'}</small>
        </div>
        <TemperatureRangeChart days={archive.days} />
      </section>

      <section className="section-block archive-table-section">
        <div className="section-heading">
          <div>
            <span>таблица</span>
            <h2>Суточные значения</h2>
          </div>
          <small>°C · мм</small>
        </div>
        <DailyTable days={archive.days} />
      </section>
    </>
  );
}

function SummaryCard({ label, value, unit, note, tone }: { label: string; value: string; unit: string; note: string; tone: string }) {
  return <article className={`archive-summary-card archive-summary-${tone}`}><span>{label}</span><strong>{value}<small>{unit}</small></strong><p>{note}</p></article>;
}

function Delta({ label, value, unit }: { label: string; value?: number; unit: string }) {
  const tone = typeof value === 'number' && Math.abs(value) >= 0.05 ? (value > 0 ? 'up' : 'down') : 'flat';
  const text = typeof value === 'number' ? `${value > 0 ? '+' : ''}${value.toFixed(1)}` : '—';
  return <div className={`archive-delta archive-delta-${tone}`}><span>{label}</span><strong>{text}<small>{unit}</small></strong></div>;
}

function TemperatureRangeChart({ days }: { days: WeatherArchiveDay[] }) {
  const data = useMemo(() => days.filter((day) => day.temp_min !== undefined && day.temp_avg !== undefined && day.temp_max !== undefined), [days]);
  if (!data.length) return <div className="archive-empty">В этом периоде пока нет полных суточных температурных наблюдений.</div>;

  const allValues = data.flatMap((day) => [day.temp_min!, day.temp_avg!, day.temp_max!]);
  const min = Math.min(...allValues);
  const max = Math.max(...allValues);
  const range = max - min || 1;
  const point = (value: number) => 88 - ((value - min) / range) * 70;
  const x = (index: number) => data.length === 1 ? 50 : 4 + index / (data.length - 1) * 92;
  const avgPoints = data.map((day, index) => `${x(index).toFixed(2)},${point(day.temp_avg!).toFixed(2)}`).join(' ');

  return (
    <div className="archive-chart-wrap">
      <div className="archive-chart-scale"><span>{max.toFixed(0)}°</span><span>{min.toFixed(0)}°</span></div>
      <svg className="archive-temperature-chart" viewBox="0 0 100 100" preserveAspectRatio="none" role="img" aria-label={`Диапазон температуры от ${min.toFixed(1)} до ${max.toFixed(1)} градусов`}>
        {data.map((day, index) => <line key={day.date} className="archive-range-line" x1={x(index)} x2={x(index)} y1={point(day.temp_min!)} y2={point(day.temp_max!)} />)}
        <polyline className="archive-average-line" points={avgPoints} />
        {data.map((day, index) => <circle key={`${day.date}-avg`} className="archive-average-dot" cx={x(index)} cy={point(day.temp_avg!)} r="1.25" />)}
      </svg>
      <div className="archive-chart-legend"><span><i className="range" />min–max</span><span><i className="average" />средняя</span></div>
    </div>
  );
}

function DailyTable({ days }: { days: WeatherArchiveDay[] }) {
  if (!days.length) return <div className="archive-empty">Наблюдений за выбранный период нет.</div>;
  return (
    <div className="archive-table-scroll">
      <table className="archive-table">
        <thead><tr><th>Дата</th><th>Мин.</th><th>Средняя</th><th>Макс.</th><th>Осадки</th></tr></thead>
        <tbody>{[...days].reverse().map((day) => <tr key={day.date}><th scope="row">{formatDay(day.date)}</th><td>{formatTemperature(day.temp_min)}</td><td>{formatTemperature(day.temp_avg)}</td><td>{formatTemperature(day.temp_max)}</td><td>{day.rain_total === undefined ? '—' : day.rain_total.toFixed(1)}</td></tr>)}</tbody>
      </table>
    </div>
  );
}

function ArchiveLoading() {
  return <div className="archive-loading" aria-label="Загрузка архива"><i /><i /><i /><i /></div>;
}

function currentMonth() {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

function formatTemperature(value?: number) {
  return typeof value === 'number' ? value.toFixed(1) : '—';
}

function comparisonDelta(current?: number, previous?: number) {
  return typeof current === 'number' && typeof previous === 'number' ? current - previous : undefined;
}

function formatDay(value: string) {
  return new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short' }).format(new Date(value));
}

function formatPeriodDates(start: string, end: string) {
  const formatter = new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short' });
  return `${formatter.format(new Date(start))} — ${formatter.format(new Date(end))}`;
}
