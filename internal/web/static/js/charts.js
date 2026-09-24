// Dashboard charts. The API labels remain intact for accessible tables and range text.
let charts = {};
let currentInterval = '5m';
let lastAppliedInterval = '5m';
let sharedTooltipIndex = null;
let chartRequestID = 0;

const CHART_BLUE = '#2563eb';
const CHART_SECONDARY = '#64748b';

function formatAccessibleNumber(value, fractionDigits = 1) {
    return Number(value).toLocaleString('ru-RU', { minimumFractionDigits: fractionDigits, maximumFractionDigits: fractionDigits });
}

function formatRangeLabel(labels) {
    const validLabels = Array.isArray(labels) ? labels.filter(Boolean) : [];
    if (!validLabels.length) return 'Нет данных за выбранный период.';
    return `Данные: ${validLabels[0]} — ${validLabels.at(-1)}`;
}

function displayTimeLabel(label) {
    const parts = String(label).split(' ');
    return parts.length > 1 ? parts[1] : parts[0];
}

function chartValues(values) {
    // Chart.js treats null as a gap. Do not coerce missing API values to zero here.
    return Array.isArray(values) ? values.map(value => Number.isFinite(value) ? value : null) : [];
}

function formatTooltipNumber(value) { return Number.isFinite(value) ? value.toFixed(1) : 'нет данных'; }

function getThemeColors() {
    const isDark = document.documentElement.classList.contains('dark');
    return {
        gridColor: isDark ? 'rgba(203, 213, 225, 0.16)' : 'rgba(148, 163, 184, 0.28)',
        textColor: isDark ? '#cbd5e1' : '#475569',
        crosshairColor: isDark ? 'rgba(226, 232, 240, 0.45)' : 'rgba(71, 85, 105, 0.35)'
    };
}

function configureChartAccessibility(chart, name, formatValue) {
    if (!chart?.canvas) return;
    chart.accessibility = { name, formatValue };
    const details = document.getElementById(`${chart.canvas.id}-details`);
    details?.addEventListener('toggle', () => { if (details.open) renderChartDataTable(chart); });
    updateChartAccessibility(chart);
}

function updateChartAccessibility(chart) {
    if (!chart?.accessibility) return;
    const { name, formatValue } = chart.accessibility;
    const summary = document.getElementById(`${chart.canvas.id}-summary`);
    if (summary) {
        const series = chart.data.datasets.map(dataset => {
            const values = dataset.data.filter(Number.isFinite);
            if (!values.length) return `${dataset.label}: нет данных.`;
            const current = values.at(-1);
            return `${dataset.label}: последнее значение ${formatValue(current, dataset)}, минимум ${formatValue(Math.min(...values), dataset)}, максимум ${formatValue(Math.max(...values), dataset)}.`;
        });
        summary.textContent = `${name}. ${series.join(' ')}`;
    }
    if (document.getElementById(`${chart.canvas.id}-details`)?.open) renderChartDataTable(chart);
}

function renderChartDataTable(chart) {
    const container = document.getElementById(`${chart.canvas.id}-table`);
    if (!container || !chart.accessibility) return;
    const { name, formatValue } = chart.accessibility;
    if (window.matchMedia('(max-width: 639px)').matches) {
        renderChartDataCards(chart, container, name, formatValue);
        return;
    }
    const table = document.createElement('table');
    table.className = 'min-w-full whitespace-nowrap text-left text-xs tabular-nums';
    table.innerHTML = `<caption class="sr-only">${name}: значения по времени</caption>`;
    const thead = document.createElement('thead');
    thead.className = 'bg-gray-50 text-gray-600 dark:bg-gray-900 dark:text-gray-300';
    const headerRow = document.createElement('tr');
    ['Время', ...chart.data.datasets.map(dataset => dataset.label)].forEach(label => {
        const cell = document.createElement('th'); cell.className = 'px-3 py-2 font-semibold'; cell.scope = 'col'; cell.textContent = label; headerRow.append(cell);
    });
    thead.append(headerRow); table.append(thead);
    const tbody = document.createElement('tbody');
    chart.data.labels.forEach((label, index) => {
        const row = document.createElement('tr'); row.className = 'border-t border-gray-100 dark:border-gray-700';
        const time = document.createElement('th'); time.className = 'px-3 py-2 font-medium'; time.scope = 'row'; time.textContent = label; row.append(time);
        chart.data.datasets.forEach(dataset => {
            const cell = document.createElement('td'); cell.className = 'px-3 py-2';
            const value = dataset.data[index]; cell.textContent = Number.isFinite(value) ? formatValue(value, dataset) : '—'; row.append(cell);
        });
        tbody.append(row);
    });
    table.append(tbody); container.replaceChildren(table);
}

