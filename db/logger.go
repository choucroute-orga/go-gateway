package db

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	gormlogger "gorm.io/gorm/logger"
)

type Gormlogger struct {
	log *logrus.Entry
}

func NewGormLogger() Gormlogger {
	return Gormlogger{
		log: logrus.WithFields(logrus.Fields{"module": "gorm"}),
	}
}

// LogMode We don't want to be able to change the log level, we force it in .env
func (g Gormlogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return NewGormLogger()
}

func (g Gormlogger) Info(ctx context.Context, s string, i ...interface{}) {
	g.log.WithContext(ctx).Infof(s, i...)
}

func (g Gormlogger) Warn(ctx context.Context, s string, i ...interface{}) {
	g.log.WithContext(ctx).Warnf(s, i...)
}

func (g Gormlogger) Error(ctx context.Context, s string, i ...interface{}) {
	g.log.WithContext(ctx).Errorf(s, i...)
}

func (g Gormlogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	// Fc function gives an explanation of the trace
	explain, affected := fc()
	l := g.log.WithContext(ctx).
		WithField("startTime", begin.Format("2006-01-02T15:04:05.999Z07:00"))

	if err != nil {
		l = l.WithError(err)
	}

	l.Trace(explain, affected)
}
