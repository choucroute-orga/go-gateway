package messages

import (
	"encoding/json"
	"gateway/configuration"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

const (
	AddIngredientShoppingList = "inventory-add-ingredient-shopping-list"
	DeadLetterQueueName       = "dead-letter-queue"
)

var logger = logrus.WithFields(logrus.Fields{
	"context": "messages",
})

func New(conf *configuration.Configuration) *amqp.Connection {
	conn, err := amqp.Dial(conf.RabbitURL)
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

func PublishInventoryShoppingListQueue(l *logrus.Entry, conn *amqp.Connection, ingredient IngredientShoppingList) error {

	var err error
	q, ch, err := GetIngredientShoppingListQueue(conn)
	if err != nil {
		return err
	}

	jsonMessage, err := json.Marshal(ingredient)
	if err != nil {
		return err
	}

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonMessage,
		})
	logger.WithFields(logrus.Fields{"message": string(jsonMessage), "queue": q.Name}).Info("Published the Ingredient inventory message")
	return err
}

func GetIngredientShoppingListQueue(conn *amqp.Connection) (*amqp.Queue, *amqp.Channel, error) {
	ch, err := OpenChannel(conn)
	if err != nil {
		logger.WithError(err).Error("Failed to open a channel")
		return nil, nil, err
	}

	q, err := ch.QueueDeclare(
		AddIngredientShoppingList, // name
		true,                      // durable
		false,                     // delete when unused
		false,                     // exclusive
		false,                     // no-wait
		nil,                       // arguments
	)
	if err != nil {
		logger.WithError(err).Error("Failed to declare a queue")
		return nil, nil, err
	}

	return &q, ch, nil
}

func OpenChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		logger.WithError(err).Error("Failed to open a channel")
		return nil, err
	}
	return ch, nil
}