function renderChartDataCards(chart, container, name, formatValue) {
    const list = document.createElement('ol');
    list.className = 'space-y-2 p-2';
    list.setAttribute('aria-label', `${name}: значения по времени`);
    chart.data.labels.forEach((label, index) => {
        const item = document.createElement('li'); item.className = 'ui-metric-card p-3';
        const time = document.createElement('time'); time.className = 'ui-tabular text-xs font-semibold text-gray-900 dark:text-white'; time.textContent = label; item.append(time);
        const values = document.createElement('dl'); values.className = 'mt-2 grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-1';
        chart.data.datasets.forEach(dataset => {
            const term = document.createElement('dt'); term.className = 'min-w-0 text-xs text-gray-600 dark:text-gray-300'; term.textContent = dataset.label; values.append(term);
            const value = document.createElement('dd'); value.className = 'ui-tabular text-right text-xs font-semibold text-gray-900 dark:text-white';
            const point = dataset.data[index]; value.textContent = Number.isFinite(point) ? formatValue(point, dataset) : '—'; values.append(value);
        });
        item.append(values); list.append(item);
    });
    container.replaceChildren(list);
}

function renderCombinedRainTable(dailyChart, rateChart) {
    const container = document.getElementById('rainChart-table');
    if (!container || !dailyChart || !rateChart) return;
    const table = document.createElement('table');
    table.className = 'min-w-full whitespace-nowrap text-left text-xs tabular-nums';
    table.innerHTML = '<caption class="sr-only">Осадки: значения по времени</caption>';
    const head = document.createElement('thead');
    head.className = 'bg-gray-50 text-gray-600 dark:bg-gray-900 dark:text-gray-300';
    head.innerHTML = '<tr><th scope="col" class="px-3 py-2 font-semibold">Время</th><th scope="col" class="px-3 py-2 font-semibold">Осадки за день, мм</th><th scope="col" class="px-3 py-2 font-semibold">Интенсивность, мм/ч</th></tr>';
    table.append(head);
    const body = document.createElement('tbody');
    dailyChart.data.labels.forEach((label, index) => {
        const daily = dailyChart.data.datasets[0].data[index];
        const rate = rateChart.data.datasets[0].data[index];
        const row = document.createElement('tr'); row.className = 'border-t border-gray-100 dark:border-gray-700';
        row.innerHTML = `<th scope="row" class="px-3 py-2 font-medium"></th><td class="px-3 py-2"></td><td class="px-3 py-2"></td>`;
        row.children[0].textContent = label;
        row.children[1].textContent = Number.isFinite(daily) ? `${formatAccessibleNumber(daily)} мм` : '—';
        row.children[2].textContent = Number.isFinite(rate) ? `${formatAccessibleNumber(rate)} мм/ч` : '—';
        body.append(row);
    });
    table.append(body); container.replaceChildren(table);
}

function refreshChartAccessibility(chartCollection = charts) { Object.values(chartCollection).forEach(updateChartAccessibility); }

let chartDataLayoutIsMobile = typeof window !== 'undefined' && window.matchMedia('(max-width: 639px)').matches;
if (typeof window !== 'undefined') window.addEventListener('resize', () => {
    const isMobile = window.matchMedia('(max-width: 639px)').matches;
    if (isMobile === chartDataLayoutIsMobile) return;
    chartDataLayoutIsMobile = isMobile;
    document.querySelectorAll('details[id$="-details"][open]').forEach(details => {
        const chart = Chart.getChart(document.getElementById(details.id.replace(/-details$/, '')));
        if (chart) renderChartDataTable(chart);
    });
});

const crosshairPlugin = {
    id: 'crosshair',
    afterDraw(chart) {
        if (sharedTooltipIndex === null || !chart.tooltip?._active?.length) return;
        const element = chart.tooltip._active[0].element;
        const yScale = chart.scales.y;
        if (!element || !yScale) return;
        const ctx = chart.ctx;
        ctx.save(); ctx.beginPath(); ctx.moveTo(element.x, yScale.top); ctx.lineTo(element.x, yScale.bottom);
        ctx.lineWidth = 1; ctx.strokeStyle = getThemeColors().crosshairColor; ctx.setLineDash([4, 4]); ctx.stroke(); ctx.restore();
    }
};

function validTooltipElements(chart, dataIndex) {
    if (dataIndex === null || dataIndex < 0 || dataIndex >= chart.data.labels.length) return [];
    return chart.data.datasets.flatMap((dataset, datasetIndex) => {
        const element = chart.getDatasetMeta(datasetIndex)?.data[dataIndex];
        return Number.isFinite(dataset.data[dataIndex]) && element ? [{ datasetIndex, index: dataIndex }] : [];
    });
}

