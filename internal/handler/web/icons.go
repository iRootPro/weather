package web

import (
	"html/template"
	"strings"
)

// Lucide icon paths below are copied from lucide-icons/lucide at
// f12b0de177fbc2a6795e99be065887e72b237123 (tag 0.468.0). See
// THIRD_PARTY_NOTICES.md for the ISC license and attribution.
//
// This is deliberately a closed mapping: templates can select only reviewed,
// static markup and never interpolate SVG supplied by data sources.
var lucidePaths = map[string]string{
	"sun":             `<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/>`,
	"cloud-sun":       `<path d="M12 2v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="M20 12h2"/><path d="m19.07 4.93-1.41 1.41"/><path d="M15.947 12.65a4 4 0 0 0-5.925-4.128"/><path d="M13 22H7a5 5 0 1 1 4.9-6H13a3 3 0 0 1 0 6Z"/>`,
	"cloud":           `<path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/>`,
	"cloud-fog":       `<path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 17H7"/><path d="M17 21H9"/>`,
	"cloud-drizzle":   `<path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M8 19v1"/><path d="M8 14v1"/><path d="M16 19v1"/><path d="M16 14v1"/><path d="M12 21v1"/><path d="M12 16v1"/>`,
	"cloud-rain":      `<path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/>`,
	"cloud-snow":      `<path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M8 15h.01"/><path d="M8 19h.01"/><path d="M12 17h.01"/><path d="M12 21h.01"/><path d="M16 15h.01"/><path d="M16 19h.01"/>`,
	"cloud-lightning": `<path d="M6 16.326A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 .5 8.973"/><path d="m13 12-3 5h4l-3 5"/>`,
	"thermometer":     `<path d="M14 4v10.54a4 4 0 1 1-4 0V4a2 2 0 0 1 4 0Z"/>`,
	"droplets":        `<path d="M7 16.3c2.2 0 4-1.83 4-4.05 0-1.16-.57-2.26-1.71-3.19S7.29 6.75 7 5.3c-.29 1.45-1.14 2.84-2.29 3.76S3 11.1 3 12.25c0 2.22 1.8 4.05 4 4.05z"/><path d="M12.56 6.6A10.97 10.97 0 0 0 14 3.02c.5 2.5 2 4.9 4 6.5s3 3.5 3 5.5a6.98 6.98 0 0 1-11.91 4.97"/>`,
	"gauge":           `<path d="m12 14 4-4"/><path d="M3.34 19a10 10 0 1 1 17.32 0"/>`,
	"wind":            `<path d="M12.8 19.6A2 2 0 1 0 14 16H2"/><path d="M17.5 8a2.5 2.5 0 1 1 2 4H2"/><path d="M9.8 4.4A2 2 0 1 1 11 8H2"/>`,
	"droplet":         `<path d="M12 22a7 7 0 0 0 7-7c0-2-1-3.9-3-5.5s-3.5-4-4-6.5c-.5 2.5-2 4.9-4 6.5C6 11.1 5 13 5 15a7 7 0 0 0 7 7z"/>`,
	"waves":           `<path d="M2 6c.6.5 1.2 1 2.5 1C7 7 7 5 9.5 5c2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1"/><path d="M2 12c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1"/><path d="M2 18c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1"/>`,
	"magnet":          `<path d="m6 15-4-4 6.75-6.77a7.79 7.79 0 0 1 11 11L13 22l-4-4 6.39-6.36a2.14 2.14 0 0 0-3-3L6 15"/><path d="m5 8 4 4"/><path d="m12 15 4 4"/>`,
	"sun-moon":        `<path d="M12 8a2.83 2.83 0 0 0 4 4 4 4 0 1 1-4-4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.9 4.9 1.4 1.4"/><path d="m17.7 17.7 1.4 1.4"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.3 17.7-1.4 1.4"/><path d="m19.1 4.9-1.4 1.4"/>`,
	"moon":            `<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>`,
	"sunrise":         `<path d="M12 2v8"/><path d="m4.93 10.93 1.41 1.41"/><path d="M2 18h2"/><path d="M20 18h2"/><path d="m19.07 10.93-1.41 1.41"/><path d="M22 22H2"/><path d="m8 6 4-4 4 4"/><path d="M16 18a4 4 0 0 0-8 0"/>`,
	"sunset":          `<path d="M12 10V2"/><path d="m4.93 10.93 1.41 1.41"/><path d="M2 18h2"/><path d="M20 18h2"/><path d="m19.07 10.93-1.41 1.41"/><path d="M22 22H2"/><path d="m16 6-4 4-4-4"/><path d="M16 18a4 4 0 0 0-8 0"/>`,
	"chevron-right":   `<path d="m9 18 6-6-6-6"/>`,
	"arrow-up":        `<path d="m5 12 7-7 7 7"/><path d="M12 19V5"/>`,
	"circle-alert":    `<circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/>`,
}

