package repository

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rodrigocitadin/url-shortener/internal/entities"
)

type queueURLRepository struct {
	channel   *amqp.Channel
	queueName string
	fallback  URLRepository
}

func NewQueueURLRepository(ch *amqp.Channel, fallback URLRepository) URLRepository {
	_, err := ch.QueueDeclare(
		"urls_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{"x-dead-letter-exchange": "urls_dlx"},
	)

	if err != nil {
		log.Fatal("Error declaring queue", err)
	}

	return &queueURLRepository{
		channel:   ch,
		queueName: "urls_queue",
		fallback:  fallback,
	}
}

func (r *queueURLRepository) Save(urlEntity *entities.URLEntity) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(urlEntity)
	if err != nil {
		return err
	}

	err = r.channel.PublishWithContext(ctx, "", r.queueName, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	})

	if err != nil {
		log.Printf("RabbitMQ error: %v. Using Fallback to DB.", err)
		return r.fallback.Save(urlEntity)
	}

	return nil
}

func (r *queueURLRepository) Find(shortCode string) (*entities.URLEntity, error) {
	return r.fallback.Find(shortCode)
}
