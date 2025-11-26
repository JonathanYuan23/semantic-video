package daemon

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"semanticvideo/internal/extract"
)

// startJob schedules a new simulated extraction job for a video.
func (s *Server) startJob(videoID string, reindex bool) (*Job, error) {
	s.mu.Lock()
	video, ok := s.videos[videoID]
	if !ok {
		s.mu.Unlock()
		return nil, errNotFound
	}
	if reindex {
		video.FramesExtracted = 0
		video.FramesUploaded = 0
		video.TotalFramesExpected = 0
		video.IndexStatus = "pending"
		video.LastError = nil
	}
	video.FramesExtracted = 0
	video.FramesUploaded = 0
	video.TotalFramesExpected = 0
	video.IndexStatus = "extracting"
	video.LastError = nil

	jobID := newID("job_")
	now := time.Now().UTC()
	job := &Job{
		ID:        jobID,
		VideoID:   videoID,
		Type:      "extract_and_upload",
		Status:    "queued",
		Progress:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.jobs[jobID] = job
	cancelCh := make(chan struct{})
	s.jobCancel[jobID] = cancelCh
	s.mu.Unlock()

	go s.runJob(jobID, cancelCh)
	return job, nil
}

// cancelJob attempts to stop a running job and mark the video as failed.
func (s *Server) cancelJob(videoID string) error {
	s.mu.Lock()
	var jobID string
	for id, job := range s.jobs {
		if job.VideoID == videoID && (job.Status == "running" || job.Status == "queued") {
			jobID = id
			break
		}
	}
	if jobID == "" {
		s.mu.Unlock()
		return errNotFound
	}
	cancelCh := s.jobCancel[jobID]
	close(cancelCh)
	now := time.Now().UTC()
	job := s.jobs[jobID]
	job.Status = "failed"
	job.UpdatedAt = now
	if v, ok := s.videos[videoID]; ok {
		v.IndexStatus = "failed"
		msg := "cancelled"
		v.LastError = &msg
	}
	delete(s.jobCancel, jobID)
	s.mu.Unlock()
	return nil
}

// runJob simulates extraction and upload progress over time until completion or cancellation.
func (s *Server) runJob(jobID string, cancelCh <-chan struct{}) {
	defer func() {
		s.mu.Lock()
		delete(s.jobCancel, jobID)
		s.mu.Unlock()
	}()

	s.mu.Lock()
	job, jobExists := s.jobs[jobID]
	if !jobExists {
		s.mu.Unlock()
		return
	}
	video, videoExists := s.videos[job.VideoID]
	if !videoExists {
		s.mu.Unlock()
		return
	}
	videoPath := video.Path
	framesDir := s.framesDirForVideo(videoPath)
	cfg := extract.Config{
		FrameRate: s.config.FrameRate,
		FrameSize: s.config.FrameSize,
	}
	expected := video.TotalFramesExpected
	if expected == 0 {
		expected = s.estimateFrames(video)
	}
	if expected <= 0 {
		expected = 1
	}
	video.TotalFramesExpected = expected
	now := time.Now().UTC()
	job.Status = "running"
	job.Progress = 0
	job.UpdatedAt = now
	video.IndexStatus = "extracting"
	video.LastError = nil
	s.mu.Unlock()

	if err := os.MkdirAll(s.framesRoot, 0o755); err != nil {
		s.failJob(jobID, fmt.Errorf("prepare frames root: %w", err))
		return
	}
	_ = os.RemoveAll(framesDir)

	errCh := make(chan error, 1)
	go func(path string) {
		errCh <- extract.ExtractFramesForVideo(path, s.framesRoot, cfg)
	}(videoPath)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cancelCh:
			s.markJobCancelled(jobID)
			return
		case err := <-errCh:
			if err != nil {
				s.failJob(jobID, fmt.Errorf("extract frames: %w", err))
				return
			}
			if err := s.indexFrames(jobID, framesDir, cancelCh); err != nil {
				if errors.Is(err, context.Canceled) {
					s.markJobCancelled(jobID)
				} else {
					s.failJob(jobID, fmt.Errorf("index frames: %w", err))
				}
				return
			}
			s.completeJob(jobID)
			return
		case <-ticker.C:
			if err := s.refreshJobProgress(jobID, framesDir); err != nil {
				s.failJob(jobID, fmt.Errorf("monitor frames: %w", err))
				return
			}
		}
	}
}

// estimateFrames returns a rough frame count based on duration and configured frame rate.
func (s *Server) estimateFrames(video *Video) int {
	expected := int(math.Ceil(float64(video.DurationSeconds) * s.config.FrameRate))
	return expected
}

