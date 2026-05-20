package api

import (
	"net/http"
	"time"

	"github.com/iRootPro/weather/internal/service"
)

type HydroHandler struct {
	hydroService *service.HydroService
}

func NewHydroHandler(hydroService *service.HydroService) *HydroHandler {
	return &HydroHandler{hydroService: hydroService}
}

func (h *HydroHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	if h.hydroService == nil {
		http.Error(w, "Hydro service not configured", http.StatusServiceUnavailable)
		return
	}
	snap, err := h.hydroService.GetSnapshot(r.Context(), time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, snap)
}

func (h *HydroHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if h.hydroService == nil {
		http.Error(w, "Hydro service not configured", http.StatusServiceUnavailable)
		return
	}
	from, to, err := parseTimeRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data, err := h.hydroService.GetRange(r.Context(), from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, data)
}
