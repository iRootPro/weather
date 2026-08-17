import type { MouseEvent } from 'react';

export type AppTab = 'now' | 'evening' | 'risks' | 'forecast' | 'archive';

type AppTabsProps = {
  active: AppTab;
  scenario?: string;
};

const tabs: Array<{ key: AppTab; label: string; href: string }> = [
  { key: 'now', label: 'сейчас', href: '/app/' },
  { key: 'evening', label: 'вечер', href: '/app/evening' },
  { key: 'risks', label: 'риски', href: '/app/risks' },
  { key: 'forecast', label: 'прогноз', href: '/app/forecast' },
  { key: 'archive', label: 'архив', href: '/app/archive' }
];

export function AppTabs({ active, scenario }: AppTabsProps) {
  return (
    <nav className="section-nav app-tabs" aria-label="Разделы дашборда">
      {tabs.map((tab) => {
        const href = withScenario(tab.href, scenario);
        return (
          <a key={tab.key} className={tab.key === active ? 'active' : undefined} aria-current={tab.key === active ? 'page' : undefined} href={href} onClick={(event) => navigateInApp(event, href)}>
            {tab.label}
          </a>
        );
      })}
    </nav>
  );
}

export function withScenario(path: string, scenario?: string) {
  return scenario ? `${path}?scenario=${encodeURIComponent(scenario)}` : path;
}

export function navigateInApp(event: MouseEvent<HTMLAnchorElement>, href: string) {
  if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;

  const target = new URL(href, window.location.origin);
  if (target.origin !== window.location.origin || !target.pathname.startsWith('/app')) return;

  event.preventDefault();
  if (window.location.pathname + window.location.search !== target.pathname + target.search) {
    window.history.pushState({}, '', target.pathname + target.search + target.hash);
  }
  window.dispatchEvent(new PopStateEvent('popstate'));
}
