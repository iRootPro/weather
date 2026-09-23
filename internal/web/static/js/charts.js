// Dashboard Charts
let charts = {};
let currentInterval = '5m';
let lastAppliedInterval = '5m';
let sharedTooltipIndex = null;
let chartRequestID = 0;

function formatAccessibleNumber(value, fractionDigits = 1) {
    return Number(value).toLocaleString('ru-RU', {
        minimumFractionDigits: fractionDigits,
        maximumFractionDigits: fractionDigits
    });
}

function configureChartAccessibility(chart, name, formatValue) {
    if (!chart?.canvas) return;

    chart.accessibility = { name, formatValue };
    const details = document.getElementById(`${chart.canvas.id}-details`);
    if (details) {
        details.addEventListener('toggle', () => {
            if (details.open) renderChartDataTable(chart);
        });
    }
    updateChartAccessibility(chart);
}

function updateChartAccessibility(chart) {
    if (!chart?.accessibility) return;

    const { name, formatValue } = chart.accessibility;
    const summary = document.getElementById(`${chart.canvas.id}-summary`);
    if (summary) {
        const series = chart.data.datasets.map(dataset => {
            const values = dataset.data.filter(value => Number.isFinite(value));
            if (values.length === 0) return `${dataset.label}: нет данных.`;

            const current = values.at(-1);
            const min = Math.min(...values);
            const max = Math.max(...values);
            const previous = values.at(-2);
            let trend = '';
            if (previous !== undefined) {
                const delta = current - previous;
                if (delta > 0) trend = ` Рост на ${formatValue(delta, dataset)} от предыдущего значения.`;
                if (delta < 0) trend = ` Снижение на ${formatValue(Math.abs(delta), dataset)} от предыдущего значения.`;
                if (delta === 0) trend = ' Без изменения от предыдущего значения.';
            }
            return `${dataset.label}: последнее значение ${formatValue(current, dataset)}, минимум ${formatValue(min, dataset)}, максимум ${formatValue(max, dataset)}.${trend}`;
        });
        summary.textContent = `${name}. ${series.join(' ')}`;
    }

    const details = document.getElementById(`${chart.canvas.id}-details`);
    if (details?.open) renderChartDataTable(chart);
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

    const caption = document.createElement('caption');
    caption.className = 'sr-only';
    caption.textContent = `${name}: значения по времени`;
    table.append(caption);

    const thead = document.createElement('thead');
    thead.className = 'bg-gray-50 text-gray-600 dark:bg-gray-900 dark:text-gray-300';
    const headerRow = document.createElement('tr');
    ['Время', ...chart.data.datasets.map(dataset => dataset.label)].forEach(label => {
        const cell = document.createElement('th');
        cell.className = 'px-3 py-2 font-semibold';
        cell.scope = 'col';
        cell.textContent = label;
        headerRow.append(cell);
    });
    thead.append(headerRow);
    table.append(thead);

    const tbody = document.createElement('tbody');
    chart.data.labels.forEach((label, index) => {
        const row = document.createElement('tr');
        row.className = 'border-t border-gray-100 dark:border-gray-700';

        const time = document.createElement('th');
        time.className = 'px-3 py-2 font-medium';
        time.scope = 'row';
        time.textContent = label;
        row.append(time);

        chart.data.datasets.forEach(dataset => {
            const cell = document.createElement('td');
            cell.className = 'px-3 py-2';
            const value = dataset.data[index];
            cell.textContent = Number.isFinite(value) ? formatValue(value, dataset) : '—';
            row.append(cell);
        });
        tbody.append(row);
    });
    table.append(tbody);

    container.replaceChildren(table);
}

