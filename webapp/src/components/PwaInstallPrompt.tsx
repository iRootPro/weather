import { useEffect, useState } from 'react';

type BeforeInstallPromptEvent = Event & {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
};

export function PwaInstallPrompt() {
  const [installEvent, setInstallEvent] = useState<BeforeInstallPromptEvent | null>(null);
  const [hidden, setHidden] = useState(() => window.localStorage.getItem('weather-pwa-install-hidden') === '1');
  const [offlineReady] = useState(false);

  useEffect(() => {
    const onBeforeInstall = (event: Event) => {
      event.preventDefault();
      setInstallEvent(event as BeforeInstallPromptEvent);
    };
    const onOfflineReady = () => {
      // Service worker readiness is useful, but an automatic bottom sheet
      // was covering dashboard content during normal viewing and QA.
      // Keep the event consumed silently; explicit install prompts still work.
    };

    window.addEventListener('beforeinstallprompt', onBeforeInstall);
    window.addEventListener('pwa-offline-ready', onOfflineReady);
    return () => {
      window.removeEventListener('beforeinstallprompt', onBeforeInstall);
      window.removeEventListener('pwa-offline-ready', onOfflineReady);
    };
  }, []);

  if (hidden || (!installEvent && !offlineReady)) return null;

  const dismiss = () => {
    window.localStorage.setItem('weather-pwa-install-hidden', '1');
    setHidden(true);
  };

  const install = async () => {
    if (!installEvent) return;
    await installEvent.prompt();
    const choice = await installEvent.userChoice.catch(() => null);
    if (choice?.outcome === 'accepted') dismiss();
    setInstallEvent(null);
  };

  return (
    <aside className="install-prompt" role="status">
      <div>
        <span aria-hidden="true">📲</span>
        <div>
          <b>{installEvent ? 'Установить погодное приложение' : 'Приложение готово офлайн'}</b>
          <small>{installEvent ? 'Будет открываться как отдельное приложение на телефоне.' : 'Главный экран кешируется и быстрее открывается.'}</small>
        </div>
      </div>
      <div className="install-actions">
        {installEvent && <button onClick={install}>Установить</button>}
        <button className="ghost" onClick={dismiss}>Скрыть</button>
      </div>
    </aside>
  );
}
