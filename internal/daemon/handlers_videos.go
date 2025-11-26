package daemon

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// handleVideos godoc
// @Summary List or register videos
// @Description GET lists tracked videos; POST registers a new video for extraction.
// @Tags videos
// @Accept json
// @Produce json
// @Param request body AddVideoRequest true "Video to register"
// @Success 200 {array} Video
// @Success 200 {object} AddVideoResponse
// @Failure 400 {object} ErrorResponse
// @Router /videos [get]
// @Router /videos [post]
func (s *Server) handleVideos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		list := make([]Video, 0, len(s.videos))
		for _, v := range s.videos {
			copyVideo := *v
			list = append(list, copyVideo)
		}
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json payload")
			return
		}
		if strings.TrimSpace(req.Path) == "" {
			writeError(w, http.StatusBadRequest, "path is required")
			return
		}
		s.mu.Lock()
		if id, exists := s.videoByPath[req.Path]; exists {
			s.mu.Unlock()
			writeJSON(w, http.StatusOK, map[string]string{
				"video_id": id,
				"status":   "already_exists",
			})
			return
		}
		videoID := newID("vid_")
		video := &Video{
			ID:          videoID,
			Path:        req.Path,
			IndexStatus: "pending",
		}
		s.videos[videoID] = video
		s.videoByPath[req.Path] = videoID
		s.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]string{
			"video_id": videoID,
			"status":   "scheduled",
		})
	}
}

// handleGetVideo godoc
// @Summary Get video details
// @Description Returns stored metadata and indexing status for a video.
// @Tags videos
// @Produce json
// @Param videoID path string true "Video ID"
// @Success 200 {object} Video
// @Failure 404 {object} ErrorResponse
// @Router /videos/{videoID} [get]
func (s *Server) handleGetVideo(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "videoID")
	s.mu.RLock()
	video, ok := s.videos[videoID]
	if ok {
		copyVideo := *video
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, copyVideo)
		return
	}
	s.mu.RUnlock()
	writeError(w, http.StatusNotFound, "video not found")
}

// handleExtract godoc
// @Summary Start extraction job
// @Description Starts an extraction and upload job for the given video.
// @Tags videos
// @Accept json
// @Produce json
// @Param videoID path string true "Video ID"
// @Param request body ExtractRequest false "Extraction options"
// @Success 200 {object} StartJobResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /videos/{videoID}/extract [post]
func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "videoID")
	var req struct {
		Reindex bool `json:"reindex"`
	}
	_ = decodeJSON(r, &req)
	job, err := s.startJob(videoID, req.Reindex)
	if err != nil {
		if errors.Is(err, errNotFound) {
			writeError(w, http.StatusNotFound, "video not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started", "job_id": job.ID})
}

// handleCancel godoc
// @Summary Cancel extraction job
// @Description Attempts to cancel an active job for the given video.
// @Tags videos
// @Produce json
// @Param videoID path string true "Video ID"
// @Success 200 {object} CancelJobResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /videos/{videoID}/cancel [post]
func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "videoID")
	if err := s.cancelJob(videoID); err != nil {
		if errors.Is(err, errNotFound) {
			writeError(w, http.StatusNotFound, "video not found or no active job")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelling"})
}

// handleVideoFile streams a registered video's file contents.
func (s *Server) handleVideoFile(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "videoID")
	s.mu.RLock()
	video, ok := s.videos[videoID]
	s.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	if strings.TrimSpace(video.Path) == "" {
		writeError(w, http.StatusNotFound, "video path missing")
		return
	}

	http.ServeFile(w, r, video.Path)
}
