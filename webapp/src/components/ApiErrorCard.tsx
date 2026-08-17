export function ApiErrorCard({ title, message, onRetry }: { title: string; message: string; onRetry: () => void }) {
  return (
    <main className="page-shell error-shell">
      <section className="error-card api-error-card">
        <span aria-hidden="true">⚠️</span>
        <h1>{title}</h1>
        <p>{cleanError(message)}</p>
        <div className="api-error-grid" aria-label="Что можно проверить">
          <div>
            <b>API</b>
            <small>сервер может быть недоступен или долго отвечать</small>
          </div>
          <div>
            <b>Станция</b>
            <small>если MQTT молчит, текущие данные могут отсутствовать</small>
          </div>
          <div>
            <b>Прогноз</b>
            <small>может продолжать работать отдельно от станции</small>
          </div>
        </div>
        <div className="api-error-actions">
          <button onClick={onRetry}>Попробовать ещё раз</button>
          <a href="/health">Проверить сервер</a>
          <a href="/">Старая панель</a>
        </div>
      </section>
    </main>
  );
}

function cleanError(message: string) {
  return message.replace(/^Error:\s*/i, '').trim() || 'Неизвестная ошибка загрузки.';
}
