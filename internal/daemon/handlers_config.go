package daemon

import (
	"net/http"
)

// handleHealth godoc
// @Summary Health check
// @Description Returns service health and version.
// @Tags system
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": Version,
	})
}

// handleConfig godoc
// @Summary Get or update configuration
// @Description Returns the current configuration on GET and updates selected fields on PUT.
// @Tags config
// @Accept json
// @Produce json
// @Param request body ConfigUpdateRequest false "Fields to update (PUT only)"
// @Success 200 {object} Config
// @Success 200 {object} StatusResponse "Update acknowledgment"
// @Failure 400 {object} ErrorResponse
// @Router /config [get]
// @Router /config [put]
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		cfg := s.config
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var req struct {
			FrameRate       *float64 `json:"frame_rate"`
			FrameSize       *[2]int  `json:"frame_size"`
			UploadBatchSize *int     `json:"upload_batch_size"`
			CloudBaseURL    *string  `json:"cloud_base_url"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json payload")
			return
		}
		s.mu.Lock()
		if req.FrameRate != nil {
			s.config.FrameRate = *req.FrameRate
		}
		if req.FrameSize != nil {
			s.config.FrameSize = *req.FrameSize
		}
		if req.UploadBatchSize != nil {
			s.config.UploadBatchSize = *req.UploadBatchSize
		}
		if req.CloudBaseURL != nil {
			s.config.CloudBaseURL = *req.CloudBaseURL
		}
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
