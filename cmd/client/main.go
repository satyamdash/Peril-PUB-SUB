package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	conn_string := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(conn_string)
	if err != nil {
		fmt.Println(err)
		return
	}

	amqp_chan, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()
	fmt.Println("Connection was successful")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println(err)
		return
	}
	// args := amqp.Table{
	// 	"x-dead-letter-exchange": "peril_dlx",
	// }

	// pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", pubsub.Durable, nil)
	pubsub.SubscribeGob(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", pubsub.Durable, handlerLog())
	gamestate := gamelogic.NewGameState(username)
	fmt.Println(gamestate.GetUsername())

	if err := pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "army_moves"+"."+username, "army_moves.*", pubsub.Transient, handlerMove(gamestate, amqp_chan)); err != nil {
		fmt.Println(err)
		return
	}
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		<-signalChan

	}()
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "war", routing.WarRecognitionsPrefix+".*", pubsub.Durable, handlerWar(gamestate, amqp_chan))

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			gamestate.CommandSpawn(words)
		case "move":
			armymove, err := gamestate.CommandMove(words)
			if err != nil {
				return
			}
			pubsub.PublishJSON(amqp_chan, routing.ExchangePerilTopic, "army_moves"+"."+username, armymove)
			fmt.Println("move was published successfully")
		case "war":

		case "status":
			gamestate.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
		default:
			fmt.Println(fmt.Errorf("command invalid"))
			continue
		}

	}

}

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(mov gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		moveOutcome, _ := gs.HandleMove(mov)
		if moveOutcome == gamelogic.MoveOutcomeMakeWar {
			warMsg := gamelogic.RecognitionOfWar{
				Attacker: mov.Player,
				Defender: gs.Player,
			}
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix+"."+gs.Player.Username, warMsg); err != nil {
				return pubsub.NackRequeue
			}
		}
		return pubsub.Ack
	}
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		waroutcome, winner, loser := gs.HandleWar(rw)
		switch waroutcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			msg := winner + "won a war against" + loser
			gl := &routing.GameLog{
				CurrentTime: time.Now(),
				Message:     msg,
				Username:    gs.Player.Username,
			}
			PublishGamelog(ch, routing.ExchangePerilTopic, gs.Player.Username, *gl)
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			msg := winner + "won a war against" + loser
			gl := &routing.GameLog{
				CurrentTime: time.Now(),
				Message:     msg,
				Username:    gs.Player.Username,
			}
			PublishGamelog(ch, routing.ExchangePerilTopic, gs.Player.Username, *gl)
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			msg := "A war between" + winner + "and" + loser + "resulted in a draw"
			gl := &routing.GameLog{
				CurrentTime: time.Now(),
				Message:     msg,
				Username:    gs.Player.Username,
			}
			PublishGamelog(ch, routing.ExchangePerilTopic, gs.Player.Username, *gl)
			return pubsub.Ack
		default:
			fmt.Println(fmt.Errorf("errror"))
			return pubsub.NackDiscard
		}
	}
}

func handlerLog() func(routing.GameLog) pubsub.AckType {
	return func(gl routing.GameLog) pubsub.AckType {
		defer fmt.Print("> ")

		if err := gamelogic.WriteLog(gl); err != nil {
			fmt.Println("failed to write log:", err)
			return pubsub.NackRequeue
		}

		return pubsub.Ack
	}
}

func PublishGamelog(ch *amqp.Channel, exchange string, username string, gamelogslug routing.GameLog) pubsub.AckType {
	err := pubsub.PublishGob(ch, exchange, routing.GameLogSlug+"."+username, gamelogslug)
	if err != nil {
		return pubsub.NackRequeue
	}
	return pubsub.Ack
}
