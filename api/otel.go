package api

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

var initResourcesOnce sync.Once

func initResource() *resource.Resource {
	var r *resource.Resource
	initResourcesOnce.Do(func() {
		extraResources, _ := resource.New(
			context.Background(),
			resource.WithOS(),
			resource.WithProcess(),
			resource.WithContainer(),
			resource.WithHost(),
		)
		r, _ = resource.Merge(
			resource.Default(),
			extraResources,
		)
	})
	return r
}

func initTracerProvider() *trace.TracerProvider {
	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		logrus.Errorf("new otlp trace grpc exporter failed: %v", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(initResource()),
	)
	return tp
}

func initLoggerProvider() *log.LoggerProvider {
	ctx := context.Background()

	exporter, err := otlploghttp.New(ctx)
	if err != nil {
		logrus.Fatalf("new logger provider failed: %v", err)
	}

	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	)
	return lp
}

func initMeterProvider() *metric.MeterProvider {
	ctx := context.Background()

	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		logrus.Fatalf("new otlp metric grpc exporter failed: %v", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(initResource()),
	)
	otel.SetMeterProvider(mp)
	return mp
}

func InitOtel() *trace.TracerProvider {
	tp := initTracerProvider()
	loggerProvider := initLoggerProvider()
	meterProvider := initMeterProvider()

	otel.SetTracerProvider(tp)
	global.SetLoggerProvider(loggerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}
