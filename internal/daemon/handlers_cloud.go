package daemon

import (
	"net/http"
	"strings"
)

// handleCloudStatus godoc
// @Summary Get cloud status
// @Description Returns cloud connectivity status and pending upload information.
// @Tags cloud
// @Produce json
// @Success 200 {object} CloudStatus
// @Router /cloud/status [get]
func (s *Server) handleCloudStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	status := s.cloud.Status
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, status)
}

// handleCloudAuth godoc
// @Summary Store cloud access token
// @Description Saves an access token used for cloud uploads and marks the connection as active.
// @Tags cloud
// @Accept json
// @Produce json
// @Param request body CloudAuthRequest true "Access token"
// @Success 200 {object} StatusResponse
// @Failure 400 {object} ErrorResponse
// @Router /cloud/auth [post]
func (s *Server) handleCloudAuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccessToken string `json:"access_token"`
	}
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.AccessToken) == "" {
		writeError(w, http.StatusBadRequest, "access_token is required")
		return
	}

	s.mu.Lock()
	s.cloud.AccessToken = req.AccessToken
	s.config.CloudAuthStatus = "ok"
	s.cloud.Status.Connected = true
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
