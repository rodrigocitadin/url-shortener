package repository

import (
	"context"
	"encoding/json"
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
	_, _ = ch.QueueDeclare(
		"urls_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

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

	return r.channel.PublishWithContext(ctx,
		"",          // exchange
		r.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		})
}

func (r *queueURLRepository) Find(shortCode string) (*entities.URLEntity, error) {
	return r.fallback.Find(shortCode)
}
