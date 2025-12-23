package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/iRootPro/weather/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Weather API - Coming Soon")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	log.Printf("Starting API server on %s", cfg.HTTP.Addr())
	if err := http.ListenAndServe(cfg.HTTP.Addr(), nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
		os.Exit(1)
	}
}
