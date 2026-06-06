import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { fetchDashboardSnapshot } from './api/dashboard';
import { getMockDashboardSnapshot, parseDashboardScenario } from './api/mockDashboard';
import { DashboardPage } from './pages/DashboardPage';

export default function App() {
  const scenario = useMemo(() => parseDashboardScenario(new URLSearchParams(window.location.search).get('scenario')), []);

  const query = useQuery({
    queryKey: ['dashboard-snapshot', scenario ?? 'live'],
    queryFn: () => (scenario ? Promise.resolve(getMockDashboardSnapshot(scenario)) : fetchDashboardSnapshot())
  });

  return <DashboardPage query={query} scenario={scenario} />;
}
