package daemon

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		list := make([]Folder, 0, len(s.folders))
		for _, f := range s.folders {
			list = append(list, f)
		}
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
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
			Status:    "scanning",
		}
		s.folders[folderID] = folder
		s.folderByPath[req.Path] = folderID
		s.mu.Unlock()

		go s.scanFolderAndIndex(folderID, req.Path, req.Recursive)

		writeJSON(w, http.StatusOK, map[string]string{
			"folder_id": folderID,
			"status":    "scanning",
		})
	}
}

func (s *Server) scanFolderAndIndex(folderID, root string, recursive bool) {
	videoPaths, err := collectVideoPaths(root, recursive)
	if err != nil {
		log.Printf("scan folder %s failed: %v", root, err)
		s.mu.Lock()
		if f, ok := s.folders[folderID]; ok {
			f.Status = "error"
			s.folders[folderID] = f
		}
		s.mu.Unlock()
		return
	}

	for _, vp := range videoPaths {
		id, exists, err := s.addVideoPath(vp)
		if err != nil {
			log.Printf("add video %s failed: %v", vp, err)
			continue
		}
		if !exists {
			if _, err := s.startJob(id, false); err != nil {
				log.Printf("start job for %s failed: %v", id, err)
			}
		}
	}

	s.mu.Lock()
	if f, ok := s.folders[folderID]; ok {
		f.Status = "scanned"
		s.folders[folderID] = f
	}
	s.mu.Unlock()
}

func collectVideoPaths(root string, recursive bool) ([]string, error) {
	if !recursive {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		paths := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if isVideoFileName(e.Name()) {
				paths = append(paths, filepath.Join(root, e.Name()))
			}
		}
		return paths, nil
	}

	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if isVideoFileName(d.Name()) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

func isVideoFileName(name string) bool {
	lower := strings.ToLower(name)
	switch filepath.Ext(lower) {
	case ".mp4", ".mov", ".mkv", ".avi", ".m4v", ".webm":
		return true
	default:
		return false
	}
}

func (s *Server) addVideoPath(path string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.videoByPath[path]; ok {
		return id, true, nil
	}
	videoID := newID("vid_")
	video := &Video{
		ID:          videoID,
		Path:        path,
		IndexStatus: "pending",
	}
	s.videos[videoID] = video
	s.videoByPath[path] = videoID
	return videoID, false, nil
}
