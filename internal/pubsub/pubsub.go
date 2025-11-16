package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	json_bytes, err := json.Marshal(val)
	if err != nil {
		return err
	}

	params := &amqp.Publishing{
		ContentType: "application/json",
		Body:        json_bytes,
	}

	ctx := context.Background()
	if err := ch.PublishWithContext(ctx, exchange, key, false, false, *params); err != nil {
		return err
	}
	return nil
}

type SimpleQueueType string

const (
	Durable   SimpleQueueType = "durable"
	Transient SimpleQueueType = "transient"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	amqp_chan, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	var amqp_queue amqp.Queue

	switch queueType {
	case Transient:
		amqp_queue, err = amqp_chan.QueueDeclare(queueName, false, true, true, false, nil)
		if err != nil {
			return nil, amqp.Queue{}, err
		}
	case Durable:
		amqp_queue, err = amqp_chan.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			return nil, amqp.Queue{}, err
		}
	}
	if err := amqp_chan.QueueBind(queueName, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, err
	}

	return amqp_chan, amqp_queue, err
}
