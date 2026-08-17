import { navigateInApp, withScenario, type AppTab } from './AppTabs';

type SidebarItem = {
  key: AppTab | 'map';
  label: string;
  icon: IconName;
  href: string;
};

type IconName = 'sun' | 'home' | 'moon' | 'risk' | 'forecast' | 'map' | 'settings';

const items: SidebarItem[] = [
  { key: 'now', label: 'Сейчас', icon: 'home', href: '/app/' },
  { key: 'evening', label: 'Вечер', icon: 'moon', href: '/app/evening' },
  { key: 'risks', label: 'Риски', icon: 'risk', href: '/app/risks' },
  { key: 'forecast', label: 'Прогноз', icon: 'forecast', href: '/app/forecast' },
  { key: 'map', label: 'Карта', icon: 'map', href: '/app/risks' }
];

export function AppSidebar({ active, scenario }: { active: AppTab; scenario?: string }) {
  return (
    <aside className="app-sidebar" aria-label="Основная навигация">
      <div className="sidebar-brand">
        <div className="sidebar-logo" aria-hidden="true"><LineIcon name="sun" /></div>
        <strong>Погодный<br />ассистент</strong>
      </div>

      <nav className="sidebar-nav">
        {items.map((item) => {
          const href = withScenario(item.href, scenario);
          const isActive = item.key === active;
          return (
            <a key={item.key} className={isActive ? 'active' : undefined} href={href} onClick={(event) => navigateInApp(event, href)}>
              <span aria-hidden="true"><LineIcon name={item.icon} /></span>
              {item.label}
            </a>
          );
        })}
      </nav>

      <a className="sidebar-settings" href="/help">
        <span aria-hidden="true"><LineIcon name="settings" /></span>
        Настройки
      </a>
    </aside>
  );
}

function LineIcon({ name }: { name: IconName }) {
  const common = { fill: 'none', stroke: 'currentColor', strokeWidth: 1.85, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const };

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      {name === 'sun' && <><circle cx="12" cy="12" r="3.8" {...common} /><path d="M12 2.8v2.1M12 19.1v2.1M4.2 4.2l1.5 1.5M18.3 18.3l1.5 1.5M2.8 12h2.1M19.1 12h2.1M4.2 19.8l1.5-1.5M18.3 5.7l1.5-1.5" {...common} /></>}
      {name === 'home' && <><path d="M4.5 11.2 12 5l7.5 6.2" {...common} /><path d="M6.8 10.3v8.2h10.4v-8.2" {...common} /><path d="M10 18.5v-5h4v5" {...common} /></>}
      {name === 'moon' && <path d="M18.6 15.2A7 7 0 0 1 8.8 5.4 7.2 7.2 0 1 0 18.6 15.2Z" {...common} />}
      {name === 'risk' && <><path d="m12 4.2 8.2 14.2H3.8L12 4.2Z" {...common} /><path d="M12 9.2v4.3M12 17.1h.01" {...common} /></>}
      {name === 'forecast' && <><path d="M17.8 7.7A6.5 6.5 0 1 0 19 14" {...common} /><path d="M19.2 5.8v4.3h-4.3" {...common} /></>}
      {name === 'map' && <><path d="M12 21s6-5.1 6-11a6 6 0 0 0-12 0c0 5.9 6 11 6 11Z" {...common} /><circle cx="12" cy="10" r="2.1" {...common} /></>}
      {name === 'settings' && <><path d="M12 15.2a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4Z" {...common} /><path d="M19.4 13.8a7.7 7.7 0 0 0 .1-1.8l2-1.5-2-3.4-2.4 1a8 8 0 0 0-1.5-.9L15.3 4H9.7l-.4 3.2c-.5.2-1 .5-1.5.9l-2.4-1-2 3.4 2 1.5a7.7 7.7 0 0 0 .1 1.8l-2 1.5 2 3.4 2.4-1c.5.4 1 .7 1.5.9l.4 3.2h5.6l.4-3.2c.5-.2 1-.5 1.5-.9l2.4 1 2-3.4-2.2-1.5Z" {...common} /></>}
    </svg>
  );
}
