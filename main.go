package main

import (
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
	logger.Info("Choucroute API Gateway Starting...")

	conf := configuration.New()

	val := validation.New(conf)
	r := api.New(val)
	v1 := r.Group(conf.ListenRoute)
	amqp := messages.New(conf)
	h := api.NewApiHandler(amqp, conf)

	h.Register(v1, conf)
	r.Logger.Fatal(r.Start(fmt.Sprintf("%v:%v", conf.ListenAddress, conf.ListenPort)))
}
