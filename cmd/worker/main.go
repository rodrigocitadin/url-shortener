package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rodrigocitadin/url-shortener/internal/entities"
	"github.com/rodrigocitadin/url-shortener/internal/repository"
)

var maxRetries = 3

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

func setupQueues(ch *amqp.Channel) <-chan amqp.Delivery {
	err := ch.ExchangeDeclare("urls_dlx", "fanout", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	qDlq, err := ch.QueueDeclare("urls_dlq", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = ch.QueueBind(qDlq.Name, "", "urls_dlx", false, nil)
	if err != nil {
		log.Fatal(err)
	}

	args := amqp.Table{"x-dead-letter-exchange": "urls_dlx"}
	_, err = ch.QueueDeclare("urls_queue", true, false, false, false, args)
	if err != nil {
		log.Fatal("Error declaring the main queue:", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := ch.Consume("urls_queue", "", false, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	return msgs
}

func main() {
	maxRetriesEnv := os.Getenv("MAX_RETRIES")
	if maxRetriesEnv != "" {
		num, err := strconv.Atoi(maxRetriesEnv)
		if err == nil {
			maxRetries = num
		}
	}

	dsnsEnv := os.Getenv("SHARD_DSNS")
	if dsnsEnv == "" {
		log.Fatal("SHARD_DSNS env not defined")
	}

	sm, err := repository.NewShardManager(strings.Split(dsnsEnv, ","))
	if err != nil {
		log.Fatal(err)
	}

	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	msgs := setupQueues(ch)

	log.Printf("Worker started, MaxRetries=%d. Waiting for messages...", maxRetries)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for d := range msgs {
			go processMessage(d, sm, ch)
		}
	}()

	<-stop
	log.Println("Worker shutting down...")
}

func processMessage(d amqp.Delivery, sm *repository.ShardManager, ch *amqp.Channel) {
	var entity entities.URLEntity
	if err := json.Unmarshal(d.Body, &entity); err != nil {
		log.Printf("Error decoding JSON: %v. Sending to DLQ.", err)
		d.Nack(false, false)
		return
	}

	db := sm.GetShard(entity.Shortcode)
	repo := repository.NewDatabaseURLRepository(db)

	log.Printf("Processing shortcode: %s", entity.Shortcode)

	err := repo.Save(&entity)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			log.Printf("Duplicated key (%s). Ack.", entity.Shortcode)
			d.Ack(false)
			return
		}

		currentRetries := getRetryCount(d)
		if currentRetries >= maxRetries {
			log.Printf("Max Retries (%d) was reached for %s. Sending it to the DLQ.", maxRetries, entity.Shortcode)
			d.Nack(false, false)
			return
		}

		log.Printf("Error (%s): %v. Attempt %d/%d.", entity.Shortcode, err, currentRetries+1, maxRetries)

		if isInfrastructureError(err) {
			log.Printf("CRITICAL DB ERROR: %v. Database seems down.", err)
			log.Println("Worker sleeping for 5s to avoid overload...")
			time.Sleep(10 * time.Second)
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
			log.Printf("Error retrying: %v. Giving Nack(true) as a replacement", errPub)
			d.Nack(false, true) // Fallback
		} else {
			d.Ack(false)
		}

	} else {
		log.Printf("Success processing: %s", entity.Shortcode)
		d.Ack(false)
	}
}
