package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

type demoWeatherScenario struct {
	ID, Name                   string
	Temp, Wind, Gust, Rain, UV float32
}

var attentionDemoScenarios = []demoWeatherScenario{
	{"normal", "Обычная погода", 22, 2, 3, 0, 2},
	{"heat", "Жара", 36, 2, 3, 0, 2},
	{"cold", "Мороз", -14, 2, 3, 0, 0},
	{"wind", "Сильный ветер", 22, 8, 18, 0, 2},
	{"rain", "Ливень", 18, 2, 3, 9, 0},
	{"uv", "Высокий UV", 26, 2, 3, 0, 9},
	{"combined", "Жара + ветер + UV", 36, 11, 18, 0, 9},
	{"moderate", "Жёлтые сигналы", 31, 6, 11, 0, 6},
	{"all", "Все пять сигналов", 36, 11, 18, 9, 9},
	{"stale", "Данные устарели", 36, 11, 18, 0, 9},
	{"geomagnetic", "Геомагнитное возмущение", 22, 2, 3, 0, 2},
	{"storm", "Магнитная буря G3", 22, 2, 3, 0, 2},
	{"stale-geomagnetic", "Станция устарела, Kp свежий", 36, 2, 3, 0, 2},
	{"six", "Все шесть сигналов", 36, 11, 18, 9, 9},
}

// NewAttentionDemo serves synthetic observations using the production partial and
// attention rules. It is only registered by cmd/weather-demo, never by the API.
func NewAttentionDemo(templatesDir string) (http.Handler, error) {
	h := &Handler{templatesDir: templatesDir}
	partial, err := h.parsePartial("current_weather.html")
	if err != nil {
		return nil, err
	}
	base, err := os.ReadFile(filepath.Join(templatesDir, "base.html"))
	if err != nil {
		return nil, err
	}
	// Reuse the actual styles without loading analytics or live widgets.
	_, css, ok := strings.Cut(string(base), "<style>")
	if !ok {
		return nil, fmt.Errorf("base template has no inline styles")
	}
	css, _, ok = strings.Cut(css, "</style>")
	if !ok {
		return nil, fmt.Errorf("base template has unterminated styles")
	}
	page, err := template.New("demo").Parse(attentionDemoHTML)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		selected := attentionDemoScenarios[0]
		for _, s := range attentionDemoScenarios {
			if s.ID == r.URL.Query().Get("scenario") {
				selected = s
			}
		}
		now := time.Now()
		var widget bytes.Buffer
		err := partial.Execute(&widget, demoCurrentWeatherData(now, selected))
		if err != nil {
			http.Error(w, "Demo render failed", http.StatusInternalServerError)
			return
		}
		var output bytes.Buffer
		err = page.Execute(&output, map[string]any{
			"CSS":       template.CSS(css),              // trusted repository stylesheet
			"Widget":    template.HTML(widget.String()), // escaped by the production template
			"Scenarios": attentionDemoScenarios, "Selected": selected.ID,
		})
		if err != nil {
			http.Error(w, "Demo render failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(output.Bytes())
	}), nil
}

