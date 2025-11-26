package daemon

import (
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
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
	vectorClient *VectorDBClient
	stateless    bool
	cleanupDirs  []string
	cleanupOnce  sync.Once
}

func NewServer() *Server {
	cfg := Config{
		FrameRate:       1.0,
		FrameSize:       [2]int{384, 384},
		UploadBatchSize: 50,
		CloudBaseURL:    "https://api.example.com",
		CloudUserID:     "user_123",
		CloudAuthStatus: "missing_token",
		VectorDBURL:     "http://localhost:8000",
		Stateless:       false,
	}

	framesRoot := os.Getenv("FRAMES_ROOT")
	stateless := os.Getenv("STATELESS_TEST") == "1" || os.Getenv("STATELESS_MODE") == "1"
	if stateless {
		if tmp, err := os.MkdirTemp("", "semantic-video-frames-"); err == nil {
			framesRoot = tmp
		}
	}

	if framesRoot == "" {
		framesRoot = "frames"
	}

	vectorURL := os.Getenv("VECTORDB_URL")
	if vectorURL == "" {
		vectorURL = cfg.VectorDBURL
	}

	cleanupDirs := []string{}
	if stateless {
		cleanupDirs = append(cleanupDirs, framesRoot)
	}

	cfg.VectorDBURL = vectorURL
	cfg.Stateless = stateless

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
		framesRoot:   framesRoot,
		vectorClient: NewVectorDBClient(vectorURL),
		stateless:    stateless,
		cleanupDirs:  cleanupDirs,
	}
}

// Routes returns the HTTP handler for all endpoints.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	// Logging
	r.Use(logRequestMiddleware)

	// CORS to allow local client
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

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
	r.MethodFunc(http.MethodGet, "/folders", s.handleFolders)
	r.MethodFunc(http.MethodPost, "/folders", s.handleFolders)

	// Videos
	r.MethodFunc(http.MethodGet, "/videos", s.handleVideos)
	r.MethodFunc(http.MethodPost, "/videos", s.handleVideos)
	r.Route("/videos/{videoID}", func(r chi.Router) {
		r.MethodFunc(http.MethodGet, "/", s.handleGetVideo)
		r.MethodFunc(http.MethodPost, "/extract", s.handleExtract)
		r.MethodFunc(http.MethodPost, "/cancel", s.handleCancel)
		r.MethodFunc(http.MethodGet, "/file", s.handleVideoFile)
	})

	// Jobs
	r.MethodFunc(http.MethodGet, "/jobs", s.handleJobs)

	// Search proxy
	r.MethodFunc(http.MethodPost, "/search", s.handleSearch)
	r.MethodFunc(http.MethodPost, "/search_video", s.handleSearch)

	// Cloud
	r.MethodFunc(http.MethodGet, "/cloud/status", s.handleCloudStatus)
	r.MethodFunc(http.MethodPost, "/cloud/auth", s.handleCloudAuth)

	return r
}

// Cleanup removes temporary data when stateless mode is enabled.
func (s *Server) Cleanup() {
	if !s.stateless {
		return
	}
	s.cleanupOnce.Do(func() {
		for _, dir := range s.cleanupDirs {
			_ = os.RemoveAll(dir)
		}
	})
}