function renderChartDataCards(chart, container, name, formatValue) {
    const list = document.createElement('ol');
    list.className = 'space-y-2 p-2';
    list.setAttribute('aria-label', `${name}: значения по времени`);

    chart.data.labels.forEach((label, index) => {
        const item = document.createElement('li');
        item.className = 'ui-metric-card p-3';

        const time = document.createElement('time');
        time.className = 'ui-tabular text-xs font-semibold text-gray-900 dark:text-white';
        time.textContent = label;
        item.append(time);

        const values = document.createElement('dl');
        values.className = 'mt-2 grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-1';
        chart.data.datasets.forEach(dataset => {
            const label = document.createElement('dt');
            label.className = 'min-w-0 text-xs text-gray-600 dark:text-gray-300';
            label.textContent = dataset.label;
            values.append(label);

            const value = document.createElement('dd');
            value.className = 'ui-tabular text-right text-xs font-semibold text-gray-900 dark:text-white';
            const dataPoint = dataset.data[index];
            value.textContent = Number.isFinite(dataPoint) ? formatValue(dataPoint, dataset) : '—';
            values.append(value);
        });
        item.append(values);
        list.append(item);
    });

    container.replaceChildren(list);
}

function refreshChartAccessibility(chartCollection) {
    Object.values(chartCollection).forEach(updateChartAccessibility);
}

let chartDataLayoutIsMobile = window.matchMedia('(max-width: 639px)').matches;

window.addEventListener('resize', () => {
    const isMobile = window.matchMedia('(max-width: 639px)').matches;
    if (isMobile === chartDataLayoutIsMobile) return;

    chartDataLayoutIsMobile = isMobile;
    document.querySelectorAll('details[id$="-details"][open]').forEach(details => {
        const canvasID = details.id.replace(/-details$/, '');
        const chart = Chart.getChart(document.getElementById(canvasID));
        if (chart) renderChartDataTable(chart);
    });
});

// Get theme-specific colors
function getThemeColors() {
    const isDark = document.documentElement.classList.contains('dark');
    return {
        gridColor: isDark ? 'rgba(156, 163, 175, 0.15)' : '#f3f4f6', // Barely visible gray grid in dark mode
        textColor: isDark ? '#9ca3af' : '#6b7280',
        crosshairColor: isDark ? 'rgba(255, 255, 255, 0.3)' : 'rgba(0, 0, 0, 0.2)'
    };
}

// Crosshair plugin to draw vertical line
const crosshairPlugin = {
    id: 'crosshair',
    afterDraw: (chart) => {
        if (sharedTooltipIndex !== null && chart.tooltip?._active?.length) {
            const ctx = chart.ctx;
            const x = chart.tooltip._active[0].element.x;
            const topY = chart.scales.y.top;
            const bottomY = chart.scales.y.bottom;

            ctx.save();
            ctx.beginPath();
            ctx.moveTo(x, topY);
            ctx.lineTo(x, bottomY);
            ctx.lineWidth = 1;
            ctx.strokeStyle = getThemeColors().crosshairColor;
            ctx.setLineDash([5, 5]);
            ctx.stroke();
            ctx.restore();
        }
    }
};

// Sync tooltips across all charts
function syncChartTooltips(sourceChart, dataIndex) {
    sharedTooltipIndex = dataIndex;

    Object.values(charts).forEach(chart => {
        if (!chart || chart === sourceChart) return;

        if (dataIndex !== null && dataIndex < chart.data.labels.length) {
            const meta = chart.getDatasetMeta(0);
            if (meta && meta.data[dataIndex]) {
                chart.tooltip.setActiveElements([{
                    datasetIndex: 0,
                    index: dataIndex
                }]);
                chart.update('none');
            }
        } else {
            chart.tooltip.setActiveElements([]);
            chart.update('none');
        }
    });
}

