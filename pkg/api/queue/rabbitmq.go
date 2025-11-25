package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/google/uuid"
)

const (
	ScanJobQueue        = "trufflehog_scan_jobs"
	WebhookQueue        = "trufflehog_webhook_deliveries"
	// New queues for GitScout integration - TruffleHog publishes, GitScout consumes
	ScanStatusQueue     = "scan_status_updates"
	ScanResultsQueue    = "scan_results_queue"
	// Queue for GitScout to request scans - pure queue-based flow
	ScanRequestQueue    = "scan_request_queue"
)

type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

type JobMessage struct {
	JobID     uuid.UUID              `json:"job_id"`
	RepoURL   string                 `json:"repo_url"`
	Options   map[string]interface{} `json:"options"`
	CreatedAt time.Time              `json:"created_at"`
}

func NewRabbitMQQueue() (*RabbitMQQueue, error) {
	url := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare queues
	_, err = ch.QueueDeclare(
		ScanJobQueue, // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare scan job queue: %w", err)
	}

	_, err = ch.QueueDeclare(
		WebhookQueue, // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare webhook queue: %w", err)
	}

	// Priority arguments - must match GitScout's queue configuration
	priorityArgs := amqp.Table{
		"x-max-priority": int32(4), // Support 4 priority levels
	}

	// Declare scan status updates queue (TruffleHog publishes, GitScout consumes)
	_, err = ch.QueueDeclare(
		ScanStatusQueue, // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		priorityArgs,    // arguments - must match GitScout's config
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare scan status queue: %w", err)
	}

	// Declare scan results queue (TruffleHog publishes, GitScout consumes)
	_, err = ch.QueueDeclare(
		ScanResultsQueue, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		priorityArgs,     // arguments - must match GitScout's config
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare scan results queue: %w", err)
	}

	// Declare scan request queue (GitScout publishes, TruffleHog consumes)
	_, err = ch.QueueDeclare(
		ScanRequestQueue, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare scan request queue: %w", err)
	}

	return &RabbitMQQueue{
		conn:    conn,
		channel: ch,
	}, nil
}

func (q *RabbitMQQueue) Close() error {
	if q.channel != nil {
		q.channel.Close()
	}
	if q.conn != nil {
		q.conn.Close()
	}
	return nil
}

func (q *RabbitMQQueue) EnqueueScanJob(ctx context.Context, job *JobMessage) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	err = q.channel.PublishWithContext(ctx,
		"",           // exchange
		ScanJobQueue, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent, // make message persistent
		})
	if err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	return nil
}

func (q *RabbitMQQueue) DequeueScanJob(ctx context.Context, timeout time.Duration) (*JobMessage, error) {
	msgs, err := q.channel.Consume(
		ScanJobQueue, // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register consumer: %w", err)
	}

	// Create a channel to receive the message
	msgChan := make(chan amqp.Delivery, 1)

	// Start a goroutine to handle the message
	go func() {
		select {
		case msg := <-msgs:
			msgChan <- msg
		case <-time.After(timeout):
			msgChan <- amqp.Delivery{} // empty delivery to indicate timeout
		case <-ctx.Done():
			msgChan <- amqp.Delivery{} // empty delivery to indicate cancellation
		}
	}()

	// Wait for either a message or timeout
	msg := <-msgChan

	// If we got an empty delivery, it means timeout or cancellation
	if msg.Body == nil {
		return nil, nil
	}

	var job JobMessage
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		msg.Nack(false, true) // negative acknowledge, requeue
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Acknowledge the message
	if err := msg.Ack(false); err != nil {
		log.Printf("Failed to acknowledge message: %v", err)
	}

	return &job, nil
}

func (q *RabbitMQQueue) EnqueueWebhook(ctx context.Context, webhookData map[string]interface{}) error {
	data, err := json.Marshal(webhookData)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook data: %w", err)
	}

	err = q.channel.PublishWithContext(ctx,
		"",            // exchange
		WebhookQueue,  // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent, // make message persistent
		})
	if err != nil {
		return fmt.Errorf("failed to publish webhook: %w", err)
	}

	return nil
}

func (q *RabbitMQQueue) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	queue, err := q.channel.QueueInspect(queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect queue: %w", err)
	}
	return int64(queue.Messages), nil
}

