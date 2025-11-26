package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VectorDBClient wraps simple HTTP calls to the vectordb service.
type VectorDBClient struct {
	baseURL string
	http    *http.Client
}

type UploadImageRequest struct {
	FilePath    string
	VideoID     string
	VideoPath   string
	FrameNumber int
	Timestamp   float64
	FrameRate   float64
}

func NewVectorDBClient(baseURL string) *VectorDBClient {
	return &VectorDBClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// UploadImage sends a single frame/image to the vectordb /upload_image endpoint.
func (c *VectorDBClient) UploadImage(ctx context.Context, req UploadImageRequest) (string, error) {
	if c == nil {
		return "", fmt.Errorf("vectordb client not configured")
	}
	if c.baseURL == "" {
		return "", fmt.Errorf("vectordb base URL is empty")
	}
	if strings.TrimSpace(req.FilePath) == "" {
		return "", fmt.Errorf("file path is required")
	}

	file, err := os.Open(req.FilePath)
	if err != nil {
		return "", fmt.Errorf("open frame: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(req.FilePath))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copy frame: %w", err)
	}
	if req.VideoID != "" {
		_ = writer.WriteField("video_id", req.VideoID)
		_ = writer.WriteField("video_path", req.VideoPath)
		if req.FrameNumber > 0 {
			_ = writer.WriteField("frame_number", strconv.Itoa(req.FrameNumber))
		}
		if req.FrameRate > 0 {
			_ = writer.WriteField("frame_rate", strconv.FormatFloat(req.FrameRate, 'f', -1, 64))
		}
		if req.Timestamp >= 0 {
			_ = writer.WriteField("timestamp", strconv.FormatFloat(req.Timestamp, 'f', -1, 64))
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("finalize form: %w", err)
	}

	endpoint := c.baseURL + "/upload_image"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("vectordb request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vectordb upload failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode vectordb response: %w", err)
	}
	if payload.ID == "" {
		return "", fmt.Errorf("vectordb returned empty id")
	}
	return payload.ID, nil
}

type vectorTimestamp struct {
	Start          float64 `json:"start"`
	End            float64 `json:"end"`
	RelevanceScore float64 `json:"relevance_score"`
}

type vectorVideoResult struct {
	VideoID           string            `json:"video_id"`
	VideoPath         string            `json:"video_path"`
	Timestamps        []vectorTimestamp `json:"timestamps"`
	MaxRelevanceScore float64           `json:"max_relevance_score"`
}

func (c *VectorDBClient) SearchVideos(ctx context.Context, query string, topK int, clusterThreshold float64) ([]vectorVideoResult, error) {
	if c == nil {
		return nil, fmt.Errorf("vectordb client not configured")
	}
	if strings.TrimSpace(c.baseURL) == "" {
		return nil, fmt.Errorf("vectordb base URL is empty")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if topK <= 0 {
		topK = 5
	}
	if clusterThreshold <= 0 {
		clusterThreshold = 5.0
	}

	endpoint := c.baseURL + "/search_video"
	body := map[string]interface{}{
		"query":             query,
		"top_k":             topK,
		"cluster_threshold": clusterThreshold,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vectordb request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vectordb search failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var payload struct {
		Results []vectorVideoResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode vectordb response: %w", err)
	}

	return payload.Results, nil
}
