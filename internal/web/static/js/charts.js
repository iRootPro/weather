// Dashboard Charts
let charts = {};
let currentInterval = '1h';

function initCharts() {
    const commonOptions = {
        responsive: true,
        maintainAspectRatio: false,
        interaction: {
            intersect: false,
            mode: 'index'
        },
        plugins: {
            legend: {
                display: false
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
                    font: { size: 10 }
                }
            },
            y: {
                grid: { color: '#f3f4f6' },
                ticks: {
                    padding: 4,
                    font: { size: 10 }
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
                        callback: (value) => value.toFixed(0) + '°',
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
                legend: { display: true, position: 'top', labels: { font: { size: 10 } } },
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
}

async function loadChartData(interval) {
    currentInterval = interval;

    // Update button states
    document.querySelectorAll('.chart-interval-btn').forEach(btn => {
        if (btn.dataset.interval === interval) {
            btn.classList.remove('bg-gray-200', 'text-gray-700');
            btn.classList.add('bg-blue-500', 'text-white');
        } else {
            btn.classList.remove('bg-blue-500', 'text-white');
            btn.classList.add('bg-gray-200', 'text-gray-700');
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
            `/api/weather/chart?from=${fromStr}&to=${toStr}&interval=${interval}&fields=temp_outdoor,humidity_outdoor,pressure_relative,wind_speed,wind_gust,solar_radiation`
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

    } catch (error) {
        console.error('Error loading chart data:', error);
    }
}

function updateCharts(interval) {
    loadChartData(interval);
}

// Auto-refresh charts every 5 minutes
setInterval(() => {
    if (typeof charts.temp !== 'undefined') {
        loadChartData(currentInterval);
    }
}, 5 * 60 * 1000);
