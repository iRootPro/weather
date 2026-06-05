package api

import (
	"net/http"

	"github.com/iRootPro/weather/internal/service"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GET /api/dashboard/snapshot
func (h *DashboardHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	if h.dashboardService == nil {
		http.Error(w, "Dashboard service not configured", http.StatusServiceUnavailable)
		return
	}

	snapshot, err := h.dashboardService.GetSnapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, snapshot)
}
