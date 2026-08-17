import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { fetchDashboardSnapshot } from './api/dashboard';
import { getMockDashboardSnapshot, parseDashboardScenario } from './api/mockDashboard';
import { LiveRefreshBadge } from './components/LiveRefreshBadge';
import { PwaInstallPrompt } from './components/PwaInstallPrompt';
import { ArchivePage } from './pages/ArchivePage';
import { CurrentDetailPage } from './pages/CurrentDetailPage';
import { DashboardPage } from './pages/DashboardPage';
import { EveningPage } from './pages/EveningPage';
import { ForecastPage } from './pages/ForecastPage';
import { RiskDetailPage, type RiskKind } from './pages/RiskDetailPage';
import { RisksOverviewPage } from './pages/RisksOverviewPage';

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
  const [locationKey, setLocationKey] = useState(() => `${window.location.pathname}${window.location.search}`);
  const scenario = useMemo(() => parseDashboardScenario(new URLSearchParams(window.location.search).get('scenario')), [locationKey]);
  const route = useMemo(() => window.location.pathname.replace(/\/+$/, '') || '/app', [locationKey]);
  const [updateSW, setUpdateSW] = useState<SWUpdater | null>(null);

  useEffect(() => {
    const onRouteChange = () => setLocationKey(`${window.location.pathname}${window.location.search}`);
    window.addEventListener('popstate', onRouteChange);
    return () => window.removeEventListener('popstate', onRouteChange);
  }, []);

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

    if (route === '/app/archive') {
      return <ArchivePage scenario={scenario} />;
    }

    if (route === '/app/evening') {
      return <EveningPage query={query} scenario={scenario} />;
    }

    if (route === '/app/risks') {
      return <RisksOverviewPage query={query} scenario={scenario} />;
    }

    if (route === '/app/current') {
      return <CurrentDetailPage query={query} scenario={scenario} />;
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
      {!scenario && <LiveRefreshBadge snapshot={query.data} isFetching={query.isFetching} />}
      <PwaInstallPrompt />
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
