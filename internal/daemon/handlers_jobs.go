package daemon

import "net/http"

// handleJobs godoc
// @Summary List jobs
// @Description Returns all extraction and upload jobs with progress.
// @Tags jobs
// @Produce json
// @Success 200 {array} Job
// @Router /jobs [get]
func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	list := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		copyJob := *j
		list = append(list, copyJob)
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, list)
}
