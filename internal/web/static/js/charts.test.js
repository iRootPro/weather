const test = require('node:test');
const assert = require('node:assert/strict');
const { chartValues, displayTimeLabel, formatRangeLabel, formatTooltipNumber, loadChartData, refreshChartAccessibility, syncChartTooltips, updateChartColors } = require('./charts.js');

function classList() {
    const names = new Set();
    return { toggle: (name, enabled) => enabled ? names.add(name) : names.delete(name), contains: name => names.has(name) };
}

function installChartDOM() {
    const buttons = ['5m', '15m', '1h'].map(interval => ({ dataset: { interval }, classList: classList(), setAttribute(name, value) { this[name] = value; } }));
    const range = { textContent: '' };
    const status = { textContent: '' };
    global.document = {
        querySelectorAll: selector => selector === '.chart-interval-btn' ? buttons : [],
        getElementById: id => ({ 'chart-range': range, 'chart-status': status }[id] || null),
        documentElement: { classList: { contains: () => false } }
    };
    return { buttons, range, status };
}

test('keeps full returned labels for the selected data range', () => {
    const labels = ['2026-09-23 23:55', '2026-09-24 00:00'];

    assert.equal(formatRangeLabel(labels), 'Данные: 2026-09-23 23:55 — 2026-09-24 00:00');
    assert.equal(displayTimeLabel(labels[0]), '23:55');
});

test('converts only absent or invalid values into chart gaps', () => {
    assert.deepEqual(chartValues([0, 1.5, null, undefined, Number.NaN]), [0, 1.5, null, null, null]);
});

test('describes an empty chart response without inventing a range', () => {
    assert.equal(formatRangeLabel([]), 'Нет данных за выбранный период.');
});

test('formats an absent tooltip value without calling toFixed', () => {
    assert.equal(formatTooltipNumber(null), 'нет данных');
    assert.equal(formatTooltipNumber(2.5), '2.5');
});

test('synchronizes every valid series and leaves gaps inactive', () => {
    const active = [];
    const target = {
        data: { labels: ['a', 'b'], datasets: [{ data: [1, null] }, { data: [2, 3] }] },
        getDatasetMeta: () => ({ data: [{}, {}] }),
        tooltip: { setActiveElements: elements => active.push(elements) }, update: () => {}
    };
    const source = { data: { labels: ['a', 'b'], datasets: [] } };

    syncChartTooltips(source, 0, { source, target });
    syncChartTooltips(source, 1, { source, target });

    assert.deepEqual(active[0], [{ datasetIndex: 0, index: 0 }, { datasetIndex: 1, index: 0 }]);
    assert.deepEqual(active[1], [{ datasetIndex: 1, index: 1 }]);
});

test('refreshes an explicitly supplied history chart collection', () => {
    const summary = { textContent: '' };
    global.document = { getElementById: id => id === 'history-summary' ? summary : null };
    const historyChart = {
        canvas: { id: 'history' },
        accessibility: { name: 'История', formatValue: value => `${value}` },
        data: { datasets: [{ label: 'Температура', data: [5] }] }
    };

    refreshChartAccessibility({ historyChart });

    assert.match(summary.textContent, /История\. Температура: последнее значение 5/);
});

test('retains the newest chart response when requests complete out of order', async () => {
    const { range } = installChartDOM();
    let resolveFirst;
    global.fetch = url => url.includes('interval=5m')
        ? new Promise(resolve => { resolveFirst = resolve; })
        : Promise.resolve({ ok: true, json: async () => ({ labels: ['2026-09-24 12:00'], datasets: {} }) });

    const first = loadChartData('5m');
    await loadChartData('15m');
    resolveFirst({ ok: true, json: async () => ({ labels: ['2026-09-24 11:00'], datasets: {} }) });
    await first;

    assert.equal(range.textContent, 'Данные: 2026-09-24 12:00 — 2026-09-24 12:00');
});

test('reports a loading error and restores the previously applied interval', async () => {
    const { buttons, status } = installChartDOM();
    global.fetch = async () => ({ ok: false, status: 503 });
    const originalError = console.error;
    console.error = () => {};

    await loadChartData('1h');

    console.error = originalError;
    assert.equal(status.textContent, 'Не удалось загрузить графики. Повторите попытку.');
    assert.equal(buttons.find(button => button.dataset.interval === '15m').classList.contains('ui-button-primary'), true);
    updateChartColors();
});
