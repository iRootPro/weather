import type { AttentionCard, DashboardSnapshot, NearForecastItem, Severity } from './dashboard';

export const dashboardScenarios = ['calm', 'uv', 'storm', 'water', 'stale', 'rain', 'wind'] as const;
export type DashboardScenario = (typeof dashboardScenarios)[number];

const baseTime = '2026-06-06T20:25:00+03:00';

const scenarioLabels: Record<DashboardScenario, string> = {
  calm: 'спокойно',
  uv: 'UV',
  storm: 'буря',
  water: 'вода',
  stale: 'нет данных',
  rain: 'дождь',
  wind: 'ветер'
};

export function parseDashboardScenario(value: string | null): DashboardScenario | undefined {
  if (!value) return undefined;
  return dashboardScenarios.includes(value as DashboardScenario) ? (value as DashboardScenario) : undefined;
}

export function getDashboardScenarioLabel(scenario: DashboardScenario) {
  return scenarioLabels[scenario];
}

export function getMockDashboardSnapshot(scenario: DashboardScenario): DashboardSnapshot {
  const snapshot = buildBaseSnapshot();

  switch (scenario) {
    case 'uv':
      snapshot.headline = {
        title: 'Солнце активное',
        summary: 'Высокий UV — лучше перенести прогулку в тень',
        severity: 'warning',
        icon: '☀️'
      };
      snapshot.summary = 'сейчас 31.4°; сухо; ветер слабый; UV очень высокий.';
      snapshot.current_weather = {
        ...snapshot.current_weather!,
        temperature: 31.4,
        feels_like: 33.2,
        humidity: 39,
        uv_index: 8,
        icon: '☀️',
        title: 'Жарко',
        subtitle: 'Ощущается как 33.2°'
      };
      snapshot.cards = [card('uv-high', 'solar', 'Очень высокий UV', 'Кожа быстро обгорает на прямом солнце', '8', 'UV', 'warning', 76, '☀️', 'UV-индекс выше безопасного уровня')];
      snapshot.quiet.items = ['ветер', 'дождь', 'геомагнитка', 'вода'];
      break;

    case 'storm':
      snapshot.headline = {
        title: 'Магнитная буря',
        summary: 'Kp 6 — возможны головная боль и сбои чувствительной техники',
        severity: 'danger',
        icon: '🧲'
      };
      snapshot.summary = 'магнитная активность высокая; погода спокойная; вода и ветер в норме.';
      snapshot.cards = [
        card('geomagnetic-storm', 'geomagnetic', 'Магнитная буря', 'Kp 6, высокий уровень возмущений', '6', 'Kp', 'danger', 90, '🧲', 'геомагнитная активность достигла уровня бури'),
        card('uv-high', 'solar', 'UV заметный', 'Днём лучше держаться тени', '6', 'UV', 'warning', 58, '☀️', 'контекстный дневной риск')
      ];
      snapshot.quiet.items = ['ветер', 'дождь', 'вода'];
      break;

    case 'water':
      snapshot.headline = {
        title: 'Вода растёт',
        summary: 'Уровень поднялся на 34 см за сутки',
        severity: 'warning',
        icon: '🌊'
      };
      snapshot.summary = 'уровень воды растёт; дождя рядом нет; ветер слабый.';
      snapshot.cards = [
        card('hydro-rising', 'hydro', 'Рост уровня воды', '+34 см за сутки', '+34', 'см', 'warning', 82, '🌊', 'быстрый рост уровня по гидропосту'),
        card('rain-clear', 'rain', 'Дождя рядом нет', 'Ближайшие часы без осадков', '0', '%', 'normal', 35, '🌤️', 'прогноз без значимых осадков')
      ];
      snapshot.quiet.items = ['ветер', 'дождь', 'геомагнитка'];
      break;

    case 'stale':
      snapshot.station_status = {
        ok: false,
        last_seen_at: '2026-06-06T18:07:00+03:00',
        age_minutes: 138,
        label: 'данные устарели',
        severity: 'danger'
      };
      snapshot.headline = {
        title: 'Станция молчит',
        summary: 'Последнее наблюдение было больше двух часов назад',
        severity: 'danger',
        icon: '📡'
      };
      snapshot.summary = 'данные метеостанции устарели; прогноз доступен, но текущие датчики не обновляются.';
      snapshot.cards = [card('station-stale', 'station', 'Нет свежих данных', 'Последнее наблюдение в 18:07', '2ч+', '', 'danger', 95, '📡', 'станция давно не присылала MQTT-сообщения')];
      snapshot.quiet.items = ['прогноз', 'архив'];
      break;

    case 'rain':
      snapshot.headline = {
        title: 'Скоро дождь',
        summary: 'Вероятность осадков растёт в ближайшие два часа',
        severity: 'warning',
        icon: '🌧️'
      };
      snapshot.summary = 'к 22:00 вероятен дождь; ветер умеренный; температура начнёт снижаться.';
      snapshot.near_forecast = buildForecast('rain');
      snapshot.cards = [card('forecast-rain', 'rain', 'Дождь в ближайшие часы', 'Пик вероятности около 22:00', '78', '%', 'warning', 80, '🌧️', 'почасовой прогноз показывает осадки')];
      snapshot.quiet.items = ['вода', 'геомагнитка'];
      break;

    case 'wind':
      snapshot.headline = {
        title: 'Порывистый ветер',
        summary: 'Порывы до 17 м/с — лучше закрепить лёгкие предметы',
        severity: 'warning',
        icon: '💨'
      };
      snapshot.summary = 'ветер усилился; дождя нет; вода и геомагнитка спокойны.';
      snapshot.current_weather = {
        ...snapshot.current_weather!,
        wind_speed: 9.4,
        wind_gust: 17.2,
        subtitle: 'Ветер 9.4 м/с, порывы до 17.2 м/с'
      };
      snapshot.cards = [card('wind-gust', 'wind', 'Сильные порывы ветра', 'До 17.2 м/с на открытых местах', '17.2', 'м/с', 'warning', 84, '💨', 'порывы ветра выше комфортного уровня')];
      snapshot.quiet.items = ['дождь', 'геомагнитка', 'вода'];
      break;

    case 'calm':
    default:
      break;
  }

  return snapshot;
}