// ScanStatusMessage represents a scan job status update published to GitScout
type ScanStatusMessage struct {
	JobID       uuid.UUID `json:"job_id"`
	RepoURL     string    `json:"repo_url"`
	Status      string    `json:"status"` // queued, running, completed, failed, cancelled
	Progress    int       `json:"progress"`
	ErrorMsg    string    `json:"error_message,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	// Metrics (populated on completion)
	ChunksScanned     int64 `json:"chunks_scanned,omitempty"`
	BytesScanned      int64 `json:"bytes_scanned,omitempty"`
	SecretsFound      int   `json:"secrets_found,omitempty"`
	VerifiedSecrets   int   `json:"verified_secrets,omitempty"`
	UnverifiedSecrets int   `json:"unverified_secrets,omitempty"`
}

// ScanResultMessage represents a single scan result (secret) published to GitScout
type ScanResultMessage struct {
	ID              uuid.UUID              `json:"id"`
	JobID           uuid.UUID              `json:"job_id"`
	RepoURL         string                 `json:"repo_url"`
	DetectorType    string                 `json:"detector_type"`
	DetectorName    string                 `json:"detector_name"`
	Secret          string                 `json:"secret"`
	RedactedSecret  string                 `json:"redacted_secret"`
	Verified        bool                   `json:"verified"`
	VerifyError     string                 `json:"verification_error,omitempty"`
	FilePath        string                 `json:"file_path,omitempty"`
	LineNumber      int                    `json:"line_number,omitempty"`
	CommitHash      string                 `json:"commit_hash,omitempty"`
	Fingerprint     string                 `json:"fingerprint"`
	RawResult       map[string]interface{} `json:"raw_result"`
	SourceMetadata  map[string]interface{} `json:"source_metadata"`
	Timestamp       time.Time              `json:"timestamp"`
}

// ScanRequestMessage represents a scan request from GitScout (pure queue-based flow)
type ScanRequestMessage struct {
	JobID     uuid.UUID              `json:"job_id"`
	RepoURL   string                 `json:"repo_url"`
	Options   map[string]interface{} `json:"options"`
	Priority  int                    `json:"priority"`
	CreatedAt time.Time              `json:"created_at"`
}

// PublishScanStatus publishes a scan status update to GitScout
func (q *RabbitMQQueue) PublishScanStatus(ctx context.Context, status *ScanStatusMessage) error {
	status.Timestamp = time.Now()
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	err = q.channel.PublishWithContext(ctx,
		"",              // exchange
		ScanStatusQueue, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		return fmt.Errorf("failed to publish scan status: %w", err)
	}

	log.Printf("Published scan status: job=%s status=%s progress=%d", status.JobID, status.Status, status.Progress)
	return nil
}

// PublishScanResult publishes a scan result (secret) to GitScout
func (q *RabbitMQQueue) PublishScanResult(ctx context.Context, result *ScanResultMessage) error {
	result.Timestamp = time.Now()
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	err = q.channel.PublishWithContext(ctx,
		"",               // exchange
		ScanResultsQueue, // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		return fmt.Errorf("failed to publish scan result: %w", err)
	}

	log.Printf("Published scan result: job=%s detector=%s verified=%t", result.JobID, result.DetectorName, result.Verified)
	return nil
}

// ConsumeScanRequests starts consuming scan requests from GitScout
func (q *RabbitMQQueue) ConsumeScanRequests(ctx context.Context, handler func(*ScanRequestMessage) error) error {
	msgs, err := q.channel.Consume(
		ScanRequestQueue, // queue
		"",               // consumer
		false,            // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	if err != nil {
		return fmt.Errorf("failed to register scan request consumer: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				var request ScanRequestMessage
				if err := json.Unmarshal(msg.Body, &request); err != nil {
					log.Printf("Failed to unmarshal scan request: %v", err)
					msg.Nack(false, false) // discard invalid message
					continue
				}

				if err := handler(&request); err != nil {
					log.Printf("Failed to handle scan request: %v", err)
					msg.Nack(false, true) // requeue for retry
					continue
				}

				msg.Ack(false)
			}
		}
	}()

	return nil
}

// Note: Status tracking methods are now handled via RabbitMQ messages to GitScout
// GitScout is the single source of truth for all database operations

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
