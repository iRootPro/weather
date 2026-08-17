export type WeatherChartData = {
  labels: string[];
  datasets: Record<string, number[]>;
};

export type WeatherArchiveSummary = {
  temp_avg?: number;
  temp_min?: number;
  temp_max?: number;
  rain_total: number;
  days_in_period: number;
  days_with_data: number;
  coverage_percent: number;
};

export type WeatherArchiveDay = {
  date: string;
  temp_min?: number;
  temp_avg?: number;
  temp_max?: number;
  rain_total?: number;
};

export type WeatherArchive = {
  period: 'month' | 'season' | 'year';
  label: string;
  start_date: string;
  end_date: string;
  summary: WeatherArchiveSummary;
  comparison: {
    label: string;
    available: boolean;
    summary: WeatherArchiveSummary;
  };
  days: WeatherArchiveDay[];
};

export async function fetchWeatherArchive(params: { period: WeatherArchive['period']; month?: string }): Promise<WeatherArchive> {
  const search = new URLSearchParams({ period: params.period });
  if (params.month) search.set('month', params.month);

  const response = await fetch(`/api/weather/archive?${search.toString()}`, {
    headers: { Accept: 'application/json' }
  });

  if (!response.ok) {
    const text = await response.text().catch(() => '');
    throw new Error(text || `Ошибка API: ${response.status}`);
  }

  return response.json();
}

export async function fetchWeatherChart(params: { from: Date; to: Date; interval?: string; fields: string[] }): Promise<WeatherChartData> {
  const search = new URLSearchParams({
    from: params.from.toISOString(),
    to: params.to.toISOString(),
    interval: params.interval ?? '1h',
    fields: params.fields.join(',')
  });

  const response = await fetch(`/api/weather/chart?${search.toString()}`, {
    headers: { Accept: 'application/json' }
  });

  if (!response.ok) {
    const text = await response.text().catch(() => '');
    throw new Error(text || `Ошибка API: ${response.status}`);
  }

  return response.json();
}
