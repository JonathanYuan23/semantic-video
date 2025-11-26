package daemon

import (
	"net/http"
)

// handleSearch proxies text queries to the vectordb service and returns timestamped results.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Query            string  `json:"query"`
		TopK             int     `json:"top_k"`
		ClusterThreshold float64 `json:"cluster_threshold"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	if req.TopK <= 0 {
		req.TopK = 5
	}
	if req.ClusterThreshold <= 0 {
		req.ClusterThreshold = 5.0
	}

	client := s.vectorClient
	if client == nil {
		writeError(w, http.StatusServiceUnavailable, "vectordb client not configured")
		return
	}

	results, err := client.SearchVideos(r.Context(), req.Query, req.TopK, req.ClusterThreshold)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}
