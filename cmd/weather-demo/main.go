// weather-demo runs a loopback-only UI preview with synthetic weather data.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/iRootPro/weather/internal/handler/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8091", "loopback address for the demo server")
	flag.Parse()
	demo, err := web.NewAttentionDemo("internal/web/templates")
	if err != nil {
		log.Fatal(err)
	}
	forecastDemo, err := web.NewForecastDemo("internal/web/templates")
	if err != nil {
		log.Fatal(err)
	}
	dashboardDemo, err := web.NewDashboardDemo("internal/web/templates")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /", demo)
	mux.Handle("GET /forecast", forecastDemo)
	mux.Handle("GET /dashboard", dashboardDemo)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/web/static"))))
	server := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Weather UI demo: http://%s", *addr)
	log.Fatal(server.ListenAndServe())
}
