// Dashboard Charts
let charts = {};
let currentInterval = '1h';

// Get theme-specific colors
function getThemeColors() {
    const isDark = document.documentElement.classList.contains('dark');
    return {
        gridColor: isDark ? 'rgba(156, 163, 175, 0.15)' : '#f3f4f6', // Barely visible gray grid in dark mode
        textColor: isDark ? '#9ca3af' : '#6b7280'
    };
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
            }
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
        }
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
        }
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
        }
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
        }
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
            }
        });
    }

    // Rain chart
    const rainCanvas = document.getElementById('rainChart');
    if (rainCanvas) {
        charts.rain = new Chart(rainCanvas, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Осадки',
                    data: [],
                    backgroundColor: 'rgba(6, 182, 212, 0.7)',
                    borderColor: '#06b6d4',
                    borderWidth: 1
                }]
            },
            options: {
                ...commonOptions,
                plugins: {
                    ...commonOptions.plugins,
                    tooltip: {
                        ...commonOptions.plugins.tooltip,
                        callbacks: {
                            label: (ctx) => ctx.dataset.label + ': ' + ctx.raw.toFixed(2) + ' мм/ч'
                        }
                    }
                },
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        min: 0,
                        ticks: {
                            callback: (value) => value.toFixed(1) + ' мм',
                            font: { size: 10 }
                        }
                    }
                }
            }
        });
    }
}

async function loadChartData(interval) {
    currentInterval = interval;

    // Update button states
    const isDark = document.documentElement.classList.contains('dark');
    document.querySelectorAll('.chart-interval-btn').forEach(btn => {
        if (btn.dataset.interval === interval) {
            btn.classList.remove('bg-gray-200', 'dark:bg-gray-700', 'text-gray-700', 'dark:text-gray-300');
            btn.classList.add('bg-blue-500', 'text-white');
        } else {
            btn.classList.remove('bg-blue-500', 'text-white');
            btn.classList.add('bg-gray-200', 'dark:bg-gray-700', 'text-gray-700', 'dark:text-gray-300');
        }
    });

    // Calculate date range (last 24 hours)
    const to = new Date();
    const from = new Date(to);
    from.setHours(from.getHours() - 24);

    const fromStr = from.toISOString().split('T')[0];
    const toStr = to.toISOString().split('T')[0];

    try {
        const response = await fetch(
            `/api/weather/chart?from=${fromStr}&to=${toStr}&interval=${interval}&fields=temp_outdoor,humidity_outdoor,pressure_relative,wind_speed,wind_gust,solar_radiation,rain_rate`
        );
        const data = await response.json();

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
            charts.rain.data.datasets[0].data = data.datasets.rain_rate;
            charts.rain.update();

            // Show/hide no-rain message
            const noRainMsg = document.getElementById('noRainMessage');
            const rainCanvas = document.getElementById('rainChart');
            if (noRainMsg && rainCanvas) {
                const hasRain = data.datasets.rain_rate && data.datasets.rain_rate.some(v => v > 0);
                if (hasRain) {
                    noRainMsg.classList.add('hidden');
                    rainCanvas.classList.remove('hidden');
                } else {
                    noRainMsg.classList.remove('hidden');
                    rainCanvas.classList.add('hidden');
                }
            }
        }

    } catch (error) {
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
    if (typeof charts.temp !== 'undefined') {
        loadChartData(currentInterval);
    }
}, 5 * 60 * 1000);
