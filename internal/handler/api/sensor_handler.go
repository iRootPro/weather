package api

import (
	"net/http"

	"github.com/iRootPro/weather/internal/service"
)

type SensorHandler struct {
	sensorService *service.SensorService
}

func NewSensorHandler(sensorService *service.SensorService) *SensorHandler {
	return &SensorHandler{sensorService: sensorService}
}

// GET /api/sensors
func (h *SensorHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	sensors, err := h.sensorService.GetAll(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, sensors)
}

// GET /api/sensors/{code}
func (h *SensorHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		http.Error(w, "sensor code is required", http.StatusBadRequest)
		return
	}

	sensor, err := h.sensorService.GetByCode(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, sensor)
}
