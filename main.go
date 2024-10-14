package main

import (
	"context"
	"fmt"
	"gateway/api"
	"gateway/configuration"
	"gateway/messages"
	"gateway/validation"

	"github.com/sirupsen/logrus"
)

var logger = logrus.WithFields(logrus.Fields{
	"context": "main",
})

func main() {
	configuration.SetupLogging()
	logger.Info("Choucroute API Gateway Starting...")

	conf := configuration.New()

	val := validation.New(conf)
	r := api.New(val)
	v1 := r.Group(conf.ListenRoute)
	amqp := messages.New(conf)
	h := api.NewApiHandler(amqp, conf)

	h.Register(v1, conf)
	tp := api.InitOtel()
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		cancel()
		if err := tp.Shutdown(ctx); err != nil {
			logger.WithError(err).Error("Error shutting down tracer provider")
		}
		if err := amqp.Close(); err != nil {
			logger.WithError(err).Error("Error closing amqp connection")
		}
	}()

	r.Logger.Fatal(r.Start(fmt.Sprintf("%v:%v", conf.ListenAddress, conf.ListenPort)))

}
