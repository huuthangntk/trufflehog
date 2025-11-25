package queue

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/trufflesecurity/trufflehog/v3/pkg/api/db"
	"github.com/trufflesecurity/trufflehog/v3/pkg/api/webhooks"
)

type Worker struct {
	id            int
	queue         *RabbitMQQueue
	database      *db.Database
	webhookMgr    *webhooks.WebhookManager
	maxConcurrent int
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

type WorkerPool struct {
	workers []*Worker
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

func NewWorker(id int, queue *RabbitMQQueue, database *db.Database, webhookMgr *webhooks.WebhookManager) *Worker {
	maxConcurrent := 5
	if val := os.Getenv("MAX_CONCURRENT_SCANS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			maxConcurrent = n
		}
	}

	return &Worker{
		id:            id,
		queue:         queue,
		database:      database,
		webhookMgr:    webhookMgr,
		maxConcurrent: maxConcurrent,
		stopCh:        make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

func (w *Worker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()

	log.Printf("Worker %d started", w.id)

	for {
		select {
		case <-w.stopCh:
			log.Printf("Worker %d stopping", w.id)
			return
		case <-ctx.Done():
			log.Printf("Worker %d context cancelled", w.id)
			return
		default:
			job, err := w.queue.DequeueScanJob(ctx, 5*time.Second)
			if err != nil {
				log.Printf("Worker %d: error dequeuing job: %v", w.id, err)
				time.Sleep(1 * time.Second)
				continue
			}

			if job == nil {
				continue
			}

			log.Printf("Worker %d: processing job %s", w.id, job.JobID)
			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *JobMessage) {
	jobID := job.JobID

	// Publish status update to RabbitMQ (GitScout will handle DB update)
	w.queue.PublishScanStatus(ctx, &ScanStatusMessage{
		JobID:    jobID,
		RepoURL:  job.RepoURL,
		Status:   "running",
		Progress: 0,
	})

	// Trigger scan.started webhook
	w.webhookMgr.TriggerEvent(ctx, "scan.started", map[string]interface{}{
		"job_id":   jobID.String(),
		"repo_url": job.RepoURL,
		"status":   "running",
	}, &jobID)

	// Execute scan
	scanMetrics, err := w.executeScan(ctx, job)

	if err != nil {
		errMsg := err.Error()
		// Publish failed status to RabbitMQ
		w.queue.PublishScanStatus(ctx, &ScanStatusMessage{
			JobID:    jobID,
			RepoURL:  job.RepoURL,
			Status:   "failed",
			Progress: 100,
			ErrorMsg: errMsg,
		})

		// Trigger scan.failed webhook
		w.webhookMgr.TriggerEvent(ctx, "scan.failed", map[string]interface{}{
			"job_id":   jobID.String(),
			"repo_url": job.RepoURL,
			"status":   "failed",
			"error":    errMsg,
		}, &jobID)

		log.Printf("Worker %d: job %s failed: %v", w.id, jobID, err)
		return
	}

	// Publish completed status to RabbitMQ with metrics
	w.queue.PublishScanStatus(ctx, &ScanStatusMessage{
		JobID:             jobID,
		RepoURL:           job.RepoURL,
		Status:            "completed",
		Progress:          100,
		ChunksScanned:     scanMetrics.ChunksScanned,
		BytesScanned:      scanMetrics.BytesScanned,
		SecretsFound:      scanMetrics.SecretsFound,
		VerifiedSecrets:   scanMetrics.VerifiedSecrets,
		UnverifiedSecrets: scanMetrics.UnverifiedSecrets,
	})

	// Trigger scan.completed webhook
	w.webhookMgr.TriggerEvent(ctx, "scan.completed", map[string]interface{}{
		"job_id":             jobID.String(),
		"repo_url":           job.RepoURL,
		"status":             "completed",
		"chunks_scanned":     scanMetrics.ChunksScanned,
		"bytes_scanned":      scanMetrics.BytesScanned,
		"secrets_found":      scanMetrics.SecretsFound,
		"verified_secrets":   scanMetrics.VerifiedSecrets,
		"unverified_secrets": scanMetrics.UnverifiedSecrets,
	}, &jobID)

	log.Printf("Worker %d: job %s completed successfully", w.id, jobID)
}

// ScanMetrics holds the metrics from a scan execution
type ScanMetrics struct {
	ChunksScanned     int64
	BytesScanned      int64
	SecretsFound      int
	VerifiedSecrets   int
	UnverifiedSecrets int
}

func (w *Worker) executeScan(ctx context.Context, job *JobMessage) (*ScanMetrics, error) {
	// Find trufflehog binary
	trufflehogBin, err := findTrufflehogBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find trufflehog binary: %w", err)
	}

	// Create temporary directory for results
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("trufflehog-scan-%s", job.JobID.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build command arguments
	args := []string{
		"git",
		job.RepoURL,
		"--json",
		"--no-update",
	}

	// Add optional parameters
	if branch := getStringOption(job.Options, "branch", ""); branch != "" {
		args = append(args, "--branch", branch)
	}
	if sinceCommit := getStringOption(job.Options, "since_commit", ""); sinceCommit != "" {
		args = append(args, "--since-commit", sinceCommit)
	}
	if maxDepth := getIntOption(job.Options, "max_depth", 0); maxDepth > 0 {
		args = append(args, "--max-depth", strconv.Itoa(maxDepth))
	}
	if getBoolOption(job.Options, "no_verification", false) {
		args = append(args, "--no-verification")
	}
	if getBoolOption(job.Options, "only_verified", false) {
		args = append(args, "--only-verified")
	}

	// Execute trufflehog
	cmd := exec.CommandContext(ctx, trufflehogBin, args...)
	
	// Capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start trufflehog: %w", err)
	}

	// Process results in real-time
	scanner := bufio.NewScanner(stdout)
	metrics := &ScanMetrics{}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON result
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			log.Printf("Failed to parse result line: %v", err)
			continue
		}

		// Parse and publish result to RabbitMQ (GitScout will handle storage)
		scanResult := w.parseResultMessage(job.JobID, job.RepoURL, result)
		if err := w.queue.PublishScanResult(ctx, scanResult); err != nil {
			log.Printf("Failed to publish scan result: %v", err)
		}

		// Count results
		metrics.SecretsFound++
		if scanResult.Verified {
			metrics.VerifiedSecrets++
		} else {
			metrics.UnverifiedSecrets++
		}

		// Publish progress update periodically
		if metrics.SecretsFound%10 == 0 {
			w.queue.PublishScanStatus(ctx, &ScanStatusMessage{
				JobID:           job.JobID,
				RepoURL:         job.RepoURL,
				Status:          "running",
				Progress:        50, // Approximate progress
				SecretsFound:    metrics.SecretsFound,
				VerifiedSecrets: metrics.VerifiedSecrets,
			})
		}
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		// Check if it's just a non-zero exit (which is normal if secrets found)
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 183 means secrets found, which is not an error
			if exitErr.ExitCode() != 183 {
				return nil, fmt.Errorf("trufflehog exited with error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("trufflehog execution error: %w", err)
		}
	}

	return metrics, nil
}

func findTrufflehogBinary() (string, error) {
	// Check common locations
	locations := []string{
		"/opt/trufflehog/trufflehog",           // Deployed location
		"/root/trufflehog/trufflehog",          // Build directory
		"/usr/local/bin/trufflehog",            // System install
		"/usr/bin/trufflehog",                  // System install
		"./trufflehog",                         // Current directory
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	// Try PATH
	path, err := exec.LookPath("trufflehog")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("trufflehog binary not found in common locations")
}

func (w *Worker) parseResultMessage(jobID uuid.UUID, repoURL string, result map[string]interface{}) *ScanResultMessage {
	msg := &ScanResultMessage{
		ID:             uuid.New(),
		JobID:          jobID,
		RepoURL:        repoURL,
		RawResult:      result,
		SourceMetadata: make(map[string]interface{}),
	}

	// Extract common fields
	if detectorName, ok := result["DetectorName"].(string); ok {
		msg.DetectorName = detectorName
	}
	if detectorType, ok := result["DetectorType"].(float64); ok {
		msg.DetectorType = fmt.Sprintf("%d", int(detectorType))
	}
	if verified, ok := result["Verified"].(bool); ok {
		msg.Verified = verified
	}
	if raw, ok := result["Raw"].(string); ok {
		msg.Secret = raw
		msg.RedactedSecret = redactSecret(raw)
	}

	// Extract source metadata
	if sourceMetadata, ok := result["SourceMetadata"].(map[string]interface{}); ok {
		msg.SourceMetadata = sourceMetadata
		
		// Extract specific fields
		if data, ok := sourceMetadata["Data"].(map[string]interface{}); ok {
			if git, ok := data["Git"].(map[string]interface{}); ok {
				if commit, ok := git["commit"].(string); ok {
					msg.CommitHash = commit
				}
				if file, ok := git["file"].(string); ok {
					msg.FilePath = file
				}
				if line, ok := git["line"].(float64); ok {
					msg.LineNumber = int(line)
				}
			}
		}
	}

	// Generate fingerprint for deduplication (same as GitScout's logic)
	msg.Fingerprint = generateFingerprint(repoURL, msg.FilePath, msg.DetectorName, msg.Secret)

	return msg
}

// generateFingerprint creates a unique fingerprint for deduplication
func generateFingerprint(repoURL, filePath, detectorName, secret string) string {
	data := fmt.Sprintf("%s:%s:%s:%s", repoURL, filePath, detectorName, secret)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func redactSecret(secret string) string {
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}

func getStringOption(options map[string]interface{}, key, defaultValue string) string {
	if val, ok := options[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getIntOption(options map[string]interface{}, key string, defaultValue int) int {
	if val, ok := options[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

func getBoolOption(options map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := options[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// WorkerPool management

func NewWorkerPool(numWorkers int, queue *RabbitMQQueue, database *db.Database, webhookMgr *webhooks.WebhookManager) *WorkerPool {
	pool := &WorkerPool{
		workers: make([]*Worker, numWorkers),
		stopCh:  make(chan struct{}),
	}

	for i := 0; i < numWorkers; i++ {
		pool.workers[i] = NewWorker(i+1, queue, database, webhookMgr)
	}

	return pool
}

func (p *WorkerPool) Start(ctx context.Context) {
	for _, worker := range p.workers {
		worker.Start(ctx)
	}
}

func (p *WorkerPool) Stop() {
	for _, worker := range p.workers {
		worker.Stop()
	}
}

