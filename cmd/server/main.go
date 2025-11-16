package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
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

	defer conn.Close()
	fmt.Println("Connection was successful")
	amqp_chan, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
		return
	}
	gamelogic.PrintServerHelp()

	for {
		str := gamelogic.GetInput()
		if len(str) == 0 {
			continue
		}
		switch str[0] {
		case "pause":
			fmt.Println("Input is Pause")
			playingstate_param := &routing.PlayingState{
				IsPaused: true,
			}

			if err := pubsub.PublishJSON(amqp_chan, routing.ExchangePerilDirect, routing.PauseKey, *playingstate_param); err != nil {
				fmt.Println(err)
				return
			}
		case "resume":
			fmt.Println("Input is resume")
			playingstate_param := &routing.PlayingState{
				IsPaused: false,
			}

			if err := pubsub.PublishJSON(amqp_chan, routing.ExchangePerilDirect, routing.PauseKey, *playingstate_param); err != nil {
				fmt.Println(err)
				return
			}
		case "quit":
			fmt.Println("Input is quit")
		default:
			fmt.Println("Command not understood")
		}
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		<-signalChan
	}

}
