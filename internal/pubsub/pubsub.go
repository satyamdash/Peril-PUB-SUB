package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

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
	queueType SimpleQueueType,
	amqp_table amqp.Table, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	amqp_chan, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	var amqp_queue amqp.Queue

	switch queueType {
	case Transient:
		amqp_queue, err = amqp_chan.QueueDeclare(queueName, false, true, true, false, amqp_table)
		if err != nil {
			return nil, amqp.Queue{}, err
		}
	case Durable:
		amqp_queue, err = amqp_chan.QueueDeclare(queueName, true, false, false, false, amqp_table)
		if err != nil {
			return nil, amqp.Queue{}, err
		}
	}
	if err := amqp_chan.QueueBind(amqp_queue.Name, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, err
	}

	return amqp_chan, amqp_queue, err
}

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	args := amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	}
	amqp_chan, amqp_queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType, args)
	if err != nil {
		return err
	}
	amqp_delievery, err := amqp_chan.Consume(amqp_queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	var val T
	go func() {
		for delievery := range amqp_delievery {

			json.Unmarshal(delievery.Body, &val)
			ack := handler(val)
			switch ack {
			case Ack:
				fmt.Println("Ack Recived")
				delievery.Ack(false)
			case NackDiscard:
				fmt.Println("NAck Recived")
				delievery.Nack(false, false)
			case NackRequeue:
				fmt.Println("NAck requeq Recived")
				delievery.Nack(false, true)
			}
		}
	}()

	return nil

}
