package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	conn_string := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(conn_string)
	if err != nil {
		fmt.Println(err)
		return
	}
	amqp_chan, err := conn.Channel()
	playingstate_param := &routing.PlayingState{
		IsPaused: true,
	}

	if err := pubsub.PublishJSON(amqp_chan, routing.ExchangePerilDirect, routing.PauseKey, *playingstate_param); err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()
	fmt.Println("Connection was successful")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
