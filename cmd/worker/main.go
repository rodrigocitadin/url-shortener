package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgconn"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rodrigocitadin/url-shortener/internal/entities"
	"github.com/rodrigocitadin/url-shortener/internal/repository"
)

func main() {
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

	_, err = ch.QueueDeclare(
		"urls_queue", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		log.Fatal(err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := ch.Consume(
		"urls_queue", // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Worker started. Waiting for messages...")

	go func() {
		for d := range msgs {
			var entity entities.URLEntity
			if err := json.Unmarshal(d.Body, &entity); err != nil {
				log.Printf("Error decoding JSON: %v", err)
				d.Nack(false, false) // DQL in the future
				continue
			}

			db := sm.GetShard(entity.Shortcode)
			repo := repository.NewDatabaseURLRepository(db)

			log.Printf("Processing shortcode: %s", entity.Shortcode)

			err := repo.Save(&entity)
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					log.Printf(
						"Duplicate Key detected for '%s'. Discarding message to avoid loop.",
						entity.Shortcode,
					)
					d.Ack(false)
				} else {
					log.Printf("Error saving to DB: %v. Requeueing...", err)
					d.Nack(false, true) // requeue
				}
			} else {
				d.Ack(false)
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Worker shutting down...")
}