func demoCurrentWeatherData(now time.Time, selected demoWeatherScenario) map[string]any {
	observed := now
	if selected.ID == "stale" || selected.ID == "stale-geomagnetic" {
		observed = now.Add(-45 * time.Minute)
	}
	current := &models.WeatherData{Time: observed, TempOutdoor: &selected.Temp, WindSpeed: &selected.Wind, WindGust: &selected.Gust, RainRate: &selected.Rain, UVIndex: &selected.UV}
	pressure, previousPressure := float32(755), float32(755)
	if selected.ID == "all" || selected.ID == "six" {
		pressure = 751
	}
	current.PressureRelative = &pressure
	previous := &models.WeatherData{Time: observed.Add(-time.Hour), PressureRelative: &previousPressure}
	var geomagnetic GeomagneticCardData
	kp := float32(0)
	switch selected.ID {
	case "geomagnetic", "stale-geomagnetic":
		kp = 4.3
	case "storm", "six":
		kp = 7
	}
	if kp > 0 {
		status := models.ClassifyKp(kp)
		geomagnetic = GeomagneticCardData{
			HasData: true, Kp: kp, StatusHeading: statusHeading(status, kp),
			StatusText: status.TextColor(), StatusGradient: status.TailwindGradient(), IsAttention: true,
			Attention: buildGeomagneticSignal(&models.GeomagneticKp{SlotTime: now.Add(-time.Hour), Kp: kp}, now),
		}
	}
	return map[string]any{
		"Attention":       buildCurrentAttention(current, previous, now).withGeomagnetic(geomagnetic.Attention),
		"Geomagnetic":     geomagnetic,
		"ObservationTime": observed.Format("15:04"), "UpdatedAt": now.Format("15:04"),
		"TempOutdoor": selected.Temp, "TempFeelsLike": selected.Temp, "HumidityOutdoor": 55,
		"PressureRelative": pressure, "WindSpeed": selected.Wind, "WindGust": selected.Gust,
		"HasWindGust": true, "WindDirectionStr": "СЗ", "WindDirection": 315,
		"RainDaily": selected.Rain, "UVIndex": selected.UV, "SolarRadiation": 420.0, "Illuminance": 50400.0,
	}
}

const attentionDemoHTML = `<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Демо подсветки погоды</title>
<script>if(localStorage.getItem('weather-demo-dark')==='true')document.documentElement.classList.add('dark');</script>
<script src="/static/js/vendor/tailwind.min.js"></script><script>tailwind.config={darkMode:'class'};</script><style>{{.CSS}}</style></head>
<body><main class="max-w-5xl mx-auto p-4 sm:p-8">
<h1 class="text-2xl font-bold">Подсветка погодных показателей</h1>
<p class="mt-2 mb-5 ui-text-muted">Локальное демо · все значения вымышленные</p>
<form class="flex flex-wrap gap-3 items-end mb-6" method="get">
<label class="font-semibold">Сценарий<select name="scenario" class="ui-field block mt-2" onchange="this.form.submit()">{{range .Scenarios}}<option value="{{.ID}}" {{if eq .ID $.Selected}}selected{{end}}>{{.Name}}</option>{{end}}</select></label>
<button class="ui-button-secondary" type="submit">Показать</button>
<button class="ui-button-secondary" type="button" id="theme" aria-pressed="false">Тёмная тема</button>
<label class="flex items-center gap-2 min-h-11"><input type="checkbox" id="mobile">Ширина телефона</label>
</form><div id="preview">{{.Widget}}</div>
<p class="mt-5 text-sm ui-text-muted">Ссылки показателей отключены в демо. Выберите сценарий, чтобы сравнить спокойные и выделенные блоки.</p>
</main><script>
const theme=document.getElementById('theme');
function syncTheme(){theme.setAttribute('aria-pressed',document.documentElement.classList.contains('dark'));}
theme.onclick=()=>{document.documentElement.classList.toggle('dark');localStorage.setItem('weather-demo-dark',document.documentElement.classList.contains('dark'));syncTheme();};syncTheme();
document.querySelectorAll('#preview a').forEach(a=>{a.removeAttribute('href');a.style.cursor='default';});
document.getElementById('mobile').onchange=e=>{const url=new URL(location.href);url.searchParams.set('phone',e.target.checked?'1':'0');location.href=url;};
if(new URLSearchParams(location.search).get('phone')==='1'){
document.getElementById('mobile').checked=true;
const frame=document.createElement('iframe');frame.title='Мобильный вид';frame.style='width:390px;max-width:100%;height:1100px;border:0';
const url=new URL(location.href);url.searchParams.delete('phone');url.searchParams.set('frame','1');frame.src=url;document.getElementById('preview').replaceChildren(frame);
}
if(new URLSearchParams(location.search).has('frame')){document.querySelector('form').remove();document.querySelector('h1').remove();document.querySelector('main > p').remove();}
window.addEventListener('storage',e=>{if(e.key==='weather-demo-dark'){document.documentElement.classList.toggle('dark',e.newValue==='true');syncTheme();}});
</script><script src="/static/js/attention.js?v=1"></script></body></html>`
