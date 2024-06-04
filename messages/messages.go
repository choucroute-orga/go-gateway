package messages

import (
	"gateway/configuration"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

const ()

var logger = logrus.WithFields(logrus.Fields{
	"context": "messages",
})

func New(conf *configuration.Configuration) *amqp.Connection {
	conn, err := amqp.Dial(conf.RabbitURI)
	if err != nil {
		panic(err)
	}
	logger.Info("Connected to RabbitMQ!")
	return conn
}

func GetInventoryShoppingListQueue(conn *amqp.Connection) (*amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		logger.WithError(err).Error("Failed to open a channel")
	}

	q, err := ch.QueueDeclare(
		"inventory-add-recipes-shopping-list", // name
		true,                                  // durable
		false,                                 // delete when unused
		false,                                 // exclusive
		false,                                 // no-wait
		nil,                                   // arguments
	)
	if err != nil {
		logger.WithError(err).Error("Failed to declare a queue")
		return nil, err
	}

	defer ch.Close()

	return &q, nil
}
