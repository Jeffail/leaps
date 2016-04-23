package binder

import (
	"os"

	"github.com/jeffail/util/log"
	"github.com/jeffail/util/metrics"
)

func loggerAndStats() (log.Modular, metrics.Aggregator) {
	logConf := log.NewLoggerConfig()
	logConf.LogLevel = "OFF"

	logger := log.NewLogger(os.Stdout, logConf)
	stats := metrics.DudType{}

	return logger, stats
}
