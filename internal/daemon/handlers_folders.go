package daemon

import (
	"net/http"
	"strings"
)

// handleFolders godoc
// @Summary Track a folder for video discovery
// @Description Registers a folder path for scanning and returns its tracking ID.
// @Tags folders
// @Accept json
// @Produce json
// @Param request body AddFolderRequest true "Folder to track"
// @Success 200 {object} AddFolderResponse
// @Failure 400 {object} ErrorResponse
// @Router /folders [post]
func (s *Server) handleFolders(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
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
	if id, exists := s.folderByPath[req.Path]; exists {
		f := s.folders[id]
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{
			"folder_id": f.ID,
			"status":    "already_exists",
		})
		return
	}
	folderID := newID("fld_")
	folder := Folder{
		ID:        folderID,
		Path:      req.Path,
		Recursive: req.Recursive,
		Status:    "scheduled",
	}
	s.folders[folderID] = folder
	s.folderByPath[req.Path] = folderID
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{
		"folder_id": folderID,
		"status":    "scheduled",
	})
}
