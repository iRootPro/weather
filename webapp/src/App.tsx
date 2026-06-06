import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { fetchDashboardSnapshot } from './api/dashboard';
import { getMockDashboardSnapshot, parseDashboardScenario } from './api/mockDashboard';
import { DashboardPage } from './pages/DashboardPage';
import { ForecastPage } from './pages/ForecastPage';
import { RiskDetailPage, type RiskKind } from './pages/RiskDetailPage';

const riskRoutes: Record<string, RiskKind> = {
  '/app/geomagnetic': 'geomagnetic',
  '/app/water': 'water',
  '/app/rain': 'rain',
  '/app/wind': 'wind',
  '/app/uv': 'uv',
  '/app/station': 'station'
};

type SWUpdater = (reloadPage?: boolean) => Promise<void>;

export default function App() {
  const scenario = useMemo(() => parseDashboardScenario(new URLSearchParams(window.location.search).get('scenario')), []);
  const route = window.location.pathname.replace(/\/+$/, '') || '/app';
  const [updateSW, setUpdateSW] = useState<SWUpdater | null>(null);

  useEffect(() => {
    const onUpdateReady = (event: Event) => {
      setUpdateSW(() => (event as CustomEvent<SWUpdater>).detail);
    };
    window.addEventListener('pwa-update-ready', onUpdateReady);
    return () => window.removeEventListener('pwa-update-ready', onUpdateReady);
  }, []);

  const query = useQuery({
    queryKey: ['dashboard-snapshot', scenario ?? 'live'],
    queryFn: () => (scenario ? Promise.resolve(getMockDashboardSnapshot(scenario)) : fetchDashboardSnapshot())
  });

  const page = (() => {
    if (route === '/app/forecast') {
      return <ForecastPage query={query} scenario={scenario} />;
    }

    const riskKind = riskRoutes[route];
    if (riskKind) {
      return <RiskDetailPage query={query} scenario={scenario} kind={riskKind} />;
    }

    return <DashboardPage query={query} scenario={scenario} />;
  })();

  return (
    <>
      {page}
      {updateSW && (
        <div className="update-toast" role="status">
          <span>Доступна новая версия приложения</span>
          <button onClick={() => updateSW(true)}>Обновить</button>
          <button className="ghost" onClick={() => setUpdateSW(null)}>Позже</button>
        </div>
      )}
    </>
  );
}
