package daemon

import (
	"errors"
	"time"
)

// Config holds global frame extraction and upload settings.
type Config struct {
	FrameRate       float64 `json:"frame_rate" example:"1.0"`
	FrameSize       [2]int  `json:"frame_size" swaggertype:"array,integer" example:"400,400"`
	UploadBatchSize int     `json:"upload_batch_size" example:"50"`
	CloudBaseURL    string  `json:"cloud_base_url" example:"https://api.example.com"`
	CloudUserID     string  `json:"cloud_user_id" example:"user_123"`
	CloudAuthStatus string  `json:"cloud_auth_status" example:"missing_token"`
}

// Folder represents a tracked folder to scan for videos.
type Folder struct {
	ID        string `json:"folder_id" example:"fld_abcd1234"`
	Path      string `json:"path" example:"/videos"`
	Recursive bool   `json:"recursive" example:"true"`
	Status    string `json:"status" example:"scheduled"`
}

// Video tracks a single video and its extraction progress.
type Video struct {
	ID                  string     `json:"video_id" example:"vid_abcd1234"`
	Path                string     `json:"path" example:"/videos/sample.mp4"`
	DurationSeconds     int        `json:"duration_seconds,omitempty" example:"120"`
	IndexStatus         string     `json:"index_status" example:"indexing"`
	FramesExtracted     int        `json:"frames_extracted" example:"80"`
	FramesUploaded      int        `json:"frames_uploaded" example:"80"`
	TotalFramesExpected int        `json:"total_frames_expected" example:"120"`
	LastIndexedAt       *time.Time `json:"last_indexed_at" example:"2024-01-01T12:00:00Z"`
	LastError           *string    `json:"last_error" example:"failed to decode frame"`
}

// Job represents a simulated extraction and upload job.
type Job struct {
	ID        string    `json:"job_id" example:"job_abcd1234"`
	VideoID   string    `json:"video_id" example:"vid_abcd1234"`
	Type      string    `json:"type" example:"extract_and_upload"`
	Status    string    `json:"status" example:"running"`
	Progress  float64   `json:"progress" example:"0.42"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-01T12:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2024-01-01T12:05:00Z"`
}

type CloudStatus struct {
	UserID               string     `json:"user_id" example:"user_123"`
	Connected            bool       `json:"connected" example:"true"`
	LastSuccessfulUpload *time.Time `json:"last_successful_upload" example:"2024-01-01T12:10:00Z"`
	PendingBatches       int        `json:"pending_batches" example:"0"`
}

type CloudState struct {
	AccessToken string
	Status      CloudStatus
}

// ErrorResponse represents a standard error payload.
type ErrorResponse struct {
	Error string `json:"error" example:"description of the error"`
}

// HealthResponse describes the health endpoint payload.
type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Version string `json:"version" example:"0.1.0"`
}

// ConfigUpdateRequest allows partial configuration updates.
type ConfigUpdateRequest struct {
	FrameRate       *float64 `json:"frame_rate" example:"2.0"`
	FrameSize       *[2]int  `json:"frame_size" swaggertype:"array,integer" example:"640,480"`
	UploadBatchSize *int     `json:"upload_batch_size" example:"100"`
	CloudBaseURL    *string  `json:"cloud_base_url" example:"https://api.example.com"`
}

// StatusResponse is a generic status wrapper.
type StatusResponse struct {
	Status string `json:"status" example:"ok"`
}

// AddFolderRequest is the payload to track a folder.
type AddFolderRequest struct {
	Path      string `json:"path" example:"/videos"`
	Recursive bool   `json:"recursive" example:"true"`
}

// AddFolderResponse returns the tracked folder ID.
type AddFolderResponse struct {
	FolderID string `json:"folder_id" example:"fld_abcd1234"`
	Status   string `json:"status" example:"scheduled"`
}

// AddVideoRequest registers a new video for extraction.
type AddVideoRequest struct {
	Path string `json:"path" example:"/videos/sample.mp4"`
}

// AddVideoResponse returns the created video ID.
type AddVideoResponse struct {
	VideoID string `json:"video_id" example:"vid_abcd1234"`
	Status  string `json:"status" example:"scheduled"`
}

// ExtractRequest toggles whether to reindex a video.
type ExtractRequest struct {
	Reindex bool `json:"reindex" example:"false"`
}

// StartJobResponse provides the started job ID.
type StartJobResponse struct {
	Status string `json:"status" example:"started"`
	JobID  string `json:"job_id" example:"job_abcd1234"`
}

// CancelJobResponse indicates a cancellation attempt.
type CancelJobResponse struct {
	Status string `json:"status" example:"cancelling"`
}

// CloudAuthRequest is the payload to store an access token.
type CloudAuthRequest struct {
	AccessToken string `json:"access_token" example:"token_abc123"`
}

var errNotFound = errors.New("not found")