function buildBaseSnapshot(): DashboardSnapshot {
  return {
    generated_at: baseTime,
    station_status: {
      ok: true,
      last_seen_at: baseTime,
      age_minutes: 1,
      label: 'данные свежие',
      severity: 'normal'
    },
    headline: {
      title: 'Сейчас спокойно',
      summary: 'Нет показателей, которые требуют внимания',
      severity: 'calm',
      icon: '🟢'
    },
    summary: 'сейчас 22.0°; сухо; ветер слабый; остальное в норме.',
    current_weather: {
      observed_at: baseTime,
      temperature: 22,
      feels_like: 22,
      temperature_delta: -2.5,
      humidity: 66,
      pressure: 744,
      pressure_delta: 0,
      wind_speed: 0,
      wind_gust: 2.4,
      rain_rate: 0,
      uv_index: 0,
      icon: '🌤️',
      title: 'Комфортно',
      subtitle: 'Ощущается как 22.0°'
    },
    near_forecast: buildForecast('calm'),
    cards: [],
    quiet: {
      title: 'Остальное спокойно',
      items: ['ветер', 'дождь', 'геомагнитка', 'вода']
    }
  };
}

function card(
  id: string,
  domain: string,
  title: string,
  subtitle: string,
  value: string,
  unit: string,
  severity: Severity,
  priority: number,
  icon: string,
  reason: string
): AttentionCard {
  return {
    id,
    domain,
    title,
    subtitle,
    value,
    unit,
    severity,
    priority,
    reason,
    icon,
    detail_url: `/detail/${domain}`
  };
}

function buildForecast(kind: 'calm' | 'rain'): NearForecastItem[] {
  const calmRows = [
    ['2026-06-06T21:00:00+03:00', 25, 25, 0, 0, 3.2, 'Преимущественно ясно', '🌤️'],
    ['2026-06-06T22:00:00+03:00', 24, 24, 3, 0, 2.8, 'Ясно', '🌙'],
    ['2026-06-06T23:00:00+03:00', 22, 22, 0, 0, 2.3, 'Ясно', '🌙'],
    ['2026-06-07T00:00:00+03:00', 20, 20, 0, 0, 1.9, 'Ясно', '🌙'],
    ['2026-06-07T01:00:00+03:00', 20, 20, 0, 0, 1.8, 'Ясно', '🌙'],
    ['2026-06-07T02:00:00+03:00', 19, 19, 0, 0, 1.6, 'Ясно', '🌙'],
    ['2026-06-07T03:00:00+03:00', 18, 18, 0, 0, 1.5, 'Ясно', '🌙'],
    ['2026-06-07T04:00:00+03:00', 17, 17, 0, 0, 1.4, 'Преимущественно ясно', '🌙']
  ] as const;

  const rainRows = [
    ['2026-06-06T21:00:00+03:00', 24, 24, 34, 0.1, 5.4, 'Пасмурно', '☁️'],
    ['2026-06-06T22:00:00+03:00', 22, 22, 78, 1.8, 6.2, 'Дождь', '🌧️'],
    ['2026-06-06T23:00:00+03:00', 21, 21, 72, 2.2, 5.9, 'Дождь', '🌧️'],
    ['2026-06-07T00:00:00+03:00', 20, 20, 45, 0.7, 4.8, 'Небольшой дождь', '🌧️'],
    ['2026-06-07T01:00:00+03:00', 20, 20, 18, 0, 3.7, 'Облачно', '☁️'],
    ['2026-06-07T02:00:00+03:00', 19, 19, 8, 0, 3.1, 'Облачно', '☁️'],
    ['2026-06-07T03:00:00+03:00', 18, 18, 0, 0, 2.4, 'Ясно', '🌙'],
    ['2026-06-07T04:00:00+03:00', 17, 17, 0, 0, 2.1, 'Ясно', '🌙']
  ] as const;

  return (kind === 'rain' ? rainRows : calmRows).map(([time, temperature, feelsLike, precipitationProbability, precipitation, windSpeed, weatherDescription, icon]) => ({
    time,
    temperature,
    feels_like: feelsLike,
    precipitation_probability: precipitationProbability,
    precipitation,
    wind_speed: windSpeed,
    weather_description: weatherDescription,
    icon
  }));
}
