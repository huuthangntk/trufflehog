package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/engine"
	"github.com/trufflesecurity/trufflehog/v3/pkg/sources"
	"github.com/trufflesecurity/trufflehog/v3/pkg/sources/git"
)

type ScanRequest struct {
	RepoURL     string   `json:"repo_url"`
	WebhookURL  string   `json:"webhook_url"`
	Verify      bool     `json:"verify"`
	IncludeOnly []string `json:"include_only,omitempty"`
}

type ScanResponse struct {
	ScanID    string `json:"scan_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type ScanResult struct {
	ScanID       string              `json:"scan_id"`
	Status       string              `json:"status"`
	RepoURL      string              `json:"repo_url"`
	StartedAt    string              `json:"started_at"`
	CompletedAt  string              `json:"completed_at,omitempty"`
	TotalSecrets int                 `json:"total_secrets"`
	Verified     int                 `json:"verified"`
	Unverified   int                 `json:"unverified"`
	Secrets      []SecretResult      `json:"secrets,omitempty"`
	Error        string              `json:"error,omitempty"`
}

type SecretResult struct {
	DetectorType string            `json:"detector_type"`
	DetectorName string            `json:"detector_name"`
	Verified     bool              `json:"verified"`
	Raw          string            `json:"raw,omitempty"`
	Redacted     string            `json:"redacted"`
	ExtraData    map[string]string `json:"extra_data,omitempty"`
	SourceName   string            `json:"source_name"`
	SourceType   string            `json:"source_type"`
}

type WebhookPayload struct {
	Event      string     `json:"event"`
	ScanResult ScanResult `json:"scan_result"`
	Timestamp  string     `json:"timestamp"`
}

type Server struct {
	engine        *engine.Engine
	scans         map[string]*ScanResult
	scansMutex    sync.RWMutex
	webhookClient *http.Client
}

func NewServer() (*Server, error) {
	e, err := engine.Start(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start engine: %w", err)
	}

	return &Server{
		engine:        e,
		scans:         make(map[string]*ScanResult),
		webhookClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *Server) HandleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.RepoURL == "" {
		http.Error(w, "repo_url is required", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	scanResult := &ScanResult{
		ScanID:    scanID,
		Status:    "pending",
		RepoURL:   req.RepoURL,
		StartedAt: now,
	}

	s.scansMutex.Lock()
	s.scans[scanID] = scanResult
	s.scansMutex.Unlock()

	go s.performScan(scanID, req)

	response := ScanResponse{
		ScanID:    scanID,
		Status:    "pending",
		Message:   "Scan initiated successfully",
		CreatedAt: now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleGetScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scanID := r.URL.Query().Get("scan_id")
	if scanID == "" {
		http.Error(w, "scan_id parameter is required", http.StatusBadRequest)
		return
	}

	s.scansMutex.RLock()
	scanResult, exists := s.scans[scanID]
	s.scansMutex.RUnlock()

	if !exists {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scanResult)
}

func (s *Server) HandleListScans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.scansMutex.RLock()
	scans := make([]*ScanResult, 0, len(s.scans))
	for _, scan := range s.scans {
		scans = append(scans, scan)
	}
	s.scansMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scans": scans,
		"total": len(scans),
	})
}

func (s *Server) performScan(scanID string, req ScanRequest) {
	s.scansMutex.Lock()
	scanResult := s.scans[scanID]
	scanResult.Status = "running"
	s.scansMutex.Unlock()

	ctx := context.Background()
	
	gitSource := &git.Source{}
	conn, err := gitSource.Init(ctx, "trufflehog-api", 0, 0, req.Verify)
	if err != nil {
		s.updateScanError(scanID, fmt.Sprintf("Failed to initialize git source: %v", err))
		s.sendWebhook(req.WebhookURL, scanID)
		return
	}

	if err := conn.SetSourceUnit(ctx, sources.SourceUnit{
		ID:   scanID,
		Kind: "git",
	}); err != nil {
		s.updateScanError(scanID, fmt.Sprintf("Failed to set source unit: %v", err))
		s.sendWebhook(req.WebhookURL, scanID)
		return
	}

	var secrets []SecretResult
	resultsChan := make(chan detectors.Result, 100)
	
	go func() {
		for result := range resultsChan {
			secret := SecretResult{
				DetectorType: result.DetectorType.String(),
				DetectorName: result.DetectorName,
				Verified:     result.Verified,
				Redacted:     result.Redacted,
				ExtraData:    result.ExtraData,
			}
			
			if result.SourceMetadata != nil {
				secret.SourceName = result.SourceMetadata.GetData().GetGit().GetRepository()
				secret.SourceType = "git"
			}
			
			secrets = append(secrets, secret)
		}
	}()

	// Note: This is a simplified version. In production, you'd integrate with the actual engine
	// and properly handle the scanning process
	close(resultsChan)

	s.scansMutex.Lock()
	scanResult.Status = "completed"
	scanResult.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	scanResult.Secrets = secrets
	scanResult.TotalSecrets = len(secrets)
	
	for _, secret := range secrets {
		if secret.Verified {
			scanResult.Verified++
		} else {
			scanResult.Unverified++
		}
	}
	s.scansMutex.Unlock()

	s.sendWebhook(req.WebhookURL, scanID)
}

func (s *Server) updateScanError(scanID, errorMsg string) {
	s.scansMutex.Lock()
	defer s.scansMutex.Unlock()
	
	if scanResult, exists := s.scans[scanID]; exists {
		scanResult.Status = "failed"
		scanResult.Error = errorMsg
		scanResult.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	}
}

func (s *Server) sendWebhook(webhookURL, scanID string) {
	if webhookURL == "" {
		return
	}

	s.scansMutex.RLock()
	scanResult := s.scans[scanID]
	s.scansMutex.RUnlock()

	payload := WebhookPayload{
		Event:      "scan.completed",
		ScanResult: *scanResult,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, nil)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TruffleHog-API/1.0")
	req.Header.Set("X-TruffleHog-Event", "scan.completed")
	
	resp, err := s.webhookClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/api/v1/scan", s.HandleScan)
	mux.HandleFunc("/api/v1/scan/status", s.HandleGetScan)
	mux.HandleFunc("/api/v1/scans", s.HandleListScans)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}