func icon(name string) template.HTML {
	path, ok := lucidePaths[name]
	if !ok {
		name, path = "circle-alert", lucidePaths["circle-alert"]
	}
	return template.HTML(`<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" aria-hidden="true" data-icon="` + name + `">` + path + `</svg>`)
}

func weatherIcon(code int16) template.HTML {
	name := "circle-alert"
	switch {
	case code == 0:
		name = "sun"
	case code == 1 || code == 2:
		name = "cloud-sun"
	case code == 3:
		name = "cloud"
	case code == 45 || code == 48:
		name = "cloud-fog"
	case code >= 51 && code <= 57:
		name = "cloud-drizzle"
	case code >= 61 && code <= 67 || code >= 80 && code <= 82:
		name = "cloud-rain"
	case code >= 71 && code <= 77 || code >= 85 && code <= 86:
		name = "cloud-snow"
	case code == 95 || code == 96 || code == 99:
		name = "cloud-lightning"
	}
	return template.HTML(strings.Replace(string(icon(name)), `class="ui-icon"`, `class="ui-icon ui-weather-icon"`, 1))
}

// moonPhaseIcon keeps visual phase distinctions local to the web surface.
// The inner arcs use rx=4 and ry=9, so they do not normalize to the outer
// circle (r=9). This keeps crescents and gibbous phases visibly distinct.
func moonPhaseIcon(phase string) template.HTML {
	paths := map[string]string{
		"Новолуние":          `<circle cx="12" cy="12" r="8.5" fill="currentColor" fill-opacity=".18"/><circle cx="12" cy="12" r="8.5"/>`,
		"Растущая луна":      `<path d="M12 3A9 9 0 0 1 12 21A4 9 0 0 0 12 3Z" fill="currentColor" fill-opacity=".32"/><circle cx="12" cy="12" r="9"/>`,
		"Первая четверть":    `<path d="M12 3A9 9 0 0 1 12 21Z" fill="currentColor" fill-opacity=".52"/><circle cx="12" cy="12" r="9"/>`,
		"Прибывающая луна":   `<path d="M12 3A9 9 0 0 1 12 21A4 9 0 0 1 12 3Z" fill="currentColor" fill-opacity=".72"/><circle cx="12" cy="12" r="9"/>`,
		"Полнолуние":         `<circle cx="12" cy="12" r="8.5" fill="currentColor" fill-opacity=".82"/><circle cx="12" cy="12" r="8.5"/>`,
		"Убывающая луна":     `<path d="M12 3A9 9 0 0 0 12 21A4 9 0 0 0 12 3Z" fill="currentColor" fill-opacity=".72"/><circle cx="12" cy="12" r="9"/>`,
		"Последняя четверть": `<path d="M12 3A9 9 0 0 0 12 21Z" fill="currentColor" fill-opacity=".52"/><circle cx="12" cy="12" r="9"/>`,
		"Стареющая луна":     `<path d="M12 3A9 9 0 0 0 12 21A4 9 0 0 1 12 3Z" fill="currentColor" fill-opacity=".32"/><circle cx="12" cy="12" r="9"/>`,
	}
	path, ok := paths[phase]
	if !ok {
		return icon("circle-alert")
	}
	return template.HTML(`<svg class="ui-icon ui-moon-phase" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" aria-hidden="true" data-moon-phase="true">` + path + `</svg>`)
}

func eventIcon(eventType string) template.HTML {
	icons := map[string]string{
		"rain_start":    "cloud-rain",
		"rain_end":      "cloud-rain",
		"temp_drop":     "thermometer",
		"temp_rise":     "thermometer",
		"wind_gust":     "wind",
		"pressure_drop": "gauge",
		"pressure_rise": "gauge",
	}
	name, ok := icons[eventType]
	if !ok {
		name = "circle-alert"
	}
	return icon(name)
}
