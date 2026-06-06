import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { fetchDashboardSnapshot } from './api/dashboard';
import { getMockDashboardSnapshot, parseDashboardScenario } from './api/mockDashboard';
import { DashboardPage } from './pages/DashboardPage';
import { ForecastPage } from './pages/ForecastPage';

export default function App() {
  const scenario = useMemo(() => parseDashboardScenario(new URLSearchParams(window.location.search).get('scenario')), []);
  const route = window.location.pathname.replace(/\/+$/, '') || '/app';

  const query = useQuery({
    queryKey: ['dashboard-snapshot', scenario ?? 'live'],
    queryFn: () => (scenario ? Promise.resolve(getMockDashboardSnapshot(scenario)) : fetchDashboardSnapshot())
  });

  if (route === '/app/forecast') {
    return <ForecastPage query={query} scenario={scenario} />;
  }

  return <DashboardPage query={query} scenario={scenario} />;
}
