package main

/*
	this is a simple wrestic snapshot webhook handler for lagoon

	https://nesv.github.io/golang/2014/02/25/worker-queues-in-go.html
	workerqueues maybe in the event rabbit goes away
*/

import (
	"log"
	"net/http"
	"os"

	"github.com/shreddedbacon/backup-handler/internal/handler"
)

var (
	httpListenPort = os.Getenv("HTTP_LISTEN_PORT")
)

func main() {
	// we want all of these vars, else fail
	if len(os.Getenv("BROKER_ADDRESS")) == 0 {
		log.Fatalln("BROKER_ADDRESS not set")
	}
	if len(os.Getenv("BROKER_PORT")) == 0 {
		log.Fatalln("BROKER_PORT not set")
	}
	if len(os.Getenv("BROKER_USER")) == 0 {
		log.Fatalln("BROKER_USER not set")
	}
	if len(os.Getenv("BROKER_PASS")) == 0 {
		log.Fatalln("BROKER_PASS not set")
	}
	if len(os.Getenv("JWT_SECRET")) == 0 {
		log.Fatalln("JWT_SECRET not set")
	}
	if len(os.Getenv("JWT_AUDIENCE")) == 0 {
		log.Fatalln("JWT_AUDIENCE not set")
	}
	if len(os.Getenv("GRAPHQL_ENDPOINT")) == 0 {
		log.Fatalln("GRAPHQL_ENDPOINT not set")
	}
	if len(os.Getenv("HTTP_LISTEN_PORT")) == 0 {
		httpListenPort = "3000"
	}

	broker := handler.RabbitBroker{
		Hostname:     os.Getenv("BROKER_ADDRESS"),
		Username:     os.Getenv("BROKER_USER"),
		Password:     os.Getenv("BROKER_PASS"),
		Port:         os.Getenv("BROKER_PORT"),
		QueueName:    "lagoon-webhooks:queue",
		ExchangeName: "lagoon-webhooks",
	}
	graphQL := handler.GraphQLEndpoint{
		Endpoint:        os.Getenv("GRAPHQL_ENDPOINT"),
		TokenSigningKey: os.Getenv("JWT_SECRET"),
		JWTAudience:     os.Getenv("JWT_AUDIENCE"),
	}
	backupHandler, err := handler.NewBackupHandler(broker, graphQL)
	if err != nil {
		panic(err)
	}
	// handle the requests
	http.HandleFunc("/", backupHandler.WebhookHandler)
	http.ListenAndServe(":"+httpListenPort, nil)
}