function initCharts() {
    const themeColors = getThemeColors();

    const commonOptions = {
        responsive: true,
        maintainAspectRatio: false,
        interaction: {
            intersect: false,
            mode: 'index'
        },
        onHover: function(event, activeElements) {
            if (activeElements.length > 0) {
                syncChartTooltips(this, activeElements[0].index);
            } else {
                syncChartTooltips(this, null);
            }
        },
        plugins: {
            legend: {
                display: false,
                labels: {
                    color: themeColors.textColor
                }
            },
            tooltip: {
                backgroundColor: 'rgba(0, 0, 0, 0.8)',
                padding: 8,
                titleFont: { size: 12 },
                bodyFont: { size: 11 }
            },
            crosshair: true
        },
        scales: {
            x: {
                grid: { display: false },
                ticks: {
                    maxRotation: 0,
                    autoSkip: true,
                    maxTicksLimit: 6,
                    font: { size: 10 },
                    color: themeColors.textColor
                }
            },
            y: {
                grid: { color: themeColors.gridColor },
                ticks: {
                    padding: 4,
                    font: { size: 10 },
                    color: themeColors.textColor
                }
            }
        },
        elements: {
            point: { radius: 1, hoverRadius: 4 },
            line: { tension: 0.3, borderWidth: 2 }
        }
    };

    // Temperature chart
    charts.temp = new Chart(document.getElementById('tempChart'), {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Температура',
                data: [],
                borderColor: '#f97316',
                backgroundColor: 'rgba(249, 115, 22, 0.1)',
                fill: true
            }]
        },
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                tooltip: {
                    ...commonOptions.plugins.tooltip,
                    callbacks: {
                        label: (ctx) => ctx.dataset.label + ': ' + ctx.raw.toFixed(1) + '°C'
                    }
                }
            },
            scales: {
                ...commonOptions.scales,
                y: {
                    ...commonOptions.scales.y,
                    ticks: {
                        callback: (value) => value.toFixed(1) + '°',
                        font: { size: 10 }
                    }
                }
            }
        },
        plugins: [crosshairPlugin]
    });

    // Humidity chart
    charts.humidity = new Chart(document.getElementById('humidityChart'), {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Влажность',
                data: [],
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true
            }]
        },
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                tooltip: {
                    ...commonOptions.plugins.tooltip,
                    callbacks: {
                        label: (ctx) => ctx.dataset.label + ': ' + Math.round(ctx.raw) + '%'
                    }
                }
            },
            scales: {
                ...commonOptions.scales,
                y: {
                    ...commonOptions.scales.y,
                    min: 0,
                    max: 100,
                    ticks: {
                        callback: (value) => value + '%',
                        font: { size: 10 }
                    }
                }
            }
        },
        plugins: [crosshairPlugin]
    });

    // Pressure chart
    charts.pressure = new Chart(document.getElementById('pressureChart'), {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Давление',
                data: [],
                borderColor: '#8b5cf6',
                backgroundColor: 'rgba(139, 92, 246, 0.1)',
                fill: true
            }]
        },
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                tooltip: {
                    ...commonOptions.plugins.tooltip,
                    callbacks: {
                        label: (ctx) => ctx.dataset.label + ': ' + ctx.raw.toFixed(1) + ' мм'
                    }
                }
            },
            scales: {
                ...commonOptions.scales,
                y: {
                    ...commonOptions.scales.y,
                    ticks: {
                        callback: (value) => value.toFixed(1) + ' мм',
                        font: { size: 10 }
                    }
                }
            }
        },
        plugins: [crosshairPlugin]
    });

    // Wind chart
    charts.wind = new Chart(document.getElementById('windChart'), {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'Скорость',
                    data: [],
                    borderColor: '#14b8a6',
                    backgroundColor: 'rgba(20, 184, 166, 0.1)',
                    fill: true
                },
                {
                    label: 'Порывы',
                    data: [],
                    borderColor: '#f43f5e',
                    backgroundColor: 'transparent',
                    borderDash: [5, 5]
                }
            ]
        },
        options: {
            ...commonOptions,
            plugins: {
                ...commonOptions.plugins,
                legend: {
                    display: true,
                    position: 'top',
                    labels: {
                        font: { size: 10 },
                        usePointStyle: true,
                        pointStyle: 'line',
                        color: getThemeColors().textColor
                    }
                },
                tooltip: {
                    ...commonOptions.plugins.tooltip,
                    callbacks: {
                        label: (ctx) => ctx.dataset.label + ': ' + ctx.raw.toFixed(1) + ' м/с'
                    }
                }
            },
            scales: {
                ...commonOptions.scales,
                y: {
                    ...commonOptions.scales.y,
                    min: 0,
                    ticks: {
                        callback: (value) => value.toFixed(0) + ' м/с',
                        font: { size: 10 }
                    }
                }
            }
        },
        plugins: [crosshairPlugin]
    });

    // Solar/Illuminance chart
    const solarCanvas = document.getElementById('solarChart');
    if (solarCanvas) {
        charts.solar = new Chart(solarCanvas, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Освещённость',
                    data: [],
                    borderColor: '#eab308',
                    backgroundColor: 'rgba(234, 179, 8, 0.1)',
                    fill: true
                }]
            },
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    tooltip: {
                        ...commonOptions.plugins.tooltip,
                        callbacks: {
                            label: (ctx) => ctx.dataset.label + ': ' + Math.round(ctx.raw * 120) + ' люкс'
                        }
                    }
                },
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        min: 0,
                        ticks: {
                            callback: (value) => Math.round(value * 120) + ' лк',
                            font: { size: 10 }
                        }
                    }
                }
            },
            plugins: [crosshairPlugin]
        });
    }

    // Rain chart
    const rainCanvas = document.getElementById('rainChart');
    if (rainCanvas) {
        charts.rain = new Chart(rainCanvas, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: 'Дневные осадки (мм)',
                        data: [],
                        borderColor: 'rgb(6, 182, 212)',
                        backgroundColor: 'rgba(6, 182, 212, 0.1)',
                        borderWidth: 2,
                        tension: 0.4,
                        fill: true
                    },
                    {
                        label: 'Интенсивность (мм/ч)',
                        data: [],
                        borderColor: 'rgb(59, 130, 246)',
                        borderDash: [5, 5],
                        borderWidth: 2,
                        tension: 0.4,
                        fill: false
                    }
                ]
            },
            options: {
                ...commonOptions,
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        min: 0,
                        ticks: {
                            callback: (value) => value.toFixed(1),
                            font: { size: 10 }
                        }
                    }
                }
            },
            plugins: [crosshairPlugin]
        });
    }

    configureChartAccessibility(charts.temp, 'Температура', value => `${formatAccessibleNumber(value)} °C`);
    configureChartAccessibility(charts.humidity, 'Влажность', value => `${formatAccessibleNumber(value, 0)} %`);
    configureChartAccessibility(charts.pressure, 'Давление', value => `${formatAccessibleNumber(value)} мм рт. ст.`);
    configureChartAccessibility(charts.wind, 'Ветер', value => `${formatAccessibleNumber(value)} м/с`);
    configureChartAccessibility(charts.solar, 'Освещённость', value => `${formatAccessibleNumber(value * 120, 0)} лк`);
    configureChartAccessibility(charts.rain, 'Осадки', (value, dataset) => {
        const unit = dataset.label.includes('Интенсивность') ? 'мм/ч' : 'мм';
        return `${formatAccessibleNumber(value)} ${unit}`;
    });
}

