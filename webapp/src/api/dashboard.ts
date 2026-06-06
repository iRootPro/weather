export type Severity = 'calm' | 'normal' | 'info' | 'warning' | 'danger';

export type StationStatus = {
  ok: boolean;
  last_seen_at?: string;
  age_minutes?: number;
  label: string;
  severity: Severity;
};

export type DashboardHeadline = {
  title: string;
  summary?: string;
  severity: Severity;
  icon?: string;
};

export type CurrentWeatherSummary = {
  observed_at: string;
  temperature?: number;
  feels_like?: number;
  temperature_delta?: number;
  humidity?: number;
  pressure?: number;
  pressure_delta?: number;
  wind_speed?: number;
  wind_gust?: number;
  rain_rate?: number;
  uv_index?: number;
  icon: string;
  title: string;
  subtitle: string;
};

export type NearForecastItem = {
  time: string;
  temperature: number;
  feels_like: number;
  precipitation_probability: number;
  precipitation: number;
  wind_speed: number;
  weather_description: string;
  icon: string;
};

export type AttentionCard = {
  id: string;
  domain: string;
  title: string;
  subtitle?: string;
  value?: string;
  unit?: string;
  severity: Severity;
  priority: number;
  reason?: string;
  action?: string;
  icon?: string;
  detail_url?: string;
};

export type QuietSummary = {
  title: string;
  items: string[];
};

export type DashboardSnapshot = {
  generated_at: string;
  station_status: StationStatus;
  headline: DashboardHeadline;
  summary?: string;
  current_weather?: CurrentWeatherSummary;
  near_forecast?: NearForecastItem[];
  cards: AttentionCard[];
  quiet: QuietSummary;
};

export async function fetchDashboardSnapshot(): Promise<DashboardSnapshot> {
  const response = await fetch('/api/dashboard/snapshot', {
    headers: { Accept: 'application/json' }
  });

  if (!response.ok) {
    const text = await response.text().catch(() => '');
    throw new Error(text || `Ошибка API: ${response.status}`);
  }

  return response.json();
}
