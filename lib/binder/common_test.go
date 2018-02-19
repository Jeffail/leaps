package binder

import (
	"os"

	"github.com/Jeffail/leaps/lib/util/service/log"
	"github.com/Jeffail/leaps/lib/util/service/metrics"
)

func loggerAndStats() (log.Modular, metrics.Type) {
	logConf := log.NewLoggerConfig()
	logConf.LogLevel = "OFF"

	logger := log.NewLogger(os.Stdout, logConf)
	stats := metrics.DudType{}

	return logger, stats
}
