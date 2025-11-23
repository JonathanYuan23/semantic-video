package daemon

import (
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Server stores all in-memory state and exposes HTTP handlers.
type Server struct {
	mu           sync.RWMutex
	config       Config
	folders      map[string]Folder
	videos       map[string]*Video
	jobs         map[string]*Job
	jobCancel    map[string]chan struct{}
	folderByPath map[string]string
	videoByPath  map[string]string
	cloud        CloudState
	framesRoot   string
}

func NewServer() *Server {
	cfg := Config{
		FrameRate:       1.0,
		FrameSize:       [2]int{400, 400},
		UploadBatchSize: 50,
		CloudBaseURL:    "https://api.example.com",
		CloudUserID:     "user_123",
		CloudAuthStatus: "missing_token",
	}

	framesRoot := os.Getenv("FRAMES_ROOT")
	if framesRoot == "" {
		framesRoot = "frames"
	}

	return &Server{
		config:       cfg,
		folders:      make(map[string]Folder),
		videos:       make(map[string]*Video),
		jobs:         make(map[string]*Job),
		jobCancel:    make(map[string]chan struct{}),
		folderByPath: make(map[string]string),
		videoByPath:  make(map[string]string),
		cloud: CloudState{
			Status: CloudStatus{
				UserID: cfg.CloudUserID,
			},
		},
		framesRoot: framesRoot,
	}
}

// Routes returns the HTTP handler for all endpoints.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	// Swagger docs
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Config and health
	r.Get("/health", s.handleHealth)
	r.MethodFunc(http.MethodGet, "/config", s.handleConfig)
	r.MethodFunc(http.MethodPut, "/config", s.handleConfig)

	// Folders
	r.MethodFunc(http.MethodPost, "/folders", s.handleFolders)

	// Videos
	r.MethodFunc(http.MethodGet, "/videos", s.handleVideos)
	r.MethodFunc(http.MethodPost, "/videos", s.handleVideos)
	r.Route("/videos/{videoID}", func(r chi.Router) {
		r.MethodFunc(http.MethodGet, "/", s.handleGetVideo)
		r.MethodFunc(http.MethodPost, "/extract", s.handleExtract)
		r.MethodFunc(http.MethodPost, "/cancel", s.handleCancel)
	})

	// Jobs
	r.MethodFunc(http.MethodGet, "/jobs", s.handleJobs)

	// Cloud
	r.MethodFunc(http.MethodGet, "/cloud/status", s.handleCloudStatus)
	r.MethodFunc(http.MethodPost, "/cloud/auth", s.handleCloudAuth)

	return r
}