async function loadChartData(interval) {
    const requestID = ++chartRequestID;
    const previousInterval = lastAppliedInterval;
    currentInterval = interval;

    // Update button states
    document.querySelectorAll('.chart-interval-btn').forEach(btn => {
        if (btn.dataset.interval === interval) {
            btn.classList.remove('ui-button-secondary');
            btn.classList.add('ui-button-primary');
            btn.setAttribute('aria-pressed', 'true');
        } else {
            btn.classList.remove('ui-button-primary');
            btn.classList.add('ui-button-secondary');
            btn.setAttribute('aria-pressed', 'false');
        }
    });
    const status = document.getElementById('chart-status');
    if (status) status.textContent = 'Загрузка графиков.';

    // Calculate date range (last 24 hours)
    const to = new Date();
    const from = new Date(to);
    from.setHours(from.getHours() - 24);

    const fromStr = from.toISOString().split('T')[0];
    const toStr = to.toISOString().split('T')[0];

    try {
        const response = await fetch(
            `/api/weather/chart?from=${fromStr}&to=${toStr}&interval=${interval}&fields=temp_outdoor,humidity_outdoor,pressure_relative,wind_speed,wind_gust,solar_radiation,rain_rate,rain_daily`
        );
        if (!response.ok) throw new Error(`chart request failed: ${response.status}`);
        const data = await response.json();
        if (requestID !== chartRequestID) return;

        // Format labels for display
        const labels = data.labels.map(label => {
            const parts = label.split(' ');
            return parts.length > 1 ? parts[1] : label;
        });

        // Update temperature chart
        charts.temp.data.labels = labels;
        charts.temp.data.datasets[0].data = data.datasets.temp_outdoor;
        charts.temp.update();

        // Update humidity chart
        charts.humidity.data.labels = labels;
        charts.humidity.data.datasets[0].data = data.datasets.humidity_outdoor;
        charts.humidity.update();

        // Update pressure chart
        charts.pressure.data.labels = labels;
        charts.pressure.data.datasets[0].data = data.datasets.pressure_relative;
        charts.pressure.update();

        // Update wind chart
        charts.wind.data.labels = labels;
        charts.wind.data.datasets[0].data = data.datasets.wind_speed;
        charts.wind.data.datasets[1].data = data.datasets.wind_gust;
        charts.wind.update();

        // Update solar chart
        if (charts.solar) {
            charts.solar.data.labels = labels;
            charts.solar.data.datasets[0].data = data.datasets.solar_radiation;
            charts.solar.update();
        }

        // Update rain chart
        if (charts.rain) {
            charts.rain.data.labels = labels;
            charts.rain.data.datasets[0].data = data.datasets.rain_daily || [];
            charts.rain.data.datasets[1].data = data.datasets.rain_rate || [];

            // Динамическое масштабирование для малых значений
            // Учитываем оба dataset'а для определения масштаба
            const rainDailyValues = data.datasets.rain_daily || [];
            const rainRateValues = data.datasets.rain_rate || [];
            const maxDaily = Math.max(...rainDailyValues.filter(v => v != null), 0);
            const maxRate = Math.max(...rainRateValues.filter(v => v != null), 0);
            const maxRain = Math.max(maxDaily, maxRate);

            // Если максимальное значение очень маленькое, установим меньший масштаб
            if (maxRain < 0.5) {
                charts.rain.options.scales.y.suggestedMax = 0.5;
            } else if (maxRain < 2) {
                charts.rain.options.scales.y.suggestedMax = 2;
            } else {
                charts.rain.options.scales.y.suggestedMax = undefined;
            }

            charts.rain.update();
        }

        refreshChartAccessibility(charts);
        lastAppliedInterval = interval;

    } catch (error) {
        if (requestID !== chartRequestID) return;
        document.querySelectorAll('.chart-interval-btn').forEach(btn => {
            const active = btn.dataset.interval === previousInterval;
            btn.classList.toggle('ui-button-primary', active);
            btn.classList.toggle('ui-button-secondary', !active);
            btn.setAttribute('aria-pressed', String(active));
        });
        currentInterval = previousInterval;
        if (status) status.textContent = 'Не удалось загрузить графики. Повторите попытку.';
        console.error('Error loading chart data:', error);
    }
}