func (s *Server) framesDirForVideo(videoPath string) string {
	base := filepath.Base(videoPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(s.framesRoot, name)
}

func (s *Server) refreshJobProgress(jobID, framesDir string) error {
	frames, err := countExtractedFrames(framesDir)
	if err != nil {
		return err
	}
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	video := s.videos[job.VideoID]
	if frames > video.FramesExtracted {
		video.FramesExtracted = frames
	}
	if frames > video.TotalFramesExpected {
		video.TotalFramesExpected = frames
	}
	s.updateProgressLocked(video, job, true)
	job.UpdatedAt = time.Now().UTC()
	video.IndexStatus = "extracting"
	s.mu.Unlock()
	return nil
}

func (s *Server) indexFrames(jobID, framesDir string, cancelCh <-chan struct{}) error {
	framePaths, err := listFrameFiles(framesDir)
	if err != nil {
		return err
	}
	if len(framePaths) == 0 {
		return fmt.Errorf("no frames extracted")
	}
	if s.vectorClient == nil {
		return fmt.Errorf("vectordb client missing")
	}

	s.mu.Lock()
	job := s.jobs[jobID]
	video := s.videos[job.VideoID]
	video.IndexStatus = "indexing"
	video.FramesUploaded = 0
	video.FramesExtracted = len(framePaths)
	video.TotalFramesExpected = len(framePaths)
	s.updateProgressLocked(video, job, true)
	s.mu.Unlock()

	for i, framePath := range framePaths {
		select {
		case <-cancelCh:
			return context.Canceled
		default:
		}

		frameNumber := i + 1
		timestamp := float64(frameNumber-1) / s.config.FrameRate
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		_, err := s.vectorClient.UploadImage(ctx, UploadImageRequest{
			FilePath:    framePath,
			VideoID:     video.ID,
			VideoPath:   video.Path,
			FrameNumber: frameNumber,
			Timestamp:   timestamp,
			FrameRate:   s.config.FrameRate,
		})
		cancel()
		if err != nil {
			return err
		}

		s.markFrameUploaded(jobID, i+1, len(framePaths))
	}

	return nil
}

func (s *Server) markFrameUploaded(jobID string, uploaded, total int) {
	now := time.Now().UTC()
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return
	}
	video := s.videos[job.VideoID]
	if uploaded > video.FramesUploaded {
		video.FramesUploaded = uploaded
	}
	if total > video.TotalFramesExpected {
		video.TotalFramesExpected = total
	}
	if video.FramesExtracted < total {
		video.FramesExtracted = total
	}
	s.updateProgressLocked(video, job, true)
	job.UpdatedAt = now
	video.IndexStatus = "indexing"
	s.mu.Unlock()
}

func (s *Server) updateProgressLocked(video *Video, job *Job, clamp bool) {
	expected := video.TotalFramesExpected
	if expected <= 0 {
		expected = video.FramesUploaded
		if expected <= 0 {
			expected = 1
		}
		video.TotalFramesExpected = expected
	}
	progress := float64(video.FramesUploaded) / float64(expected)
	if clamp && progress >= 1 && (job == nil || job.Status != "done") {
		progress = math.Nextafter(1, 0)
	}
	if job != nil && progress > job.Progress {
		job.Progress = progress
	}
}

func (s *Server) completeJob(jobID string) {
	now := time.Now().UTC()
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return
	}
	video := s.videos[job.VideoID]
	if video.FramesExtracted < video.FramesUploaded {
		video.FramesExtracted = video.FramesUploaded
	}
	if video.TotalFramesExpected < video.FramesUploaded {
		video.TotalFramesExpected = video.FramesUploaded
	}
	job.Status = "done"
	job.Progress = 1
	job.UpdatedAt = now
	video.IndexStatus = "indexed"
	video.LastError = nil
	video.LastIndexedAt = &now
	s.cloud.Status.Connected = s.cloud.AccessToken != ""
	s.cloud.Status.PendingBatches = 0
	s.cloud.Status.LastSuccessfulUpload = &now
	s.mu.Unlock()
}

func (s *Server) failJob(jobID string, err error) {
	msg := err.Error()
	now := time.Now().UTC()
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return
	}
	job.Status = "failed"
	job.Progress = 0
	job.UpdatedAt = now
	if video, exists := s.videos[job.VideoID]; exists {
		video.IndexStatus = "failed"
		video.LastError = &msg
	}
	s.mu.Unlock()
}

func (s *Server) markJobCancelled(jobID string) {
	now := time.Now().UTC()
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if ok {
		job.Status = "failed"
		job.Progress = 0
		job.UpdatedAt = now
	}
	if job != nil {
		if video, exists := s.videos[job.VideoID]; exists {
			video.IndexStatus = "failed"
			msg := "cancelled"
			video.LastError = &msg
		}
	}
	s.mu.Unlock()
}

func countExtractedFrames(dir string) (int, error) {
	frames, err := listFrameFiles(dir)
	if err != nil {
		return 0, err
	}
	return len(frames), nil
}

func listFrameFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	frames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "frame_") && strings.HasSuffix(name, ".jpg") {
			frames = append(frames, filepath.Join(dir, name))
		}
	}
	sort.Strings(frames)
	return frames, nil
}