function syncChartTooltips(sourceChart, dataIndex, chartCollection = charts) {
    sharedTooltipIndex = dataIndex;
    Object.values(chartCollection).forEach(chart => {
        if (!chart || chart === sourceChart) return;
        chart.tooltip.setActiveElements(validTooltipElements(chart, dataIndex));
        chart.update('none');
    });
}

function commonOptions() {
    const theme = getThemeColors();
    return {
        responsive: true, maintainAspectRatio: false, normalized: true,
        interaction: { intersect: false, mode: 'index' },
        onHover(event, active) { syncChartTooltips(this, active.length ? active[0].index : null); },
        plugins: {
            legend: { display: false }, crosshair: true,
            tooltip: { backgroundColor: 'rgba(15, 23, 42, 0.94)', padding: 9, titleFont: { family: 'Golos Text', size: 12 }, bodyFont: { family: 'Golos Text', size: 12 } }
        },
        scales: {
            x: { grid: { display: false }, ticks: { callback(value) { return displayTimeLabel(this.getLabelForValue(value)); }, maxRotation: 0, autoSkip: true, maxTicksLimit: 5, color: theme.textColor, font: { family: 'Golos Text', size: 12 } } },
            y: { grid: { color: theme.gridColor }, ticks: { padding: 6, color: theme.textColor, font: { family: 'Golos Text', size: 12 } } }
        },
        elements: { point: { radius: 0, hoverRadius: 4 }, line: { borderWidth: 2, tension: 0.25, spanGaps: false } }
    };
}

function createChart(id, dataset, options = {}) {
    const canvas = document.getElementById(id);
    if (!canvas) return null;
    const base = commonOptions();
    return new Chart(canvas, {
        type: 'line', data: { labels: [], datasets: [dataset] },
        options: { ...base, ...options, plugins: { ...base.plugins, ...options.plugins }, scales: { ...base.scales, ...options.scales } }, plugins: [crosshairPlugin]
    });
}

function initCharts() {
    charts.temp = createChart('tempChart', { label: 'Температура', data: [], borderColor: CHART_BLUE, fill: false }, { plugins: { tooltip: { callbacks: { label: ctx => `Температура: ${formatTooltipNumber(ctx.raw)} °C` } } } });
    charts.humidity = createChart('humidityChart', { label: 'Влажность', data: [], borderColor: CHART_BLUE, fill: false }, { scales: { y: { ...commonOptions().scales.y, min: 0, max: 100 } }, plugins: { tooltip: { callbacks: { label: ctx => `Влажность: ${Number.isFinite(ctx.raw) ? Math.round(ctx.raw) : 'нет данных'} %` } } } });
    charts.pressure = createChart('pressureChart', { label: 'Давление', data: [], borderColor: CHART_BLUE, fill: false }, { plugins: { tooltip: { callbacks: { label: ctx => `Давление: ${formatTooltipNumber(ctx.raw)} мм рт. ст.` } } } });
    charts.wind = createChart('windChart', { label: 'Скорость', data: [], borderColor: CHART_BLUE, fill: false }, { plugins: { tooltip: { callbacks: { label: ctx => `${ctx.dataset.label}: ${formatTooltipNumber(ctx.raw)} м/с` } } } });
    if (charts.wind) charts.wind.data.datasets.push({ label: 'Порывы', data: [], borderColor: CHART_SECONDARY, borderDash: [6, 4], fill: false });
    charts.rainDaily = createChart('rainDailyChart', { label: 'Осадки за день', data: [], borderColor: CHART_BLUE, fill: false }, { scales: { y: { ...commonOptions().scales.y, min: 0 } }, plugins: { tooltip: { callbacks: { label: ctx => `Осадки за день: ${formatTooltipNumber(ctx.raw)} мм` } } } });
    charts.rainRate = createChart('rainRateChart', { label: 'Интенсивность', data: [], borderColor: CHART_SECONDARY, borderDash: [6, 4], fill: false }, { scales: { y: { ...commonOptions().scales.y, min: 0 } }, plugins: { tooltip: { callbacks: { label: ctx => `Интенсивность: ${formatTooltipNumber(ctx.raw)} мм/ч` } } } });
    charts.solar = createChart('solarChart', { label: 'Освещённость', data: [], borderColor: CHART_BLUE, fill: false }, { scales: { y: { ...commonOptions().scales.y, min: 0 } }, plugins: { tooltip: { callbacks: { label: ctx => `Освещённость: ${Number.isFinite(ctx.raw) ? Math.round(ctx.raw * 120) : 'нет данных'} лк` } } } });
    configureChartAccessibility(charts.temp, 'Температура', value => `${formatAccessibleNumber(value)} °C`);
    configureChartAccessibility(charts.humidity, 'Влажность', value => `${formatAccessibleNumber(value, 0)} %`);
    configureChartAccessibility(charts.pressure, 'Давление', value => `${formatAccessibleNumber(value)} мм рт. ст.`);
    configureChartAccessibility(charts.wind, 'Ветер', value => `${formatAccessibleNumber(value)} м/с`);
    configureChartAccessibility(charts.rainDaily, 'Осадки за день', value => `${formatAccessibleNumber(value)} мм`);
    configureChartAccessibility(charts.rainRate, 'Интенсивность осадков', value => `${formatAccessibleNumber(value)} мм/ч`);
    configureChartAccessibility(charts.solar, 'Освещённость', value => `${formatAccessibleNumber(value * 120, 0)} лк`);
    document.getElementById('rainChart-details')?.addEventListener('toggle', event => {
        if (event.currentTarget.open) renderCombinedRainTable(charts.rainDaily, charts.rainRate);
    });
}

