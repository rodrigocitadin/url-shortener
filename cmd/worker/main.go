package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rodrigocitadin/url-shortener/internal/entities"
	"github.com/rodrigocitadin/url-shortener/internal/repository"
)

var maxRetries = 3

var (
	jobsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "worker_jobs_processed_total",
		Help: "Total jobs processed by the worker",
	}, []string{"status", "shard"}) // status: success, error, dlq

	jobDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "worker_job_duration_seconds",
		Help:    "Time taken to process a job",
		Buckets: prometheus.DefBuckets, // Adicionado buckets padrão
	})
)

func isInfrastructureError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "failed to connect")
}

func getRetryCount(d amqp.Delivery) int {
	if d.Headers == nil {
		return 0
	}
	if v, ok := d.Headers["x-retry-count"]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int32:
			return int(val)
		case int64:
			return int(val)
		}
	}
	return 0
}

func setupQueues(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	// DLX (Dead Letter Exchange)
	err := ch.ExchangeDeclare("urls_dlx", "fanout", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// DLQ (Dead Letter Queue)
	qDlq, err := ch.QueueDeclare("urls_dlq", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare dlq: %w", err)
	}

	err = ch.QueueBind(qDlq.Name, "", "urls_dlx", false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind dlq: %w", err)
	}

	// Main queue with DLX
	args := amqp.Table{"x-dead-letter-exchange": "urls_dlx"}
	_, err = ch.QueueDeclare("urls_queue", true, false, false, false, args)
	if err != nil {
		return nil, fmt.Errorf("failed to declare main queue: %w", err)
	}

	// Backpressure
	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to define Qos: %w", err)
	}

	msgs, err := ch.Consume("urls_queue", "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to consume: %w", err)
	}

	return msgs, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	maxRetriesEnv := os.Getenv("MAX_RETRIES")
	if maxRetriesEnv != "" {
		num, err := strconv.Atoi(maxRetriesEnv)
		if err == nil {
			maxRetries = num
		}
	}

	dsnsEnv := os.Getenv("SHARD_DSNS")
	if dsnsEnv == "" {
		slog.Error("Shards env not filled", "error", errors.New("Undefined shards env"))
		os.Exit(1)
	}

	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		slog.Error("RabbitMQ env not filled", "error", errors.New("Undefined RABBITMQ_URL env"))
		os.Exit(1)
	}

	sm, err := repository.NewShardManager(strings.Split(dsnsEnv, ","))
	if err != nil {
		slog.Error("Failed to connect to Shards", "error", err)
		os.Exit(1)
	}

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		slog.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		slog.Error("Failed to create amqp channel", "error", err)
		os.Exit(1)
	}
	defer ch.Close()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		slog.Info("Metrics server listening", "port", 2112)
		if err := http.ListenAndServe(":2112", nil); err != nil {
			slog.Error("Metrics server failed", "error", err)
		}
	}()

	msgs, err := setupQueues(ch)
	if err != nil {
		slog.Error("Failed to setup amqp queues", "error", err)
		os.Exit(1)
	}

	slog.Info("Worker started. Waiting for messages...", "max_retries", maxRetries)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			go processMessage(d, sm, ch)
		}
		forever <- true
	}()

	<-stop
	slog.Info("Worker shutting down...")
}

func processMessage(d amqp.Delivery, sm *repository.ShardManager, ch *amqp.Channel) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		jobDuration.Observe(duration)
	}()

	var entity entities.URLEntity
	if err := json.Unmarshal(d.Body, &entity); err != nil {
		slog.Error("Error decoding JSON: Sending to DLQ.", "error", err)
		d.Nack(false, false)
		return
	}

	shardIdx := sm.GetShardIndex(entity.Shortcode)
	shardLabel := "shard-" + strconv.Itoa(shardIdx)

	db := sm.GetShard(shardIdx)
	repo := repository.NewDatabaseURLRepository(db)

	slog.Info("Processing shortcode", "shortcode", entity.Shortcode, "shard", shardLabel)

	err := repo.Save(&entity)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			slog.Error("Duplicated key detected", "shortcode", entity.Shortcode)
			d.Ack(false)
			jobsProcessed.WithLabelValues("success", shardLabel).Inc()
			return
		}

		currentRetries := getRetryCount(d)
		if currentRetries >= maxRetries {
			slog.Warn(
				"Max Retries was reachedl. Sending it to the DLQ",
				"shortcode", entity.Shortcode, "retries", currentRetries)
			d.Nack(false, false)
			jobsProcessed.WithLabelValues("dlq", shardLabel).Inc()
			return
		}

		slog.Error(
			"Error with message",
			"shortcode", entity.Shortcode,
			"error", err,
			"retries", currentRetries+1,
			"shard", shardLabel,
		)

		if isInfrastructureError(err) {
			slog.Error("Critical Infrastructure Error. Sleeping to avoid overload", "error", err)
			time.Sleep(5 * time.Second)
			d.Nack(false, true)
			return
		}

		newHeaders := d.Headers
		if newHeaders == nil {
			newHeaders = make(amqp.Table)
		}
		newHeaders["x-retry-count"] = currentRetries + 1

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		errPub := ch.PublishWithContext(ctx,
			"",
			d.RoutingKey,
			false,
			false,
			amqp.Publishing{
				Headers:      newHeaders,
				ContentType:  d.ContentType,
				Body:         d.Body,
				DeliveryMode: d.DeliveryMode,
			},
		)

		if errPub != nil {
			d.Nack(false, true) // Fallback
			slog.Error("Error to republish retry: Giving Nack(true) as fallback", "error", errPub)
		} else {
			d.Ack(false)
		}

		jobsProcessed.WithLabelValues("retry", shardLabel).Inc()
	} else {
		d.Ack(false)
		slog.Info("Successfully processed", "shortcode", entity.Shortcode, "shard", shardLabel)
		jobsProcessed.WithLabelValues("success", shardLabel).Inc()
	}
}