function updateCharts(interval) {
    loadChartData(interval);
}

// Update chart colors when theme changes
function updateChartColors() {
    const themeColors = getThemeColors();

    Object.values(charts).forEach(chart => {
        if (!chart) return;

        // Update grid colors
        if (chart.options.scales?.y?.grid) {
            chart.options.scales.y.grid.color = themeColors.gridColor;
        }

        // Update tick colors
        if (chart.options.scales?.x?.ticks) {
            chart.options.scales.x.ticks.color = themeColors.textColor;
        }
        if (chart.options.scales?.y?.ticks) {
            chart.options.scales.y.ticks.color = themeColors.textColor;
        }

        // Update legend colors
        if (chart.options.plugins?.legend?.labels) {
            chart.options.plugins.legend.labels.color = themeColors.textColor;
        }

        chart.update('none'); // Update without animation
    });
}

// Listen for theme changes
window.addEventListener('themeChanged', updateChartColors);

// Auto-refresh charts every 5 minutes
setInterval(() => {
    if (!document.hidden && typeof charts.temp !== 'undefined') {
        loadChartData(currentInterval);
    }
}, 5 * 60 * 1000);

document.addEventListener('visibilitychange', () => {
    if (!document.hidden && typeof charts.temp !== 'undefined') {
        loadChartData(currentInterval);
    }
});
