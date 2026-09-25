// weather-demo runs a loopback-only UI preview with synthetic weather data.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/iRootPro/weather/internal/handler/web"
)

func main() {
	demo, err := web.NewAttentionDemo("internal/web/templates")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /", demo)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/web/static"))))
	server := &http.Server{Addr: "127.0.0.1:8091", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Print("Weather UI demo: http://127.0.0.1:8091")
	log.Fatal(server.ListenAndServe())
}
