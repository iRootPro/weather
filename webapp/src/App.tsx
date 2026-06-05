import { useQuery } from '@tanstack/react-query';
import { fetchDashboardSnapshot } from './api/dashboard';
import { DashboardPage } from './pages/DashboardPage';

export default function App() {
  const query = useQuery({
    queryKey: ['dashboard-snapshot'],
    queryFn: fetchDashboardSnapshot
  });

  return <DashboardPage query={query} />;
}