function updateIntervalButtons(interval) {
    document.querySelectorAll('.chart-interval-btn').forEach(button => {
        const active = button.dataset.interval === interval;
        button.classList.toggle('ui-button-primary', active); button.classList.toggle('ui-button-secondary', !active); button.setAttribute('aria-pressed', String(active));
    });
}

function applyChartData(data) {
    const labels = Array.isArray(data.labels) ? data.labels : [];
    const datasets = data.datasets || {};
    const bindings = [
        ['temp', 0, 'temp_outdoor'], ['humidity', 0, 'humidity_outdoor'], ['pressure', 0, 'pressure_relative'],
        ['wind', 0, 'wind_speed'], ['wind', 1, 'wind_gust'], ['rainDaily', 0, 'rain_daily'], ['rainRate', 0, 'rain_rate'], ['solar', 0, 'solar_radiation']
    ];
    Object.values(charts).forEach(chart => { chart.data.labels = labels; });
    bindings.forEach(([chartName, datasetIndex, field]) => { if (charts[chartName]) charts[chartName].data.datasets[datasetIndex].data = chartValues(datasets[field]); });
    Object.values(charts).forEach(chart => chart.update());
    document.getElementById('chart-range').textContent = formatRangeLabel(labels);
    refreshChartAccessibility();
    if (document.getElementById('rainChart-details')?.open) renderCombinedRainTable(charts.rainDaily, charts.rainRate);
}

async function loadChartData(interval) {
    const requestID = ++chartRequestID;
    const previousInterval = lastAppliedInterval;
    currentInterval = interval; updateIntervalButtons(interval);
    const status = document.getElementById('chart-status');
    if (status) status.textContent = 'Загрузка графиков.';
    const to = new Date(); const from = new Date(to); from.setHours(from.getHours() - 24);
    const fromStr = from.toISOString().split('T')[0]; const toStr = to.toISOString().split('T')[0];
    try {
        const response = await fetch(`/api/weather/chart?from=${fromStr}&to=${toStr}&interval=${interval}&fields=temp_outdoor,humidity_outdoor,pressure_relative,wind_speed,wind_gust,solar_radiation,rain_rate,rain_daily`);
        if (!response.ok) throw new Error(`chart request failed: ${response.status}`);
        const data = await response.json();
        if (requestID !== chartRequestID) return;
        applyChartData(data); lastAppliedInterval = interval;
        if (status) status.textContent = formatRangeLabel(data.labels);
    } catch (error) {
        if (requestID !== chartRequestID) return;
        updateIntervalButtons(previousInterval); currentInterval = previousInterval;
        if (status) status.textContent = 'Не удалось загрузить графики. Повторите попытку.';
        console.error('Error loading chart data:', error);
    }
}

function updateCharts(interval) { loadChartData(interval); }
function updateChartColors() {
    const theme = getThemeColors();
    Object.values(charts).forEach(chart => {
        chart.options.scales.y.grid.color = theme.gridColor;
        chart.options.scales.x.ticks.color = theme.textColor;
        chart.options.scales.y.ticks.color = theme.textColor;
        chart.update('none');
    });
}

if (typeof window !== 'undefined') {
    window.addEventListener('themeChanged', updateChartColors);
    setInterval(() => { if (!document.hidden && charts.temp) loadChartData(currentInterval); }, 5 * 60 * 1000);
    document.addEventListener('visibilitychange', () => { if (!document.hidden && charts.temp) loadChartData(currentInterval); });
}

if (typeof module !== 'undefined') module.exports = { applyChartData, chartValues, displayTimeLabel, formatRangeLabel, formatTooltipNumber, loadChartData, refreshChartAccessibility, syncChartTooltips, updateChartColors, validTooltipElements };
